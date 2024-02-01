package psm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/dbconvert"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TableSpec is the configuration for the state machine's table mapping.
// The generated code provides a default which is derived from the patterns and
// annotations on the proto message, however, the proto message is designed to
// specify the wire data, not the storage mechanism, so consuming code may need
// to override some of the defaults to map to the database.
// The generated default is called DefaultFooPSMTableSpec
type TableSpec[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] struct {
	// Primary Key derives the *State* primary key from the event.
	PrimaryKey func(E) (map[string]interface{}, error)

	// StateColumns stores non-primary-key columns from the state into the
	// database
	StateColumns func(S) (map[string]interface{}, error)

	StateTable string
	EventTable string

	StateDataColumn string // default: state
	EventDataColumn string // default: data
	EventColumns    func(E) (map[string]interface{}, error)

	EventPrimaryKeyFieldPaths []string
	StatePrimaryKeyFieldPaths []string
}

func (spec *TableSpec[S, ST, E, IE]) setDefaults() {
	if spec.StateDataColumn == "" {
		spec.StateDataColumn = "state"
	}
	if spec.EventDataColumn == "" {
		spec.EventDataColumn = "data"
	}
}

func (spec TableSpec[S, ST, E, IE]) StateTableSpec() StateTableSpec {
	spec.setDefaults()
	return StateTableSpec{
		StateTable:      spec.StateTable,
		StateDataColumn: spec.StateDataColumn,
		EventTable:      spec.EventTable,
		EventDataColumn: spec.EventDataColumn,

		StatePrimaryKeyPaths: spec.StatePrimaryKeyFieldPaths,
		EventPrimaryKeyPaths: spec.EventPrimaryKeyFieldPaths,

		EventTypeName: (*new(E)).ProtoReflect().Descriptor().FullName(),
		StateTypeName: (*new(S)).ProtoReflect().Descriptor().FullName(),
	}
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

// EventTypeConverter is implemented by the gnerated code, it could all
// be derived at runtime from proto reflect but the discovert runs a lot of
// loops. This is faster.
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
	spec        TableSpec[S, ST, E, IE]
	conversions EventTypeConverter[S, ST, E, IE]
	*Eventer[S, ST, E, IE]

	hooks []StateHook[S, ST, E, IE]
}

func (sm StateMachine[S, ST, E, IE]) StateTableSpec() StateTableSpec {
	return sm.spec.StateTableSpec()
}

type StateHook[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] func(context.Context, sqrlx.Transaction, S, E) error

func (sm *StateMachine[S, ST, E, IE]) AddHook(hook StateHook[S, ST, E, IE]) {
	sm.hooks = append(sm.hooks, hook)
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

// StateMachineConfig allows the generated code to build a default
// machine, but expose options to the user to override the defaults
type StateMachineConfig[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
] struct {
	conversions EventTypeConverter[S, ST, E, IE]
	spec        TableSpec[S, ST, E, IE]
	systemActor *SystemActor
}

func NewStateMachineConfig[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
](
	defaultConversions EventTypeConverter[S, ST, E, IE],
	defaultSpec TableSpec[S, ST, E, IE],
) *StateMachineConfig[S, ST, E, IE] {
	return &StateMachineConfig[S, ST, E, IE]{
		conversions: defaultConversions,
		spec:        defaultSpec,
	}
}

func (cb *StateMachineConfig[S, ST, E, IE]) WithTableSpec(spec TableSpec[S, ST, E, IE]) *StateMachineConfig[S, ST, E, IE] {
	cb.spec = spec
	return cb
}

func (cb *StateMachineConfig[S, ST, E, IE]) WithEventTypeConverter(conversions EventTypeConverter[S, ST, E, IE]) *StateMachineConfig[S, ST, E, IE] {
	cb.conversions = conversions
	return cb
}

func (cb *StateMachineConfig[S, ST, E, IE]) WithSystemActor(systemActor SystemActor) *StateMachineConfig[S, ST, E, IE] {
	cb.systemActor = &systemActor
	return cb
}

func (cb *StateMachineConfig[S, ST, E, IE]) WithStateColumns(stateColumns func(S) (map[string]interface{}, error)) *StateMachineConfig[S, ST, E, IE] {
	cb.spec.StateColumns = stateColumns
	return cb
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

	(&cb.spec).setDefaults()

	ee := &Eventer[S, ST, E, IE]{
		conversions: cb.conversions,
		SystemActor: cb.systemActor,
	}

	return &StateMachine[S, ST, E, IE]{
		spec:        cb.spec,
		conversions: cb.conversions,
		Eventer:     ee,
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
	if stateSpec.StateColumns != nil {
		additionalColumns, err = stateSpec.StateColumns(state)
		if err != nil {
			return fmt.Errorf("extra state columns: %w", err)
		}
	}

	stateJSON, err := dbconvert.MarshalProto(state)
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
		log.WithFields(ctx, map[string]interface{}{
			"stateKeyMap": stateKeyMap,
			"stateSetMap": stateSetMap,
			"error":       err.Error(),
		}).Error("failed to upsert state")
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

	for _, hook := range sm.hooks {
		if err := hook(ctx, tx, state, event); err != nil {
			return fmt.Errorf("hook: %w", err)
		}
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

type DBStateMachine[S IState[ST], ST IStatusEnum, E IEvent[IE], IE IInnerEvent] struct {
	*StateMachine[S, ST, E, IE]
	db Transactor
}

// Transition transitions the state machine in a new transaction from the state
// machine's database pool
func (sm *DBStateMachine[S, ST, E, IE]) Transition(ctx context.Context, events ...E) (S, error) {
	var state S
	if err := sm.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		var err error
		for _, event := range events {
			state, err = sm.runTx(ctx, tx, event)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return state, err
	}

	return state, nil
}
