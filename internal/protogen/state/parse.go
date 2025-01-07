package state

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pentops/j5/gen/j5/ext/v1/ext_j5pb"
	"github.com/pentops/protostate/psm"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	eventMetadataProtoName = protoreflect.FullName("j5.state.v1.EventMetadata")
	stateMetadataProtoName = protoreflect.FullName("j5.state.v1.StateMetadata")
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
	inOrder       []*StateEntityGenerateSet
}

func WalkFile(file *protogen.File) ([]*PSMEntity, error) {
	se := &sourceSet{
		stateMachines: make(map[string]*StateEntityGenerateSet),
	}
	for _, message := range file.Messages {
		if err := se.checkMessage(message); err != nil {
			return nil, err
		}
	}

	stateMachines := make([]*PSMEntity, 0, len(se.stateMachines))
	for _, stateSet := range se.inOrder {
		converted, err := BuildStateSet(*stateSet)
		if err != nil {
			return nil, err
		}
		stateMachines = append(stateMachines, converted)
	}

	return stateMachines, nil
}

func (se *sourceSet) checkMessage(message *protogen.Message) error {

	ww := walkingMessage{
		message: message,
	}

	var keyOptions *ext_j5pb.PSMOptions
	var keyMessage *protogen.Message

	for _, field := range message.Fields {
		if field.Message == nil {
			ww.unusedFields = append(ww.unusedFields, field)
			continue
		}

		if field.Message.Desc.FullName() == stateMetadataProtoName {
			if ww.metadataField != nil {
				return fmt.Errorf("message %s has multiple metadata fields", message.Desc.FullName())
			}
			ww.isStateMessage = true
			ww.metadataField = field
			continue
		}

		if field.Message.Desc.FullName() == eventMetadataProtoName {
			if ww.metadataField != nil {
				return fmt.Errorf("message %s has multiple metadata fields", message.Desc.FullName())
			}
			ww.isEventMessage = true
			ww.metadataField = field
			continue
		}

		stateObjectAnnotation := proto.GetExtension(field.Message.Desc.Options(), ext_j5pb.E_Psm).(*ext_j5pb.PSMOptions)
		if stateObjectAnnotation != nil && strings.HasSuffix(string(field.Message.Desc.Name()), "Keys") {
			if keyOptions != nil {
				return fmt.Errorf("message has multiple PSM Key fields %s and %s", field.Desc.FullName(), ww.keyField.Desc.FullName())
			}
			keyOptions = stateObjectAnnotation
			keyMessage = field.Message
			ww.keyField = field
			continue
		}

		ww.unusedFields = append(ww.unusedFields, field)

	}

	if ww.metadataField == nil {
		return nil // This message is not a state machine.
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

	stateSet, ok := se.stateMachines[keyOptions.EntityName]
	if !ok {
		stateSet = NewStateEntityGenerateSet(keyMessage, keyOptions)
		se.stateMachines[keyOptions.EntityName] = stateSet
		se.inOrder = append(se.inOrder, stateSet)
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

	options    *ext_j5pb.PSMOptions
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

func NewStateEntityGenerateSet(keyMessage *protogen.Message, keyOptions *ext_j5pb.PSMOptions) *StateEntityGenerateSet {
	return &StateEntityGenerateSet{
		name:       keyOptions.EntityName,
		fullName:   fmt.Sprintf("%s/%s", keyMessage.Desc.ParentFile().FullName(), keyOptions.EntityName),
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

	fieldNames := make([]string, 0, len(ww.unusedFields))
	for _, field := range ww.unusedFields {
		fieldNames = append(fieldNames, string(field.Desc.Name()))
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
		return fmt.Errorf("missing data field, have %s", strings.Join(fieldNames, ", "))
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
		fieldNames := make([]string, 0, len(ww.unusedFields))
		for _, field := range ww.unusedFields {
			fieldNames = append(fieldNames, string(field.Desc.Name()))
		}
		return fmt.Errorf("should have exactly three fields, (metadata, keys, event) but has (%s)", strings.Join(fieldNames, ", "))
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
		return nil, fmt.Errorf("state object %s: %w", src.options.EntityName, err)
	}

	namePrefix := strcase.ToCamel(src.options.EntityName)

	ss := &PSMEntity{
		state:         src.state,
		event:         src.event,
		specifiedName: src.options.EntityName,
		namePrefix:    namePrefix,
		machineName:   namePrefix + "PSM",
		eventName:     namePrefix + "PSMEvent",
		keyMessage:    src.keyMessage,
	}

	spec, err := psm.BuildQueryTableSpec(src.state.message.Desc, src.event.message.Desc)
	if err != nil {
		return nil, err
	}

	ss.tableMap = &spec.TableMap

	return ss, nil
}
