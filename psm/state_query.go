package psm

import (
	"context"
	"fmt"

	"github.com/pentops/protostate/internal/pgstore"
	"github.com/pentops/protostate/pquery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// QueryTableSpec the TableMap with descriptors for the messages, without using
// generic parameters.
type QueryTableSpec struct {
	EventType protoreflect.MessageDescriptor
	StateType protoreflect.MessageDescriptor
	TableMap
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

type TenantFilterProvider interface {
	GetRequiredTenantKeys(ctx context.Context) (map[string]string, error)
}

type TenantFilterProviderFunc func(ctx context.Context) (map[string]string, error)

func (f TenantFilterProviderFunc) GetRequiredTenantKeys(ctx context.Context) (map[string]string, error) {
	return f(ctx)
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

	eventJoinMap := pquery.JoinFields{}
	requestReflect := (*new(GetREQ)).ProtoReflect().Descriptor()

	unmappedRequestFields := map[protoreflect.Name]protoreflect.FieldDescriptor{}
	reqFields := requestReflect.Fields()
	for i := 0; i < reqFields.Len(); i++ {
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
		fieldNames := make([]string, 0, len(unmappedRequestFields))
		for key := range unmappedRequestFields {
			fieldNames = append(fieldNames, string(key))
		}
		return nil, fmt.Errorf("unmapped fields in Get request: %v", fieldNames)
	}

	getSpec := pquery.GetSpec[GetREQ, GetRES]{
		TableName:  smSpec.State.TableName,
		DataColumn: smSpec.State.Root.ColumnName,
	}

	if options.AuthJoin != nil {
		getSpec.AuthJoin = []*pquery.LeftJoin{options.AuthJoin}
	}

	if options.Auth != nil {
		getSpec.Auth = options.Auth
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

		if msg.FullName() == smSpec.EventType.FullName() {
			eventsInGet = field.Name()
		} else if msg.FullName() == smSpec.StateType.FullName() {
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
			FallbackSortColumns: statePrimaryKeys,
			Auth:                getSpec.Auth,
			AuthJoin:            getSpec.AuthJoin,
		},
		RequestFilter: smSpec.ListRequestFilter,
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

	eventListSpec := pquery.ListSpec[ListEventsREQ, ListEventsRES]{
		TableSpec: pquery.TableSpec{
			TableName:  smSpec.Event.TableName,
			DataColumn: smSpec.Event.Root.ColumnName,
			Auth:       getSpec.Auth,
			AuthJoin:   getSpec.AuthJoin,
			FallbackSortColumns: []pgstore.ProtoFieldSpec{{
				ColumnName: smSpec.Event.ID.ColumnName,
			}},
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
