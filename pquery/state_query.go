package pquery

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.daemonl.com/sqrlx"
)

type aliasSet int

func (as *aliasSet) Next() string {
	*as++
	return fmt.Sprintf("_gc_alias_%d", *as)
}

func newAliasSet() *aliasSet {
	return new(aliasSet)
}

type Transactor interface {
	Transact(ctx context.Context, opts *sqrlx.TxOptions, callback sqrlx.Callback) error
}

type AuthProvider interface {
	AuthFilter(ctx context.Context) (map[string]interface{}, error)
}

type AuthProviderFunc func(ctx context.Context) (map[string]interface{}, error)

func (f AuthProviderFunc) AuthFilter(ctx context.Context) (map[string]interface{}, error) {
	return f(ctx)
}

// MethodDescriptor is the RequestResponse pair in the gRPC Method
type MethodDescriptor struct {
	Request  protoreflect.MessageDescriptor
	Response protoreflect.MessageDescriptor
}

type StateQuerySpec struct {
	// The table of the main entity
	TableName string

	// The JSONB column containing the state
	DataColumn string

	// The column containing the primary key, UUID, of the main entity
	PrimaryKeyColumn string

	// The field in the request message containing the primary key for Get
	// requests
	PrimaryKeyRequestField protoreflect.Name

	// The field in the response message containing the state, defaults to
	// 'state' but usually should be the name of the entity
	StateResponseField protoreflect.Name

	// The gRPC Method pair for the Get request. Required
	Get *MethodDescriptor

	// The gRPC Method pair for the List request. Required
	List *MethodDescriptor

	// The gRPC Method pair for the ListEvents request. Optional
	ListEvents *MethodDescriptor

	// A function to filter the entities to the current user
	Auth AuthProvider

	// If the map which AuthProvider returns does not directly map to the main
	// table, this is a join to the main table. All filters will then be applied
	// to the joined table instead.
	AuthJoin *LeftJoin

	Events      *GetJoinSpec
	EventsInGet bool
}

func (gc StateQuerySpec) validate() error {
	if gc.TableName == "" {
		return fmt.Errorf("missing TableName")
	}
	if gc.DataColumn == "" {
		return fmt.Errorf("missing DataColumn")
	}
	if gc.PrimaryKeyColumn == "" {
		return fmt.Errorf("missing PrimaryKeyColumn")
	}
	if gc.PrimaryKeyRequestField == "" {
		return fmt.Errorf("missing PrimaryKeyRequestField")
	}

	if gc.Get == nil {
		return fmt.Errorf("missing Get")
	}
	if gc.List == nil {
		return fmt.Errorf("missing List")
	}

	if gc.ListEvents != nil {
		if gc.Events == nil {
			return fmt.Errorf("missing Events, but ListEvents is specified")
		}
	}

	if gc.Events != nil {
		if gc.Events.FieldInParent == "" {
			gc.Events.FieldInParent = protoreflect.Name("events")
		}
		if err := gc.Events.validate(); err != nil {
			return fmt.Errorf("Events: %w", err)
		}
	}

	return nil
}

// StateQuerySet is a shortcut for manually specifying three different query
// types following the 'standard model':
// 1. A getter for a single state
// 2. A lister for the main state
// 3. A lister for the events of the main state
type StateQuerySet struct {
	getter      *Getter
	mainLister  *Lister
	eventLister *Lister
}

func (gc *StateQuerySet) Get(ctx context.Context, db Transactor, reqMsg proto.Message, resMsg proto.Message) error {
	return gc.getter.Get(ctx, db, reqMsg, resMsg)
}

func (gc *StateQuerySet) List(ctx context.Context, db Transactor, reqMsg proto.Message, resMsg proto.Message) error {
	return gc.mainLister.List(ctx, db, reqMsg, resMsg)
}

func (gc *StateQuerySet) ListEvents(ctx context.Context, db Transactor, reqMsg proto.Message, resMsg proto.Message) error {
	return gc.eventLister.List(ctx, db, reqMsg, resMsg)
}

func NewStateQuery(spec StateQuerySpec) (*StateQuerySet, error) {
	if err := spec.validate(); err != nil {
		return nil, fmt.Errorf("state query for '%s' was invalid: %w", spec.TableName, err)
	}

	getSpec := GetSpec{
		TableName:              spec.TableName,
		DataColumn:             spec.DataColumn,
		Auth:                   spec.Auth,
		Method:                 spec.Get,
		PrimaryKeyColumn:       spec.PrimaryKeyColumn,
		PrimaryKeyRequestField: spec.PrimaryKeyRequestField,
		StateResponseField:     spec.StateResponseField,
		AuthJoin:               spec.AuthJoin,
	}
	if spec.EventsInGet {
		getSpec.Join = spec.Events
	}
	getter, err := NewGetter(getSpec)
	if err != nil {
		return nil, fmt.Errorf("build getter for state query '%s': %w", spec.TableName, err)
	}

	listSpec := ListSpec{
		TableName:  spec.TableName,
		DataColumn: spec.DataColumn,
		Auth:       spec.Auth,
		Method:     spec.List,
	}
	if spec.AuthJoin != nil {
		listSpec.AuthJoin = []*LeftJoin{spec.AuthJoin}
	}
	lister, err := NewLister(listSpec)
	if err != nil {
		return nil, fmt.Errorf("build main lister for state query '%s': %w", spec.TableName, err)
	}

	ss := &StateQuerySet{
		getter:     getter,
		mainLister: lister,
	}

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
}
