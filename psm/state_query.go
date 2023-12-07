package psm

import (
	"context"
	"fmt"
	"strings"

	"github.com/pentops/protostate/pquery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

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
	getter      *pquery.Getter[GetREQ, GetRES]
	mainLister  *pquery.Lister[ListREQ, ListRES]
	eventLister *pquery.Lister[ListEventsREQ, ListEventsRES]
}

func (gc *StateQuerySet[
	GetREQ, GetRES,
	ListREQ, ListRES,
	ListEventsREQ, ListEventsRES,
]) Get(ctx context.Context, db Transactor, reqMsg GetREQ, resMsg GetRES) error {
	return gc.getter.Get(ctx, db, reqMsg, resMsg)
}

func (gc *StateQuerySet[
	GetREQ, GetRES,
	ListREQ, ListRES,
	ListEventsREQ, ListEventsRES,
]) List(ctx context.Context, db Transactor, reqMsg proto.Message, resMsg proto.Message) error {
	return gc.mainLister.List(ctx, db, reqMsg, resMsg)
}

func (gc *StateQuerySet[
	GetREQ, GetRES,
	ListREQ, ListRES,
	ListEventsREQ, ListEventsRES,
]) ListEvents(ctx context.Context, db Transactor, reqMsg proto.Message, resMsg proto.Message) error {
	return gc.eventLister.List(ctx, db, reqMsg, resMsg)
}

type StateQuerySpec[
	GetREQ pquery.GetRequest,
	GetRES pquery.GetResponse,
	ListREQ pquery.ListRequest,
	ListRES pquery.ListResponse,
	ListEventsREQ pquery.ListRequest,
	ListEventsRES pquery.ListResponse,

] struct {
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
	smSpec QuerySpec,
	spec StateQuerySpec[GetREQ, GetRES, ListREQ, ListRES, ListEventsREQ, ListEventsRES],
) (*StateQuerySet[GetREQ, GetRES, ListREQ, ListRES, ListEventsREQ, ListEventsRES], error) {

	getSpec := pquery.GetSpec[GetREQ, GetRES]{
		TableName:  smSpec.StateTable,
		DataColumn: smSpec.StateDataColumn,
		Auth:       spec.Auth,
		AuthJoin:   spec.AuthJoin,
	}

	pkFields := map[string]protoreflect.FieldDescriptor{}
	eventJoinMap := pquery.JoinFields{}
	requestReflect := (*new(GetREQ)).ProtoReflect().Descriptor()

	for i := 0; i < requestReflect.Fields().Len(); i++ {
		field := requestReflect.Fields().Get(i)
		fullKey := string(field.Name())
		rootKey := strings.TrimPrefix(fullKey, smSpec.StateTable+"_")
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
		if smSpec.EventTable == "" {
			return nil, fmt.Errorf("missing EventTable in state spec for %s", smSpec.StateTable)
		}
		if smSpec.EventDataColumn == "" {
			return nil, fmt.Errorf("missing EventDataColumn in state spec for %s", smSpec.StateTable)
		}
		getSpec.Join = &pquery.GetJoinSpec{
			TableName:     smSpec.EventTable,
			DataColumn:    smSpec.EventDataColumn,
			FieldInParent: eventsInGet,
			On:            eventJoinMap,
		}
	}

	getter, err := pquery.NewGetter(getSpec)
	if err != nil {
		return nil, fmt.Errorf("build getter for state query '%s': %w", smSpec.StateTable, err)
	}

	listSpec := pquery.ListSpec[ListREQ, ListRES]{
		TableName:  smSpec.StateTable,
		DataColumn: smSpec.StateDataColumn,
		Auth:       spec.Auth,
	}
	if spec.AuthJoin != nil {
		listSpec.AuthJoin = []*pquery.LeftJoin{spec.AuthJoin}
	}

	lister, err := pquery.NewLister(listSpec)
	if err != nil {
		return nil, fmt.Errorf("build main lister for state query '%s': %w", smSpec.EventTable, err)
	}

	querySet := &StateQuerySet[GetREQ, GetRES, ListREQ, ListRES, ListEventsREQ, ListEventsRES]{
		getter:     getter,
		mainLister: lister,
	}

	if spec.SkipEvents {
		return querySet, nil
	}

	eventsAuthJoin := []*pquery.LeftJoin{{
		// Main is the events table, joining to the state table
		TableName: smSpec.StateTable,
		On:        eventJoinMap.Reverse(),
	}}

	if spec.AuthJoin != nil {
		eventsAuthJoin = append(eventsAuthJoin, spec.AuthJoin)
	}

	eventListSpec := pquery.ListSpec[ListEventsREQ, ListEventsRES]{
		TableName:  smSpec.EventTable,
		DataColumn: smSpec.EventDataColumn,
		Auth:       spec.Auth,
		AuthJoin:   eventsAuthJoin,
	}
	eventLister, err := pquery.NewLister(eventListSpec)
	if err != nil {
		return nil, fmt.Errorf("build event lister for state query '%s' lister: %w", smSpec.EventTable, err)
	}

	querySet.eventLister = eventLister

	return querySet, nil
}
