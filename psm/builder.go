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

	tableName *string
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) SystemActor(systemActor SystemActor) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.systemActor = systemActor
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

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) DeriveKeyValues(cbFunc func(K) (map[string]string, error)) *StateMachineConfig[K, S, ST, SD, E, IE] {
	smc.keyValues = cbFunc
	return smc
}

func (smc *StateMachineConfig[K, S, ST, SD, E, IE]) apply() error {
	if smc.tableMap == nil {
		state := (*new(S)).ProtoReflect().Descriptor()
		event := (*new(E)).ProtoReflect().Descriptor()
		tableMap, err := tableMapFromStateAndEvent(state, event)
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

	state := (*new(S)).ProtoReflect().Descriptor()
	event := (*new(E)).ProtoReflect().Descriptor()

	return &QueryTableSpec{
		TableMap:  *smc.tableMap,
		StateType: state,
		EventType: event,
	}, nil
}
