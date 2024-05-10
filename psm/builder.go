package psm

// StateMachineConfig allows the generated code to build a default
// machine, but expose options to the user to override the defaults
type StateMachineConfig[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] struct {
	spec        PSMTableSpec[K, S, ST, E, IE]
	systemActor SystemActor
}

func NewStateMachineConfig[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
](
	defaultSpec PSMTableSpec[K, S, ST, E, IE],
) *StateMachineConfig[K, S, ST, E, IE] {
	return &StateMachineConfig[K, S, ST, E, IE]{
		spec: defaultSpec,
	}
}

func (cb *StateMachineConfig[K, S, ST, E, IE]) SystemActor(systemActor SystemActor) *StateMachineConfig[K, S, ST, E, IE] {
	cb.systemActor = systemActor
	return cb
}

func (cb *StateMachineConfig[K, S, ST, E, IE]) StateTableName(stateTable string) *StateMachineConfig[K, S, ST, E, IE] {
	cb.spec.State.TableName = stateTable
	return cb
}

func (cb *StateMachineConfig[K, S, ST, E, IE]) StoreExtraStateColumns(stateColumns func(S) (map[string]interface{}, error)) *StateMachineConfig[K, S, ST, E, IE] {
	cb.spec.State.StoreExtraColumns = stateColumns
	return cb
}

func (cb *StateMachineConfig[K, S, ST, E, IE]) EventTableName(eventTable string) *StateMachineConfig[K, S, ST, E, IE] {
	cb.spec.Event.TableName = eventTable
	return cb
}

func (cb *StateMachineConfig[K, S, ST, E, IE]) StoreExtraEventColumns(eventColumns func(E) (map[string]interface{}, error)) *StateMachineConfig[K, S, ST, E, IE] {
	cb.spec.Event.StoreExtraColumns = eventColumns
	return cb
}

func (cb *StateMachineConfig[K, S, ST, E, IE]) PrimaryKey(primaryKey func(E) (map[string]interface{}, error)) *StateMachineConfig[K, S, ST, E, IE] {
	cb.spec.PrimaryKey = primaryKey
	return cb
}

func (cb *StateMachineConfig[K, S, ST, E, IE]) StoreEventStateSnapshot() *StateMachineConfig[K, S, ST, E, IE] {
	cb.spec.EventStateSnapshotColumn = &cb.spec.State.DataColumn
	return cb
}

func (cb *StateMachineConfig[K, S, ST, E, IE]) EventStateSnapshotColumn(name string) *StateMachineConfig[K, S, ST, E, IE] {
	cb.spec.EventStateSnapshotColumn = &name
	return cb
}

func (cb *StateMachineConfig[K, S, ST, E, IE]) NewStateMachine() (*StateMachine[K, S, ST, E, IE], error) {
	return NewStateMachine[K, S, ST, E, IE](cb)
}
