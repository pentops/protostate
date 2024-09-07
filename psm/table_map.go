package psm

import (
	"fmt"
	"strings"
	"unicode"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/pentops/j5/gen/j5/ext/v1/ext_j5pb"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/j5/gen/j5/schema/v1/schema_j5pb"
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

		keyColumn, err := keyFieldColumn(field)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.FullName(), err)
		}

		tm.KeyColumns[idx] = *keyColumn
	}

	return tm, nil
}

// TODO: This is a crazy amount of code for what it does. It's copied (and
// modified) from J5 code, creating a string field from the various 'hints'
// which coud be present. A: PSM should be based on J5, not proto... but not
// right now, but B: it should use the J5 schema builder anyway. Maybe once it
// is stable and has a single type annotation.
func keyFieldColumn(field protoreflect.FieldDescriptor) (*KeyColumn, error) {

	if field.Kind() != protoreflect.StringKind {
		return nil, fmt.Errorf("key fields must be strings, %s is %s", field.Name(), field.Kind().String())
	}

	annotation := proto.GetExtension(field.Options(), ext_j5pb.E_Key).(*ext_j5pb.PSMKeyFieldOptions)
	if annotation == nil {
		annotation = &ext_j5pb.PSMKeyFieldOptions{}
	}

	format, err := stringFieldFormat(field)
	if err != nil {
		return nil, fmt.Errorf("field %s: %w", field.FullName(), err)
	}

	kc := &KeyColumn{
		ColumnName: string(field.Name()), // Use proto names as column names.
		ProtoName:  field.Name(),
		Primary:    annotation.PrimaryKey,
		Required:   annotation.PrimaryKey || !field.HasOptionalKeyword(),
		Format:     format,
	}

	return kc, nil
}

func stringFieldFormat(field protoreflect.FieldDescriptor) (*schema_j5pb.KeyFormat, error) {

	validateConstraint := proto.GetExtension(field.Options(), validate.E_Field).(*validate.FieldConstraints)

	if validateConstraint != nil && validateConstraint.Type != nil {
		stringConstraint, ok := validateConstraint.Type.(*validate.FieldConstraints_String_)
		if !ok {
			return nil, fmt.Errorf("wrong constraint type for string: %T", validateConstraint.Type)
		}

		constraint := stringConstraint.String_
		switch wkt := constraint.WellKnown.(type) {
		case *validate.StringRules_Uuid:
			if wkt.Uuid {
				return &schema_j5pb.KeyFormat{Type: &schema_j5pb.KeyFormat_Uuid{Uuid: &schema_j5pb.KeyFormat_UUID{}}}, nil
			}

		}
	}

	listConstraint := proto.GetExtension(field.Options(), list_j5pb.E_Field).(*list_j5pb.FieldConstraint)
	listRules := listConstraint.GetString_()
	if fk := listRules.GetForeignKey(); fk != nil {
		switch fk.Type.(type) {
		case *list_j5pb.ForeignKeyRules_UniqueString:
			return &schema_j5pb.KeyFormat{Type: &schema_j5pb.KeyFormat_Informal_{Informal: &schema_j5pb.KeyFormat_Informal{}}}, nil
		case *list_j5pb.ForeignKeyRules_Uuid:
			return &schema_j5pb.KeyFormat{Type: &schema_j5pb.KeyFormat_Uuid{Uuid: &schema_j5pb.KeyFormat_UUID{}}}, nil
		}
	}

	return nil, nil
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
