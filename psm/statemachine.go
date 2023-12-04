package psm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/elgris/sqrl"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.daemonl.com/sqrlx"
)

type Metadata struct {
	Actor     proto.Message
	Timestamp time.Time
	EventID   string
}

type StateSpec[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] struct {
	EmptyState        func(E) S
	PrimaryKey        func(E) map[string]interface{}
	AdditionalColumns func(E) map[string]interface{}
	EventForeignKey   func(E) map[string]interface{}

	Conversions Converter[S, ST, E, IE]

	StateTable string
	EventTable string
}

type Converter[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] interface {
	Wrap(context.Context, S, IE) E
	Unwrap(E) IE
	StateLabel(S) string
	EventLabel(IE) string
	EventMetadata(E) *Metadata
}

func (spec StateSpec[S, ST, E, IE]) Validate() error {

	if spec.EmptyState == nil {
		return fmt.Errorf("missing EmptyState func")
	}

	if spec.PrimaryKey == nil {
		return fmt.Errorf("missing PrimaryKey func")
	}

	// AdditionalColumns may be nil

	if spec.EventForeignKey == nil {
		return fmt.Errorf("missing EventForeignKey func")
	}

	if spec.StateTable == "" {
		return fmt.Errorf("missing StateTable func")
	}

	if spec.EventTable == "" {
		return fmt.Errorf("missing EventTable func")
	}

	return nil
}

type StateMachine[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] struct {
	db   *sqrlx.Wrapper
	spec StateSpec[S, ST, E, IE]
	*Eventer[S, ST, E, IE]
}

func NewStateMachine[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
](db *sqrlx.Wrapper, spec StateSpec[S, ST, E, IE]) (*StateMachine[S, ST, E, IE], error) {

	if err := spec.Validate(); err != nil {
		return nil, err
	}

	ee := &Eventer[S, ST, E, IE]{
		WrapEvent:   spec.Conversions.Wrap,
		UnwrapEvent: spec.Conversions.Unwrap,
		StateLabel:  spec.Conversions.StateLabel,
		EventLabel:  spec.Conversions.EventLabel,
	}

	return &StateMachine[S, ST, E, IE]{
		db:      db,
		spec:    spec,
		Eventer: ee,
	}, nil
}

func (sm *StateMachine[S, ST, E, IE]) getCurrentState(ctx context.Context, tx sqrlx.Transaction, event E) (S, error) {

	primaryKey := sm.spec.PrimaryKey(event)

	selectQuery := sq.Select("state").From(sm.spec.StateTable)
	for k, v := range primaryKey {
		selectQuery = selectQuery.Where(sq.Eq{k: v})
	}

	state := sm.spec.EmptyState(event)

	var stateJSON []byte
	err := tx.SelectRow(ctx, selectQuery).Scan(&stateJSON)
	if errors.Is(err, sql.ErrNoRows) {
		// OK, leave empty state alone
	} else if err != nil {
		return state, fmt.Errorf("selecting current state: %w", err)
	} else {
		if err := protojson.Unmarshal(stateJSON, state); err != nil {
			return state, err
		}
	}
	return state, nil
}

func (sm *StateMachine[S, ST, E, IE]) store(
	ctx context.Context,
	tx sqrlx.Transaction,
	state S,
	event E,
) error {
	metadata := sm.spec.Conversions.EventMetadata(event)

	stateSpec := sm.spec

	eventJSON, err := protojson.Marshal(event)
	if err != nil {
		return err
	}

	actorJSON, err := protojson.Marshal(metadata.Actor)
	if err != nil {
		return err
	}

	eventMap := map[string]interface{}{
		"id":        metadata.EventID,
		"timestamp": metadata.Timestamp,
		"event":     eventJSON,
		"actor":     actorJSON,
	}

	primaryKey := stateSpec.PrimaryKey(event)

	additionalColumns := map[string]interface{}{}
	if stateSpec.AdditionalColumns != nil {
		additionalColumns = stateSpec.AdditionalColumns(event)
	}

	for k, v := range additionalColumns {
		eventMap[k] = v
	}

	for k, v := range stateSpec.EventForeignKey(event) {
		eventMap[k] = v
	}

	stateJSON, err := protojson.Marshal(state)
	if err != nil {
		return err
	}

	_, err = tx.Insert(ctx, sqrlx.Upsert(sm.spec.StateTable).
		KeyMap(primaryKey).
		SetMap(additionalColumns).
		Set("state", stateJSON))

	if err != nil {
		return fmt.Errorf("upsert state: %w", err)
	}

	_, err = tx.Insert(ctx, sq.Insert(sm.spec.EventTable).SetMap(eventMap))
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
