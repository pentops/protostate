package psm

// StateMachineConfig allows the generated code to build a default
// machine, but expose options to the user to override the defaults
type StateMachineConfig[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
] struct {
	conversions EventTypeConverter[S, ST, E, IE]
	spec        PSMTableSpec[S, ST, E, IE]
	systemActor *SystemActor
}

func NewStateMachineConfig[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
](
	defaultConversions EventTypeConverter[S, ST, E, IE],
	defaultSpec PSMTableSpec[S, ST, E, IE],
) *StateMachineConfig[S, ST, E, IE] {
	return &StateMachineConfig[S, ST, E, IE]{
		conversions: defaultConversions,
		spec:        defaultSpec,
	}
}

func (cb *StateMachineConfig[S, ST, E, IE]) EventTypeConverter(conversions EventTypeConverter[S, ST, E, IE]) *StateMachineConfig[S, ST, E, IE] {
	cb.conversions = conversions
	return cb
}

func (cb *StateMachineConfig[S, ST, E, IE]) SystemActor(systemActor SystemActor) *StateMachineConfig[S, ST, E, IE] {
	cb.systemActor = &systemActor
	return cb
}

func (cb *StateMachineConfig[S, ST, E, IE]) StateTableName(stateTable string) *StateMachineConfig[S, ST, E, IE] {
	cb.spec.State.TableName = stateTable
	return cb
}

func (cb *StateMachineConfig[S, ST, E, IE]) StoreExtraStateColumns(stateColumns func(S) (map[string]interface{}, error)) *StateMachineConfig[S, ST, E, IE] {
	cb.spec.State.StoreExtraColumns = stateColumns
	return cb
}

func (cb *StateMachineConfig[S, ST, E, IE]) EventTableName(eventTable string) *StateMachineConfig[S, ST, E, IE] {
	cb.spec.Event.TableName = eventTable
	return cb
}

func (cb *StateMachineConfig[S, ST, E, IE]) StoreExtraEventColumns(eventColumns func(E) (map[string]interface{}, error)) *StateMachineConfig[S, ST, E, IE] {
	cb.spec.Event.StoreExtraColumns = eventColumns
	return cb
}

func (cb *StateMachineConfig[S, ST, E, IE]) PrimaryKey(primaryKey func(E) (map[string]interface{}, error)) *StateMachineConfig[S, ST, E, IE] {
	cb.spec.PrimaryKey = primaryKey
	return cb
}

func (cb *StateMachineConfig[S, ST, E, IE]) NewStateMachine() (*StateMachine[S, ST, E, IE], error) {
	return NewStateMachine[S, ST, E, IE](cb)
}
