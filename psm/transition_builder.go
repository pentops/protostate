package psm

// From creates a new transition builder for the state machine, which will run
// when the machine is transitioning from one of the given states. If the given
// list is empty (From()), matches ALL starting states.
func (ee *StateMachine[K, S, ST, SD, E, IE]) From(states ...ST) BuilderFrom[K, S, ST, SD, E, IE] {
	return BuilderFrom[K, S, ST, SD, E, IE]{
		sm:         ee,
		fromStatus: states,
	}
}

func (ee *StateMachine[K, S, ST, SD, E, IE]) GeneralHook(hook GeneralStateHook[K, S, ST, SD, E, IE]) {
	ee.hooks = append(ee.hooks, hook)
}

type BuilderFrom[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	sm         *StateMachine[K, S, ST, SD, E, IE]
	fromStatus []ST
}

// OnEvent identifies a specific transition from state(s) for an event.
func (tb BuilderFrom[K, S, ST, SD, E, IE]) OnEvent(event string) *TransitionBuilder[K, S, ST, SD, E, IE] {
	return &TransitionBuilder[K, S, ST, SD, E, IE]{
		sm: tb.sm,
		eventFilter: eventFilter[K, S, ST, SD, E, IE]{
			fromStatus: tb.fromStatus,
			onEvents:   []string{event},
		},
	}
}

// Mutate is a shortcut for OnEvent().Mutate() with the event type matching callback
// function.
func (tb BuilderFrom[K, S, ST, SD, E, IE]) Mutate(
	transition ITransitionHandler[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	eventType := transition.eventType()
	return tb.OnEvent(eventType).Mutate(transition)
}

// Hook is a shortcut for OnEvent().Hook() with the event type matching callback
// function.
func (tb BuilderFrom[K, S, ST, SD, E, IE]) Hook(
	hook IStateHookHandler[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	eventType := hook.eventType()
	return tb.OnEvent(eventType).Hook(hook)
}

type TransitionBuilder[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	sm                *StateMachine[K, S, ST, SD, E, IE]
	eventFilter       eventFilter[K, S, ST, SD, E, IE]
	transitionWrapper *TransitionWrapper[K, S, ST, SD, E, IE]
}

func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) transition() *TransitionWrapper[K, S, ST, SD, E, IE] {
	if tb.transitionWrapper == nil {
		tb.transitionWrapper = &TransitionWrapper[K, S, ST, SD, E, IE]{
			handler:     nil, // set by Mutate
			toStatus:    0,   // set by SetStatus
			eventFilter: tb.eventFilter,
		}

		tb.sm.Eventer.Register(tb.transitionWrapper)
	}
	return tb.transitionWrapper
}

// Mutate sets the function which will merge data from the event into the state
// data. A transition can have zero or one mutate function, calling this
// function replaces any previous function.
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) Mutate(
	transition ITransitionHandler[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.transition().handler = transition
	return tb
}

// SetStatus sets status which the state machine will have after the transition.
// Not all transitions will change the status, some will only mutate.
// Calling this function replaces any previous value.
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) SetStatus(
	status ST,
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.transition().toStatus = status
	return tb
}

// Hook creates and registers a new hook to run after the transition. Multiple
// hooks can be attached to the same transition.
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) Hook(
	hook IStateHookHandler[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {

	typedHook := &HookWrapper[K, S, ST, SD, E, IE]{
		handler:     hook,
		eventFilter: tb.eventFilter,
	}

	tb.sm.addHook(typedHook)
	return tb
}

// Noop registers a transition which does nothing, but prevents the machine from
// erroring when the conditions are met.
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) Noop() *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.transition()
	return tb
}
