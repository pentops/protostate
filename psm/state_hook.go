package psm

/*
// StateHook runs after a state machine transition. Very similar to a
// TransitionHandler, but it has access to the database transaction and is
// designed for either business logic requiring the database, or for more
// generic hooks which need to run after all transitions.
type StateHook[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event but untyped
	SE IInnerEvent, // Typed Inner Event, the specifically typed event *interface*
] func(context.Context, sqrlx.Transaction, S, E) error



// GenericStateHook is a StateHook that should do something after all or most
// transitions, e.g. publishing to a global event bus or updating a cache.
type GenericStateHook[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
] func(context.Context, sqrlx.Transaction, S, E) error

func (hook GenericStateHook[S, ST, E, IE]) RunStateHook(ctx context.Context, tx sqrlx.Transaction, baton StateHookBaton[E, IE], state S, innerEvent IE) error {
	outerEvent := baton.FullCause()
	return hook(ctx, tx, state, outerEvent)
}


// TypedStateHook is a StateHook that should do something after a specific type
// of event, it runs like a Transition but can run business logic involving the
// database.
type TypedStateHook[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
	SE IInnerEvent, // Typed Inner Event, the specifically typed event *interface*
] struct {
	handler     TypedStateHookHandler[S, ST, E, IE, SE]
	afterStatus []ST
}

type TypedStateHookHandler[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
	SE IInnerEvent,
] func(context.Context, sqrlx.Transaction, StateHookBaton[E, IE], S, SE) error

func (h TypedStateHook[S, ST, E, IE, SE]) RunStateHook(ctx context.Context, tx sqrlx.Transaction, baton StateHookBaton[E, IE], state S, innerEvent IE) error {

	if len(h.afterStatus) > 0 {
		// Check if the current status is in the list of statuses
		// that this hook should run after.
		status := state.GetStatus()
		match := false
		for _, s := range h.afterStatus {
			if status == s {
				match = true
				break
			}
		}
		if !match {
			return nil
		}

	}

	asType, ok := any(innerEvent).(SE)
	if !ok {
		// In transitions this returns an error because they are
		// pre-filtered/selected as **the** transition to run.
		// here multiple hooks or no hook can run, so the filtering is done on-the-fly.
		return nil
	}

	return h.handler(ctx, tx, baton, state, asType)
}*/
