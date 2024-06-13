package psm

import (
	"context"

	"github.com/pentops/o5-messaging/o5msg"
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

type transitionHookSet[
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
	mutations  []TransitionMutation[K, S, ST, SD, E, IE]
	logic      []TransitionLogicHook[K, S, ST, SD, E, IE]
	data       []TransitionDataHook[K, S, ST, SD, E, IE]
}

func (hs *transitionHookSet[K, S, ST, SD, E, IE]) RunTransitionMutations(
	ctx context.Context,
	state S,
	event E,
) error {
	sd := state.PSMData()
	innerEvent := event.UnwrapPSMEvent()

	if hs.noop {
		return nil
	}

	if hs.toStatus != 0 {
		state.SetStatus(hs.toStatus)
	}

	for _, mutation := range hs.mutations {
		err := mutation.TransitionMutation(sd, innerEvent)
		if err != nil {
			return err
		}
	}
	return nil
}

func (hs *transitionHookSet[K, S, ST, SD, E, IE]) RunTransitionHooks(
	ctx context.Context,
	tx sqrlx.Transaction,
	baton HookBaton[K, S, ST, SD, E, IE],
	state S,
	event E,
) error {
	innerEvent := event.UnwrapPSMEvent()
	/*
		asType, ok := any(innerType).(SE)
		if !ok {
			name := innerType.ProtoReflect().Descriptor().FullName()
			return fmt.Errorf("unexpected event type in transition: %s [IE] does not match [SE] (%T)", name, new(SE))

		}*/

	for _, logic := range hs.logic {
		err := logic.TransitionLogicHook(ctx, baton, state, innerEvent)
		if err != nil {
			return err
		}
	}

	for _, data := range hs.data {
		err := data.TransitionDataHook(ctx, tx, state, innerEvent)
		if err != nil {
			return err
		}
	}

	return nil
}

type hookSet[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	logic     []GeneralLogicHook[K, S, ST, SD, E, IE]
	stateData []GeneralStateDataHook[K, S, ST, SD, E, IE]
	eventData []GeneralEventDataHook[K, S, ST, SD, E, IE]

	transitions []*transitionHookSet[K, S, ST, SD, E, IE]
}

func (hs *hookSet[K, S, ST, SD, E, IE]) LogicHook(hook GeneralLogicHook[K, S, ST, SD, E, IE]) {
	hs.logic = append(hs.logic, hook)
}

func (hs *hookSet[K, S, ST, SD, E, IE]) StateDataHook(hook GeneralStateDataHook[K, S, ST, SD, E, IE]) {
	hs.stateData = append(hs.stateData, hook)
}

func (hs *hookSet[K, S, ST, SD, E, IE]) EventDataHook(hook GeneralEventDataHook[K, S, ST, SD, E, IE]) {
	hs.eventData = append(hs.eventData, hook)
}

type TransitionHookSet[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	RunTransitionMutations(context.Context, S, E) error
	RunTransitionHooks(context.Context, sqrlx.Transaction, HookBaton[K, S, ST, SD, E, IE], S, E) error
}

func (hs hookSet[K, S, ST, SD, E, IE]) FindHooks(status ST, event E) []TransitionHookSet[K, S, ST, SD, E, IE] {

	hooks := []TransitionHookSet[K, S, ST, SD, E, IE]{}

	wantEventType := event.UnwrapPSMEvent().PSMEventKey()
	for _, ths := range hs.transitions {
		if ths.eventType != wantEventType {
			continue
		}
		for _, fromStatus := range ths.fromStatus {
			if fromStatus == status {
				hooks = append(hooks, ths)
				break
			}
		}
	}

	return hooks
}
