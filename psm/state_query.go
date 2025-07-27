package psm

import (
	"context"
	"fmt"

	"github.com/pentops/j5/lib/j5reflect"
	"github.com/pentops/j5/lib/j5schema"
	"github.com/pentops/protostate/pquery"
)

// QueryTableSpec the TableMap with descriptors for the messages, without using
// generic parameters.
type QueryTableSpec struct {
	EventType *j5schema.ObjectSchema
	StateType *j5schema.ObjectSchema
	TableMap
}

// QuerySpec is the configuration for the query service side of the state
// machine. Can be partially derived from the state machine table spec, but
// contains types relating to the query service so cannot be fully derived.
type QuerySpec struct {
	QueryTableSpec

	GetMethod        *j5schema.MethodSchema
	ListMethod       *j5schema.MethodSchema
	ListEventsMethod *j5schema.MethodSchema

	ListRequestFilter       func(j5reflect.Object) (map[string]any, error)
	ListEventsRequestFilter func(j5reflect.Object) (map[string]any, error)
}

func (qs *QuerySpec) Validate() error {
	err := qs.QueryTableSpec.Validate()
	if err != nil {
		return fmt.Errorf("validate QueryTableSpec: %w", err)
	}
	if qs.GetMethod == nil {
		return fmt.Errorf("missing GetMethod in QuerySpec")
	}
	if qs.ListMethod == nil {
		return fmt.Errorf("missing ListMethod in QuerySpec")
	}
	if qs.ListEventsMethod == nil {
		return fmt.Errorf("missing ListEventsMethod in QuerySpec")
	}
	return nil
}

// StateQuerySet is a shortcut for manually specifying three different query
// types following the 'standard model':
// 1. A getter for a single state
// 2. A lister for the main state
// 3. A lister for the events of the main state
type StateQuerySet struct {
	Getter      *pquery.Getter
	MainLister  *pquery.Lister
	EventLister *pquery.Lister
}

func (gc *StateQuerySet) SetQueryLogger(logger pquery.QueryLogger) {
	gc.Getter.SetQueryLogger(logger)
	gc.MainLister.SetQueryLogger(logger)
	gc.EventLister.SetQueryLogger(logger)
}

func (gc *StateQuerySet) Get(ctx context.Context, db Transactor, reqMsg, resMsg j5reflect.Object) error {
	return gc.Getter.Get(ctx, db, reqMsg, resMsg)
}

func (gc *StateQuerySet) List(ctx context.Context, db Transactor, reqMsg, resMsg j5reflect.Object) error {
	return gc.MainLister.List(ctx, db, reqMsg, resMsg)
}

func (gc *StateQuerySet) ListEvents(ctx context.Context, db Transactor, reqMsg, resMsg j5reflect.Object) error {
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

func BuildStateQuerySet(
	smSpec QuerySpec,
	options StateQueryOptions,
) (*StateQuerySet, error) {

	if err := smSpec.Validate(); err != nil {
		return nil, fmt.Errorf("validate state machine spec: %w", err)
	}

	requestReflect := smSpec.GetMethod.Request

	unmappedRequestFields := map[string]*j5schema.ObjectProperty{}
	for _, field := range requestReflect.Properties {
		unmappedRequestFields[field.JSONName] = field
	}

	getPkFields := []KeyColumn{}
	for _, keyColumn := range smSpec.KeyColumns {
		_, ok := unmappedRequestFields[keyColumn.JSONFieldName]
		if !ok {
			continue
		}
		delete(unmappedRequestFields, keyColumn.JSONFieldName)
		getPkFields = append(getPkFields, keyColumn)
	}

	if len(unmappedRequestFields) > 0 {
		fieldNames := make([]string, 0, len(unmappedRequestFields))
		for key := range unmappedRequestFields {
			fieldNames = append(fieldNames, string(key))
		}
		return nil, fmt.Errorf("unmapped fields in Get request: %v", fieldNames)
	}

	getSpec := pquery.GetSpec{
		Method:     smSpec.GetMethod,
		TableName:  smSpec.State.TableName,
		DataColumn: smSpec.State.Root.ColumnName,
	}

	if options.AuthJoin != nil {
		getSpec.AuthJoin = []*pquery.LeftJoin{options.AuthJoin}
	}

	if options.Auth != nil {
		getSpec.Auth = options.Auth
	}

	getSpec.PrimaryKey = func(req j5reflect.Object) (map[string]any, error) {
		out := map[string]any{}
		for _, col := range getPkFields {
			value, ok, err := req.GetField(col.JSONFieldName)
			if err != nil {
				return nil, fmt.Errorf("get primary key field '%s': %w", col.JSONFieldName, err)
			}
			if !ok {
				return nil, fmt.Errorf("missing primary key field '%s'", col.JSONFieldName)
			}
			scalarValue, ok := value.AsScalar()
			if !ok {
				return nil, fmt.Errorf("primary key field '%s' is not a scalar value", col.JSONFieldName)
			}
			out[col.ColumnName], err = scalarValue.ToGoValue()
			if err != nil {
				return nil, fmt.Errorf("convert primary key field '%s' to Go value: %w", col.JSONFieldName, err)
			}
		}
		return out, nil
	}

	var eventsInGet *j5schema.ObjectProperty

	getResponseReflect := smSpec.GetMethod.Response
	for _, field := range getResponseReflect.Properties {
		switch ft := field.Schema.(type) {
		case *j5schema.ObjectField:
			name := ft.ObjectSchema().FullName()
			if name == smSpec.StateType.FullName() {
				if getSpec.StateResponseField != "" {
					return nil, fmt.Errorf("multiple state fields in Get response: %s and %s", getSpec.StateResponseField, field.FullName())
				}
				getSpec.StateResponseField = field.JSONName
			} else {
				return nil, fmt.Errorf("unexpected object field in Get response: %s", name)
			}

		case *j5schema.ArrayField:
			itemSchema, ok := ft.ItemSchema.(*j5schema.ObjectField)
			if !ok {
				return nil, fmt.Errorf("expected array field in Get response, got: %T", ft.ItemSchema)
			}

			name := itemSchema.ObjectSchema().FullName()
			if name == smSpec.EventType.FullName() {
				if eventsInGet != nil {
					return nil, fmt.Errorf("multiple event fields in Get response: %s and %s", eventsInGet.FullName(), field.FullName())
				}
				eventsInGet = field
			} else {
				return nil, fmt.Errorf("unexpected array field in Get response: %s", name)
			}

		default:
			return nil, fmt.Errorf("unexpected field in Get response: %s", field.FullName())
		}
	}

	eventJoinMap := pquery.JoinFields{}
	for _, keyColumn := range smSpec.KeyColumns {
		if keyColumn.Primary {
			eventJoinMap = append(eventJoinMap, pquery.JoinField{
				RootColumn: keyColumn.ColumnName,
				JoinColumn: keyColumn.ColumnName,
			})
		}

	}

	if eventsInGet != nil {
		if smSpec.Event.TableName == "" {
			return nil, fmt.Errorf("missing EventTable in state spec for %s", smSpec.State.TableName)
		}
		if smSpec.Event.Root == nil {
			return nil, fmt.Errorf("missing EventDataColumn in state spec for %s", smSpec.State.TableName)
		}
		getSpec.Join = &pquery.GetJoinSpec{
			TableName:     smSpec.Event.TableName,
			DataColumn:    smSpec.Event.Root.ColumnName,
			FieldInParent: eventsInGet.JSONName,
			On:            eventJoinMap,
		}
	}

	getter, err := pquery.NewGetter(getSpec)
	if err != nil {
		return nil, fmt.Errorf("build getter for state query '%s': %w", smSpec.State.TableName, err)
	}

	statePrimaryKeys := []pquery.ProtoField{}

	for _, field := range smSpec.KeyColumns {
		if !field.Primary {
			continue
		}
		keyPath := field.JSONFieldName
		statePrimaryKeys = append(statePrimaryKeys,
			pquery.NewJSONField(keyPath, &field.ColumnName),
		)
	}

	listSpec := pquery.ListSpec{
		Method: smSpec.ListMethod,
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

	querySet := &StateQuerySet{
		Getter:     getter,
		MainLister: lister,
	}

	if options.SkipEvents {
		return querySet, nil
	}

	eventListSpec := pquery.ListSpec{
		Method: smSpec.ListEventsMethod,
		TableSpec: pquery.TableSpec{
			TableName:  smSpec.Event.TableName,
			DataColumn: smSpec.Event.Root.ColumnName,
			Auth:       getSpec.Auth,
			AuthJoin:   getSpec.AuthJoin,
			FallbackSortColumns: []pquery.ProtoField{
				pquery.NewJSONField("metadata.eventId", &smSpec.Event.ID.ColumnName),
			},
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
