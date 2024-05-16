package psm

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
	spec        PSMTableSpec[K, S, ST, SD, E, IE]
	systemActor SystemActor
}

func NewStateMachineConfig[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
](
	defaultSpec PSMTableSpec[K, S, ST, SD, E, IE],
) *StateMachineConfig[K, S, ST, SD, E, IE] {
	return &StateMachineConfig[K, S, ST, SD, E, IE]{
		spec: defaultSpec,
	}
}

func (cb *StateMachineConfig[K, S, ST, SD, E, IE]) SystemActor(systemActor SystemActor) *StateMachineConfig[K, S, ST, SD, E, IE] {
	cb.systemActor = systemActor
	return cb
}

func (cb *StateMachineConfig[K, S, ST, SD, E, IE]) StateTableName(stateTable string) *StateMachineConfig[K, S, ST, SD, E, IE] {
	cb.spec.State.TableName = stateTable
	return cb
}

func (cb *StateMachineConfig[K, S, ST, SD, E, IE]) StoreExtraStateColumns(stateColumns func(S) (map[string]interface{}, error)) *StateMachineConfig[K, S, ST, SD, E, IE] {
	cb.spec.State.StoreExtraColumns = stateColumns
	return cb
}

func (cb *StateMachineConfig[K, S, ST, SD, E, IE]) EventTableName(eventTable string) *StateMachineConfig[K, S, ST, SD, E, IE] {
	cb.spec.Event.TableName = eventTable
	return cb
}

func (cb *StateMachineConfig[K, S, ST, SD, E, IE]) StoreExtraEventColumns(eventColumns func(E) (map[string]interface{}, error)) *StateMachineConfig[K, S, ST, SD, E, IE] {
	cb.spec.Event.StoreExtraColumns = eventColumns
	return cb
}

func (cb *StateMachineConfig[K, S, ST, SD, E, IE]) PrimaryKey(primaryKey func(K) (map[string]interface{}, error)) *StateMachineConfig[K, S, ST, SD, E, IE] {
	cb.spec.PrimaryKey = primaryKey
	return cb
}

func (cb *StateMachineConfig[K, S, ST, SD, E, IE]) StoreEventStateSnapshot() *StateMachineConfig[K, S, ST, SD, E, IE] {
	cb.spec.EventStateSnapshotColumn = &cb.spec.State.DataColumn
	return cb
}

func (cb *StateMachineConfig[K, S, ST, SD, E, IE]) EventStateSnapshotColumn(name string) *StateMachineConfig[K, S, ST, SD, E, IE] {
	cb.spec.EventStateSnapshotColumn = &name
	return cb
}

func (cb *StateMachineConfig[K, S, ST, SD, E, IE]) NewStateMachine() (*StateMachine[K, S, ST, SD, E, IE], error) {
	return NewStateMachine[K, S, ST, SD, E, IE](cb)
}
