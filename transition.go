package genericstate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.daemonl.com/sqrlx"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/genericstate/sm"
)

type StateSpec[
	S sm.IState[ST], // Outer State Entity
	ST sm.IStatusEnum, // Status Enum in State Entity
	E sm.IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE any, // Inner Event, the typed event
] struct {
	Transition func(ctx context.Context, state S, event E) error

	EmptyState        func(E) S
	EventData         func(E) proto.Message
	EventMetadata     func(E) *Metadata
	PrimaryKey        func(E) map[string]interface{}
	AdditionalColumns func(E) map[string]interface{}
	EventForeignKey   func(E) map[string]interface{}

	StateTable string
	EventTable string
}

type StateMachine[
	S sm.IState[Status], // Outer State Entity
	Status sm.IStatusEnum, // Status Enum in State Entity
	E sm.IEvent[InnerEvent], // Event Wrapper, with IDs and Metadata
	InnerEvent any, // Inner Event, the typed event
] struct {
	db   *sqrlx.Wrapper
	spec StateSpec[S, Status, E, InnerEvent]
	*sm.Eventer[S, Status, E, InnerEvent]
}

func NewStateMachine[
	S sm.IState[ST], // Outer State Entity
	ST sm.IStatusEnum, // Status Enum in State Entity
	E sm.IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE any, // Inner Event, the typed event
](db *sqrlx.Wrapper, spec StateSpec[S, ST, E, IE]) (*StateMachine[S, ST, E, IE], error) {

	if spec.Transition == nil {
		return nil, fmt.Errorf("missing Transition func")
	}

	if spec.EmptyState == nil {
		return nil, fmt.Errorf("missing EmptyState func")
	}

	if spec.EventData == nil {
		return nil, fmt.Errorf("missing EventData func")
	}

	if spec.EventMetadata == nil {
		return nil, fmt.Errorf("missing EventMetadata func")
	}

	if spec.PrimaryKey == nil {
		return nil, fmt.Errorf("missing PrimaryKey func")
	}

	// AdditionalColumns may be nil

	if spec.EventForeignKey == nil {
		return nil, fmt.Errorf("missing EventForeignKey func")
	}

	if spec.StateTable == "" {
		return nil, fmt.Errorf("missing StateTable func")
	}

	if spec.EventTable == "" {
		return nil, fmt.Errorf("missing EventTable func")
	}

	ee := &sm.Eventer[S, ST, E, IE]{
		WrapEvent:   func(state S, event IE) E {},
		UnwrapEvent: func(event E) IE {},
		StateLabel: func(state S) string {
			return state.GetStatus().ShortString()
		},
		EventLabel: func(event IE) string {
			pr := proto.MessageReflect(event)
		},
	}

	return &StateMachine[S, ST, E, IE]{
		db:      db,
		spec:    spec,
		Eventer: ee,
	}, nil
}

func (sm *StateMachine[S, ST, E, IE]) getCurrentState(ctx context.Context, tx sqrlx.Transaction, prepared *preparedTransition[E, S]) (S, error) {

	selectQuery := sq.Select("state").From(sm.spec.StateTable)
	for k, v := range prepared.primaryKey {
		selectQuery = selectQuery.Where(sq.Eq{k: v})
	}

	state := prepared.state

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

func (sm *StateMachine[S, ST, E, IE]) storeEvent(ctx context.Context, tx sqrlx.Transaction, prepared *preparedTransition[E, S]) error {

	stateJSON, err := protojson.Marshal(prepared.state)
	if err != nil {
		return err
	}

	_, err = tx.Insert(ctx, sqrlx.Upsert(sm.spec.StateTable).
		KeyMap(prepared.primaryKey).
		SetMap(prepared.additionalColumns).
		Set("state", stateJSON))

	if err != nil {
		return fmt.Errorf("upsert state: %w", err)
	}

	_, err = tx.Insert(ctx, sq.Insert(sm.spec.EventTable).SetMap(prepared.eventMap))
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

type preparedTransition[E proto.Message, S proto.Message] struct {
	primaryKey        map[string]interface{}
	additionalColumns map[string]interface{}
	eventMap          map[string]interface{}

	actorJSON []byte
	eventJSON []byte

	event E
	state S
}

type Metadata struct {
	Actor     proto.Message
	Timestamp time.Time
	EventID   string
}

func (sm *StateMachine[S, ST, E, IE]) prepare(ctx context.Context, eventData E) (*preparedTransition[E, S], error) {
	metadata := sm.spec.EventMetadata(eventData)

	stateSpec := sm.spec

	eventJSON, err := protojson.Marshal(stateSpec.EventData(eventData))
	if err != nil {
		return nil, err
	}

	actorJSON, err := protojson.Marshal(metadata.Actor)
	if err != nil {
		return nil, err
	}

	eventMap := map[string]interface{}{
		"id":        metadata.EventID,
		"timestamp": metadata.Timestamp,
		"event":     eventJSON,
		"actor":     actorJSON,
	}

	primaryKey := stateSpec.PrimaryKey(eventData)

	additionalColumns := map[string]interface{}{}
	if stateSpec.AdditionalColumns != nil {
		additionalColumns = stateSpec.AdditionalColumns(eventData)
	}

	for k, v := range additionalColumns {
		eventMap[k] = v
	}

	for k, v := range stateSpec.EventForeignKey(eventData) {
		eventMap[k] = v
	}

	state := stateSpec.EmptyState(eventData)

	prepared := &preparedTransition[E, S]{
		primaryKey:        primaryKey,
		additionalColumns: additionalColumns,
		eventMap:          eventMap,
		actorJSON:         actorJSON,
		eventJSON:         eventJSON,
		event:             eventData,
		state:             state,
	}

	return prepared, nil
}

func (pt *preparedTransition[E, S]) FinalState() S {
	return pt.state
}

func (sm *StateMachine[S, ST, E, IE]) runTx(ctx context.Context, tx sqrlx.Transaction, prepared *preparedTransition[E, S]) error {

	var err error
	prepared.state, err = sm.getCurrentState(ctx, tx, prepared)
	if err != nil {
		return err
	}

	if err := sm.spec.Transition(ctx, prepared.state, prepared.event); err != nil {
		return fmt.Errorf("state transition: %w", err)
	}

	if err := sm.storeEvent(ctx, tx, prepared); err != nil {
		return fmt.Errorf("store event: %w", err)
	}

	return nil
}

func (sm *StateMachine[S, ST, E, IE]) PrepareTransition(ctx context.Context, eventData E) (*preparedTransition[E, S], error) {
	return sm.prepare(ctx, eventData)
}

func (sm *StateMachine[S, ST, E, IE]) TransitionPrepared(ctx context.Context, tx sqrlx.Transaction, prepared *preparedTransition[E, S]) error {
	return sm.runTx(ctx, tx, prepared)
}

// TransitionInTx uses an existing transaction to transition the state machine.
func (sm *StateMachine[S, ST, E, IE]) TransitionInTx(ctx context.Context, tx sqrlx.Transaction, eventData E) error {
	prepared, err := sm.prepare(ctx, eventData)
	if err != nil {
		return err
	}
	return sm.runTx(ctx, tx, prepared)
}

// Transition transitions the state machine in a new transaction from the state
// machine's database pool
func (sm *StateMachine[S, ST, E, IE]) Transition(ctx context.Context, eventData E) error {

	prepared, err := sm.prepare(ctx, eventData)
	if err != nil {
		return err
	}

	return sm.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return sm.runTx(ctx, tx, prepared)
	})
}
