package psm

import (
	"context"
	"fmt"

	"github.com/pentops/o5-messaging.go/o5msg"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type HookBaton[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	SideEffect(o5msg.Message)
	ChainEvent(IE)
	FullCause() E
	AsCause() *psm_pb.Cause
}

type hookBaton[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	sideEffects []o5msg.Message

	chainEvents []IE
	causedBy    E
}

func (td *hookBaton[K, S, ST, SD, E, IE]) ChainEvent(inner IE) {
	td.chainEvents = append(td.chainEvents, inner)
}

func (td *hookBaton[K, S, ST, SD, E, IE]) SideEffect(msg o5msg.Message) {
	td.sideEffects = append(td.sideEffects, msg)
}

func (td *hookBaton[K, S, ST, SD, E, IE]) FullCause() E {
	return td.causedBy
}

func (td *hookBaton[K, S, ST, SD, E, IE]) AsCause() *psm_pb.Cause {
	causeMetadata := td.causedBy.PSMMetadata()
	return &psm_pb.Cause{
		Type: &psm_pb.Cause_PsmEvent{
			PsmEvent: &psm_pb.PSMEventCause{
				EventId:      causeMetadata.EventId,
				StateMachine: td.causedBy.PSMKeys().PSMFullName(),
			},
		},
	}
}

type ITransitionHandler[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	runTransition(context.Context, SD, E) error
	eventType() string
}

type IStateHookHandler[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	handlesEvent(E) bool
	eventType() string
	runStateHook(context.Context, sqrlx.Transaction, HookBaton[K, S, ST, SD, E, IE], S, E) error
}

type PSMMutationFunc[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
	SE IInnerEvent,
] func(SD, SE) error

func (f PSMMutationFunc[K, S, ST, SD, E, IE, SE]) runTransition( // nolint: unused // used when implementing ITransitionHandler
	ctx context.Context,
	stateData SD,
	event E,
) error {
	// Cast the interface ET IInnerEvent to the specific type of event which
	// this func handles
	innerType := event.UnwrapPSMEvent()
	asType, ok := any(innerType).(SE)
	if !ok {

		name := innerType.ProtoReflect().Descriptor().FullName()

		return fmt.Errorf("unexpected event type in transition: %s [IE] does not match [SE] (%T)", name, new(SE))
	}

	return f(stateData, asType)
}

func (f PSMMutationFunc[K, S, ST, SD, E, IE, SE]) eventType() string { // nolint: unused // used when implementing ITransitionHandler
	return (*new(SE)).PSMEventKey()
}

type PSMHookFunc[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
	SE IInnerEvent,
] func(context.Context, sqrlx.Transaction, HookBaton[K, S, ST, SD, E, IE], S, SE) error

func (f PSMHookFunc[K, S, ST, SD, E, IE, SE]) runStateHook( // nolint: unused // Implements IStateHookHandler

	ctx context.Context,
	tx sqrlx.Transaction,
	baton HookBaton[K, S, ST, SD, E, IE],
	state S,
	event E,
) error {
	// Cast the interface ET IInnerEvent to the specific type of event which
	// this func handles
	innerType := event.UnwrapPSMEvent()
	asType, ok := any(innerType).(SE)
	if !ok {
		name := innerType.ProtoReflect().Descriptor().FullName()
		return fmt.Errorf("unexpected event typein hook: %s [IE] does not match [SE] (%T)", name, new(SE))
	}
	return f(ctx, tx, baton, state, asType)
}

func (f PSMHookFunc[K, S, ST, SD, E, IE, SE]) handlesEvent(outerEvent E) bool { // nolint: unused // Implements IStateHookHandler
	// Check if the parameter passed as ET (IInnerEvent) is the specific type
	// (IE, also IInnerEvent, but typed) which this transition handles
	event := outerEvent.UnwrapPSMEvent()
	_, ok := any(event).(SE)
	return ok
}

func (f PSMHookFunc[K, S, ST, SD, E, IE, SE]) eventType() string { // nolint: unused // linter seems broken.
	return (*new(SE)).PSMEventKey()
}

type eventFilter[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,

] struct {
	fromStatus []ST
	onEvents   []string
}

func (ef eventFilter[K, S, ST, SD, E, IE]) Matches(currentStatus ST, outerEvent E) bool {
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

	if ef.onEvents != nil {
		eventKey := outerEvent.UnwrapPSMEvent().PSMEventKey()
		didMatch := false
		for _, onEvent := range ef.onEvents {
			if onEvent == eventKey {
				didMatch = true
				break
			}
		}
		if !didMatch {
			return false
		}
	}

	return true
}

type TransitionWrapper[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	handler  ITransitionHandler[K, S, ST, SD, E, IE]
	toStatus ST
	eventFilter[K, S, ST, SD, E, IE]
}

func (f TransitionWrapper[K, S, ST, SD, E, IE]) RunTransition(
	ctx context.Context,
	state S,
	event E,
) error {
	sd := state.PSMData()
	if f.toStatus != 0 {
		state.SetStatus(f.toStatus)
	}
	if f.handler != nil {
		err := f.handler.runTransition(ctx, sd, event)
		if err != nil {
			return err
		}
	}
	return nil
}

type HookWrapper[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	handler IStateHookHandler[K, S, ST, SD, E, IE]
	eventFilter[K, S, ST, SD, E, IE]
}

func (f HookWrapper[K, S, ST, SD, E, IE]) RunStateHook(
	ctx context.Context,
	tx sqrlx.Transaction,
	baton HookBaton[K, S, ST, SD, E, IE],
	state S,
	event E,
) error {
	return f.handler.runStateHook(ctx, tx, baton, state, event)
}

// GeneralStateHook is a StateHook that should do something after all or most
// transitions, e.g. publishing to a global event bus or updating a cache.
type GeneralStateHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] func(context.Context, sqrlx.Transaction, HookBaton[K, S, ST, SD, E, IE], S, E) error

func (hook GeneralStateHook[K, S, ST, SD, E, IE]) RunStateHook(
	ctx context.Context,
	tx sqrlx.Transaction,
	baton HookBaton[K, S, ST, SD, E, IE],
	state S,
	event E,
) error {
	return hook(ctx, tx, baton, state, event)
}

func (hook GeneralStateHook[K, S, ST, SD, E, IE]) Matches(ST, E) bool {
	return true
}
