package main

import (
	"fmt"

	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/protostate/pquery"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func buildQuerySet(qs queryServiceGenerateSet) (*PSMQuerySet, error) {

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

	if qs.getMethod == nil {
		return nil, fmt.Errorf("service %s does not have a get method", qs.name)
	}

	if qs.listMethod == nil {
		return nil, fmt.Errorf("service %s does not have a list method", qs.name)
	}

	if qs.listEventsMethod == nil {
		return nil, fmt.Errorf("service %s does not have a list events method", qs.name)
	}

	var statePkFields []string
	if ss != nil {
		statePkFields = ss.statePkFields
	}

	listReflectionSet, err := pquery.BuildListReflection(qs.listMethod.Input.Desc, qs.listMethod.Output.Desc, pquery.WithTieBreakerFields(statePkFields...))
	if err != nil {
		return nil, fmt.Errorf("query service %s is not compatible with PSM: %w", qs.listMethod.Desc.FullName(), err)
	}

	goServiceName := stateGoName(qs.name)

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

	if qs.listEventsMethod != nil {
		var fallbackPkFields []string
		if ss != nil {
			fallbackPkFields = ss.eventPkFields
		}
		listEventsReflectionSet, err := pquery.BuildListReflection(qs.listEventsMethod.Input.Desc, qs.listEventsMethod.Output.Desc, pquery.WithTieBreakerFields(fallbackPkFields...))
		if err != nil {
			return nil, fmt.Errorf("query service %s is not compatible with PSM: %w", qs.listMethod.Desc.FullName(), err)
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
	}
	return ww, nil
}

type queryPkFields struct {
	statePkFields []string
	eventPkFields []string
}

func deriveQueryPKFromDescriptors(stateMessage, eventMessage protoreflect.MessageDescriptor) (*queryPkFields, error) {

	// this function mirrors builfEventFieldDescriptors, but uses only the
	// descriptors, as the messages will likely not be in the same file as the
	// service, i.e. we won't have the protogen wrappers.
	out := &queryPkFields{}

	var metadataField protoreflect.FieldDescriptor

	for idx := 0; idx < eventMessage.Fields().Len(); idx++ {
		field := eventMessage.Fields().Get(idx)
		fieldOpt := proto.GetExtension(field.Options(), psm_pb.E_EventField).(*psm_pb.EventField)
		if fieldOpt == nil {
			continue
		}

		if fieldOpt.Metadata {
			if metadataField != nil {
				return nil, fmt.Errorf("event message %s has more than one metadata field", eventMessage.Name())
			}
			metadataField = field
			continue
		}

		if fieldOpt.StateKey || fieldOpt.StateField {

			matchingStateField := eventMessage.Fields().ByName(field.Name())
			if matchingStateField == nil {
				return nil, fmt.Errorf("event field %s does not exist in event message", field.Name())
			}

			if err := descriptorTypesMatch(field, matchingStateField); err != nil {
				return nil, fmt.Errorf("event field %s does not match event field: %w", field.Name(), err)
			}

			out.statePkFields = append(out.statePkFields, string(field.Name()))
		}
	}

	if metadataField == nil {
		return nil, fmt.Errorf("event message %s does not have a metadata field", eventMessage.Name())
	}
	if metadataField.Kind() != protoreflect.MessageKind {
		return nil, fmt.Errorf("metadata field %s is not a message", metadataField.Name())
	}

	mdFields := metadataFields(metadataField.Message())

	if mdFields.id == nil {
		return nil, fmt.Errorf("metadata message %s does not have an id field", metadataField.Message().Name())
	}

	out.eventPkFields = []string{
		fmt.Sprintf("%s.%s", metadataField.Name(), mdFields.id.Name()),
	}

	return out, nil
}

// attempts to walk through the query methods to find the descriptors for the
// state and event messages.
func deriveStateDescriptorFromQueryDescriptor(src queryServiceGenerateSet) (*queryPkFields, error) {

	if src.getMethod == nil {
		return nil, fmt.Errorf("no get nethod, cannot derive state fields")
	}

	var repeatedEventMessage *protogen.Message
	var stateMessage *protogen.Message
	var eventMessage *protogen.Message
	for _, field := range src.getMethod.Output.Fields {
		if field.Message == nil {
			continue
		}
		if field.Desc.Cardinality() == protoreflect.Repeated {
			if repeatedEventMessage != nil {
				return nil, fmt.Errorf("state get response %s should have exactly one repeated field", src.getMethod.Desc.FullName())
			}
			repeatedEventMessage = field.Message
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

	stateOptions := proto.GetExtension(stateMessage.Desc.Options(), psm_pb.E_State).(*psm_pb.StateObjectOptions)
	if stateOptions == nil {
		// No state, can't add fallbacks.
		return nil, nil
	}

	// derive from the event list in the get response
	if repeatedEventMessage != nil {
		eventOptions := proto.GetExtension(repeatedEventMessage.Desc.Options(), psm_pb.E_Event).(*psm_pb.EventObjectOptions)
		if eventOptions == nil {
			return nil, nil
		}

		eventMessage = repeatedEventMessage

		return deriveQueryPKFromDescriptors(stateMessage.Desc, eventMessage.Desc)
	}

	// derive from the list response
	if src.listEventsMethod != nil {

		for _, field := range src.listEventsMethod.Output.Fields {
			if field.Message == nil {
				continue
			}
			if field.Desc.Cardinality() != protoreflect.Repeated {
				continue
			}
			repeatedEventMessage = field.Message
			break
		}
		if repeatedEventMessage == nil {
			return nil, nil
		}
		eventOptions := proto.GetExtension(repeatedEventMessage.Desc.Options(), psm_pb.E_Event).(*psm_pb.EventObjectOptions)
		if eventOptions == nil {
			return nil, nil
		}

		return deriveQueryPKFromDescriptors(stateMessage.Desc, repeatedEventMessage.Desc)
	}

	// Last ditch effort, try to find the event message in the file
	stateFileMessages := stateMessage.Desc.ParentFile().Messages()
	for i := 0; i < stateFileMessages.Len(); i++ {
		msg := stateFileMessages.Get(i)
		eventOptions := proto.GetExtension(msg.Options(), psm_pb.E_Event).(*psm_pb.EventObjectOptions)
		if eventOptions == nil {
			continue
		}
		if eventOptions.Name != stateOptions.Name {
			continue
		}

		return deriveQueryPKFromDescriptors(stateMessage.Desc, msg)
	}

	return nil, nil
}
