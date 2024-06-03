package psm

import (
	"context"
	"fmt"

	"github.com/pentops/protostate/pgstore"
	"github.com/pentops/protostate/pquery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// QueryTableSpec is the subset of TableSpec which does not relate to the
// specific event and state types of the state machine.
type QueryTableSpec struct {
	EventTypeName protoreflect.FullName
	StateTypeName protoreflect.FullName
	TableMap
}

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
	if tm.Event.Cause == nil {
		return fmt.Errorf("missing Event.Cause in TableMap")
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

type EventTableSpec struct {
	TableName string

	// The entire event mesage as JSONB
	Root *pgstore.ProtoFieldSpec

	// a UUID holding the primary key of the event
	// TODO: Multi-column ID for Events?
	ID *pgstore.ProtoFieldSpec

	// timestamptz The time of the event
	Timestamp *pgstore.ProtoFieldSpec

	// int, The descrete integer for the event in the state machine
	Sequence *pgstore.ProtoFieldSpec

	// jsonb, the event cause
	Cause *pgstore.ProtoFieldSpec

	// jsonb, holds the state after the event
	StateSnapshot *pgstore.ProtoFieldSpec
}

type StateTableSpec struct {
	TableName string

	// The entire state message, as a JSONB
	Root *pgstore.ProtoFieldSpec
}

type KeyColumn struct {
	ColumnName string
	ProtoName  protoreflect.Name
	Primary    bool
	Required   bool
	Unique     bool
}

type EntityTableSpec struct {
	TableName  string
	DataColumn string

	// PKFieldPaths is a list of 'field paths' which constitute the primary key
	// of the entity, dot-notated protobuf field names.
	PKFieldPaths []string

	Fields []KeyField
}

type KeyField struct {
	ColumnName *string // Optional, stores in the table as a column.
	Primary    bool
	Unique     bool
	Path       *pgstore.Path
}

func (spec QueryTableSpec) BuildStateMachineMigration() ([]byte, error) {

	stateTable, err := buildStateTable(spec)
	if err != nil {
		return nil, err
	}

	eventTable, err := buildEventTable(spec)
	if err != nil {
		return nil, err
	}

	fileData, err := pgstore.PrintCreateMigration(stateTable, eventTable)
	if err != nil {
		return nil, err
	}
	return fileData, nil
}

func buildStateTable(spec QueryTableSpec) (*pgstore.CreateTableBuilder, error) {
	tt := pgstore.CreateTable(spec.State.TableName).
		Column("id", "uuid", pgstore.PrimaryKey).
		Column(spec.State.Root.ColumnName, "jsonb", pgstore.NotNull)

	return tt, nil
}

func buildEventTable(spec QueryTableSpec) (*pgstore.CreateTableBuilder, error) {
	tt := pgstore.CreateTable(spec.Event.TableName).
		Column("id", "uuid", pgstore.PrimaryKey).
		Column("timestamp", "timestamptz", pgstore.NotNull).
		Column("sequence", "int", pgstore.NotNull).
		Column("cause", "jsonb", pgstore.NotNull).
		Column("data", "jsonb", pgstore.NotNull).
		Column("state", "jsonb", pgstore.NotNull)

	return tt, nil
}

// QuerySpec is the configuration for the query service side of the state
// machine. Can be partially derived from the state machine table spec, but
// contains types relating to the query service so cannot be fully derived.
type QuerySpec[
	GetREQ pquery.GetRequest,
	GetRES pquery.GetResponse,
	ListREQ pquery.ListRequest,
	ListRES pquery.ListResponse,
	ListEventsREQ pquery.ListRequest,
	ListEventsRES pquery.ListResponse,
] struct {
	QueryTableSpec

	ListRequestFilter       func(ListREQ) (map[string]interface{}, error)
	ListEventsRequestFilter func(ListEventsREQ) (map[string]interface{}, error)
}

// StateQuerySet is a shortcut for manually specifying three different query
// types following the 'standard model':
// 1. A getter for a single state
// 2. A lister for the main state
// 3. A lister for the events of the main state
type StateQuerySet[
	GetREQ pquery.GetRequest,
	GetRES pquery.GetResponse,
	ListREQ pquery.ListRequest,
	ListRES pquery.ListResponse,
	ListEventsREQ pquery.ListRequest,
	ListEventsRES pquery.ListResponse,
] struct {
	Getter      *pquery.Getter[GetREQ, GetRES]
	MainLister  *pquery.Lister[ListREQ, ListRES]
	EventLister *pquery.Lister[ListEventsREQ, ListEventsRES]
}

func (gc *StateQuerySet[
	GetREQ, GetRES,
	ListREQ, ListRES,
	ListEventsREQ, ListEventsRES,
]) SetQueryLogger(logger pquery.QueryLogger) {
	gc.Getter.SetQueryLogger(logger)
	gc.MainLister.SetQueryLogger(logger)
	gc.EventLister.SetQueryLogger(logger)
}

func (gc *StateQuerySet[
	GetREQ, GetRES,
	ListREQ, ListRES,
	ListEventsREQ, ListEventsRES,
]) Get(ctx context.Context, db Transactor, reqMsg GetREQ, resMsg GetRES) error {
	return gc.Getter.Get(ctx, db, reqMsg, resMsg)
}

func (gc *StateQuerySet[
	GetREQ, GetRES,
	ListREQ, ListRES,
	ListEventsREQ, ListEventsRES,
]) List(ctx context.Context, db Transactor, reqMsg proto.Message, resMsg proto.Message) error {
	return gc.MainLister.List(ctx, db, reqMsg, resMsg)
}

func (gc *StateQuerySet[
	GetREQ, GetRES,
	ListREQ, ListRES,
	ListEventsREQ, ListEventsRES,
]) ListEvents(ctx context.Context, db Transactor, reqMsg proto.Message, resMsg proto.Message) error {
	return gc.EventLister.List(ctx, db, reqMsg, resMsg)
}

type StateQueryOptions struct {
	Auth       pquery.AuthProvider
	AuthJoin   *pquery.LeftJoin
	SkipEvents bool
}

func BuildStateQuerySet[
	GetREQ pquery.GetRequest,
	GetRES pquery.GetResponse,
	ListREQ pquery.ListRequest,
	ListRES pquery.ListResponse,
	ListEventsREQ pquery.ListRequest,
	ListEventsRES pquery.ListResponse,
](
	smSpec QuerySpec[GetREQ, GetRES, ListREQ, ListRES, ListEventsREQ, ListEventsRES],
	options StateQueryOptions,
) (*StateQuerySet[GetREQ, GetRES, ListREQ, ListRES, ListEventsREQ, ListEventsRES], error) {

	getSpec := pquery.GetSpec[GetREQ, GetRES]{
		TableName:  smSpec.State.TableName,
		DataColumn: smSpec.State.Root.ColumnName,
		Auth:       options.Auth,
		AuthJoin:   options.AuthJoin,
	}

	eventJoinMap := pquery.JoinFields{}
	requestReflect := (*new(GetREQ)).ProtoReflect().Descriptor()

	unmappedRequestFields := map[protoreflect.Name]protoreflect.FieldDescriptor{}
	for i := 0; i < requestReflect.Fields().Len(); i++ {
		field := requestReflect.Fields().Get(i)
		unmappedRequestFields[field.Name()] = field
	}

	pkFields := map[string]protoreflect.FieldDescriptor{}
	for _, keyColumn := range smSpec.KeyColumns {
		matchingRequestField, ok := unmappedRequestFields[keyColumn.ProtoName]
		if ok {
			delete(unmappedRequestFields, keyColumn.ProtoName)
			pkFields[keyColumn.ColumnName] = matchingRequestField
		}

		if keyColumn.Primary {
			eventJoinMap = append(eventJoinMap, pquery.JoinField{
				RootColumn: keyColumn.ColumnName,
				JoinColumn: keyColumn.ColumnName,
			})
		}
	}

	if len(unmappedRequestFields) > 0 {
		return nil, fmt.Errorf("unmapped fields in Get request: %v", unmappedRequestFields)
	}

	getSpec.PrimaryKey = func(req GetREQ) (map[string]interface{}, error) {
		refl := req.ProtoReflect()
		out := map[string]interface{}{}
		for k, v := range pkFields {
			out[k] = refl.Get(v).Interface()
		}
		return out, nil
	}

	var eventsInGet protoreflect.Name

	getResponseReflect := (*new(GetRES)).ProtoReflect().Descriptor()
	for i := 0; i < getResponseReflect.Fields().Len(); i++ {
		field := getResponseReflect.Fields().Get(i)
		msg := field.Message()
		if msg == nil {
			continue
		}

		if msg.FullName() == smSpec.EventTypeName {
			eventsInGet = field.Name()
		} else if msg.FullName() == smSpec.StateTypeName {
			getSpec.StateResponseField = field.Name()
		}
	}

	if eventsInGet != "" {
		if smSpec.Event.TableName == "" {
			return nil, fmt.Errorf("missing EventTable in state spec for %s", smSpec.State.TableName)
		}
		if smSpec.Event.Root == nil {
			return nil, fmt.Errorf("missing EventDataColumn in state spec for %s", smSpec.State.TableName)
		}
		getSpec.Join = &pquery.GetJoinSpec{
			TableName:     smSpec.Event.TableName,
			DataColumn:    smSpec.Event.Root.ColumnName,
			FieldInParent: eventsInGet,
			On:            eventJoinMap,
		}
	}

	getter, err := pquery.NewGetter(getSpec)
	if err != nil {
		return nil, fmt.Errorf("build getter for state query '%s': %w", smSpec.State.TableName, err)
	}

	statePrimaryKeys := []pgstore.ProtoFieldSpec{}

	for _, field := range smSpec.KeyColumns {
		if !field.Primary {
			continue
		}
		statePrimaryKeys = append(statePrimaryKeys, pgstore.ProtoFieldSpec{
			ColumnName: field.ColumnName,
			// No Path
		})
	}

	listSpec := pquery.ListSpec[ListREQ, ListRES]{
		TableSpec: pquery.TableSpec{
			TableName:           smSpec.State.TableName,
			DataColumn:          smSpec.State.Root.ColumnName,
			Auth:                options.Auth,
			FallbackSortColumns: statePrimaryKeys,
		},
		RequestFilter: smSpec.ListRequestFilter,
	}
	if options.AuthJoin != nil {
		listSpec.AuthJoin = []*pquery.LeftJoin{options.AuthJoin}
	}

	lister, err := pquery.NewLister(listSpec)
	if err != nil {
		return nil, fmt.Errorf("build main lister for state query '%s': %w", smSpec.State.TableName, err)
	}

	querySet := &StateQuerySet[GetREQ, GetRES, ListREQ, ListRES, ListEventsREQ, ListEventsRES]{
		Getter:     getter,
		MainLister: lister,
	}

	if options.SkipEvents {
		return querySet, nil
	}

	eventsAuthJoin := []*pquery.LeftJoin{{
		// Main is the events table, joining to the state table
		TableName: smSpec.State.TableName,
		On:        eventJoinMap.Reverse(),
	}}

	if options.AuthJoin != nil {
		eventsAuthJoin = append(eventsAuthJoin, options.AuthJoin)
	}

	eventListSpec := pquery.ListSpec[ListEventsREQ, ListEventsRES]{
		TableSpec: pquery.TableSpec{
			TableName:           smSpec.Event.TableName,
			DataColumn:          smSpec.Event.Root.ColumnName,
			Auth:                options.Auth,
			AuthJoin:            eventsAuthJoin,
			FallbackSortColumns: []pgstore.ProtoFieldSpec{*smSpec.Event.ID},
		},
		RequestFilter: smSpec.ListEventsRequestFilter,
	}

	eventLister, err := pquery.NewLister(eventListSpec)
	if err != nil {
		return nil, fmt.Errorf("build event lister for state query '%s' lister: %w", smSpec.Event.TableName, err)
	}

	querySet.EventLister = eventLister

	return querySet, nil
}
