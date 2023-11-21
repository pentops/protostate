package genericstate

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

	Get        *MethodDescriptor
	List       *MethodDescriptor
	ListEvents *MethodDescriptor

	Auth AuthProvider

	Events *JoinSpec
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

type JoinSpec struct {
	TableName        string
	DataColumn       string
	ForeignKeyColumn string

	FieldInParent protoreflect.Name
}

func (gc JoinSpec) validate() error {
	if gc.TableName == "" {
		return fmt.Errorf("missing TableName")
	}
	if gc.DataColumn == "" {
		return fmt.Errorf("missing DataColumn")
	}
	if gc.ForeignKeyColumn == "" {
		return fmt.Errorf("missing ForeignKeyColumn")
	}
	if gc.FieldInParent == "" {
		return fmt.Errorf("missing FieldInParent")
	}

	return nil
}

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

type join struct {
	Table            string
	DataColunn       string
	ForeignKeyColumn string

	fieldInParent  protoreflect.FieldDescriptor // wraps the ListFooEventResponse type
	eventListField protoreflect.FieldDescriptor // the events array inside the response
}

func NewStateQuery(spec StateQuerySpec) (*StateQuerySet, error) {
	if err := spec.validate(); err != nil {
		return nil, err
	}

	getter, err := NewGetter(GetSpec{
		TableName:              spec.TableName,
		DataColumn:             spec.DataColumn,
		Auth:                   spec.Auth,
		Method:                 spec.Get,
		Join:                   spec.Events,
		PrimaryKeyColumn:       spec.PrimaryKeyColumn,
		PrimaryKeyRequestField: spec.PrimaryKeyRequestField,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build getter: %w", err)
	}

	lister, err := NewLister(ListSpec{
		TableName:  spec.TableName,
		DataColumn: spec.DataColumn,
		Auth:       spec.Auth,
		Method:     spec.List,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build lister: %w", err)
	}

	ss := &StateQuerySet{
		getter:     getter,
		mainLister: lister,
	}

	if spec.Events != nil {
		ss.eventLister, err = NewLister(ListSpec{
			TableName:  spec.Events.TableName,
			DataColumn: spec.Events.DataColumn,
			Auth:       spec.Auth,
			Method:     spec.ListEvents,
			AuthJoin: &LeftJoin{
				// Main is the events table, joining to the state table
				TableName:     spec.TableName,
				MainKeyColumn: spec.Events.ForeignKeyColumn,
				JoinKeyColumn: spec.PrimaryKeyColumn,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to build lister: %w", err)
		}

	}
	return ss, nil
}

/*
func (gc *StateQuery) ListEvents(ctx context.Context, reqMsg proto.Message, resMsg proto.Message) error {

	ll := lister{
		db: gc.DB,
		selecQuery: func(ctx context.Context) (*sq.SelectBuilder, error) {

			selectQuery := sq.
				Select(fmt.Sprintf("%s.%s", EventAlias, gc.EventDataColumn)).
				From(fmt.Sprintf("%s AS %s", gc.StateTableName, StateAlias)).
				LeftJoin(fmt.Sprintf(
					"%s AS %s ON %s.%s = %s.%s",
					gc.EventTableName,
					EventAlias,
					EventAlias,
					gc.EventForeignKeyColumn,
					StateAlias,
					gc.PrimaryKeyColumn,
				))

			if gc.Auth != nil {

				authFilter, err := gc.Auth.AuthFilter(ctx)
				if err != nil {
					return nil, err
				}
				for k, v := range authFilter {
					selectQuery = selectQuery.Where(sq.Eq{fmt.Sprintf("%s.%s", StateAlias, k): v})
				}
			}
			return selectQuery, nil
		},
	}
	return ll.list(ctx, reqMsg, resMsg)
}

*/
