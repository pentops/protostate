package psm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/protostate/dbconvert"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type TableSpec[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] struct {
	PrimaryKey        func(E) (map[string]interface{}, error)
	ExtraStateColumns func(S) (map[string]interface{}, error)

	StateTable string
	EventTable string

	StateDataColumn string // default: state
	EventDataColumn string // default: data
	EventColumns    func(E) (map[string]interface{}, error)
}

func (spec TableSpec[S, ST, E, IE]) Validate() error {
	if spec.PrimaryKey == nil {
		return fmt.Errorf("missing PrimaryKey func")
	}

	if spec.EventColumns == nil {
		return fmt.Errorf("missing EventColumns func")
	}

	if spec.StateTable == "" {
		return fmt.Errorf("missing StateTable func")
	}

	if spec.EventTable == "" {
		return fmt.Errorf("missing EventTable func")
	}

	return nil
}

type QuerySpec struct {
	StateTable      string
	StateDataColumn string

	EventTable      string
	EventDataColumn string

	EventTypeName protoreflect.FullName
	StateTypeName protoreflect.FullName
}

func (spec TableSpec[S, ST, E, IE]) QuerySpec() QuerySpec {
	spec.setDefaults()
	return QuerySpec{
		StateTable:      spec.StateTable,
		StateDataColumn: spec.StateDataColumn,
		EventTable:      spec.EventTable,
		EventDataColumn: spec.EventDataColumn,

		EventTypeName: (*new(E)).ProtoReflect().Descriptor().FullName(),
		StateTypeName: (*new(S)).ProtoReflect().Descriptor().FullName(),
	}
}

func (spec *TableSpec[S, ST, E, IE]) setDefaults() {
	if spec.StateDataColumn == "" {
		spec.StateDataColumn = "state"
	}
	if spec.EventDataColumn == "" {
		spec.EventDataColumn = "data"
	}
}

type EventTypeConverter[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] interface {
	Unwrap(E) IE
	StateLabel(S) string
	EventLabel(IE) string
	EmptyState(E) S
	CheckStateKeys(S, E) error
}

type Transactor interface {
	Transact(context.Context, *sqrlx.TxOptions, sqrlx.Callback) error
}

type StateMachine[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] struct {
	db          Transactor
	spec        TableSpec[S, ST, E, IE]
	conversions EventTypeConverter[S, ST, E, IE]
	*Eventer[S, ST, E, IE]
}

// stateMachineConfigBuilder allows the generated code to build a default
// machine, but expose options to the user to override the defaults
type stateMachineConfigBuilder[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
] struct {
	conversions EventTypeConverter[S, ST, E, IE]
	spec        TableSpec[S, ST, E, IE]
}

func WithTableSpec[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
](
	spec TableSpec[S, ST, E, IE],
) func(*stateMachineConfigBuilder[S, ST, E, IE]) {
	return func(cb *stateMachineConfigBuilder[S, ST, E, IE]) {
		cb.spec = spec
	}
}

func WithEventTypeConverter[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
](
	conversions EventTypeConverter[S, ST, E, IE],
) func(*stateMachineConfigBuilder[S, ST, E, IE]) {
	return func(cb *stateMachineConfigBuilder[S, ST, E, IE]) {
		cb.conversions = conversions
	}
}

func WithExtraStateColumns[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
](
	extraStateColumns func(S) (map[string]interface{}, error),
) func(*stateMachineConfigBuilder[S, ST, E, IE]) {
	return func(cb *stateMachineConfigBuilder[S, ST, E, IE]) {
		cb.spec.ExtraStateColumns = extraStateColumns
	}
}

type StateMachineOption[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
] func(*stateMachineConfigBuilder[S, ST, E, IE])

func NewStateMachine[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
](
	db Transactor,
	defaultConversions EventTypeConverter[S, ST, E, IE],
	defaultSpec TableSpec[S, ST, E, IE],
	options ...StateMachineOption[S, ST, E, IE],
) (*StateMachine[S, ST, E, IE], error) {

	cb := &stateMachineConfigBuilder[S, ST, E, IE]{
		conversions: defaultConversions,
		spec:        defaultSpec,
	}

	for _, option := range options {
		option(cb)
	}

	if err := cb.spec.Validate(); err != nil {
		return nil, err
	}

	(&cb.spec).setDefaults()

	ee := &Eventer[S, ST, E, IE]{
		UnwrapEvent: cb.conversions.Unwrap,
		StateLabel:  cb.conversions.StateLabel,
		EventLabel:  cb.conversions.EventLabel,
	}

	return &StateMachine[S, ST, E, IE]{
		db:          db,
		spec:        cb.spec,
		conversions: cb.conversions,
		Eventer:     ee,
	}, nil
}

func (sm *StateMachine[S, ST, E, IE]) getCurrentState(ctx context.Context, tx sqrlx.Transaction, event E) (S, error) {
	state := sm.conversions.EmptyState(event)

	primaryKey, err := sm.spec.PrimaryKey(event)
	if err != nil {
		return state, fmt.Errorf("primary key: %w", err)
	}

	selectQuery := sq.
		Select(sm.spec.StateDataColumn).
		From(sm.spec.StateTable)
	for k, v := range primaryKey {
		selectQuery = selectQuery.Where(sq.Eq{k: v})
	}

	var stateJSON []byte
	err = tx.SelectRow(ctx, selectQuery).Scan(&stateJSON)
	if errors.Is(err, sql.ErrNoRows) {
		// OK, leave empty state alone
		return state, nil
	}
	if err != nil {
		qq, _, _ := selectQuery.ToSql()
		return state, fmt.Errorf("selecting current state (%s): %w", qq, err)
	}

	if err := protojson.Unmarshal(stateJSON, state); err != nil {
		return state, err
	}

	if err := sm.conversions.CheckStateKeys(state, event); err != nil {
		return state, err
	}

	return state, nil
}

func (sm *StateMachine[S, ST, E, IE]) GetQuerySpec() QuerySpec {
	return sm.spec.QuerySpec()
}

func (sm *StateMachine[S, ST, E, IE]) store(
	ctx context.Context,
	tx sqrlx.Transaction,
	state S,
	event E,
) error {

	stateSpec := sm.spec

	eventMap, err := sm.spec.EventColumns(event)
	if err != nil {
		return fmt.Errorf("event columns: %w", err)
	}
	eventMap[stateSpec.EventDataColumn] = event

	primaryKey, err := stateSpec.PrimaryKey(event)
	if err != nil {
		return fmt.Errorf("primary key: %w", err)
	}

	additionalColumns := map[string]interface{}{}
	if stateSpec.ExtraStateColumns != nil {
		additionalColumns, err = stateSpec.ExtraStateColumns(state)
		if err != nil {
			return fmt.Errorf("extra state columns: %w", err)
		}
	}

	stateJSON, err := protojson.Marshal(state)
	if err != nil {
		return err
	}

	stateKeyMap, err := dbconvert.FieldsToDBValues(primaryKey)
	if err != nil {
		return fmt.Errorf("failed to map state primary key to DB values: %w", err)
	}

	stateSetMap, err := dbconvert.FieldsToDBValues(additionalColumns)
	if err != nil {
		return fmt.Errorf("failed to map state fields to DB values: %w", err)
	}

	_, err = tx.Insert(ctx, sqrlx.Upsert(sm.spec.StateTable).
		KeyMap(stateKeyMap).
		SetMap(stateSetMap).
		Set(sm.spec.StateDataColumn, stateJSON))

	if err != nil {
		return fmt.Errorf("upsert state: %w", err)
	}

	eventSetMap, err := dbconvert.FieldsToDBValues(eventMap)
	if err != nil {
		return fmt.Errorf("failed to map event fields to DB values: %w", err)
	}

	_, err = tx.Insert(ctx, sq.Insert(sm.spec.EventTable).SetMap(eventSetMap))
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil

}

func (sm *StateMachine[S, ST, E, IE]) runTx(ctx context.Context, tx sqrlx.Transaction, event E) (S, error) {

	state, err := sm.getCurrentState(ctx, tx, event)
	if err != nil {
		return state, err
	}

	if err := sm.Eventer.Run(ctx, NewSqrlxTransaction[S, E](tx, sm.store), state, event); err != nil {
		return state, fmt.Errorf("run event: %w", err)
	}

	return state, nil
}

// TransitionInTx uses an existing transaction to transition the state machine.
func (sm *StateMachine[S, ST, E, IE]) TransitionInTx(ctx context.Context, tx sqrlx.Transaction, event E) (S, error) {
	return sm.runTx(ctx, tx, event)
}

// Transition transitions the state machine in a new transaction from the state
// machine's database pool
func (sm *StateMachine[S, ST, E, IE]) Transition(ctx context.Context, event E) (S, error) {
	var state S
	if err := sm.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		stateOut, err := sm.runTx(ctx, tx, event)
		if err != nil {
			return err
		}
		state = stateOut
		return nil
	}); err != nil {
		return state, err
	}

	return state, nil
}
