package psm

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
	"github.com/pentops/j5/gen/j5/schema/v1/schema_j5pb"
	"github.com/pentops/j5/lib/j5schema"
	"github.com/pentops/j5/lib/j5query"
)

type TableMap struct {

	// KeyColumns are stored in both state and event tables.
	// Keys marked primary combine to form the primary key of the State table,
	// and therefore a foreign key from the event table.
	// Non primary keys are included but not referenced.
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

	// The entire event message as JSONB
	Root *FieldSpec

	// a UUID holding the primary key of the event
	// TODO: Multi-column ID for Events?
	ID *FieldSpec

	// timestamptz The time of the event
	Timestamp *FieldSpec

	// int, The discrete integer for the event in the state machine
	Sequence *FieldSpec

	// jsonb, holds the state after the event
	StateSnapshot *FieldSpec
}

type StateTableSpec struct {
	TableName string

	KeyField *j5schema.ObjectProperty

	// The entire state message, as a JSONB
	Root *FieldSpec
}

type KeyColumn struct {
	ColumnName string
	//ProtoField protoreflect.FieldNumber
	//ProtoName  protoreflect.Name
	JSONFieldName string

	Primary            bool
	Required           bool
	ExplicitlyOptional bool // makes strings pointers
	Unique             bool

	Schema *schema_j5pb.Field // The schema for this key field, used for validation and serialization.

	//TenantKey  *string
}

type KeyField struct {
	ColumnName *string // Optional, stores in the table as a column.
	Primary    bool
	Unique     bool
	Path       *pquery.Path
}

func safeTableName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, name)
}

func BuildQueryTableSpec(stateMessage, eventMessage *j5schema.ObjectSchema) (QueryTableSpec, error) {
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

func buildDefaultTableMap(stateKeyField *j5schema.ObjectProperty, keyMessage *j5schema.ObjectSchema) (*TableMap, error) {
	if keyMessage.Entity == nil {
		return nil, fmt.Errorf("key message %s has no PSM Entity annotation", keyMessage.Name())
	}
	entity := keyMessage.Entity

	tm := &TableMap{
		State: StateTableSpec{
			TableName: safeTableName(entity.Entity),
			KeyField:  stateKeyField,
			Root: &FieldSpec{
				ColumnName: "state",
				//PathFromRoot: psm.PathSpec{},
			},
		},
		Event: EventTableSpec{
			TableName: safeTableName(entity.Entity + "_event"),
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

	for _, field := range keyMessage.Properties {

		keyColumn, err := keyFieldColumn(field)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.FullName(), err)
		}

		tm.KeyColumns = append(tm.KeyColumns, *keyColumn)
	}

	return tm, nil
}

func keyFieldColumn(field *j5schema.ObjectProperty) (*KeyColumn, error) {

	isPrimary := field.Entity != nil && field.Entity.Primary

	kc := &KeyColumn{
		ColumnName:    strcase.ToSnake(field.JSONName),
		JSONFieldName: field.JSONName,
		//ProtoField:         desc.Number(),
		//ProtoName:          desc.Name(),
		Primary:            isPrimary,
		Required:           isPrimary || field.Required,
		ExplicitlyOptional: field.ExplicitlyOptional,
		Schema:             field.Schema.ToJ5Field(),
	}

	return kc, nil
}

func tableMapFromStateAndEvent(stateMessage, eventMessage *j5schema.ObjectSchema) (*TableMap, error) {

	var stateKeyField *j5schema.ObjectProperty
	var keyMessage *j5schema.ObjectSchema

	for _, field := range stateMessage.Properties {
		obj, ok := field.Schema.(*j5schema.ObjectField)
		if !ok {
			continue
		}
		objSchema := obj.ObjectSchema()
		if objSchema.Entity == nil {
			continue
		}
		if objSchema.Entity.Part != schema_j5pb.EntityPart_KEYS {
			continue
		}
		keyMessage = objSchema
		stateKeyField = field
		break
	}

	if stateKeyField == nil {
		return nil, fmt.Errorf("state message %s has no PSM Keys", stateMessage.FullName())
	}

	var eventKeyField *j5schema.ObjectProperty
	var eventKeyMessage *j5schema.ObjectSchema

	for _, field := range eventMessage.Properties {

		obj, ok := field.Schema.(*j5schema.ObjectField)
		if !ok {
			continue
		}
		objSchema := obj.ObjectSchema()
		if objSchema.Entity == nil {
			continue
		}
		if objSchema.Entity.Part != schema_j5pb.EntityPart_KEYS {
			continue
		}
		eventKeyMessage = objSchema
		eventKeyField = field

	}

	if eventKeyField == nil {
		return nil, fmt.Errorf("event message %s has no PSM Keys", eventMessage.FullName())
	}

	if eventKeyMessage.FullName() != keyMessage.FullName() {
		return nil, fmt.Errorf("state message %s has keys %s, but event message %s has keys %s",
			stateMessage.FullName(),
			stateKeyField.FullName(),
			eventMessage.FullName(),
			eventKeyField.FullName(),
		)
	}

	return buildDefaultTableMap(stateKeyField, keyMessage)
}
