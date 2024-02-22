package main

import (
	"fmt"

	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func buildStateSet(src stateEntityGenerateSet) (*PSMEntity, error) {

	if src.stateMessage == nil || src.stateOptions == nil {
		return nil, fmt.Errorf("state object '%s' does not have a State message (message with (psm.state.v1.state).name = '%s')", src.fullName, src.name)
	}

	if src.eventMessage == nil || src.eventOptions == nil {
		return nil, fmt.Errorf("state object '%s' does not have an Event message (message with (psm.state.v1.event).name = '%s')", src.fullName, src.name)
	}

	ss := &PSMEntity{
		stateMessage: src.stateMessage,
		eventMessage: src.eventMessage,
	}

	ss.specifiedName = src.stateOptions.Name
	ss.namePrefix = stateGoName(ss.specifiedName)
	ss.machineName = ss.namePrefix + "PSM"
	ss.eventName = ss.namePrefix + "PSMEvent"

	stateFields, err := buildStateFields(src)
	if err != nil {
		return nil, err
	}

	if src.eventMessage == nil {
		return nil, fmt.Errorf("state object %s does not have an event object", src.stateOptions.Name)
	}

	eventFields, err := buildEventFields(src)
	if err != nil {
		return nil, err
	}

	ss.eventFields = *eventFields
	ss.stateFields = *stateFields

	return ss, nil
}

type stateFieldGenerators struct {
	// In the Status message, this is the enum for the simplified status
	// is always an Enum
	statusField       *protogen.Field
	lastSequenceField *protogen.Field
}

func buildStateFields(src stateEntityGenerateSet) (*stateFieldGenerators, error) {

	out := &stateFieldGenerators{}
	for _, field := range src.stateMessage.Fields {
		switch field.Desc.Name() {
		case "status":
			out.statusField = field
		case "last_event_sequence":
			out.lastSequenceField = field
		}
	}

	if err := out.validate(); err != nil {
		return nil, fmt.Errorf("state object %s: %w", src.stateOptions.Name, err)
	}
	return out, nil

}

func (ss stateFieldGenerators) validate() error {
	if ss.statusField == nil {
		return fmt.Errorf("no 'status' field")
	}

	if ss.statusField.Enum == nil {
		return fmt.Errorf("'status' field is not an enum")
	}

	return nil
}

type eventStateFieldGenMap struct {
	eventField *protogen.Field
	stateField *protogen.Field
	isKey      bool
}

type eventFieldGenerators struct {
	eventStateKeyFields []eventStateFieldGenMap
	eventTypeField      *protogen.Field
	metadataField       *protogen.Field
	metadataFields      metadataGenerators
}

func buildEventFields(src stateEntityGenerateSet) (*eventFieldGenerators, error) {

	out := eventFieldGenerators{}
	for _, field := range src.eventMessage.Fields {
		fieldOpt := proto.GetExtension(field.Desc.Options(), psm_pb.E_EventField).(*psm_pb.EventField)
		if fieldOpt == nil {
			continue
		}

		if fieldOpt.EventType {
			if field.Message == nil {
				return nil, fmt.Errorf("event object event type field is not a message")
			}
			out.eventTypeField = field
		} else if fieldOpt.Metadata {
			if field.Message == nil {
				return nil, fmt.Errorf("event object %s metadata field is not a message", src.eventOptions.Name)
			}
			out.metadataField = field
		} else if fieldOpt.StateKey || fieldOpt.StateField {
			desc := field.Desc
			if desc.IsList() || desc.IsMap() || desc.Kind() == protoreflect.MessageKind {
				return nil, fmt.Errorf("event object %s state key field %s is not a scalar", src.eventOptions.Name, field.Desc.Name())
			}

			matchingStateField := fieldByName(src.stateMessage, field.Desc.Name())
			if matchingStateField == nil {
				return nil, fmt.Errorf("event object %s state key field %s does not exist in state object %s", src.eventOptions.Name, field.Desc.Name(), src.stateOptions.Name)
			}

			if err := descriptorTypesMatch(matchingStateField.Desc, field.Desc); err != nil {
				return nil, fmt.Errorf("event object %s state key field %s is not the same type as state object %s: %w", src.eventOptions.Name, field.Desc.Name(), src.stateOptions.Name, err)
			}

			out.eventStateKeyFields = append(out.eventStateKeyFields, eventStateFieldGenMap{
				eventField: field,
				stateField: matchingStateField,
				isKey:      fieldOpt.StateKey,
			})

		}

	}

	if out.metadataField == nil {
		return nil, fmt.Errorf("event object '%s' missing metadata field", src.eventOptions.Name)
	}

	mdFields := metadataFields(out.metadataField.Message.Desc)

	mdMsg := out.metadataField.Message
	out.metadataFields = mdFields.toGenerators(mdMsg)

	if err := out.validate(); err != nil {
		return nil, fmt.Errorf("event object %s: %w", src.eventOptions.Name, err)
	}

	return &out, nil
}

func (desc eventFieldGenerators) validate() error {

	if desc.metadataFields.id == nil {
		return fmt.Errorf("event object missing id field")
	}

	if desc.metadataFields.timestamp == nil {
		return fmt.Errorf("event object missing timestamp field")
	}

	// the oneof wrapper
	if desc.eventTypeField == nil {
		return fmt.Errorf("event object missing eventType field")
	}

	if desc.eventTypeField.Message == nil {
		return fmt.Errorf("event object event type field is not a message")
	}

	return nil
}
