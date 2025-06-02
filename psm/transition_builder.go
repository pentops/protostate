package psm

import (
	"fmt"
	"slices"
)

type transitionSpec[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	fromStatus []ST
	toStatus   ST
	noop       bool
	eventType  string
	mutations  []transitionMutation[K, S, ST, SD, E, IE]
	hooks      []transitionHook[K, S, ST, SD, E, IE]
}

type transitionSet[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	globalEventHooks []globalEventHook[K, S, ST, SD, E, IE]
	globalStateHooks []globalStateHook[K, S, ST, SD, E, IE]
	transitions      []*transitionSpec[K, S, ST, SD, E, IE]
}

func (hs *transitionSet[K, S, ST, SD, E, IE]) LogicHook(hook GeneralEventHook[K, S, ST, SD, E, IE]) {
	hs.globalEventHooks = append(hs.globalEventHooks, hook)
}

func (hs *transitionSet[K, S, ST, SD, E, IE]) StateDataHook(hook GeneralStateHook[K, S, ST, SD, E, IE]) {
	hs.globalStateHooks = append(hs.globalStateHooks, hook)
}

func (hs *transitionSet[K, S, ST, SD, E, IE]) EventDataHook(hook GeneralEventHook[K, S, ST, SD, E, IE]) {
	hs.globalEventHooks = append(hs.globalEventHooks, hook)
}

func (hs *transitionSet[K, S, ST, SD, E, IE]) PublishEvent(hook GeneralEventHook[K, S, ST, SD, E, IE]) {
	hs.globalEventHooks = append(hs.globalEventHooks, hook)
}

func (hs *transitionSet[K, S, ST, SD, E, IE]) PublishUpsert(hook GeneralStateHook[K, S, ST, SD, E, IE]) {
	hs.globalStateHooks = append(hs.globalStateHooks, hook)
}

// From creates a new transition builder for the state machine, which will run
// when the machine is transitioning from one of the given states. If the given
// list is empty (From()), matches ALL starting states.
func (hs *transitionSet[K, S, ST, SD, E, IE]) From(states ...ST) BuilderFrom[K, S, ST, SD, E, IE] {
	return BuilderFrom[K, S, ST, SD, E, IE]{
		hs:         hs,
		fromStatus: states,
	}
}

func (hs transitionSet[K, S, ST, SD, E, IE]) buildTransition(status ST, eventType string) (*transition[K, S, ST, SD, E, IE], error) {
	foundSpecs := make([]*transitionSpec[K, S, ST, SD, E, IE], 0, 1)

	for _, search := range hs.transitions {
		if search.eventType != eventType {
			continue
		}
		if len(search.fromStatus) == 0 {
			foundSpecs = append(foundSpecs, search)
			continue
		}
		if slices.Contains(search.fromStatus, status) {
			foundSpecs = append(foundSpecs, search)
		}
	}

	if len(foundSpecs) == 0 {
		return nil, fmt.Errorf("no transition found for %s on %s", status, eventType)
	}

	merged := &transition[K, S, ST, SD, E, IE]{
		fromStatus: status,
		eventType:  eventType,
	}

	for _, spec := range foundSpecs {
		merged.mutations = append(merged.mutations, spec.mutations...)
		for _, hook := range spec.hooks {
			merged.eventHooks = append(merged.eventHooks, hook)
		}

		if spec.toStatus != 0 {
			if merged.toStatus == 0 {
				merged.toStatus = spec.toStatus
			} else if merged.toStatus != spec.toStatus {
				return nil, fmt.Errorf("conflicting toStatus transitions for fromStatus %q event %q", status.ShortString(), eventType)
			}
		}
	}

	merged.eventHooks = append(merged.eventHooks, hs.globalEventHooks...)
	merged.stateHooks = append(merged.stateHooks, hs.globalStateHooks...)

	return merged, nil
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
	hook transitionMutation[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	eventType := hook.eventType()
	return tb.OnEvent(eventType).Mutate(hook)

}

// Hook is a shortcut for OnEvent().Hook() with the event type matching callback function.
func (tb BuilderFrom[K, S, ST, SD, E, IE]) Hook(
	hook transitionHook[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	eventType := hook.eventType()
	return tb.OnEvent(eventType).Hook(hook)
}

// DEPRECATED: use Hook
func (tb BuilderFrom[K, S, ST, SD, E, IE]) LogicHook(
	hook transitionHook[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	eventType := hook.eventType()
	return tb.OnEvent(eventType).Hook(hook)
}

// DEPRECATED: use Hook
func (tb BuilderFrom[K, S, ST, SD, E, IE]) DataHook(
	hook transitionHook[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	eventType := hook.eventType()
	return tb.OnEvent(eventType).Hook(hook)
}

// DEPRECATED: use Hook
func (tb BuilderFrom[K, S, ST, SD, E, IE]) LinkTo(
	link transitionHook[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	eventType := link.eventType()
	return tb.OnEvent(eventType).LinkTo(link)
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
	mutation transitionMutation[K, S, ST, SD, E, IE],
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

// Hook adds one or more TransitionHooks to the transition.
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) Hook(
	hooks ...transitionHook[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.hookSet.hooks = append(tb.hookSet.hooks, hooks...)
	return tb
}

// DEPRECATED: use Hook
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) LogicHook(
	hook transitionHook[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.hookSet.hooks = append(tb.hookSet.hooks, hook)
	return tb
}

// DEPRECATED: use Hook
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) DataHook(
	hook transitionHook[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.hookSet.hooks = append(tb.hookSet.hooks, hook)
	return tb
}

// Noop registers a transition which does nothing, but prevents the machine from
// erroring when the conditions are met.
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) Noop() *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.hookSet.noop = true
	return tb
}

// DEPRECATED: use Hook
func (tb *TransitionBuilder[K, S, ST, SD, E, IE]) LinkTo(
	hook transitionHook[K, S, ST, SD, E, IE],
) *TransitionBuilder[K, S, ST, SD, E, IE] {
	tb.hookSet.hooks = append(tb.hookSet.hooks, hook)
	return tb
}
