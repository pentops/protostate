package query

import (
	"errors"
	"fmt"

	"github.com/iancoleman/strcase"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/protostate/pquery"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type QueryServiceSourceSet struct {
	QuerySets map[string]*QueryServiceGenerateSet
}

func WalkFile(file *protogen.File) (map[string]*PSMQuerySet, error) {
	qss := &QueryServiceSourceSet{
		QuerySets: map[string]*QueryServiceGenerateSet{},
	}

	for _, service := range file.Services {
		stateQueryAnnotation := proto.GetExtension(service.Desc.Options(), psm_pb.E_StateQuery).(*psm_pb.StateQueryServiceOptions)

		for _, method := range service.Methods {
			methodOpt := proto.GetExtension(method.Desc.Options(), psm_pb.E_StateQueryMethod).(*psm_pb.StateQueryMethodOptions)
			if methodOpt == nil {
				continue
			}
			if methodOpt.Name == "" {
				if stateQueryAnnotation == nil || stateQueryAnnotation.Name == "" {
					return nil, fmt.Errorf("service %s method %s does not have a state query name, and no service default", service.GoName, method.GoName)
				}
				methodOpt.Name = stateQueryAnnotation.Name
			}

			methodSet, ok := qss.QuerySets[methodOpt.Name]
			if !ok {
				methodSet = NewQueryServiceGenerateSet(methodOpt.Name, service.Desc.FullName())
				qss.QuerySets[methodOpt.Name] = methodSet
			}
			if err := methodSet.AddMethod(method, methodOpt); err != nil {
				return nil, fmt.Errorf("adding method %s to %s: %w", method.Desc.Name(), service.Desc.FullName(), err)
			}
		}
	}

	out := map[string]*PSMQuerySet{}
	for _, qs := range qss.QuerySets {
		converted, err := BuildQuerySet(*qs)
		if err != nil {
			return nil, err
		}
		out[qs.name] = converted
	}

	return out, nil
}

type QueryServiceGenerateSet struct {

	// name of the state machine
	name string
	// for errors / debugging, includes the proto source name
	fullName string

	getMethod        *protogen.Method
	listMethod       *protogen.Method
	listEventsMethod *protogen.Method
}

func NewQueryServiceGenerateSet(name string, serviceFullName protoreflect.FullName) *QueryServiceGenerateSet {
	return &QueryServiceGenerateSet{
		name:     name,
		fullName: fmt.Sprintf("%s/%s", serviceFullName, name),
	}
}

func (qs *QueryServiceGenerateSet) AddMethod(method *protogen.Method, methodOpt *psm_pb.StateQueryMethodOptions) error {

	if methodOpt.Get {
		if qs.getMethod != nil {
			return fmt.Errorf("service %s already has a get method (%s)", qs.name, qs.getMethod.Desc.Name())
		}
		qs.getMethod = method
	} else if methodOpt.List {
		if qs.listMethod != nil {
			return fmt.Errorf("service %s already has a list method (%s)", qs.name, qs.listMethod.Desc.Name())
		}
		qs.listMethod = method
	} else if methodOpt.ListEvents {
		if qs.listEventsMethod != nil {
			return fmt.Errorf("service %s already has a list events method (%s)", qs.name, qs.listEventsMethod.Desc.Name())
		}

		qs.listEventsMethod = method

	} else {
		return fmt.Errorf("method does not have a state query type")
	}

	return nil

}

func (qs QueryServiceGenerateSet) validate() error {
	if qs.getMethod == nil {
		return fmt.Errorf("PSM Query '%s' does not have a get method", qs.fullName)
	}

	if qs.listMethod == nil {
		return fmt.Errorf("PSM Qurey '%s' does not have a list method", qs.fullName)
	}

	return nil

}

func BuildQuerySet(qs QueryServiceGenerateSet) (*PSMQuerySet, error) {
	if err := qs.validate(); err != nil {
		return nil, err
	}

	// this walks the proto to get some of the same data as the state set,
	// however it is unlkely to duplicate work, as the states are usually
	// defined in a separate file from the query service
	ss, err := deriveStateDescriptorFromQueryDescriptor(qs)
	if err != nil {
		return nil, err
	}
	if ss == nil {
		return nil, fmt.Errorf("query service %s does not have a state descriptor", qs.name)
	}

	var errs []error
	if qs.getMethod == nil {
		errs = append(errs, fmt.Errorf("service %s does not have a get method", qs.name))
	}

	if qs.listMethod == nil {
		errs = append(errs, fmt.Errorf("service %s does not have a list method", qs.name))
	}

	if qs.listEventsMethod == nil {
		errs = append(errs, fmt.Errorf("service %s does not have a list events method", qs.name))
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	var statePkFields []string
	if ss != nil {
		statePkFields = ss.statePkFields
	}

	listReflectionSet, err := pquery.BuildListReflection(qs.listMethod.Input.Desc, qs.listMethod.Output.Desc, pquery.WithTieBreakerFields(statePkFields...))
	if err != nil {
		return nil, fmt.Errorf("pquery.BuildListReflection for %s: %w", qs.listMethod.Desc.FullName(), err)
	}

	goServiceName := strcase.ToCamel(qs.name)

	ww := &PSMQuerySet{
		GoServiceName: goServiceName,
		GetREQ:        qs.getMethod.Input.GoIdent,
		GetRES:        qs.getMethod.Output.GoIdent,
		ListREQ:       qs.listMethod.Input.GoIdent,
		ListRES:       qs.listMethod.Output.GoIdent,
	}

	for _, field := range listReflectionSet.RequestFilterFields {
		genField := mapGenField(qs.listMethod.Input, field)
		ww.ListRequestFilter = append(ww.ListRequestFilter, ListFilterField{
			DBName:   string(field.Name()),
			Getter:   genField.GoName,
			Optional: field.HasOptionalKeyword(),
		})
	}

	var fallbackPkFields []string
	if ss != nil {
		fallbackPkFields = ss.eventPkFields
	}

	listEventsReflectionSet, err := pquery.BuildListReflection(qs.listEventsMethod.Input.Desc, qs.listEventsMethod.Output.Desc, pquery.WithTieBreakerFields(fallbackPkFields...))
	if err != nil {
		return nil, fmt.Errorf("pquery.BuildListReflection for %s is not compatible with PSM: %w", qs.listEventsMethod.Desc.FullName(), err)
	}

	ww.ListEventsREQ = &qs.listEventsMethod.Input.GoIdent
	ww.ListEventsRES = &qs.listEventsMethod.Output.GoIdent
	for _, field := range listEventsReflectionSet.RequestFilterFields {
		genField := mapGenField(qs.listEventsMethod.Input, field)
		ww.ListEventsRequestFilter = append(ww.ListEventsRequestFilter, ListFilterField{
			DBName:   string(field.Name()),
			Getter:   genField.GoName,
			Optional: field.HasOptionalKeyword(),
		})
	}

	return ww, nil
}

type queryPkFields struct {
	statePkFields []string
	eventPkFields []string
}

// attempts to walk through the query methods to find the descriptors for the
// state and event messages.
func deriveStateDescriptorFromQueryDescriptor(src QueryServiceGenerateSet) (*queryPkFields, error) {
	if src.getMethod == nil {
		return nil, fmt.Errorf("no get nethod, cannot derive state fields")
	}

	var eventMessage *protogen.Message
	var stateMessage *protogen.Message
	//var eventMessage *protogen.Message
	for _, field := range src.getMethod.Output.Fields {
		if field.Message == nil {
			continue
		}
		if field.Desc.Cardinality() == protoreflect.Repeated {
			if eventMessage != nil {
				return nil, fmt.Errorf("state get response %s should have exactly one repeated field", src.getMethod.Desc.FullName())
			}
			eventMessage = field.Message
			continue
		}

		if stateMessage != nil {
			return nil, fmt.Errorf("state get response %s should have exactly one field", src.getMethod.Desc.FullName())
		}
		stateMessage = field.Message
	}

	if stateMessage == nil {
		return nil, fmt.Errorf("state get response should have exactly one non-repeated field, which is a message")
	}

	if eventMessage == nil {
		if src.listEventsMethod == nil {
			return nil, fmt.Errorf("no repeated field in get response, and no list events method, cannot derive event")
		}
		for _, field := range src.listEventsMethod.Output.Fields {
			if field.Message == nil {
				continue
			}
			if field.Desc.Cardinality() == protoreflect.Repeated {
				if eventMessage != nil {
					return nil, fmt.Errorf("state get response %s should have exactly one repeated field", src.getMethod.Desc.FullName())
				}
				eventMessage = field.Message
				continue
			}
		}
	}

	var stateMetadataField *protogen.Field
	var stateKeyField *protogen.Field
	var keyMessage *protogen.Message
	var keyOptions *psm_pb.PSMOptions

	for _, field := range stateMessage.Fields {
		if field.Message == nil {
			continue
		}
		if field.Message.Desc.FullName() == stateMetadataProtoName {
			stateMetadataField = field
			continue
		}
		stateObjectAnnotation, ok := proto.GetExtension(field.Message.Desc.Options(), psm_pb.E_Psm).(*psm_pb.PSMOptions)
		if ok && stateObjectAnnotation != nil {
			keyOptions = stateObjectAnnotation
			keyMessage = field.Message
			stateKeyField = field
			continue
		}
	}

	if stateMetadataField == nil {
		return nil, fmt.Errorf("state message %s has no %s field", stateMessage.Desc.FullName(), stateMetadataProtoName)
	}
	if stateKeyField == nil {
		return nil, fmt.Errorf("state message %s has no PSM Keys", stateMessage.Desc.FullName())
	}
	if keyOptions.Name != src.name {
		return nil, fmt.Errorf("keys message %s has a different name than the query service %s", keyMessage.Desc.FullName(), src.name)
	}

	if eventMessage == nil {
		// No event, can't add fallbacks.
		return nil, fmt.Errorf("no event message for %s, cannot derive event fields", keyMessage.Desc.FullName())
	}

	var eventMetadataField *protogen.Field
	var eventKeysField *protogen.Field
	for _, field := range eventMessage.Fields {
		if field.Message == nil {
			continue
		}
		if field.Message.Desc.FullName() == eventMetadataProtoName {
			eventMetadataField = field
			continue
		}

		stateObjectAnnotation, ok := proto.GetExtension(field.Message.Desc.Options(), psm_pb.E_Psm).(*psm_pb.PSMOptions)
		if ok && stateObjectAnnotation != nil {
			if keyMessage.Desc.FullName() != field.Message.Desc.FullName() {
				return nil, fmt.Errorf("%s.%s is a %s, but %s.%s is a %s, these should be the same",
					stateMessage.Desc.FullName(),
					stateKeyField.Desc.Name(),
					keyMessage.Desc.FullName(),
					eventMessage.Desc.FullName(),
					field.Desc.Name(),
					field.Message.Desc.FullName(),
				)
			}
			eventKeysField = field
			continue
		}
	}

	if eventMetadataField == nil {
		// No event, can't add fallbacks.
		return nil, fmt.Errorf("event message %s has no %s field", eventMessage.Desc.FullName(), eventMetadataProtoName)
	}
	if eventKeysField == nil {
		return nil, fmt.Errorf("event message %s has no PSM Keys", eventMessage.Desc.FullName())
	}

	// this function mirrors builfEventFieldDescriptors, but uses only the
	// descriptors, as the messages will likely not be in the same file as the
	// service, i.e. we won't have the protogen wrappers.
	out := &queryPkFields{}

	out.eventPkFields = []string{
		fmt.Sprintf("%s.event_id", eventMetadataField.Desc.Name()),
	}

	for _, field := range keyMessage.Fields {
		out.statePkFields = append(out.statePkFields, fmt.Sprintf("%s.%s", string(stateKeyField.Desc.Name()), string(field.Desc.Name())))
	}

	return out, nil
}

func mapGenField(parent *protogen.Message, field protoreflect.FieldDescriptor) *protogen.Field {
	if field == nil {
		return nil
	}
	for _, f := range parent.Fields {
		if f.Desc.FullName() == field.FullName() {
			return f
		}
	}
	panic(fmt.Sprintf("field %s not found in parent %s", field.FullName(), parent.Desc.FullName()))
}
