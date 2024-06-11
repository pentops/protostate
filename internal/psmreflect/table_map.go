package psmreflect

import (
	"fmt"

	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/protostate/internal/pgstore"
	"github.com/pentops/protostate/psm"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type StateMachine struct {
	EventMetadataField protoreflect.FieldDescriptor
	KeyMessage         protoreflect.MessageDescriptor
}

func BuildDefaultTableMap(ss StateMachine) (*psm.TableMap, error) {

	stateObjectAnnotation, ok := proto.GetExtension(ss.KeyMessage.Options(), psm_pb.E_Psm).(*psm_pb.PSMOptions)
	if !ok || stateObjectAnnotation == nil {
		return nil, fmt.Errorf("message %s has no PSM Key field", ss.KeyMessage.Name())
	}

	tm := &psm.TableMap{
		State: psm.StateTableSpec{
			TableName: stateObjectAnnotation.Name,
			Root:      &pgstore.ProtoFieldSpec{ColumnName: "state", PathFromRoot: pgstore.ProtoPathSpec{}},
		},
		Event: psm.EventTableSpec{
			TableName: stateObjectAnnotation.Name + "_event",
			Root: &pgstore.ProtoFieldSpec{
				ColumnName:   "data",
				PathFromRoot: pgstore.ProtoPathSpec{},
			},
			ID: &pgstore.ProtoFieldSpec{
				ColumnName:   "id",
				PathFromRoot: pgstore.ProtoPathSpec{string(ss.EventMetadataField.Name()), "event_id"},
			},
			Timestamp: &pgstore.ProtoFieldSpec{
				ColumnName:   "timestamp",
				PathFromRoot: pgstore.ProtoPathSpec{string(ss.EventMetadataField.Name()), "timestamp"},
			},
			Sequence: &pgstore.ProtoFieldSpec{
				ColumnName:   "sequence",
				PathFromRoot: pgstore.ProtoPathSpec{string(ss.EventMetadataField.Name()), "sequence"},
			},
			StateSnapshot: &pgstore.ProtoFieldSpec{
				ColumnName: "state",
			},
		},
		KeyColumns: []psm.KeyColumn{{
			ColumnName: "foo_id",
			ProtoName:  protoreflect.Name("foo_id"),
			Primary:    true,
			Required:   true,
		}, {
			ColumnName: "tenant_id",
			ProtoName:  protoreflect.Name("tenant_id"),
			Primary:    false,
			Required:   false,
		}},
	}

	fields := ss.KeyMessage.Fields()
	tm.KeyColumns = make([]psm.KeyColumn, fields.Len())
	for idx := 0; idx < fields.Len(); idx++ {
		field := fields.Get(idx)

		annotation := proto.GetExtension(field.Options(), psm_pb.E_Field).(*psm_pb.FieldOptions)
		if annotation == nil {
			annotation = &psm_pb.FieldOptions{}
		}

		tm.KeyColumns[idx] = psm.KeyColumn{
			ColumnName: string(field.Name()), // Use proto names as column names.
			ProtoName:  field.Name(),
			Primary:    annotation.PrimaryKey,
			Required:   annotation.PrimaryKey || !field.HasOptionalKeyword(),
		}
	}

	return tm, nil
}

const (
	stateMetadataProtoName = protoreflect.FullName("psm.state.v1.StateMetadata")
	eventMetadataProtoName = protoreflect.FullName("psm.state.v1.EventMetadata")
)

func TableMapFromStateAndEvent(stateMessage, eventMessage protoreflect.MessageDescriptor) (*psm.TableMap, error) {

	var stateMetadataField protoreflect.FieldDescriptor
	var stateKeyField protoreflect.FieldDescriptor
	var keyMessage protoreflect.MessageDescriptor

	fields := stateMessage.Fields()
	for idx := 0; idx < fields.Len(); idx++ {
		field := fields.Get(idx)
		if field.Kind() != protoreflect.MessageKind {
			continue
		}
		msg := field.Message()
		if msg.FullName() == stateMetadataProtoName {
			stateMetadataField = field
			continue
		}
		stateObjectAnnotation, ok := proto.GetExtension(msg.Options(), psm_pb.E_Psm).(*psm_pb.PSMOptions)
		if ok && stateObjectAnnotation != nil {
			keyMessage = msg
			stateKeyField = field
			continue
		}
	}

	if stateMetadataField == nil {
		return nil, fmt.Errorf("state message %s has no %s field", stateMessage.FullName(), stateMetadataProtoName)
	}
	if stateKeyField == nil {
		return nil, fmt.Errorf("state message %s has no PSM Keys", stateMessage.FullName())
	}

	var eventMetadataField protoreflect.FieldDescriptor
	var eventKeysField protoreflect.FieldDescriptor

	fields = eventMessage.Fields()
	for idx := 0; idx < fields.Len(); idx++ {
		field := fields.Get(idx)
		if field.Kind() != protoreflect.MessageKind {
			continue
		}
		msg := field.Message()
		if msg.FullName() == eventMetadataProtoName {
			eventMetadataField = field
			continue
		}

		stateObjectAnnotation, ok := proto.GetExtension(msg.Options(), psm_pb.E_Psm).(*psm_pb.PSMOptions)
		if ok && stateObjectAnnotation != nil {
			if keyMessage.FullName() != msg.FullName() {
				return nil, fmt.Errorf("%s.%s is a %s, but %s.%s is a %s, these should be the same",
					stateMessage.FullName(),
					stateKeyField.Name(),
					keyMessage.FullName(),
					eventMessage.FullName(),
					field.Name(),
					msg.FullName(),
				)
			}
			eventKeysField = field
			continue
		}
	}

	if eventMetadataField == nil {
		return nil, fmt.Errorf("event message %s has no %s field", eventMessage.FullName(), eventMetadataProtoName)
	}
	if eventKeysField == nil {
		return nil, fmt.Errorf("event message %s has no PSM Keys", eventMessage.FullName())
	}

	ss := StateMachine{
		EventMetadataField: eventMetadataField,
		KeyMessage:         keyMessage,
	}

	return BuildDefaultTableMap(ss)
}
