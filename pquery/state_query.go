package pquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/pentops/protostate/psm"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// StateQuerySet is a shortcut for manually specifying three different query
// types following the 'standard model':
// 1. A getter for a single state
// 2. A lister for the main state
// 3. A lister for the events of the main state
type StateQuerySet[
	GetREQ GetRequest,
	GetRES proto.Message,
	ListREQ ListRequest,
	ListRES ListResponse,
	ListEventsREQ ListRequest,
	ListEventsRES ListResponse,
] struct {
	getter      *Getter[GetREQ, GetRES]
	mainLister  *Lister[ListREQ, ListRES]
	eventLister *Lister[ListEventsREQ, ListEventsRES]
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

type SourceSM interface {
	GetQuerySpec() psm.QuerySpec
}

type StateSpec[
	GetREQ GetRequest,
	GetRES GetResponse,
	ListREQ ListRequest,
	ListRES ListResponse,
	ListEventsREQ ListRequest,
	ListEventsRES ListResponse,

] struct {
	AuthFunc   AuthProvider
	AuthJoin   *LeftJoin
	SkipEvents bool
}

func FromStateMachine[
	GetREQ GetRequest,
	GetRES GetResponse,
	ListREQ ListRequest,
	ListRES ListResponse,
	ListEventsREQ ListRequest,
	ListEventsRES ListResponse,
](
	source SourceSM,
	spec StateSpec[GetREQ, GetRES, ListREQ, ListRES, ListEventsREQ, ListEventsRES],
) (*StateQuerySet[GetREQ, GetRES, ListREQ, ListRES, ListEventsREQ, ListEventsRES], error) {

	smSpec := source.GetQuerySpec()

	getSpec := GetSpec[GetREQ, GetRES]{
		TableName:  smSpec.StateTable,
		DataColumn: smSpec.StateDataColumn,
		Auth:       spec.AuthFunc,
		AuthJoin:   spec.AuthJoin,
	}

	pkFields := map[string]protoreflect.FieldDescriptor{}
	eventJoinMap := JoinFields{}
	requestReflect := (*new(GetREQ)).ProtoReflect().Descriptor()

	for i := 0; i < requestReflect.Fields().Len(); i++ {
		field := requestReflect.Fields().Get(i)
		fullKey := string(field.Name())
		rootKey := strings.TrimPrefix(fullKey, smSpec.StateTable+"_")
		pkFields[rootKey] = field
		eventJoinMap = append(eventJoinMap, JoinField{
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
		getSpec.Join = &GetJoinSpec{
			TableName:     smSpec.EventTable,
			DataColumn:    smSpec.EventDataColumn,
			FieldInParent: eventsInGet,
			On:            eventJoinMap,
		}
	}

	getter, err := NewGetter(getSpec)
	if err != nil {
		return nil, fmt.Errorf("build getter for state query '%s': %w", smSpec.StateTable, err)
	}

	listSpec := ListSpec[ListREQ, ListRES]{
		TableName:  smSpec.StateTable,
		DataColumn: smSpec.StateDataColumn,
		Auth:       spec.AuthFunc,
	}
	if spec.AuthJoin != nil {
		listSpec.AuthJoin = []*LeftJoin{spec.AuthJoin}
	}

	lister, err := NewLister(listSpec)
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

	eventsAuthJoin := []*LeftJoin{{
		// Main is the events table, joining to the state table
		TableName: smSpec.EventTable,
		On:        eventJoinMap.Reverse(),
	}}

	if spec.AuthJoin != nil {
		eventsAuthJoin = append(eventsAuthJoin, spec.AuthJoin)
	}

	eventListSpec := ListSpec[ListEventsREQ, ListEventsRES]{
		TableName:  smSpec.EventTable,
		DataColumn: smSpec.EventDataColumn,
		Auth:       spec.AuthFunc,
		AuthJoin:   eventsAuthJoin,
	}
	eventLister, err := NewLister(eventListSpec)
	if err != nil {
		return nil, fmt.Errorf("build event lister for state query '%s' lister: %w", smSpec.EventTable, err)
	}

	querySet.eventLister = eventLister

	return querySet, nil
}
