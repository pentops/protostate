package psm

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
	"github.com/pentops/j5/gen/j5/ext/v1/ext_j5pb"
	"github.com/pentops/j5/gen/j5/schema/v1/schema_j5pb"
	"github.com/pentops/j5/lib/j5schema"
	"github.com/pentops/protostate/internal/pgstore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	if tm.Event.IdempotencyHash == nil {
		return fmt.Errorf("missing Event.IdempotencyHash in TableMap")
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
	ID *FieldSpec

	// Stores a globally unique itempotency key, hashed appropriately for
	// unique-per-tenant at entry
	IdempotencyHash *FieldSpec

	// timestamptz The time of the event
	Timestamp *FieldSpec

	// int, The discrete integer for the event in the state machine
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
	ProtoField protoreflect.FieldNumber
	ProtoName  protoreflect.Name

	Primary            bool
	Required           bool
	ExplicitlyOptional bool // makes strings pointers
	Unique             bool

	Format *schema_j5pb.KeyFormat

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

var globalSchemaCache = j5schema.NewSchemaCache()

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
			IdempotencyHash: &FieldSpec{
				ColumnName: "idempotency",
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

	schema, err := globalSchemaCache.Schema(keyMessage)
	if err != nil {
		return nil, nil
	}
	objSchema, ok := schema.(*j5schema.ObjectSchema)
	if !ok {
		return nil, fmt.Errorf("expected object schema, got %T", schema)
	}
	keyFields := keyMessage.Fields()
	for _, field := range objSchema.Properties {
		if len(field.ProtoField) != 1 {
			return nil, fmt.Errorf("key field %s: expected one proto field, got %d", field.FullName(), len(field.ProtoField))
		}
		desc := keyFields.ByNumber(field.ProtoField[0])

		keyColumn, err := keyFieldColumn(field, desc)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.FullName(), err)
		}

		tm.KeyColumns = append(tm.KeyColumns, *keyColumn)
	}

	return tm, nil
}

func keyFieldColumn(field *j5schema.ObjectProperty, desc protoreflect.FieldDescriptor) (*KeyColumn, error) {

	key := field.Schema.ToJ5Field().GetKey()
	if key == nil {
		return nil, fmt.Errorf("key field %s: expected key, got %T", field.FullName(), field.Schema.ToJ5Field().Type)
	}
	isPrimary := false
	if key.Entity != nil {
		switch key.Entity.Type.(type) {
		case *schema_j5pb.EntityKey_PrimaryKey:
			isPrimary = true
		}
	}

	kc := &KeyColumn{
		ColumnName:         strcase.ToSnake(field.JSONName),
		ProtoField:         desc.Number(),
		ProtoName:          desc.Name(),
		Primary:            isPrimary,
		Required:           isPrimary || field.Required,
		ExplicitlyOptional: field.ExplicitlyOptional,
		Format:             key.Format,
	}

	return kc, nil
}

func tableMapFromStateAndEvent(stateMessage, eventMessage protoreflect.MessageDescriptor) (*TableMap, error) {

	var stateKeyField protoreflect.FieldDescriptor
	var keyMessage protoreflect.MessageDescriptor

	fields := stateMessage.Fields()
	for idx := range fields.Len() {
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
	for idx := range fields.Len() {
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
