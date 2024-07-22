package psm

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/pentops/j5/gen/j5/ext/v1/ext_j5pb"
	"github.com/pentops/protostate/internal/pgstore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type TableMap struct {

	// KeyColumns are stored in both state and event tables.
	// Keys marked primary combine to form the primary key of the State table,
	// and therefore a foreign key from the event table.
	// Non primary keys are included but not referenced.
	// All columns must be UUID.
	KeyColumns []KeyColumn

	State StateTableSpec
	Event EventTableSpec
}

func (tm *TableMap) Validate() error {

	if tm.State.TableName == "" {
		return fmt.Errorf("missing State.TableName in TableMap")
	}
	if tm.State.Root == nil {
		return fmt.Errorf("missing State.Data in TableMap")
	}
	if tm.Event.TableName == "" {
		return fmt.Errorf("missing Event.TableName in TableMap")
	}
	if tm.Event.ID == nil {
		return fmt.Errorf("missing Event.Data in TableMap")
	}
	if tm.Event.Timestamp == nil {
		return fmt.Errorf("missing Event.Timestamp in TableMap")
	}
	if tm.Event.Root == nil {
		return fmt.Errorf("missing Event.Data in TableMap")
	}
	if tm.Event.Sequence == nil {
		return fmt.Errorf("missing Event.Sequence in TableMap")
	}
	if tm.Event.StateSnapshot == nil {
		return fmt.Errorf("missing Event.StateSnapshot in TableMap")
	}
	return nil
}

type FieldSpec struct {
	ColumnName string
}

type EventTableSpec struct {
	TableName string

	// The entire event mesage as JSONB
	Root *FieldSpec

	// a UUID holding the primary key of the event
	// TODO: Multi-column ID for Events?
	ID *FieldSpec

	// timestamptz The time of the event
	Timestamp *FieldSpec

	// int, The descrete integer for the event in the state machine
	Sequence *FieldSpec

	// jsonb, holds the state after the event
	StateSnapshot *FieldSpec
}

type StateTableSpec struct {
	TableName string

	// The entire state message, as a JSONB
	Root *FieldSpec
}

type KeyColumn struct {
	ColumnName string
	ProtoName  protoreflect.Name
	Primary    bool
	Required   bool
	Unique     bool
	//TenantKey  *string
}

type KeyField struct {
	ColumnName *string // Optional, stores in the table as a column.
	Primary    bool
	Unique     bool
	Path       *pgstore.Path
}

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
	stateObjectAnnotation, ok := proto.GetExtension(keyMessage.Options(), ext_j5pb.E_Psm).(*ext_j5pb.PSMOptions)
	if !ok || stateObjectAnnotation == nil {
		return nil, fmt.Errorf("message %s has no PSM Key field", keyMessage.Name())
	}

	tm := &TableMap{
		State: StateTableSpec{
			TableName: safeTableName(stateObjectAnnotation.EntityName),
			Root: &FieldSpec{
				ColumnName: "state",
				//PathFromRoot: psm.PathSpec{},
			},
		},
		Event: EventTableSpec{
			TableName: safeTableName(stateObjectAnnotation.EntityName + "_event"),
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

		annotation := proto.GetExtension(field.Options(), ext_j5pb.E_Key).(*ext_j5pb.KeyFieldOptions)
		if annotation == nil {
			annotation = &ext_j5pb.KeyFieldOptions{}
		}

		tm.KeyColumns[idx] = KeyColumn{
			ColumnName: string(field.Name()), // Use proto names as column names.
			ProtoName:  field.Name(),
			Primary:    annotation.PrimaryKey,
			Required:   annotation.PrimaryKey || !field.HasOptionalKeyword(),
			//TenantKey:  annotation.TenantKey,
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
		stateObjectAnnotation, ok := proto.GetExtension(msg.Options(), ext_j5pb.E_Psm).(*ext_j5pb.PSMOptions)
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

		stateObjectAnnotation, ok := proto.GetExtension(msg.Options(), ext_j5pb.E_Psm).(*ext_j5pb.PSMOptions)
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
