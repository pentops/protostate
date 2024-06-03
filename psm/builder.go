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

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) SystemActor(systemActor SystemActor) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.systemActor = systemActor
	return smc
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) StateTableName(stateTable string) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.spec.State.TableName = stateTable
	return smc
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) EventTableName(eventTable string) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.spec.Event.TableName = eventTable
	return smc
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) DeriveKeyValues(cbFunc func(K) (map[string]string, error)) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.spec.KeyValues = cbFunc
	return smc
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) NewStateMachine() (*StateMachine[K, S, ST, SD, E, IE], error) {
	return NewStateMachine[K, S, ST, SD, E, IE](smc)
}
