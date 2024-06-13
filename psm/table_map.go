package psm

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func safeTableName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, name)
}

func BuildQueryTableSpec(stateMessage, eventMessage protoreflect.MessageDescriptor) (QueryTableSpec, error) {
	tableMap, err := tableMapFromStateAndEvent(stateMessage, eventMessage)
	if err != nil {
		return QueryTableSpec{}, err
	}

	return QueryTableSpec{
		TableMap:  *tableMap,
		StateType: stateMessage,
		EventType: eventMessage,
	}, nil
}

func buildDefaultTableMap(keyMessage protoreflect.MessageDescriptor) (*TableMap, error) {
	stateObjectAnnotation, ok := proto.GetExtension(keyMessage.Options(), psm_pb.E_Psm).(*psm_pb.PSMOptions)
	if !ok || stateObjectAnnotation == nil {
		return nil, fmt.Errorf("message %s has no PSM Key field", keyMessage.Name())
	}

	tm := &TableMap{
		State: StateTableSpec{
			TableName: safeTableName(stateObjectAnnotation.Name),
			Root: &FieldSpec{
				ColumnName: "state",
				//PathFromRoot: psm.PathSpec{},
			},
		},
		Event: EventTableSpec{
			TableName: safeTableName(stateObjectAnnotation.Name + "_event"),
			Root: &FieldSpec{
				ColumnName: "data",
				//PathFromRoot: psm.PathSpec{},
			},
			ID: &FieldSpec{
				ColumnName: "id",
				//PathFromRoot: psm.PathSpec{string(ss.EventMetadataField.Name()), "event_id"},
			},
			Timestamp: &FieldSpec{
				ColumnName: "timestamp",
				//PathFromRoot: psm.PathSpec{string(ss.EventMetadataField.Name()), "timestamp"},
			},
			Sequence: &FieldSpec{
				ColumnName: "sequence",
				//PathFromRoot: psm.PathSpec{string(ss.EventMetadataField.Name()), "sequence"},
			},
			StateSnapshot: &FieldSpec{
				ColumnName: "state",
			},
		},
	}

	fields := keyMessage.Fields()
	tm.KeyColumns = make([]KeyColumn, fields.Len())
	for idx := 0; idx < fields.Len(); idx++ {
		field := fields.Get(idx)

		annotation := proto.GetExtension(field.Options(), psm_pb.E_Field).(*psm_pb.FieldOptions)
		if annotation == nil {
			annotation = &psm_pb.FieldOptions{}
		}

		tm.KeyColumns[idx] = KeyColumn{
			ColumnName: string(field.Name()), // Use proto names as column names.
			ProtoName:  field.Name(),
			Primary:    annotation.PrimaryKey,
			Required:   annotation.PrimaryKey || !field.HasOptionalKeyword(),
		}
	}

	return tm, nil
}

func tableMapFromStateAndEvent(stateMessage, eventMessage protoreflect.MessageDescriptor) (*TableMap, error) {

	var stateKeyField protoreflect.FieldDescriptor
	var keyMessage protoreflect.MessageDescriptor

	fields := stateMessage.Fields()
	for idx := 0; idx < fields.Len(); idx++ {
		field := fields.Get(idx)
		if field.Kind() != protoreflect.MessageKind {
			continue
		}
		msg := field.Message()
		stateObjectAnnotation, ok := proto.GetExtension(msg.Options(), psm_pb.E_Psm).(*psm_pb.PSMOptions)
		if ok && stateObjectAnnotation != nil {
			keyMessage = msg
			stateKeyField = field
			break
		}
	}

	if stateKeyField == nil {
		return nil, fmt.Errorf("state message %s has no PSM Keys", stateMessage.FullName())
	}

	var eventKeysField protoreflect.FieldDescriptor

	fields = eventMessage.Fields()
	for idx := 0; idx < fields.Len(); idx++ {
		field := fields.Get(idx)
		if field.Kind() != protoreflect.MessageKind {
			continue
		}
		msg := field.Message()

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

	if eventKeysField == nil {
		return nil, fmt.Errorf("event message %s has no PSM Keys", eventMessage.FullName())
	}
	return buildDefaultTableMap(keyMessage)
}
