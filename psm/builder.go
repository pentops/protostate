package psm

import (
	"context"
	"fmt"

	"github.com/pentops/j5/lib/j5schema"
	"github.com/pentops/sqrlx.go/sqrlx"
)

// StateMachineConfig allows the generated code to build a default
// machine, but expose options to the user to override the defaults
type StateMachineConfig[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	//systemActor SystemActor

	keyValues func(K) (map[string]any, error)

	initialStateFunc func(context.Context, sqrlx.Transaction, K) (IE, error)

	tableMap *TableMap

	tableName *string
}

// DEPRECATED: This does nothing.
type SystemActor any

// DEPRECATED: This does nothing.
func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) SystemActor(systemActor SystemActor) *StateMachineConfig[K, S, ST, SD, E, IE] {
	//smc.systemActor = systemActor
	return smc
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) TableMap(tableMap *TableMap) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.tableMap = tableMap
	return smc
}

// TableName sets both tables to {name} and {name_event}
func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) TableName(tableName string) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.tableName = &tableName
	return smc
}

// KeyFields derives the key values from the Key entity. Should return ID Strings, and omit entries for NULL values
func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) DeriveKeyValues(cbFunc func(K) (map[string]any, error)) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.keyValues = cbFunc
	return smc
}

// InitialStateFunc is called when the state machine is created, and the state
// is not found in the database. It must return an 'initialization' event which
// will run prior to the event being processed.
// The DB transaction is available to read data from, for example, upsert data,
// but be sure that the data is either immutable, or there is an event to update
// it when it changes.
func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) InitialStateFunc(cbFunc func(context.Context, sqrlx.Transaction, K) (IE, error)) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.initialStateFunc = cbFunc
	return smc
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) schemas() (*j5schema.ObjectSchema, *j5schema.ObjectSchema, error) {
	state, ok := newJ5Message[S]().J5Reflect().RootSchema()
	if !ok {
		return nil, nil, fmt.Errorf("invalid state type %T, must be a j5 root schema", new(S))
	}
	stateObject, ok := state.(*j5schema.ObjectSchema)
	if !ok {
		return nil, nil, fmt.Errorf("invalid state type %T, must be a j5 object schema", state)
	}
	event, ok := newJ5Message[E]().J5Reflect().RootSchema()
	if !ok {
		return nil, nil, fmt.Errorf("invalid event type %T, must be a j5 root schema", new(E))
	}
	eventObject, ok := event.(*j5schema.ObjectSchema)
	if !ok {
		return nil, nil, fmt.Errorf("invalid event type %T, must be a j5 object schema", event)
	}
	return stateObject, eventObject, nil

}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) apply() error {
	if smc.tableMap == nil {
		stateObject, eventObject, err := smc.schemas()
		if err != nil {
			return err
		}
		tableMap, err := tableMapFromStateAndEvent(stateObject, eventObject)
		if err != nil {
			return err
		}
		smc.tableMap = tableMap
	}

	if smc.tableName != nil {
		smc.tableMap.State.TableName = *smc.tableName
		smc.tableMap.Event.TableName = *smc.tableName + "_event"
	}
	return nil
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) BuildStateMachine() (*StateMachine[K, S, ST, SD, E, IE], error) {
	if err := smc.apply(); err != nil {
		return nil, err
	}
	return NewStateMachine(smc)
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) BuildQueryTableSpec() (*QueryTableSpec, error) {
	if err := smc.apply(); err != nil {
		return nil, err
	}

	state, event, err := smc.schemas()
	if err != nil {
		return nil, err
	}

	return &QueryTableSpec{
		TableMap:  *smc.tableMap,
		StateType: state,
		EventType: event,
	}, nil
}
