package psm

import (
	"context"
	"fmt"
	"strings"

	"github.com/pentops/protostate/pquery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// QueryTableSpec is the subset of TableSpec which does not relate to the
// specific event and state types of the state machine.
type QueryTableSpec struct {
	EventTypeName protoreflect.FullName
	StateTypeName protoreflect.FullName

	State EntityTableSpec
	Event EntityTableSpec
}

type EntityTableSpec struct {
	TableName  string
	DataColumn string

	// PKFieldPaths is a list of 'field paths' which constitute the primary key
	// of the entity, dot-notated protobuf field names.
	PKFieldPaths []string
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
		DataColumn: smSpec.State.DataColumn,
		Auth:       options.Auth,
		AuthJoin:   options.AuthJoin,
	}

	pkFields := map[string]protoreflect.FieldDescriptor{}
	eventJoinMap := pquery.JoinFields{}
	requestReflect := (*new(GetREQ)).ProtoReflect().Descriptor()

	for i := 0; i < requestReflect.Fields().Len(); i++ {
		field := requestReflect.Fields().Get(i)
		fullKey := string(field.Name())
		rootKey := strings.TrimPrefix(fullKey, smSpec.State.TableName+"_")
		pkFields[rootKey] = field
		eventJoinMap = append(eventJoinMap, pquery.JoinField{
			RootColumn: rootKey,
			JoinColumn: fullKey,
		})
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
		if smSpec.Event.DataColumn == "" {
			return nil, fmt.Errorf("missing EventDataColumn in state spec for %s", smSpec.State.TableName)
		}
		getSpec.Join = &pquery.GetJoinSpec{
			TableName:     smSpec.Event.TableName,
			DataColumn:    smSpec.Event.DataColumn,
			FieldInParent: eventsInGet,
			On:            eventJoinMap,
		}
	}

	getter, err := pquery.NewGetter(getSpec)
	if err != nil {
		return nil, fmt.Errorf("build getter for state query '%s': %w", smSpec.State.TableName, err)
	}

	listSpec := pquery.ListSpec[ListREQ, ListRES]{
		TableName:     smSpec.State.TableName,
		DataColumn:    smSpec.State.DataColumn,
		Auth:          options.Auth,
		RequestFilter: smSpec.ListRequestFilter,
	}
	if options.AuthJoin != nil {
		listSpec.AuthJoin = []*pquery.LeftJoin{options.AuthJoin}
	}

	lister, err := pquery.NewLister(listSpec, pquery.WithTieBreakerFields(smSpec.State.PKFieldPaths...))
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
		TableName:     smSpec.Event.TableName,
		DataColumn:    smSpec.Event.DataColumn,
		Auth:          options.Auth,
		AuthJoin:      eventsAuthJoin,
		RequestFilter: smSpec.ListEventsRequestFilter,
	}

	eventLister, err := pquery.NewLister(eventListSpec, pquery.WithTieBreakerFields(smSpec.Event.PKFieldPaths...))
	if err != nil {
		return nil, fmt.Errorf("build event lister for state query '%s' lister: %w", smSpec.Event.TableName, err)
	}

	querySet.EventLister = eventLister

	return querySet, nil
}
