package main

import (
	"fmt"

	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/protobuf/proto"
)

func (src stateEntityGenerateSet) validate() error {
	if src.state == nil {
		return fmt.Errorf("missing State message (field of %s)", stateMetadataProtoName)
	}

	if src.event == nil {
		return fmt.Errorf("missing Event message (field of %s)", eventMetadataProtoName)
	}

	if src.keyMessage == nil {
		return fmt.Errorf("missing Key message (field where a message has the state annotation)")
	}

	return nil

}

func buildStateSet(src stateEntityGenerateSet) (*PSMEntity, error) {

	if err := src.validate(); err != nil {
		return nil, fmt.Errorf("state object %s: %w", src.options.Name, err)
	}

	namePrefix := stateGoName(src.options.Name)
	ss := &PSMEntity{
		state:         src.state,
		event:         src.event,
		specifiedName: src.options.Name,
		namePrefix:    namePrefix,
		machineName:   namePrefix + "PSM",
		eventName:     namePrefix + "PSMEvent",
		keyMessage:    src.keyMessage,
		keyFields:     make([]keyField, 0, len(src.keyMessage.Fields)),
	}

	for _, field := range src.keyMessage.Fields {

		annotation := proto.GetExtension(field.Desc.Options(), psm_pb.E_Field).(*psm_pb.FieldOptions)
		if annotation == nil {
			annotation = &psm_pb.FieldOptions{}
		}

		keyField := keyField{
			field:        field,
			FieldOptions: annotation,
		}
		ss.keyFields = append(ss.keyFields, keyField)
	}

	return ss, nil
}

/*


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
		return nil, fmt.Errorf("state object %s: %w", src.options.Name, err)
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
	eventTypeField *protogen.Field
	metadataField  *protogen.Field
	keyField       *protogen.Field
	metadataFields metadataGenerators
}

func buildEventFields(src stateEntityGenerateSet) (*eventFieldGenerators, error) {

	// The event should have exactly three fields: Metadata, Keys and the
	// specific Type of event
	out := eventFieldGenerators{}
	for _, field := range src.eventMessage.Fields {

		if fieldOpt.EventType {
			if field.Message == nil {
				return nil, fmt.Errorf("event object event type field is not a message")
			}
			out.eventTypeField = field
		} else if fieldOpt.Metadata {
			if field.Message == nil {
				return nil, fmt.Errorf("event object %s metadata field is not a message", src.name)
			}
			out.metadataField = field
		} else if fieldOpt.StateKey || fieldOpt.StateField {
			desc := field.Desc
			if desc.IsList() || desc.IsMap() || desc.Kind() == protoreflect.MessageKind {
				return nil, fmt.Errorf("event object %s state key field %s is not a scalar", src.name, field.Desc.Name())
			}

			matchingStateField := fieldByName(src.stateMessage, field.Desc.Name())
			if matchingStateField == nil {
				return nil, fmt.Errorf("event object %s state key field %s does not exist in state object %s", src.name, field.Desc.Name(), src.name)
			}

			if err := descriptorTypesMatch(matchingStateField.Desc, field.Desc); err != nil {
				return nil, fmt.Errorf("event object %s state key field %s is not the same type as state object %s: %w", src.name, field.Desc.Name(), src.name, err)
			}

			out.eventStateKeyFields = append(out.eventStateKeyFields, eventStateFieldGenMap{
				eventField: field,
				stateField: matchingStateField,
				isKey:      fieldOpt.StateKey,
			})

		}

	}

	if out.metadataField == nil {
		return nil, fmt.Errorf("event object '%s' missing metadata field", src.name)
	}

	mdFields := metadataFields(out.metadataField.Message.Desc)

	mdMsg := out.metadataField.Message
	out.metadataFields = mdFields.toGenerators(mdMsg)

	if err := out.validate(); err != nil {
		return nil, fmt.Errorf("event object %s: %w", src.name, err)
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

*/
