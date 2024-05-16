package state

import (
	"fmt"

	"github.com/iancoleman/strcase"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	eventMetadataProtoName = protoreflect.FullName("psm.state.v1.EventMetadata")
	stateMetadataProtoName = protoreflect.FullName("psm.state.v1.StateMetadata")
)

type walkingMessage struct {
	message        *protogen.Message
	isEventMessage bool
	isStateMessage bool
	keyField       *protogen.Field
	metadataField  *protogen.Field
	unusedFields   []*protogen.Field
}

type sourceSet struct {
	stateMachines map[string]*StateEntityGenerateSet
}

func WalkFile(file *protogen.File) (map[string]*PSMEntity, error) {
	se := &sourceSet{
		stateMachines: make(map[string]*StateEntityGenerateSet),
	}
	for _, message := range file.Messages {
		if err := se.checkMessage(message); err != nil {
			return nil, err
		}
	}

	stateMachines := make(map[string]*PSMEntity, len(se.stateMachines))
	for _, stateSet := range se.stateMachines {
		converted, err := BuildStateSet(*stateSet)
		if err != nil {
			return nil, err
		}
		stateMachines[stateSet.options.Name] = converted
	}

	return stateMachines, nil
}

func (se *sourceSet) checkMessage(message *protogen.Message) error {

	ww := walkingMessage{
		message: message,
	}

	var keyOptions *psm_pb.PSMOptions
	var keyMessage *protogen.Message

	unusedFields := make([]*protogen.Field, 0, message.Desc.Fields().Len())

	for _, field := range message.Fields {
		if field.Message == nil {
			unusedFields = append(unusedFields, field)
			continue
		}

		if field.Message.Desc.FullName() == stateMetadataProtoName {
			if ww.metadataField != nil {
				return fmt.Errorf("message %s has multiple metadata fields", message.Desc.Name())
			}
			ww.isStateMessage = true
			ww.metadataField = field
			continue
		}

		if field.Message.Desc.FullName() == eventMetadataProtoName {
			if ww.metadataField != nil {
				return fmt.Errorf("message %s has multiple metadata fields", message.Desc.Name())
			}
			ww.isEventMessage = true
			ww.metadataField = field
			continue
		}

		stateObjectAnnotation, ok := proto.GetExtension(field.Message.Desc.Options(), psm_pb.E_Psm).(*psm_pb.PSMOptions)
		if ok && stateObjectAnnotation != nil {
			if keyOptions != nil {
				return fmt.Errorf("message %s has multiple PSM Key fields", message.Desc.Name())
			}
			keyOptions = stateObjectAnnotation
			keyMessage = field.Message
			ww.keyField = field
			continue
		}

		ww.unusedFields = append(unusedFields, field)

	}

	if keyOptions == nil {
		if ww.isStateMessage {
			return fmt.Errorf("message %s does not have a PSM Key field, but has %s", message.Desc.Name(), stateMetadataProtoName)
		}
		if ww.isEventMessage {
			return fmt.Errorf("message %s does not have a PSM Key field, but has %s", message.Desc.Name(), eventMetadataProtoName)
		}
		return nil // This message is not a state machine.
	}

	if ww.metadataField == nil {
		return fmt.Errorf("message %s does not have a metadata field but, but imports the PSM Key Message (%s)", message.Desc.Name(), keyMessage.Desc.FullName())
	}

	stateSet, ok := se.stateMachines[keyOptions.Name]
	if !ok {
		stateSet = NewStateEntityGenerateSet(keyMessage, keyOptions)
		se.stateMachines[keyOptions.Name] = stateSet
	}

	if ww.isStateMessage {
		if err := stateSet.addStateMessage(ww); err != nil {
			return fmt.Errorf("message %s is not a valid PSM State: %w", message.Desc.FullName(), err)
		}
	} else if ww.isEventMessage {
		if err := stateSet.addEventMessage(ww); err != nil {
			return fmt.Errorf("message %s is not a valid PSM Event: %w", message.Desc.FullName(), err)
		}
	} else {
		return fmt.Errorf("message %s does not have a Metadata field, but imports the PSM Key Message (%s)", ww.message.Desc.FullName(), keyMessage.Desc.FullName())
	}
	return nil
}

type StateEntityGenerateSet struct {
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
	dataField     *protogen.Field
}

type stateEntityEvent struct {
	message        *protogen.Message
	keyField       *protogen.Field
	metadataField  *protogen.Field
	eventTypeField *protogen.Field
}

func NewStateEntityGenerateSet(keyMessage *protogen.Message, keyOptions *psm_pb.PSMOptions) *StateEntityGenerateSet {
	return &StateEntityGenerateSet{
		name:       keyOptions.Name,
		fullName:   fmt.Sprintf("%s/%s", keyMessage.Desc.ParentFile().FullName(), keyOptions.Name),
		options:    keyOptions,
		keyMessage: keyMessage,
	}
}

func (stateSet *StateEntityGenerateSet) addStateMessage(ww walkingMessage) error {
	if stateSet.state != nil {
		return fmt.Errorf("duplicate state object")
	}

	entity := &stateEntityState{
		message:       ww.message,
		keyField:      ww.keyField,
		metadataField: ww.metadataField,
	}

	for _, field := range ww.unusedFields {
		switch field.Desc.Name() {
		case "status":
			entity.statusField = field
			if field.Desc.Kind() != protoreflect.EnumKind {
				return fmt.Errorf("status field must be an enum")
			}
		case "data":
			entity.dataField = field
			if field.Desc.Kind() != protoreflect.MessageKind {
				return fmt.Errorf("data field must be a message")
			}
		default:
			return fmt.Errorf("field %s in %s unknown for protostate State entity", field.Desc.Name(), ww.message.Desc.Name())
		}
	}
	if entity.statusField == nil {
		return fmt.Errorf("missing status enum field")
	}
	if entity.dataField == nil {
		return fmt.Errorf("missing data field")
	}
	if entity.dataField.Desc.Kind() != protoreflect.MessageKind {
		return fmt.Errorf("data field must be a message")
	}

	stateSet.state = entity
	return nil
}

func (stateSet *StateEntityGenerateSet) addEventMessage(ww walkingMessage) error {
	if stateSet.event != nil {
		return fmt.Errorf("duplicate event object")
	}

	if len(ww.unusedFields) != 1 {
		return fmt.Errorf("should have exactly three fields, (metadata, keys, event) but has %d", len(ww.message.Fields))
	}
	entity := &stateEntityEvent{
		message:        ww.message,
		keyField:       ww.keyField,
		metadataField:  ww.metadataField,
		eventTypeField: ww.unusedFields[0],
	}

	stateSet.event = entity
	return nil
}

func (src StateEntityGenerateSet) validate() error {
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

func BuildStateSet(src StateEntityGenerateSet) (*PSMEntity, error) {

	if err := src.validate(); err != nil {
		return nil, fmt.Errorf("state object %s: %w", src.options.Name, err)
	}

	namePrefix := strcase.ToCamel(src.options.Name)

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
