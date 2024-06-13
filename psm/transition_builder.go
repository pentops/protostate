package psm

// From creates a new transition builder for the state machine, which will run
// when the machine is transitioning from one of the given states. If the given
// list is empty (From()), matches ALL starting states.
func (hs *transitionSet[K, S, ST, SD, E, IE]) From(states ...ST) BuilderFrom[K, S, ST, SD, E, IE] {
	return BuilderFrom[K, S, ST, SD, E, IE]{
		hs:         hs,
		fromStatus: states,
	}
}

type BuilderFrom[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	hs         *transitionSet[K, S, ST, SD, E, IE]
	fromStatus []ST
}

// OnEvent identifies a specific transition from state(s) for an event.
func (tb BuilderFrom[K, S, ST, SD, E, IE]) OnEvent(event string) *TransitionBuilder[K, S, ST, SD, E, IE] {
	ths := &transitionSpec[K, S, ST, SD, E, IE]{
		fromStatus: tb.fromStatus,
		eventType:  event,
	}

	tb.hs.transitions = append(tb.hs.transitions, ths)

	return &TransitionBuilder[K, S, ST, SD, E, IE]{
		fromStatus: tb.fromStatus,
		onEvent:    event,
		hookSet:    ths,
	}
}

// Mutate is a shortcut for OnEvent().Mutate() with the event type matching callback function.
func (tb BuilderFrom[K, S, ST, SD, E, IE]) Mutate(
	hook transitionMutationWrapper[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	eventType := hook.EventType()
	return tb.OnEvent(eventType).Mutate(hook)
}

// LogicHook is a shortcut for OnEvent().LogicHook() with the event type matching callback function.
func (tb BuilderFrom[K, S, ST, SD, E, IE]) LogicHook(
	hook transitionLogicHookWrapper[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	eventType := hook.EventType()
	return tb.OnEvent(eventType).LogicHook(hook)
}

// DataHook is a shortcut for OnEvent().DataHook() with the event type matching callback function.
func (tb BuilderFrom[K, S, ST, SD, E, IE]) DataHook(
	hook transitionDataHookWrapper[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	eventType := hook.EventType()
	return tb.OnEvent(eventType).DataHook(hook)
}

type TransitionBuilder[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	hookSet    *transitionSpec[K, S, ST, SD, E, IE]
	fromStatus []ST
	onEvent    string
}

// Mutate adds a function which will merge data from the event into the state
// data. A transition can have zero or one mutate function. Multiple mutation
// callbacks may be registered.
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) Mutate(
	mutation transitionMutationWrapper[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.hookSet.mutations = append(tb.hookSet.mutations, mutation)
	return tb
}

// SetStatus sets status which the state machine will have after the transition.
// Not all transitions will change the status, some will only mutate.
// Calling this function replaces any previous value.
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) SetStatus(
	status ST,
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.hookSet.toStatus = status
	return tb
}

// LogicHook adds a new TransitionLogicHook which will run after all mutations
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) LogicHook(
	hook transitionLogicHookWrapper[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.hookSet.logic = append(tb.hookSet.logic, hook)
	return tb
}

// DataHook adds a new TransitionDataHook which will run after all mutations
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) DataHook(
	hook transitionDataHookWrapper[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.hookSet.data = append(tb.hookSet.data, hook)
	return tb
}

// Noop registers a transition which does nothing, but prevents the machine from
// erroring when the conditions are met.
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) Noop() *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.hookSet.noop = true
	return tb
}

// LinkTo registers a transition into another local state machine to run in the
// same database transaction
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) LinkTo(
	link transitionLink[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.hookSet.links = append(tb.hookSet.links, link)
	return tb
}
