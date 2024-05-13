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
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PSMTableSpec is the configuration for the state machine's table mapping.
// The generated code provides a default which is derived from the patterns and
// annotations on the proto message, however, the proto message is designed to
// specify the wire data, not the storage mechanism, so consuming code may need
// to override some of the defaults to map to the database.
// The generated default is called DefaultFooPSMTableSpec
type PSMTableSpec[
	K IKeyset,
	S IState[K, ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[K, S, ST, IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] struct {
	// Primary Key derives the *State* primary key, and thus event foreign key
	// to state, from the event.
	PrimaryKey      func(K) (map[string]interface{}, error)
	EventPrimaryKey func(string, K) (map[string]interface{}, error)

	State TableSpec[S]
	Event TableSpec[E]

	// When set, stores the current state in the event table.
	EventStateSnapshotColumn *string
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
func (spec PSMTableSpec[K, S, ST, E, IE]) StateTableSpec() QueryTableSpec {
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

func (spec PSMTableSpec[K, S, ST, E, IE]) Validate() error {
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

type Transactor interface {
	Transact(context.Context, *sqrlx.TxOptions, sqrlx.Callback) error
}

// StateMachine is a database wrapper around the eventer. Using sane defaults
// with overrides for table configuration.
type StateMachine[
	K IKeyset,
	S IState[K, ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[K, S, ST, IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] struct {
	spec PSMTableSpec[K, S, ST, E, IE]
	*Eventer[K, S, ST, E, IE]
	SystemActor SystemActor

	hooks []IStateHook[K, S, ST, E, IE]
}

func (sm StateMachine[K, S, ST, E, IE]) StateTableSpec() QueryTableSpec {
	return sm.spec.StateTableSpec()
}

type SimpleSystemActor struct {
	ID    uuid.UUID
	Actor protoreflect.Value
}

func NewSystemActor(id string) (SimpleSystemActor, error) {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return SimpleSystemActor{}, fmt.Errorf("parsing id: %w", err)
	}
	return SimpleSystemActor{
		ID: idUUID,
	}, nil
}

func (sa SimpleSystemActor) NewEventID(fromEventUUID string, eventKey string) string {
	return uuid.NewMD5(sa.ID, []byte(fromEventUUID+eventKey)).String()
}

type SystemActor interface {
	NewEventID(fromEventUUID string, eventKey string) string
}

func NewStateMachine[
	K IKeyset,
	S IState[K, ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[K, S, ST, IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
](
	cb *StateMachineConfig[K, S, ST, E, IE],
) (*StateMachine[K, S, ST, E, IE], error) {

	if err := cb.spec.Validate(); err != nil {
		return nil, err
	}

	ee := &Eventer[K, S, ST, E, IE]{}

	return &StateMachine[K, S, ST, E, IE]{
		spec:        cb.spec,
		Eventer:     ee,
		SystemActor: cb.systemActor,
	}, nil
}

func (sm *StateMachine[K, S, ST, E, IE]) WithDB(db Transactor) *DBStateMachine[K, S, ST, E, IE] {
	return &DBStateMachine[K, S, ST, E, IE]{
		StateMachine: sm,
		db:           db,
	}
}

func (sm *StateMachine[K, S, ST, E, IE]) getCurrentState(ctx context.Context, tx sqrlx.Transaction, keys K) (S, error) {
	state := (*new(S)).ProtoReflect().New().Interface().(S)

	primaryKey, err := sm.spec.PrimaryKey(keys)
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
		state.SetPSMKeys(proto.Clone(keys).(K))
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

	stateKeyset := state.PSMKeys()
	if !proto.Equal(keys, stateKeyset) {
		return state, fmt.Errorf("event and state keysets do not match")
	}

	return state, nil
}

func (sm *StateMachine[K, S, ST, E, IE]) store(
	ctx context.Context,
	tx sqrlx.Transaction,
	state S,
	event E,
) error {
	var err error

	keys := state.PSMKeys()

	stateSpec := sm.spec

	stateSetMap, err := stateSpec.State.storeDBMap(state)
	if err != nil {
		return fmt.Errorf("state fields: %w", err)
	}

	primaryKey, err := stateSpec.PrimaryKey(keys)
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

	if stateSpec.EventStateSnapshotColumn != nil {
		columnName := *stateSpec.EventStateSnapshotColumn
		eventSetMap[columnName] = stateSetMap[stateSpec.State.DataColumn]
	}

	_, err = tx.Insert(ctx, sq.
		Insert(sm.spec.Event.TableName).
		SetMap(eventSetMap))
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	return nil
}

func (sm *StateMachine[K, S, ST, E, IE]) eventQuery(ctx context.Context, tx sqrlx.Transaction, event *EventSpec[K, S, ST, E, IE]) (*sq.SelectBuilder, error) {
	primaryKey, err := sm.spec.EventPrimaryKey(event.EventID, event.Keys)
	if err != nil {
		return nil, fmt.Errorf("primary key: %w", err)
	}

	pkMap, err := dbconvert.FieldsToDBValues(primaryKey)
	if err != nil {
		return nil, fmt.Errorf("failed to map event primary key to DB values: %w", err)
	}

	selectQuery := sq.
		Select(sm.spec.Event.DataColumn).
		From(sm.spec.Event.TableName).
		Where(sq.Eq(pkMap))

	return selectQuery, nil
}

// firstEventUniqueCheck checks if the event ID for the outer triggering event
// is unique in the event table. If not, it checks if the event is a repeat
// processing of the same event, and returns the state after the initial
// transition.
func (sm *StateMachine[K, S, ST, E, IE]) firstEventUniqueCheck(ctx context.Context, tx sqrlx.Transaction, event *EventSpec[K, S, ST, E, IE]) (S, bool, error) {
	var s S
	selectQuery, err := sm.eventQuery(ctx, tx, event)
	if err != nil {
		return s, false, fmt.Errorf("event query: %w", err)
	}
	if sm.spec.EventStateSnapshotColumn == nil {
		return s, false, fmt.Errorf("no snapshot column defined")
	}

	selectQuery.Column(*sm.spec.EventStateSnapshotColumn)

	var eventData, stateData []byte
	err = tx.SelectRow(ctx, selectQuery).Scan(&eventData, &stateData)
	if errors.Is(sql.ErrNoRows, err) {
		return s, false, nil
	}
	if err != nil {
		return s, false, fmt.Errorf("selecting event: %w", err)
	}

	existing := (*new(E)).ProtoReflect().New().Interface().(E)

	if err := protojson.Unmarshal(eventData, existing); err != nil {
		return s, false, fmt.Errorf("unmarshalling event: %w", err)
	}

	if !proto.Equal(existing.UnwrapPSMEvent(), event.Event) {
		return s, false, ErrDuplicateEventID
	}

	state := (*new(S)).ProtoReflect().New()
	if err := protojson.Unmarshal(stateData, state.Interface()); err != nil {
		return s, false, fmt.Errorf("unmarshalling state: %w", err)
	}

	return state.Interface().(S), true, nil
}

func (sm *StateMachine[K, S, ST, E, IE]) eventsMustBeUnique(ctx context.Context, tx sqrlx.Transaction, events ...*EventSpec[K, S, ST, E, IE]) error {
	for _, event := range events {
		if event.EventID == "" {
			continue // UUID Gen Later
		}
		selectQuery, err := sm.eventQuery(ctx, tx, event)
		if err != nil {
			return fmt.Errorf("event query: %w", err)
		}

		var data []byte
		err = tx.SelectRow(ctx, selectQuery).Scan(&data)
		if errors.Is(sql.ErrNoRows, err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("selecting event: %w", err)
		}
		return ErrDuplicateEventID
	}
	return nil

}

type EventSpec[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] struct {
	// Keys must be set, to identify the state machine.
	Keys K
	// EventID must be set for incomming events
	EventID string
	// The inner PSM Event type, must be set
	Event IE
}

func (es EventSpec[K, S, ST, E, IE]) validateIncomming() error {
	if es.EventID == "" {
		return fmt.Errorf("EventSpec.EventID must be set")
	}

	if !es.Keys.PSMIsSet() {
		return fmt.Errorf("EventSpec.Keys is required")
	}

	if !es.Event.PSMIsSet() {
		return fmt.Errorf("EventSpec.Event must be set")
	}

	return nil
}

func (sm *StateMachine[K, S, ST, E, IE]) runTx(ctx context.Context, tx sqrlx.Transaction, outerEvent *EventSpec[K, S, ST, E, IE]) (S, error) {

	if err := outerEvent.validateIncomming(); err != nil {
		return *new(S), fmt.Errorf("event validation: %w", err)
	}

	if sm.spec.EventStateSnapshotColumn == nil {
		// With no snapshot column, we can't correctly return the state after
		// the first time we received the event, so duplicates must be an error.
		if err := sm.eventsMustBeUnique(ctx, tx, outerEvent); err != nil {
			return *new(S), fmt.Errorf("events must be unique: %w", err)
		}
	} else {
		if existingState, didExist, err := sm.firstEventUniqueCheck(ctx, tx, outerEvent); err != nil {
			return existingState, err
		} else if didExist {
			return existingState, nil
		}
	}

	state, err := sm.getCurrentState(ctx, tx, outerEvent.Keys)
	if err != nil {
		return state, err
	}

	builtEvent := (*new(E)).ProtoReflect().New().Interface().(E)
	if err := builtEvent.SetPSMEvent(outerEvent.Event); err != nil {
		return state, fmt.Errorf("set event: %w", err)
	}
	builtEvent.SetPSMKeys(outerEvent.Keys)

	eventMeta := builtEvent.PSMMetadata()
	eventMeta.EventId = outerEvent.EventID
	eventMeta.Timestamp = timestamppb.Now()

	stateMeta := state.PSMMetadata()

	eventMeta.Sequence = 0
	if state.GetStatus() == 0 {
		eventMeta.Sequence = 0
		stateMeta.CreatedAt = eventMeta.Timestamp
		stateMeta.UpdatedAt = eventMeta.Timestamp
	} else {
		eventMeta.Sequence = stateMeta.LastSequence + 1
		stateMeta.LastSequence = eventMeta.Sequence
		stateMeta.UpdatedAt = eventMeta.Timestamp
		stateMeta.CreatedAt = eventMeta.Timestamp
	}

	return sm.runEvent(ctx, tx, state, builtEvent)
}

func (sm *StateMachine[K, S, ST, E, IE]) runEvent(ctx context.Context, tx sqrlx.Transaction, state S, builtEvent E) (S, error) {

	statusBefore := state.GetStatus()

	// runEvent modifies state in place
	err := sm.Eventer.RunEvent(ctx, state, builtEvent)
	if err != nil {
		return state, fmt.Errorf("event queue: %w", err)
	}

	if state.GetStatus() == 0 {
		return state, fmt.Errorf("state machine transitioned to zero status")
	}

	if err := sm.store(ctx, tx, state, builtEvent); err != nil {
		return state, err
	}

	// return the state after the first transition, not after all hooks
	// etc have run
	returnState := proto.Clone(state).(S)

	chained, err := sm.runHooks(ctx, tx, statusBefore, state, builtEvent)
	if err != nil {
		return state, fmt.Errorf("run hooks: %w", err)
	}

	if err := sm.eventsMustBeUnique(ctx, tx, chained...); err != nil {
		if errors.Is(err, ErrDuplicateEventID) {
			return state, ErrDuplicateChainedEventID
		}
		return state, err
	}

	for _, chainedEvent := range chained {
		builtChained, err := sm.buildEvent(builtEvent, chainedEvent)
		if err != nil {
			return state, fmt.Errorf("derive event: %w", err)
		}
		// discard state output, using the first transition
		_, err = sm.runEvent(ctx, tx, state, builtChained)
		if err != nil {
			return state, fmt.Errorf("chained event: %w", err)
		}
	}

	return returnState, nil
}

func (sm *StateMachine[K, S, ST, E, IE]) AddHook(hook IStateHook[K, S, ST, E, IE]) {
	sm.hooks = append(sm.hooks, hook)
}

func (sm *StateMachine[K, S, ST, E, IE]) FindHooks(status ST, event E) []IStateHook[K, S, ST, E, IE] {

	hooks := []IStateHook[K, S, ST, E, IE]{}

	for _, hook := range sm.hooks {
		if hook.Matches(status, event) {
			hooks = append(hooks, hook)
		}
	}

	return hooks
}

func (sm *StateMachine[K, S, ST, E, IE]) runHooks(ctx context.Context, tx sqrlx.Transaction, statusBefore ST, state S, event E) ([]*EventSpec[K, S, ST, E, IE], error) {

	chain := []*EventSpec[K, S, ST, E, IE]{}
	hooks := sm.FindHooks(statusBefore, event)

	for _, hook := range hooks {

		baton := &TransitionData[K, S, ST, E, IE]{
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
	}

	return chain, nil
}

// TransitionInTx uses an existing transaction to transition the state machine.
func (sm *StateMachine[K, S, ST, E, IE]) TransitionInTx(ctx context.Context, tx sqrlx.Transaction, event *EventSpec[K, S, ST, E, IE]) (S, error) {
	var state S
	var err error
	state, err = sm.runTx(ctx, tx, event)
	if err != nil {
		return state, err
	}
	return state, nil
}

func (sm *StateMachine[K, S, ST, E, IE]) Transition(ctx context.Context, db Transactor, event *EventSpec[K, S, ST, E, IE]) (S, error) {
	return sm.WithDB(db).Transition(ctx, event)
}

// buildEvent returns a new event with metadata derived from the causing
// event and system actor
func (sm *StateMachine[K, S, ST, E, IE]) buildEvent(cause E, chained *EventSpec[K, S, ST, E, IE]) (evt E, err error) {
	if sm.SystemActor == nil {
		err = fmt.Errorf("no system actor defined, cannot derive events")
		return
	}

	causeMetadata := cause.PSMMetadata()

	derived := (*new(E)).ProtoReflect().New().Interface().(E)
	if err := derived.SetPSMEvent(chained.Event); err != nil {
		return derived, fmt.Errorf("set event: %w", err)
	}

	derived.SetPSMKeys(cause.PSMKeys())

	eventMeta := derived.PSMMetadata()
	if chained.EventID != "" {
		eventMeta.EventId = chained.EventID
	} else {
		eventKey := chained.Event.PSMEventKey()
		eventMeta.EventId = sm.SystemActor.NewEventID(causeMetadata.EventId, eventKey)
	}

	eventMeta.Timestamp = timestamppb.Now()

	// Sequence is set later by the state machine
	eventMeta.Cause = &psm_pb.Cause{
		Actor: nil, // No actor on derived events, the system is no longer considered an actor.
		Source: &psm_pb.Cause_PsmEvent{
			PsmEvent: &psm_pb.PSMEventCause{
				EventId:  causeMetadata.EventId,
				Indirect: false, // This is directly caused by the current event
			},
		},
	}

	return derived, nil
}

// DBStateMachine adds the 'Transaction' method to the state machine, which
// runs the transition in a new transaction from the state machine's database
type DBStateMachine[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] struct {
	*StateMachine[K, S, ST, E, IE]
	db Transactor
}

var ErrDuplicateEventID = errors.New("duplicate event ID")
var ErrDuplicateChainedEventID = errors.New("duplicate chained event ID")

var TxOptions = &sqrlx.TxOptions{
	Isolation: sql.LevelReadCommitted,
	Retryable: true,
	ReadOnly:  false,
}

// Transition transitions the state machine in a new transaction from the state
// machine's database pool
func (sm *DBStateMachine[K, S, ST, E, IE]) Transition(ctx context.Context, event *EventSpec[K, S, ST, E, IE]) (S, error) {
	var state S
	opts := &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}

	err := sm.db.Transact(ctx, opts, func(ctx context.Context, tx sqrlx.Transaction) error {
		var err error
		state, err = sm.runTx(ctx, tx, event)
		if err != nil {
			return fmt.Errorf("run tx: %w", err)
		}

		return nil
	})
	if err != nil {
		return state, err
	}

	return state, nil
}
