package psm

import (
	"context"
	"fmt"

	"github.com/pentops/outbox.pg.go/outbox"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
)

type StateHookBaton[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] interface {
	SideEffect(outbox.OutboxMessage)
	ChainEvent(*EventSpec[K, S, ST, E, IE])
	DeriveEvent(IE)
	FullCause() E
}

type TransitionBaton[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] StateHookBaton[K, S, ST, E, IE]

type TransitionData[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] struct {
	sideEffects []outbox.OutboxMessage
	chainEvents []*EventSpec[K, S, ST, E, IE]
	causedBy    E
}

func (td *TransitionData[K, S, ST, E, IE]) ChainEvent(event *EventSpec[K, S, ST, E, IE]) {
	td.chainEvents = append(td.chainEvents, event)
}

func (td *TransitionData[K, S, ST, E, IE]) DeriveEvent(inner IE) {
	td.chainEvents = append(td.chainEvents, &EventSpec[K, S, ST, E, IE]{
		Event: inner,
	})
}

func (td *TransitionData[K, S, ST, E, IE]) SideEffect(msg outbox.OutboxMessage) {
	td.sideEffects = append(td.sideEffects, msg)
}

func (td *TransitionData[K, S, ST, E, IE]) FullCause() E {
	return td.causedBy
}

type ITransitionHandler[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] interface {
	handlesEvent(E) bool
	runTransition(context.Context, S, E) error
}

type IStateHookHandler[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] interface {
	handlesEvent(E) bool
	runStateHook(context.Context, sqrlx.Transaction, StateHookBaton[K, S, ST, E, IE], S, E) error
}

type ICombinedHandler[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] interface {
	ITransitionHandler[K, S, ST, E, IE]
	IStateHookHandler[K, S, ST, E, IE]
}

// PSMCombinedFunc is returned by the generated FooPSMFunc, it exists for
// compatibility with the combined Do func method.
type PSMCombinedFunc[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
	SE IInnerEvent,
] func(context.Context, TransitionBaton[K, S, ST, E, IE], S, SE) error

// RunTransition implements TransitionHandler, where SE is the specific event
// cast from the interface IE provided in the call
func (f PSMCombinedFunc[K, S, ST, E, IE, SE]) runTransition( // nolint: unused // used when implementing ITransitionHandler
	ctx context.Context,
	state S,
	event E,
) error {

	// Creates a baton to *discard* the chains and side effects, they will be
	// called later when the transition is used in as state hook
	baton := &TransitionData[K, S, ST, E, IE]{
		causedBy: event,
	}
	// Cast the interface ET IInnerEvent to the specific type of event which
	// this func handles
	innerType := event.UnwrapPSMEvent()
	asType, ok := any(innerType).(SE)
	if !ok {

		name := asType.ProtoReflect().Descriptor().FullName()

		return fmt.Errorf("unexpected event type (b): %s [IE] does not match [SE] (%T)", name, new(SE))
	}
	return f(ctx, baton, state, asType)
}

func (f PSMCombinedFunc[K, S, ST, E, IE, SE]) runStateHook( // nolint: unused // used when implementing IStateHookHandler
	ctx context.Context,
	tx sqrlx.Transaction,
	baton StateHookBaton[K, S, ST, E, IE],
	state S,
	event E,
) error {
	// Cast the interface ET IInnerEvent to the specific type of event which
	// this func handles
	innerType := event.UnwrapPSMEvent()
	asType, ok := any(innerType).(SE)
	if !ok {
		name := innerType.ProtoReflect().Descriptor().FullName()

		return fmt.Errorf("unexpected event type (a): %s [IE] does not match [SE] (%T)", name, new(SE))
	}
	stateClone := proto.Clone(state).(S)
	// uses clone as hook should not modify the state
	return f(ctx, baton, stateClone, asType)
}

func (f PSMCombinedFunc[K, S, ST, E, IE, SE]) handlesEvent(outerEvent E) bool { // nolint:unused // used when implementing ITransitionHandler
	// Check if the parameter passed as ET (IInnerEvent) is the specific type
	// (IE, also IInnerEvent, but typed) which this transition handles
	event := outerEvent.UnwrapPSMEvent()
	_, ok := any(event).(SE)
	return ok
}

type PSMTransitionFunc[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
	SE IInnerEvent,
] func(context.Context, S, SE) error

func (f PSMTransitionFunc[K, S, ST, E, IE, SE]) runTransition( // nolint: unused // used when implementing ITransitionHandler
	ctx context.Context,
	state S,
	event E,
) error {
	// Cast the interface ET IInnerEvent to the specific type of event which
	// this func handles
	innerType := event.UnwrapPSMEvent()
	asType, ok := any(innerType).(SE)
	if !ok {

		name := innerType.ProtoReflect().Descriptor().FullName()

		return fmt.Errorf("unexpected event type (b): %s [IE] does not match [SE] (%T)", name, new(SE))
	}

	return f(ctx, state, asType)
}

func (f PSMTransitionFunc[K, S, ST, E, IE, SE]) handlesEvent(outerEvent E) bool { // nolint: unused // used when implementing ITransitionHandler
	// Check if the parameter passed as ET (IInnerEvent) is the specific type
	// (IE, also IInnerEvent, but typed) which this transition handles
	event := outerEvent.UnwrapPSMEvent()
	_, ok := any(event).(SE)
	return ok
}

type PSMHookFunc[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
	SE IInnerEvent,
] func(context.Context, sqrlx.Transaction, StateHookBaton[K, S, ST, E, IE], S, SE) error

func (f PSMHookFunc[K, S, ST, E, IE, SE]) runStateHook( // nolint: unused // used when implementing IStateHookHandler
	ctx context.Context,
	tx sqrlx.Transaction,
	baton StateHookBaton[K, S, ST, E, IE],
	state S,
	event E,
) error {
	// Cast the interface ET IInnerEvent to the specific type of event which
	// this func handles
	innerType := event.UnwrapPSMEvent()
	asType, ok := any(innerType).(SE)
	if !ok {
		name := innerType.ProtoReflect().Descriptor().FullName()
		return fmt.Errorf("unexpected event type (c): %s [IE] does not match [SE] (%T)", name, new(SE))
	}
	return f(ctx, tx, baton, state, asType)
}

func (f PSMHookFunc[K, S, ST, E, IE, SE]) handlesEvent(outerEvent E) bool { // nolint: unused // used when implementing IStateHookHandler
	// Check if the parameter passed as ET (IInnerEvent) is the specific type
	// (IE, also IInnerEvent, but typed) which this transition handles
	event := outerEvent.UnwrapPSMEvent()
	_, ok := any(event).(SE)
	return ok
}

type eventFilter[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,

] struct {
	fromStatus    []ST
	customFilters []func(E) bool
}

func (ef eventFilter[K, S, ST, E, IE]) matches(currentStatus ST, outerEvent E) bool {
	if ef.fromStatus != nil {
		didMatch := false
		for _, fromStatus := range ef.fromStatus {
			if fromStatus == currentStatus {
				didMatch = true
				break
			}
		}
		if !didMatch {
			return false
		}
	}

	for _, filter := range ef.customFilters {
		if !filter(outerEvent) {
			return false
		}
	}
	return true
}

type TransitionWrapper[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] struct {
	handler ITransitionHandler[K, S, ST, E, IE]
	eventFilter[K, S, ST, E, IE]
}

func (f TransitionWrapper[K, S, ST, E, IE]) Matches(status ST, outerEvent E) bool {
	if !f.handler.handlesEvent(outerEvent) {
		return false
	}

	return f.eventFilter.matches(status, outerEvent)
}

func (f TransitionWrapper[K, S, ST, E, IE]) RunTransition(
	ctx context.Context,
	state S,
	event E,
) error {
	return f.handler.runTransition(ctx, state, event)
}

type HookWrapper[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] struct {
	handler IStateHookHandler[K, S, ST, E, IE]
	eventFilter[K, S, ST, E, IE]
}

func (f HookWrapper[K, S, ST, E, IE]) matches(status ST, outerEvent E) bool {
	if !f.handler.handlesEvent(outerEvent) {
		return false
	}

	return f.eventFilter.matches(status, outerEvent)
}

func (f HookWrapper[K, S, ST, E, IE]) Matches(status ST, outerEvent E) bool {
	return f.matches(status, outerEvent)
}

func (f HookWrapper[K, S, ST, E, IE]) RunStateHook(
	ctx context.Context,
	tx sqrlx.Transaction,
	baton StateHookBaton[K, S, ST, E, IE],
	state S,
	event E,
) error {
	return f.handler.runStateHook(ctx, tx, baton, state, event)
}

type StateMachineTransitionBuilder[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] struct {
	sm *StateMachine[K, S, ST, E, IE]
	eventFilter[K, S, ST, E, IE]
}

func (ee *StateMachine[K, S, ST, E, IE]) From(states ...ST) StateMachineTransitionBuilder[K, S, ST, E, IE] {
	return StateMachineTransitionBuilder[K, S, ST, E, IE]{
		sm: ee,
		eventFilter: eventFilter[K, S, ST, E, IE]{
			fromStatus: states,
		},
	}
}

func (tb StateMachineTransitionBuilder[K, S, ST, E, IE]) Where(filter func(event IE) bool) StateMachineTransitionBuilder[K, S, ST, E, IE] {
	innerFilter := func(fullEvent E) bool {
		innerEvent := fullEvent.UnwrapPSMEvent()
		return filter(innerEvent)
	}
	tb.eventFilter.customFilters = append(tb.customFilters, innerFilter)
	return tb
}

// Do is a legacy method which combines a Transition and a Hook. The function
// will run twice, the first run will transition the state machine and discard
// the side effects, the second discards state transitions and runs only side
// effects. Use Transition and Hook instead.
func (tb StateMachineTransitionBuilder[K, S, ST, E, IE]) Do(
	handler ICombinedHandler[K, S, ST, E, IE],
) StateMachineTransitionBuilder[K, S, ST, E, IE] {

	typedTransition := &TransitionWrapper[K, S, ST, E, IE]{
		handler:     handler,
		eventFilter: tb.eventFilter,
	}

	tb.sm.Eventer.Register(typedTransition)

	typedHook := &HookWrapper[K, S, ST, E, IE]{
		handler:     handler,
		eventFilter: tb.eventFilter,
	}

	tb.sm.AddHook(typedHook)

	return tb
}

func (tb StateMachineTransitionBuilder[K, S, ST, E, IE]) Transition(
	transition ITransitionHandler[K, S, ST, E, IE],
) StateMachineTransitionBuilder[K, S, ST, E, IE] {

	typedTransition := &TransitionWrapper[K, S, ST, E, IE]{
		handler:     transition,
		eventFilter: tb.eventFilter,
	}

	tb.sm.Eventer.Register(typedTransition)

	return tb
}

func (tb StateMachineTransitionBuilder[K, S, ST, E, IE]) Hook(
	hook IStateHookHandler[K, S, ST, E, IE],
) StateMachineTransitionBuilder[K, S, ST, E, IE] {

	typedHook := &HookWrapper[K, S, ST, E, IE]{
		handler:     hook,
		eventFilter: tb.eventFilter,
	}

	tb.sm.AddHook(typedHook)

	return tb
}

// StateHook runs after a state machine transition. Very similar to a
// TransitionHandler, but it has access to the database transaction and is
// designed for either business logic requiring the database, or for more
// generic hooks which need to run after all transitions.
type StateHook[
	K IKeyset,
	S IState[K, ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[K, S, ST, IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event but untyped
	SE IInnerEvent, // Typed Inner Event, the specifically typed event *interface*
] func(context.Context, sqrlx.Transaction, S, E) error

// GeneralStateHook is a StateHook that should do something after all or most
// transitions, e.g. publishing to a global event bus or updating a cache.
type GeneralStateHook[
	K IKeyset,
	S IState[K, ST],
	ST IStatusEnum,
	E IEvent[K, S, ST, IE],
	IE IInnerEvent,
] func(context.Context, sqrlx.Transaction, S, E) error

func (hook GeneralStateHook[K, S, ST, E, IE]) RunStateHook(
	ctx context.Context,
	tx sqrlx.Transaction,
	baton StateHookBaton[K, S, ST, E, IE],
	state S,
	event E,
) error {
	return hook(ctx, tx, state, event)
}

func (hook GeneralStateHook[K, S, ST, E, IE]) Matches(ST, E) bool {
	return true
}
