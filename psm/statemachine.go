package psm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/pentops/j5/j5types/date_j5t"
	"github.com/pentops/j5/lib/j5reflect"
	"github.com/pentops/j5/lib/j5schema"
	"github.com/pentops/j5/lib/j5validate"

	"buf.build/go/protovalidate"
	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/internal/dbconvert"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrDuplicateEventID = errors.New("duplicate event ID")
var ErrDuplicateChainedEventID = errors.New("duplicate chained event ID")

type Transactor interface {
	Transact(context.Context, *sqrlx.TxOptions, sqrlx.Callback) error
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
	transitionSet[K, S, ST, SD, E, IE]

	keyValueFunc func(K) (map[string]any, error)

	initialStateFunc func(context.Context, sqrlx.Transaction, K) (IE, error)

	tableMap *TableMap

	stateSchema *j5schema.ObjectSchema
	eventSchema *j5schema.ObjectSchema

	validator      *j5validate.Validator
	protoValidator protovalidate.Validator
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
	stateSchema, eventSchema, err := cb.schemas()
	if err != nil {
		return nil, fmt.Errorf("schemas: %w", err)
	}

	if cb.tableMap == nil {

		tableMap, err := tableMapFromStateAndEvent(stateSchema, eventSchema)
		if err != nil {
			return nil, err
		}
		cb.tableMap = tableMap
	}

	if err := cb.tableMap.Validate(); err != nil {
		return nil, err
	}

	pv, err := protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize protovalidate: %w", err)
	}

	return &StateMachine[K, S, ST, SD, E, IE]{
		keyValueFunc:     cb.keyValues,
		initialStateFunc: cb.initialStateFunc,
		tableMap:         cb.tableMap,
		stateSchema:      stateSchema,
		eventSchema:      eventSchema,
		validator:        j5validate.NewValidator(),
		protoValidator:   pv,
	}, nil
}

func (sm StateMachine[K, S, ST, SD, E, IE]) StateTableSpec() QueryTableSpec {
	return QueryTableSpec{
		TableMap:  *sm.tableMap,
		EventType: sm.eventSchema,
		StateType: sm.stateSchema,
	}
}

// TransitionInTx uses an existing transaction to transition the state machine.
func (sm *StateMachine[K, S, ST, SD, E, IE]) TransitionInTx(ctx context.Context, tx sqrlx.Transaction, event *EventSpec[K, S, ST, SD, E, IE]) (S, error) {
	if sm == nil {
		return *new(S), fmt.Errorf("transition in tx: state machine is nil")
	}

	var state S
	var err error
	state, err = sm.runTx(ctx, tx, event)
	if err != nil {
		return state, err
	}
	return state, nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) Transition(ctx context.Context, db Transactor, event *EventSpec[K, S, ST, SD, E, IE]) (S, error) {
	if sm == nil {
		return *new(S), fmt.Errorf("transition: state machine is nil")
	}

	return sm.WithDB(db).Transition(ctx, event)
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
	if sm == nil {
		return *new(S), fmt.Errorf("transition: state machine is nil")
	}

	var state S
	err := sm.db.Transact(ctx, TxOptions, func(ctx context.Context, tx sqrlx.Transaction) error {
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

// FollowEvent stores the event in the database and runs all database updates
// and hooks, but does not run any Logic hooks and so does not produce
// side-effects or chain events. This is used when the state machine is not the
// Leader, or when it is restoring or loading from stored event history.
func (sm *DBStateMachine[K, S, ST, SD, E, IE]) FollowEvent(ctx context.Context, event E) error {
	err := sm.db.Transact(ctx, TxOptions, func(ctx context.Context, tx sqrlx.Transaction) error {
		err := sm.followEvents(ctx, tx, []E{event})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (sm *DBStateMachine[K, S, ST, SD, E, IE]) FollowEvents(ctx context.Context, events []E) error {
	// TODO: Prepare the events prior to opening the transaction
	err := sm.db.Transact(ctx, TxOptions, func(ctx context.Context, tx sqrlx.Transaction) error {
		err := sm.followEvents(ctx, tx, events)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) getCurrentState(ctx context.Context, tx sqrlx.Transaction, keys K) (S, error) {
	state := newJ5Message[S]()
	stateRefl := state.J5Reflect()

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
		state.SetPSMKeys(keys.Clone().(K))

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

	if err := dbconvert.UnmarshalJ5(stateJSON, stateRefl); err != nil {
		return state, err
	}

	return state, nil
}

type keyValues struct {
	values []keyValue

	// when required, non-primary keys are not set, the caller can't create a
	// new state entity, but can fill in missing keys from an existing stored
	// entity.
	// This allows controllers and workers to specify only the primary key on
	// non-create events and save a database lookup
	missingRequired []string
}

type keyValue struct {
	value any
	KeyColumn
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) keyValues(keysMessage K) (*keyValues, error) {
	var rawValues map[string]any
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
	missingRequired := make([]string, 0, len(sm.tableMap.KeyColumns))

	for _, def := range sm.tableMap.KeyColumns {
		gotValue, ok := rawValues[def.ColumnName]
		if !ok || gotValue == "" {
			if def.Primary {
				return nil, fmt.Errorf("KeyValues() for %s did not return a value for required key field %s", keysMessage.PSMFullName(), def.ColumnName)
			}
			if def.Required {
				missingRequired = append(missingRequired, def.ColumnName)
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

	stateDBValue, err := dbconvert.MarshalJ5(state.J5Reflect())
	if err != nil {
		return fmt.Errorf("state field: %w", err)
	}

	eventDBValue, err := dbconvert.MarshalJ5(event.J5Reflect())
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

	insertValues := []any{}
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
		log.WithFields(ctx, map[string]any{
			"keys":  keyValues,
			"error": err.Error(),
		}).Error("failed to upsert state")
		return fmt.Errorf("upsert state: %w", err)
	}

	_, err = tx.Insert(ctx, insertEventQuery)
	if err != nil {
		log.WithFields(ctx, map[string]any{
			"keys":  keyValues,
			"error": err.Error(),
		}).Error("failed to insert event")
		return fmt.Errorf("insert event: %w", err)
	}

	return nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) eventQuery(eventID string) *sq.SelectBuilder {

	selectQuery := sq.
		Select(sm.tableMap.Event.Root.ColumnName).
		From(sm.tableMap.Event.TableName).
		Where(sq.Eq{sm.tableMap.Event.ID.ColumnName: eventID})

	return selectQuery
}

// followEventDeduplicate is similar to the firstEventUniqueCheck, but it
// compares the entire event including metadata, as this is not designed to
// handle consumer idempotency.
func (sm *StateMachine[K, S, ST, SD, E, IE]) followEventDeduplicate(ctx context.Context, tx sqrlx.Transaction, event E) (bool, error) {
	selectQuery := sm.eventQuery(event.PSMMetadata().EventId)

	var eventData, stateData []byte
	err := tx.SelectRow(ctx, selectQuery).Scan(&eventData, &stateData)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("selecting event for deduplication: %w", err)
	}

	existing := newJ5Message[E]()
	if err := dbconvert.UnmarshalJ5(eventData, existing.J5Reflect()); err != nil {
		return true, fmt.Errorf("unmarshalling event: %w", err)
	}

	if !j5reflect.DeepEqual(existing.J5Reflect(), event.J5Reflect()) {
		return true, fmt.Errorf("event %s already exists with different data", existing.PSMMetadata().EventId)
	}

	return true, nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) followEvents(ctx context.Context, tx sqrlx.Transaction, events []E) error {

	for _, event := range events {

		if err := sm.validateEvent(event); err != nil {
			return fmt.Errorf("validating event: %w", err)
		}

		exists, err := sm.followEventDeduplicate(ctx, tx, event)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		state, err := sm.getCurrentState(ctx, tx, event.PSMKeys())
		if err != nil {
			return err
		}

		if state.GetStatus() == 0 {
			newState, err := sm.runInitialEvent(ctx, tx, state)
			if err != nil {
				return err
			}
			state = newState
		}

		sm.nextStateEvent(state, event.PSMMetadata())

		err = sm.followEvent(ctx, tx, state, event)
		if err != nil {
			return fmt.Errorf("follow event %s (%s): %w", event.PSMMetadata().EventId, event.UnwrapPSMEvent().PSMEventKey(), err)
		}

	}

	return nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) runInitialEvent(ctx context.Context, tx sqrlx.Transaction, state S) (S, error) {
	if sm.initialStateFunc == nil {
		return state, nil
	}

	keys := state.PSMKeys()
	innerEvent, err := sm.initialStateFunc(ctx, tx, keys)
	if err != nil {
		return state, fmt.Errorf("initial state func: %w", err)
	}

	eventSpec := &EventSpec[K, S, ST, SD, E, IE]{
		Keys:    keys,
		EventID: uuid.NewString(),
		Event:   innerEvent,
		Cause: &psm_j5pb.Cause{
			Type: &psm_j5pb.Cause_Init{
				Init: &psm_j5pb.InitCause{},
			},
		},
	}

	prepared, err := sm.prepareEvent(state, eventSpec)
	if err != nil {
		return state, fmt.Errorf("prepare event: %w", err)
	}

	// RunEvent modifies state in place
	returnState, err := sm.runEvent(ctx, tx, state, prepared, captureFinalState)
	if err != nil {
		return state, fmt.Errorf("run event %s: %w", eventSpec.Event.PSMEventKey(), err)
	}
	return *returnState, nil

}

func assertPresentKeysMatch[K IKeyset](existing, event K) error {
	a, err := existing.PSMKeyValues()
	if err != nil {
		return fmt.Errorf("existing keys: %w", err)
	}
	eventMap, err := event.PSMKeyValues()
	if err != nil {
		return fmt.Errorf("event keys: %w", err)
	}

	for key, eventValue := range eventMap {
		existingValue, ok := a[key]
		if !ok {
			return fmt.Errorf("event key %s is not present in existing keys", key)
		}

		switch v := existingValue.(type) {
		case *date_j5t.Date:
			if eventValueAsDate, ok := eventValue.(*date_j5t.Date); !ok || !v.Equals(eventValueAsDate) {
				return status.Errorf(codes.FailedPrecondition, "event key %s value %s does not match existing value %s", key, eventValue, existingValue)
			}
		default:
			if existingValue != eventValue {
				return status.Errorf(codes.FailedPrecondition, "event key %s value %s does not match existing value %s", key, eventValue, existingValue)
			}
		}
	}

	return nil

}

func (sm *StateMachine[K, S, ST, SD, E, IE]) runTx(ctx context.Context, tx sqrlx.Transaction, outerEvent *EventSpec[K, S, ST, SD, E, IE]) (S, error) {

	if err := outerEvent.validateAndPrepare(); err != nil {
		return *new(S), err
	}

	if outerEvent.EventID == "" {
		outerEvent.EventID = uuid.NewString()
	}

	if existingState, didExist, err := sm.firstEventUniqueCheck(ctx, tx, outerEvent.EventID, outerEvent.Event); err != nil {
		return existingState, err
	} else if didExist {
		return existingState, nil
	}

	state, err := sm.getCurrentState(ctx, tx, outerEvent.Keys)
	if err != nil {
		return state, err
	}

	if state.GetStatus() == 0 {
		newState, err := sm.runInitialEvent(ctx, tx, state)
		if err != nil {
			return state, err
		}
		state = newState
	} else {
		// If this is not the first event, the event keys can be derived from the
		// state keys, so the non-primary keys of the event need not be set for every
		// event.

		err = assertPresentKeysMatch(state.PSMKeys(), outerEvent.Keys)
		if err != nil {
			return state, err
		}
		// TODO: Consider checking that any key which *is* set matches.
		// The event will be validated later using buf validate so any required non-primary keys are
		// evaluated at that point.
		outerEvent.Keys = state.PSMKeys()
	}

	prepared, err := sm.prepareEvent(state, outerEvent)
	if err != nil {
		return state, fmt.Errorf("prepare event: %w", err)
	}

	// RunEvent modifies state in place
	returnState, err := sm.runEvent(ctx, tx, state, prepared, captureInitialState) // return the state after the first transition
	if err != nil {
		return state, fmt.Errorf("run event %s: %w", outerEvent.Event.PSMEventKey(), err)
	}

	return *returnState, nil
}

// firstEventUniqueCheck checks if the event ID for the outer triggering event
// is unique in the event table. If not, it checks if the event is a repeat
// processing of the same event, and returns the state after the initial
// transition.
func (sm *StateMachine[K, S, ST, SD, E, IE]) firstEventUniqueCheck(ctx context.Context, tx sqrlx.Transaction, eventID string, data IE) (S, bool, error) {
	var s S
	selectQuery := sm.eventQuery(eventID)

	selectQuery.Column(sm.tableMap.Event.StateSnapshot.ColumnName)

	var eventData, stateData []byte
	err := tx.SelectRow(ctx, selectQuery).Scan(&eventData, &stateData)
	if errors.Is(err, sql.ErrNoRows) {
		return s, false, nil
	}
	if err != nil {
		return s, false, fmt.Errorf("selecting event: %w", err)
	}

	existing := newJ5Message[E]()

	if err := dbconvert.UnmarshalJ5(eventData, existing.J5Reflect()); err != nil {
		return s, false, fmt.Errorf("unmarshalling event: %w", err)
	}

	existingContent := existing.UnwrapPSMEvent()

	if !j5reflect.DeepEqual(existingContent.J5Reflect(), data.J5Reflect()) {
		log.WithFields(ctx, "existing", prototext.Format(existingContent), "new", prototext.Format(data)).Info("event data does not match existing event data")
		return s, false, ErrDuplicateEventID
	}

	state := newJ5Message[S]()
	if err := dbconvert.UnmarshalJ5(stateData, state.J5Reflect()); err != nil {
		return s, false, fmt.Errorf("unmarshalling state: %w", err)
	}

	return state, true, nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) eventsMustBeUnique(ctx context.Context, tx sqrlx.Transaction, events ...*EventSpec[K, S, ST, SD, E, IE]) error {
	for _, event := range events {
		if event.EventID == "" {
			continue // UUID Gen Later
		}
		selectQuery := sm.eventQuery(event.EventID)

		var data []byte
		err := tx.SelectRow(ctx, selectQuery).Scan(&data)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return fmt.Errorf("selecting event: %w", err)
		}
		return ErrDuplicateEventID
	}
	return nil

}

func (sm *StateMachine[K, S, ST, SD, E, IE]) validateEvent(event E) error {
	if sm.validator == nil {
		v := j5validate.NewValidator()
		sm.validator = v
	}

	err := sm.validator.Validate(event.J5Reflect())
	if err != nil {
		txt := prototext.Format(event)
		fmt.Printf("Event validation failed: %s\n%s\n", err.Error(), txt)

		return fmt.Errorf("validate event: %w", err)
	}
	return nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) prepareEvent(state S, spec *EventSpec[K, S, ST, SD, E, IE]) (built E, err error) {

	built = newJ5Message[E]()
	if err := built.SetPSMEvent(spec.Event); err != nil {
		return built, fmt.Errorf("set event: %w", err)
	}
	built.SetPSMKeys(spec.Keys)

	eventMeta := built.PSMMetadata()
	eventMeta.EventId = spec.EventID
	eventMeta.Timestamp = timestamppb.Now()
	eventMeta.Cause = spec.Cause

	sm.nextStateEvent(state, eventMeta)

	return

}

func (sm *StateMachine[K, S, ST, SD, E, IE]) nextStateEvent(state S, eventMeta *psm_j5pb.EventMetadata) {
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
	}
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) transitionFromLink(ctx context.Context, tx sqrlx.Transaction, cause *psm_j5pb.Cause, keys K, innerEvent IE) error { // nolint: unused // Used when the state machine is implementing LinkDestination
	event := &EventSpec[K, S, ST, SD, E, IE]{
		Keys:      keys,
		Timestamp: time.Now(),
		Event:     innerEvent,
		EventID:   uuid.NewString(),
		Cause:     cause,
	}

	_, err := sm.runTx(ctx, tx, event)
	if err != nil {
		return fmt.Errorf("run transition: %w", err)
	}

	return nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) deriveEvent(cause E, chained IE) (evt *EventSpec[K, S, ST, SD, E, IE], err error) {

	causeMetadata := cause.PSMMetadata()
	eventID := uuid.NewString()
	psmKeys := cause.PSMKeys()

	eventOut := &EventSpec[K, S, ST, SD, E, IE]{
		Keys:      psmKeys,
		Timestamp: time.Now(),
		Event:     chained,
		EventID:   eventID,
		Cause: &psm_j5pb.Cause{
			Type: &psm_j5pb.Cause_PsmEvent{
				PsmEvent: &psm_j5pb.PSMEventCause{
					EventId:      causeMetadata.EventId,
					StateMachine: psmKeys.PSMFullName(),
				},
			},
		},
	}

	return eventOut, nil
}
