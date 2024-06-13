package psm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-messaging/outbox"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/protostate/internal/dbconvert"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var ErrDuplicateEventID = errors.New("duplicate event ID")
var ErrDuplicateChainedEventID = errors.New("duplicate chained event ID")

type Transactor interface {
	Transact(context.Context, *sqrlx.TxOptions, sqrlx.Callback) error
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

func MustSystemActor(id string) SimpleSystemActor {
	actor, err := NewSystemActor(id)
	if err != nil {
		panic(err)
	}
	return actor
}

func (sa SimpleSystemActor) NewEventID(fromEventUUID string, eventKey string) string {
	return uuid.NewMD5(sa.ID, []byte(fromEventUUID+eventKey)).String()
}

type SystemActor interface {
	NewEventID(fromEventUUID string, eventKey string) string
}

// StateMachine is a database wrapper around the eventer. Using sane defaults
// with overrides for table configuration.
type StateMachine[
	K IKeyset,
	S IState[K, ST, SD], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	SD IStateData,
	E IEvent[K, S, ST, SD, IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] struct {
	*Eventer[K, S, ST, SD, E, IE]

	SystemActor SystemActor

	hookSet[K, S, ST, SD, E, IE]

	keyValueFunc func(K) (map[string]string, error)

	tableMap *TableMap
}

func NewStateMachine[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
](
	cb *StateMachineConfig[K, S, ST, SD, E, IE],
) (*StateMachine[K, S, ST, SD, E, IE], error) {

	if cb.tableMap == nil {
		tableMap, err := tableMapFromStateAndEvent(
			(*new(S)).ProtoReflect().Descriptor(),
			(*new(E)).ProtoReflect().Descriptor(),
		)
		if err != nil {
			return nil, err
		}
		cb.tableMap = tableMap
	}

	if err := cb.tableMap.Validate(); err != nil {
		return nil, err
	}

	ee := &Eventer[K, S, ST, SD, E, IE]{}

	return &StateMachine[K, S, ST, SD, E, IE]{
		keyValueFunc: cb.keyValues,
		tableMap:     cb.tableMap,
		Eventer:      ee,
		SystemActor:  cb.systemActor,
	}, nil
}

func (sm StateMachine[K, S, ST, SD, E, IE]) StateTableSpec() QueryTableSpec {
	return QueryTableSpec{
		TableMap:  *sm.tableMap,
		EventType: (*new(E)).ProtoReflect().Descriptor(),
		StateType: (*new(S)).ProtoReflect().Descriptor(),
	}
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) getCurrentState(ctx context.Context, tx sqrlx.Transaction, keys K) (S, error) {
	state := (*new(S)).ProtoReflect().New().Interface().(S)

	selectQuery := sq.
		Select(sm.tableMap.State.Root.ColumnName).
		From(sm.tableMap.State.TableName)

	allKeys, err := sm.keyValues(keys)
	if err != nil {
		return state, err
	}
	for _, key := range allKeys.values {
		if !key.Primary {
			continue
		}
		selectQuery = selectQuery.Where(sq.Eq{key.ColumnName: key.value})
	}

	var stateJSON []byte
	err = tx.SelectRow(ctx, selectQuery).Scan(&stateJSON)
	if errors.Is(err, sql.ErrNoRows) {
		state.SetPSMKeys(proto.Clone(keys).(K))

		if len(allKeys.missingRequired) > 0 {
			return state, fmt.Errorf("missing required key(s) %v in initial event", allKeys.missingRequired)
		}

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

	return state, nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) storeCallback(tx sqrlx.Transaction) eventerCallback[K, S, ST, SD, E, IE] {
	return func(ctx context.Context, statusBefore ST, state S, event E) error {
		return sm.store(ctx, tx, state, event)
	}
}

type keyValues struct {
	values []keyValue

	// when required, non-primary keys are not set, the caller can't create a
	// new state entity, but can fill in missing keys from an existing stored
	// entity.
	// This allows controllers and workers to specify only the primary key on
	// non-create events and save a database lookup
	missingRequired []protoreflect.Name
}

type keyValue struct {
	value string
	KeyColumn
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) keyValues(keysMessage K) (*keyValues, error) {
	var rawValues map[string]string
	var err error
	if sm.keyValueFunc != nil {
		rawValues, err = sm.keyValueFunc(keysMessage)
		if err != nil {
			return nil, err
		}
	} else {
		rawValues, err = keysMessage.PSMKeyValues()
		if err != nil {
			return nil, err
		}
	}
	values := make([]keyValue, 0, len(sm.tableMap.KeyColumns))
	missingRequired := make([]protoreflect.Name, 0, len(sm.tableMap.KeyColumns))

	for _, def := range sm.tableMap.KeyColumns {
		gotValue, ok := rawValues[def.ColumnName]
		if !ok || gotValue == "" {
			if def.Primary {
				return nil, fmt.Errorf("KeyValues() for %s did not return a value for required key field %s", keysMessage.PSMFullName(), def.ProtoName)
			}
			if def.Required {
				missingRequired = append(missingRequired, def.ProtoName)
			}
			if ok { // set but empty
				delete(rawValues, def.ColumnName)
			}
			continue
		}
		delete(rawValues, def.ColumnName)

		values = append(values, keyValue{value: gotValue, KeyColumn: def})

	}

	if len(rawValues) > 0 {
		return nil, fmt.Errorf("KeyValues() for %s returned unexpected keys: %v", keysMessage.PSMFullName(), rawValues)
	}

	return &keyValues{
		values:          values,
		missingRequired: missingRequired,
	}, nil

}

func (sm *StateMachine[K, S, ST, SD, E, IE]) store(
	ctx context.Context,
	tx sqrlx.Transaction,
	state S,
	event E,
) error {

	stateDBValue, err := dbconvert.MarshalProto(state)
	if err != nil {
		return fmt.Errorf("state field: %w", err)
	}

	eventDBValue, err := dbconvert.MarshalProto(event)
	if err != nil {
		return fmt.Errorf("event field: %w", err)
	}

	// TODO: This does not change during transitions, so should be calculated
	// early and once.
	keyValues, err := sm.keyValues(state.PSMKeys())
	if err != nil {
		return fmt.Errorf("key fields: %w", err)
	}
	if len(keyValues.missingRequired) > 0 {
		return fmt.Errorf("missing required key(s) %v in store", keyValues.missingRequired)
	}

	eventMeta := event.PSMMetadata()

	upsertStateQuery := sqrlx.Upsert(sm.tableMap.State.TableName)

	insertValues := []interface{}{}
	insertColumns := []string{}

	insertEventQuery := sq.Insert(sm.tableMap.Event.TableName)

	insertColumns = append(insertColumns, sm.tableMap.Event.ID.ColumnName)
	insertValues = append(insertValues, eventMeta.EventId)

	for _, key := range keyValues.values {
		if key.Primary {
			upsertStateQuery.Key(key.ColumnName, key.value)
		} else {
			upsertStateQuery.Set(key.ColumnName, key.value)
		}

		insertColumns = append(insertColumns, key.ColumnName)
		insertValues = append(insertValues, key.value)
	}

	insertColumns = append(insertColumns,
		sm.tableMap.Event.Timestamp.ColumnName,
		sm.tableMap.Event.Sequence.ColumnName,
		sm.tableMap.Event.Root.ColumnName,
		sm.tableMap.Event.StateSnapshot.ColumnName,
	)
	insertValues = append(insertValues,
		eventMeta.Timestamp.AsTime(),
		eventMeta.Sequence,
		eventDBValue,
		stateDBValue,
	)
	insertEventQuery.Columns(insertColumns...).Values(insertValues...)

	upsertStateQuery.Set(sm.tableMap.State.Root.ColumnName, stateDBValue)

	_, err = tx.Insert(ctx, upsertStateQuery)
	if err != nil {
		log.WithFields(ctx, map[string]interface{}{
			"keys":  keyValues,
			"error": err.Error(),
		}).Error("failed to upsert state")
		return fmt.Errorf("upsert state: %w", err)
	}

	_, err = tx.Insert(ctx, insertEventQuery)
	if err != nil {
		log.WithFields(ctx, map[string]interface{}{
			"keys":  keyValues,
			"error": err.Error(),
		}).Error("failed to insert event")
		return fmt.Errorf("insert event: %w", err)
	}

	return nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) eventQuery(ctx context.Context, tx sqrlx.Transaction, eventID string, keys K) (*sq.SelectBuilder, error) {

	selectQuery := sq.
		Select(sm.tableMap.Event.Root.ColumnName).
		From(sm.tableMap.Event.TableName).
		Where(sq.Eq{sm.tableMap.Event.ID.ColumnName: eventID})

	return selectQuery, nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) runTx(ctx context.Context, tx sqrlx.Transaction, outerEvent *EventSpec[K, S, ST, SD, E, IE]) (S, error) {

	if err := outerEvent.validateIncomming(); err != nil {
		return *new(S), fmt.Errorf("event %s: %w", outerEvent.Event.ProtoReflect().Descriptor().FullName(), err)
	}

	if outerEvent.EventID == "" {
		if causeEvent := outerEvent.Cause.GetPsmEvent(); causeEvent != nil {
			// Can derive an ID
			if sm.SystemActor == nil {
				return *new(S), fmt.Errorf("no system actor defined, cannot derive events, and no ID set")
			}
			outerEvent.EventID = sm.SystemActor.NewEventID(causeEvent.EventId, outerEvent.Event.PSMEventKey())
		} else {
			return *new(S), fmt.Errorf("EventSpec.EventID must be set unless the cause is a PSM Event")
		}
	}

	if existingState, didExist, err := sm.firstEventUniqueCheck(ctx, tx, outerEvent); err != nil {
		return existingState, err
	} else if didExist {
		return existingState, nil
	}

	state, err := sm.getCurrentState(ctx, tx, outerEvent.Keys)
	if err != nil {
		return state, err
	}

	if state.GetStatus() != 0 {
		// If this is not the first event, the event keys can be derived from the
		// state keys, so the non-primary keys of the event need not be set for every
		// event.
		// TODO: Consider checking that any key which *is* set matches.
		// The event will be validated later using buf validate so any required non-primary keys are
		// evaluated at that point.
		outerEvent.Keys = state.PSMKeys()
	}

	return sm.runInputEvent(ctx, tx, state, outerEvent)
}

// firstEventUniqueCheck checks if the event ID for the outer triggering event
// is unique in the event table. If not, it checks if the event is a repeat
// processing of the same event, and returns the state after the initial
// transition.
func (sm *StateMachine[K, S, ST, SD, E, IE]) firstEventUniqueCheck(ctx context.Context, tx sqrlx.Transaction, event *EventSpec[K, S, ST, SD, E, IE]) (S, bool, error) {
	var s S
	selectQuery, err := sm.eventQuery(ctx, tx, event.EventID, event.Keys)
	if err != nil {
		return s, false, fmt.Errorf("event query: %w", err)
	}

	selectQuery.Column(sm.tableMap.Event.StateSnapshot.ColumnName)

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

func (sm *StateMachine[K, S, ST, SD, E, IE]) eventsMustBeUnique(ctx context.Context, tx sqrlx.Transaction, events ...*EventSpec[K, S, ST, SD, E, IE]) error {
	for _, event := range events {
		if event.EventID == "" {
			continue // UUID Gen Later
		}
		selectQuery, err := sm.eventQuery(ctx, tx, event.EventID, event.Keys)
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

func (sm *StateMachine[K, S, ST, SD, E, IE]) runInputEvent(ctx context.Context, tx sqrlx.Transaction, state S, spec *EventSpec[K, S, ST, SD, E, IE]) (S, error) {

	var returnState S

	captureState := func(ctx context.Context, statusBefore ST, state S, event E) error {
		returnState = proto.Clone(state).(S)
		return nil
	}
	// RunEvent modifies state in place
	err := sm.Eventer.RunEvent(ctx, state, spec,
		sm.storeCallback(tx),
		captureState, // return the state after the first transition
		sm.runHooksCallback(tx),
	)

	if err != nil {
		return state, fmt.Errorf("input event %s: %w", spec.Event.PSMEventKey(), err)
	}

	return returnState, nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) runChainedEvent(ctx context.Context, tx sqrlx.Transaction, state S, spec *EventSpec[K, S, ST, SD, E, IE]) error {
	err := sm.Eventer.RunEvent(ctx, state, spec, sm.storeCallback(tx), sm.runHooksCallback(tx))
	if err != nil {
		return fmt.Errorf("chained event: %s: %w", spec.Event.PSMEventKey(), err)
	}

	return nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) runHooksCallback(tx sqrlx.Transaction) eventerCallback[K, S, ST, SD, E, IE] {
	return func(ctx context.Context, statusBefore ST, state S, event E) error {
		if err := sm.runHooks(ctx, tx, statusBefore, state, event); err != nil {
			return fmt.Errorf("run hooks: %w", err)
		}
		return nil
	}
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) runHooks(ctx context.Context, tx sqrlx.Transaction, statusBefore ST, state S, event E) error {

	chain := []*EventSpec[K, S, ST, SD, E, IE]{}
	hooks := sm.FindHooks(statusBefore, event)

	for _, hook := range hooks {

		baton := &hookBaton[K, S, ST, SD, E, IE]{
			causedBy: event,
		}

		if err := hook.RunTransitionHooks(ctx, tx, baton, state, event); err != nil {
			return fmt.Errorf("run state hook: %w", err)
		}

		for _, se := range baton.sideEffects {
			if err := outbox.Send(ctx, tx, se); err != nil {
				return fmt.Errorf("side effect outbox: %w", err)
			}
		}

		for _, chained := range baton.chainEvents {
			derived, err := sm.deriveEvent(event, chained)
			if err != nil {
				return fmt.Errorf("derive chained: %w", err)
			}
			chain = append(chain, derived)
		}
	}

	if err := sm.eventsMustBeUnique(ctx, tx, chain...); err != nil {
		if errors.Is(err, ErrDuplicateEventID) {
			return ErrDuplicateChainedEventID
		}
		return err
	}

	for _, chainedEvent := range chain {
		err := sm.runChainedEvent(ctx, tx, state, chainedEvent)
		if err != nil {
			return fmt.Errorf("chained event: %w", err)
		}
	}

	return nil
}

// TransitionInTx uses an existing transaction to transition the state machine.
func (sm *StateMachine[K, S, ST, SD, E, IE]) TransitionInTx(ctx context.Context, tx sqrlx.Transaction, event *EventSpec[K, S, ST, SD, E, IE]) (S, error) {
	var state S
	var err error
	state, err = sm.runTx(ctx, tx, event)
	if err != nil {
		return state, err
	}
	return state, nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) Transition(ctx context.Context, db Transactor, event *EventSpec[K, S, ST, SD, E, IE]) (S, error) {
	return sm.WithDB(db).Transition(ctx, event)
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) deriveEvent(cause E, chained IE) (evt *EventSpec[K, S, ST, SD, E, IE], err error) {
	if sm.SystemActor == nil {
		err = fmt.Errorf("no system actor defined, cannot derive events")
		return
	}

	eventKey := chained.PSMEventKey()
	causeMetadata := cause.PSMMetadata()
	eventID := sm.SystemActor.NewEventID(causeMetadata.EventId, eventKey)
	psmKeys := cause.PSMKeys()

	eventOut := &EventSpec[K, S, ST, SD, E, IE]{
		Keys:      psmKeys,
		Timestamp: time.Now(),
		Event:     chained,
		EventID:   eventID,
		Cause: &psm_pb.Cause{
			Type: &psm_pb.Cause_PsmEvent{
				PsmEvent: &psm_pb.PSMEventCause{
					EventId:      causeMetadata.EventId,
					StateMachine: psmKeys.PSMFullName(),
				},
			},
		},
	}

	return eventOut, nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) WithDB(db Transactor) *DBStateMachine[K, S, ST, SD, E, IE] {
	return &DBStateMachine[K, S, ST, SD, E, IE]{
		StateMachine: sm,
		db:           db,
	}
}

// DBStateMachine adds the 'Transaction' method to the state machine, which
// runs the transition in a new transaction from the state machine's database
type DBStateMachine[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	*StateMachine[K, S, ST, SD, E, IE]
	db Transactor
}

var TxOptions = &sqrlx.TxOptions{
	Isolation: sql.LevelReadCommitted,
	Retryable: true,
	ReadOnly:  false,
}

// Transition transitions the state machine in a new transaction from the state
// machine's database pool
func (sm *DBStateMachine[K, S, ST, SD, E, IE]) Transition(ctx context.Context, event *EventSpec[K, S, ST, SD, E, IE]) (S, error) {
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
			return err
		}

		return nil
	})
	if err != nil {
		return state, err
	}

	return state, nil
}
