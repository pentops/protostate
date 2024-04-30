package psm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/outbox.pg.go/outbox"
	"github.com/pentops/protostate/dbconvert"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// PSMTableSpec is the configuration for the state machine's table mapping.
// The generated code provides a default which is derived from the patterns and
// annotations on the proto message, however, the proto message is designed to
// specify the wire data, not the storage mechanism, so consuming code may need
// to override some of the defaults to map to the database.
// The generated default is called DefaultFooPSMTableSpec
type PSMTableSpec[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] struct {
	// Primary Key derives the *State* primary key, and thus event foreign key
	// to state, from the event.
	PrimaryKey func(E) (map[string]interface{}, error)

	State TableSpec[S]
	Event TableSpec[E]
}

type TableSpec[T proto.Message] struct {
	TableName         string
	StoreExtraColumns func(T) (map[string]interface{}, error)
	// DataColumn is the JSONB column which stores the main data chunk
	DataColumn   string
	PKFieldPaths []string
}

func (ts TableSpec[T]) storeDBMap(obj T) (map[string]interface{}, error) {
	var columnMap map[string]interface{}
	if ts.StoreExtraColumns != nil {
		var err error
		columnMap, err = ts.StoreExtraColumns(obj)
		if err != nil {
			return nil, fmt.Errorf("extra columns: %w", err)
		}
	}

	if columnMap == nil {
		columnMap = make(map[string]interface{})
	}

	columnMap[ts.DataColumn] = obj

	return dbconvert.FieldsToDBValues(columnMap)
}

// StateTableSpec derives the Query spec table elements from the StateMachine
// specs. The Query spec is a subset of the TableSpec
func (spec PSMTableSpec[S, ST, E, IE]) StateTableSpec() QueryTableSpec {
	return QueryTableSpec{
		State: EntityTableSpec{
			TableName:    spec.State.TableName,
			DataColumn:   spec.State.DataColumn,
			PKFieldPaths: spec.State.PKFieldPaths,
		},
		Event: EntityTableSpec{
			TableName:    spec.Event.TableName,
			DataColumn:   spec.Event.DataColumn,
			PKFieldPaths: spec.Event.PKFieldPaths,
		},
		EventTypeName: (*new(E)).ProtoReflect().Descriptor().FullName(),
		StateTypeName: (*new(S)).ProtoReflect().Descriptor().FullName(),
	}
}

func (spec PSMTableSpec[S, ST, E, IE]) Validate() error {
	if spec.PrimaryKey == nil {
		return fmt.Errorf("missing PrimaryKey func")
	}

	if spec.State.TableName == "" {
		return fmt.Errorf("missing StateTable func")
	}

	if spec.Event.TableName == "" {
		return fmt.Errorf("missing EventTable func")
	}

	return nil
}

// EventTypeConverter is implemented by the gnerated code, it could all
// be derived at runtime from proto reflect but the discovery process runs a lot of
// loops. This is faster.
type EventTypeConverter[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] interface {
	EmptyState(E) S
	CheckStateKeys(S, E) error

	DeriveChainEvent(template E, actor SystemActor, eventKey string) E
}

type Transactor interface {
	Transact(context.Context, *sqrlx.TxOptions, sqrlx.Callback) error
}

// StateMachine is a database wrapper around the eventer. Using sane defaults
// with overrides for table configuration.
type StateMachine[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] struct {
	spec        PSMTableSpec[S, ST, E, IE]
	conversions EventTypeConverter[S, ST, E, IE]
	*Eventer[S, ST, E, IE]
	SystemActor SystemActor

	hooks []IStateHook[S, ST, E, IE]
}

func (sm StateMachine[S, ST, E, IE]) StateTableSpec() QueryTableSpec {
	return sm.spec.StateTableSpec()
}

type SimpleSystemActor struct {
	ID    uuid.UUID
	Actor protoreflect.Value
}

func NewSystemActor(id string, actor proto.Message) (SimpleSystemActor, error) {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return SimpleSystemActor{}, fmt.Errorf("parsing id: %w", err)
	}
	actorValue := protoreflect.ValueOf(actor.ProtoReflect())
	return SimpleSystemActor{
		ID:    idUUID,
		Actor: actorValue,
	}, nil
}

func (sa SimpleSystemActor) NewEventID(fromEventUUID string, eventKey string) string {
	return uuid.NewMD5(sa.ID, []byte(fromEventUUID+eventKey)).String()
}

func (sa SimpleSystemActor) ActorProto() protoreflect.Value {
	return sa.Actor
}

type SystemActor interface {
	NewEventID(fromEventUUID string, eventKey string) string
	ActorProto() protoreflect.Value
}

func NewStateMachine[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
](
	cb *StateMachineConfig[S, ST, E, IE],
) (*StateMachine[S, ST, E, IE], error) {

	if err := cb.spec.Validate(); err != nil {
		return nil, err
	}

	ee := &Eventer[S, ST, E, IE]{
		conversions: cb.conversions,
	}

	return &StateMachine[S, ST, E, IE]{
		spec:        cb.spec,
		conversions: cb.conversions,
		Eventer:     ee,
		SystemActor: cb.systemActor,
	}, nil
}

func (sm *StateMachine[S, ST, E, IE]) WithDB(db Transactor) *DBStateMachine[S, ST, E, IE] {
	return &DBStateMachine[S, ST, E, IE]{
		StateMachine: sm,
		db:           db,
	}
}

func (sm *StateMachine[S, ST, E, IE]) getCurrentState(ctx context.Context, tx sqrlx.Transaction, event E) (S, error) {
	state := sm.conversions.EmptyState(event)

	primaryKey, err := sm.spec.PrimaryKey(event)
	if err != nil {
		return state, fmt.Errorf("primary key: %w", err)
	}

	selectQuery := sq.
		Select(sm.spec.State.DataColumn).
		From(sm.spec.State.TableName)
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

func (sm *StateMachine[S, ST, E, IE]) store(
	ctx context.Context,
	tx sqrlx.Transaction,
	state S,
	event E,
) error {
	var err error

	stateSpec := sm.spec

	stateSetMap, err := stateSpec.State.storeDBMap(state)
	if err != nil {
		return fmt.Errorf("state fields: %w", err)
	}

	primaryKey, err := stateSpec.PrimaryKey(event)
	if err != nil {
		return fmt.Errorf("primary key: %w", err)
	}

	stateKeyMap, err := dbconvert.FieldsToDBValues(primaryKey)
	if err != nil {
		return fmt.Errorf("failed to map state primary key to DB values: %w", err)
	}

	_, err = tx.Insert(ctx, sqrlx.
		Upsert(sm.spec.State.TableName).
		KeyMap(stateKeyMap).
		SetMap(stateSetMap))

	if err != nil {
		log.WithFields(ctx, map[string]interface{}{
			"stateKeyMap": stateKeyMap,
			"stateSetMap": stateSetMap,
			"error":       err.Error(),
		}).Error("failed to upsert state")
		return fmt.Errorf("upsert state: %w", err)
	}

	eventSetMap, err := stateSpec.Event.storeDBMap(event)
	if err != nil {
		return fmt.Errorf("event fields: %w", err)
	}

	_, err = tx.Insert(ctx, sq.
		Insert(sm.spec.Event.TableName).
		SetMap(eventSetMap))
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	return nil
}

func trySequence[S any, E any](state S, event E, isInitial bool) {
	stateSequencer, ok := any(state).(IStateSequencer)
	if !ok {
		return
	}
	eventSequencer, ok := any(event).(IEventSequencer)
	if !ok {
		return
	}

	seq := uint64(0)
	if !isInitial {
		seq = stateSequencer.LastPSMSequence() + 1
	}
	eventSequencer.SetPSMSequence(seq)
	stateSequencer.SetLastPSMSequence(seq)
}

func (sm *StateMachine[S, ST, E, IE]) runTx(ctx context.Context, tx sqrlx.Transaction, outerEvent E) (S, error) {
	state, err := sm.getCurrentState(ctx, tx, outerEvent)
	if err != nil {
		return state, err
	}

	isFirst := true
	var returnState S

	eventQueue := []E{outerEvent}

	isInitial := state.GetStatus() == 0

	for len(eventQueue) > 0 {
		innerEvent := eventQueue[0]
		eventQueue = eventQueue[1:]

		trySequence(state, innerEvent, isInitial)
		isInitial = false

		statusBefore := state.GetStatus()

		// runEvent modifies state in place
		err := sm.Eventer.RunEvent(ctx, state, innerEvent)
		if err != nil {
			return state, fmt.Errorf("event queue: %w", err)
		}

		if state.GetStatus() == 0 {
			return state, fmt.Errorf("state machine transitioned to zero status")
		}

		if err := sm.store(ctx, tx, state, innerEvent); err != nil {
			return state, err
		}

		if isFirst {
			// return the state after the first transition, not after all hooks
			// etc have run
			returnState = proto.Clone(state).(S)
			isFirst = false
		}

		chained, err := sm.runHooks(ctx, tx, statusBefore, state, innerEvent)
		if err != nil {
			return state, fmt.Errorf("run hooks: %w", err)
		}

		eventQueue = append(eventQueue, chained...)
	}

	return returnState, nil
}

func (sm *StateMachine[S, ST, E, IE]) AddHook(hook IStateHook[S, ST, E, IE]) {
	sm.hooks = append(sm.hooks, hook)
}

func (sm *StateMachine[S, ST, E, IE]) runHooks(ctx context.Context, tx sqrlx.Transaction, statusBefore ST, state S, event E) ([]E, error) {

	chain := []E{}

	for _, hook := range sm.hooks {

		if !hook.Matches(statusBefore, event) {
			continue
		}

		baton := &TransitionData[E, IE]{
			causedBy: event,
		}

		if err := hook.RunStateHook(ctx, tx, baton, state, event); err != nil {
			return nil, fmt.Errorf("run state hook: %w", err)
		}

		for _, se := range baton.sideEffects {
			if err := outbox.Send(ctx, tx, se); err != nil {
				return nil, fmt.Errorf("side effect outbox: %w", err)
			}
		}

		chain = append(chain, baton.chainEvents...)
		for _, chained := range baton.chainInnerEvents {
			derived, err := sm.deriveEvent(event, chained)
			if err != nil {
				return nil, fmt.Errorf("deriving event: %w", err)
			}
			chain = append(chain, derived)
		}
	}

	return chain, nil
}

// TransitionInTx uses an existing transaction to transition the state machine.
func (sm *StateMachine[S, ST, E, IE]) TransitionInTx(ctx context.Context, tx sqrlx.Transaction, events ...E) (S, error) {
	var state S
	var err error
	for _, event := range events {
		state, err = sm.runTx(ctx, tx, event)
		if err != nil {
			return state, err
		}
	}
	return state, nil
}

func (sm *StateMachine[S, ST, E, IE]) Transition(ctx context.Context, db Transactor, events ...E) (S, error) {
	return sm.WithDB(db).Transition(ctx, events...)
}

// deriveEvent returns a new event with metadata derived from the causing
// event and system actor
func (sm *StateMachine[S, ST, E, IE]) deriveEvent(cause E, inner IE) (evt E, err error) {
	if sm.SystemActor == nil {
		err = fmt.Errorf("no system actor defined, cannot derive events")
		return
	}
	eventKey := inner.PSMEventKey()
	derived := sm.conversions.DeriveChainEvent(cause, sm.SystemActor, eventKey)
	derived.SetPSMEvent(inner)
	return derived, nil
}

// DBStateMachine adds the 'Transaction' method to the state machine, which
// runs the transition in a new transaction from the state machine's database
type DBStateMachine[S IState[ST], ST IStatusEnum, E IEvent[IE], IE IInnerEvent] struct {
	*StateMachine[S, ST, E, IE]
	db Transactor
}

var TxOptions = &sqrlx.TxOptions{
	Isolation: sql.LevelReadCommitted,
	Retryable: true,
	ReadOnly:  false,
}

// Transition transitions the state machine in a new transaction from the state
// machine's database pool
func (sm *DBStateMachine[S, ST, E, IE]) Transition(ctx context.Context, events ...E) (S, error) {
	var state S
	opts := &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}

	err := sm.db.Transact(ctx, opts, func(ctx context.Context, tx sqrlx.Transaction) error {
		var err error
		for _, event := range events {
			state, err = sm.runTx(ctx, tx, event)
			if err != nil {
				return fmt.Errorf("run tx: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return state, err
	}

	return state, nil
}
