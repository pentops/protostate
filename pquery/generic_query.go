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

type MethodDescriptor struct {
	Request  protoreflect.MessageDescriptor
	Response protoreflect.MessageDescriptor
}

type StateQuerySpec struct {
	TableName        string
	DataColumn       string
	PrimaryKeyColumn string

	PrimaryKeyRequestField protoreflect.Name
	StateResponseField     protoreflect.Name

	Get        *MethodDescriptor
	List       *MethodDescriptor
	ListEvents *MethodDescriptor

	Auth     AuthProvider
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

	if gc.Events != nil {
		if err := gc.Events.validate(); err != nil {
			return err
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
		return nil, err
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
		return nil, fmt.Errorf("failed to build getter: %w", err)
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
		return nil, fmt.Errorf("failed to build lister: %w", err)
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
			return nil, fmt.Errorf("failed to build lister: %w", err)
		}

	}
	return ss, nil
}
