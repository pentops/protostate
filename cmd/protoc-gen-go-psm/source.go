package main

import (
	"fmt"

	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type stateEntityGenerateSet struct {
	// name of the state machine
	name string
	// for errors / debugging, includes the proto source name
	fullName string

	options    *psm_pb.PSMOptions
	keyMessage *protogen.Message

	state *stateEntityState
	event *stateEntityEvent
}

type stateEntityState struct {
	message       *protogen.Message
	keyField      *protogen.Field
	metadataField *protogen.Field
	statusField   *protogen.Field
}

type stateEntityEvent struct {
	message        *protogen.Message
	keyField       *protogen.Field
	metadataField  *protogen.Field
	eventTypeField *protogen.Field
}

// generateSet contains the protogen wrapper around the descriptors
type queryServiceGenerateSet struct {

	// name of the state machine
	name string
	// for errors / debugging, includes the proto source name
	fullName string

	getMethod        *protogen.Method
	listMethod       *protogen.Method
	listEventsMethod *protogen.Method
}

func (qs queryServiceGenerateSet) validate() error {
	if qs.getMethod == nil {
		return fmt.Errorf("PSM Query '%s' does not have a get method", qs.fullName)
	}

	if qs.listMethod == nil {
		return fmt.Errorf("PSM Qurey '%s' does not have a list method", qs.fullName)
	}

	return nil

}

type mappedSourceFile struct {
	stateSets map[string]*stateEntityGenerateSet
	querySets map[string]*queryServiceGenerateSet
}

func mapSourceFile(file *protogen.File) (*mappedSourceFile, error) {
	source := &mappedSourceFile{
		stateSets: map[string]*stateEntityGenerateSet{},
		querySets: map[string]*queryServiceGenerateSet{},
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

			methodSet, ok := source.querySets[methodOpt.Name]
			if !ok {
				methodSet = &queryServiceGenerateSet{
					name:     methodOpt.Name,
					fullName: fmt.Sprintf("%s/%s", service.Desc.FullName(), methodOpt.Name),
				}
				source.querySets[methodOpt.Name] = methodSet
			}

			if methodOpt.Get {
				methodSet.getMethod = method
			} else if methodOpt.List {
				methodSet.listMethod = method
			} else if methodOpt.ListEvents {
				methodSet.listEventsMethod = method
			} else {
				return nil, fmt.Errorf("service %s method %s does not have a state query type", service.GoName, method.GoName)
			}
		}

	}

	for _, message := range file.Messages {

		isEventMessage := false
		isStateMessage := false
		var keyMessage *protogen.Message
		var keyField *protogen.Field
		var keyOptions *psm_pb.PSMOptions
		var metadataField *protogen.Field

		unusedFields := make([]*protogen.Field, 0, message.Desc.Fields().Len())

		for _, field := range message.Fields {
			if field.Message == nil {
				unusedFields = append(unusedFields, field)
				continue
			}

			if field.Message.Desc.FullName() == stateMetadataProtoName {
				isStateMessage = true
				metadataField = field
				continue
			}

			if field.Message.Desc.FullName() == eventMetadataProtoName {
				isEventMessage = true
				metadataField = field
				continue
			}

			stateObjectAnnotation, ok := proto.GetExtension(field.Message.Desc.Options(), psm_pb.E_Psm).(*psm_pb.PSMOptions)
			if ok && stateObjectAnnotation != nil {
				keyOptions = stateObjectAnnotation
				keyMessage = field.Message
				keyField = field
				continue
			}

			unusedFields = append(unusedFields, field)

		}

		if keyOptions != nil {
			if metadataField == nil {
				return nil, fmt.Errorf("message %s does not have a metadata field but field '%s' is a PSM Keys message", message.Desc.Name(), keyField.Desc.Name())
			}
		}

		if keyOptions == nil {
			if isStateMessage || isEventMessage {
				return nil, fmt.Errorf("state object %s does not have a PSM annotation in a Keys message", message.Desc.Name())
			}
			continue
		}

		stateSet, ok := source.stateSets[keyOptions.Name]
		if !ok {
			stateSet = &stateEntityGenerateSet{
				name:       keyOptions.Name,
				fullName:   fmt.Sprintf("%s/%s", message.Desc.ParentFile().FullName(), keyOptions.Name),
				options:    keyOptions,
				keyMessage: keyMessage,
			}
			source.stateSets[keyOptions.Name] = stateSet
		}

		if isStateMessage {
			if stateSet.state != nil {
				return nil, fmt.Errorf("duplicate state object name %s", keyOptions.Name)
			}

			entity := &stateEntityState{
				message:       message,
				keyField:      keyField,
				metadataField: metadataField,
			}

			for _, field := range message.Fields {
				if field.Desc.Kind() == protoreflect.EnumKind && field.Desc.Name() == "status" {
					entity.statusField = field
				}
			}
			if entity.statusField == nil {
				return nil, fmt.Errorf("state object %s does not have a status enum field", message.Desc.Name())
			}

			stateSet.state = entity
		} else if isEventMessage {
			if stateSet.event != nil {
				return nil, fmt.Errorf("duplicate event object name %s", keyOptions.Name)
			}
			entity := &stateEntityEvent{
				message:       message,
				keyField:      keyField,
				metadataField: metadataField,
				// REFACTOR: eventTypeField:
			}

			if len(unusedFields) != 1 {
				return nil, fmt.Errorf("event object %s should have exactly three fields, but has %d", message.Desc.Name(), len(unusedFields))
			}
			entity.eventTypeField = unusedFields[0]

			stateSet.event = entity
		} else {
			return nil, fmt.Errorf("message %s does not have a Metadata field, but imports the PSM Key Message (%s)", message.Desc.Name(), keyMessage.Desc.FullName())
		}
	}

	return source, nil
}
