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
	systemActor SystemActor

	// KeyFields derives the key values from the Key entity. Should return
	// UUID Strings, and omit entries for NULL values
	keyValues func(K) (map[string]string, error)

	tableMap *TableMap
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) SystemActor(systemActor SystemActor) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.systemActor = systemActor
	return smc
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) TableMap(tableMap *TableMap) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.tableMap = tableMap
	return smc
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) DeriveKeyValues(cbFunc func(K) (map[string]string, error)) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.keyValues = cbFunc
	return smc
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) BuildStateMachine() (*StateMachine[K, S, ST, SD, E, IE], error) {
	return NewStateMachine(smc)
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) BuildQueryTableSpec() (*QueryTableSpec, error) {

	state := (*new(S)).ProtoReflect().Descriptor()
	event := (*new(E)).ProtoReflect().Descriptor()

	tableMap, err := tableMapFromStateAndEvent(state, event)
	if err != nil {
		return nil, err
	}

	return &QueryTableSpec{
		TableMap:  *tableMap,
		StateType: state,
		EventType: event,
	}, nil
}
