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
	AuthFunc         AuthProvider
	AuthJoin         *LeftJoin
	GetMethod        *MethodDescriptor[GetREQ, GetRES]
	ListMethod       *MethodDescriptor[ListREQ, ListRES]
	ListEventsMethod *MethodDescriptor[ListEventsREQ, ListEventsRES]
	EventsInGet      bool
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
		DataColumn: smSpec.DataColumn,
		Auth:       spec.AuthFunc,
		AuthJoin:   spec.AuthJoin,
		Method:     spec.GetMethod,
	}

	pkFields := map[string]protoreflect.FieldDescriptor{}
	eventJoinMap := map[string]string{}
	getReflect := getSpec.Method.Request.ProtoReflect().Descriptor()
	for i := 0; i < getReflect.Fields().Len(); i++ {
		field := getReflect.Fields().Get(i)
		fullKey := string(field.Name())
		rootKey := strings.TrimPrefix(fullKey, smSpec.StateTable+"_")
		pkFields[rootKey] = field
		eventJoinMap[fullKey] = rootKey
	}

	getSpec.PrimaryKey = func(req GetREQ) (map[string]interface{}, error) {
		refl := req.ProtoReflect()
		out := map[string]interface{}{}
		for k, v := range pkFields {
			out[k] = refl.Get(v).Interface()
		}
		return out, nil
	}

	if spec.EventsInGet {
		getSpec.Join = &GetJoinSpec{
			TableName:     smSpec.EventTable,
			DataColumn:    "data",   // TODO: make this configurable
			FieldInParent: "events", // TODO: look this up in the proto
			On:            eventJoinMap,
		}
	}

	getter, err := NewGetter(getSpec)
	if err != nil {
		return nil, fmt.Errorf("build getter for state query '%s': %w", smSpec.StateTable, err)
	}

	listSpec := ListSpec[ListREQ, ListRES]{
		TableName:  smSpec.StateTable,
		DataColumn: smSpec.DataColumn,
		Auth:       spec.AuthFunc,
		Method:     spec.ListMethod,
	}
	if spec.AuthJoin != nil {
		listSpec.AuthJoin = []*LeftJoin{spec.AuthJoin}
	}

	lister, err := NewLister(listSpec)
	if err != nil {
		return nil, fmt.Errorf("build main lister for state query '%s': %w", smSpec.EventTable, err)
	}

	/*
		if spec.Events != nil {
			eventsAuthJoin := []*LeftJoin{{
				// Main is the events table, joining to the state table
				TableName:     spec.TableName,
				MainKeyColumn: spec.Events.ForeignKeyColumn,
				JoinKeyColumn: spec.PrimaryKeyColumn,
			}}

			if spec.AuthJoin != nil {
				eventsAuthJoin = append(eventsAuthJoin, spec.AuthJoin)
			}

			ss.eventLister, err = NewLister(ListSpec{
				TableName:  spec.Events.TableName,
				DataColumn: spec.Events.DataColumn,
				Auth:       spec.Auth,
				Method:     spec.ListEvents,
				AuthJoin:   eventsAuthJoin,
			})
			if err != nil {
				return nil, fmt.Errorf("build event lister for state query '%s' lister: %w", spec.TableName, err)
			}

		}
		return ss, nil
	*/

	return &StateQuerySet[GetREQ, GetRES, ListREQ, ListRES, ListEventsREQ, ListEventsRES]{
		getter:     getter,
		mainLister: lister,
	}, nil
}
