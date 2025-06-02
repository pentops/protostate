package psm

import (
	"context"
	"time"

	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-messaging/o5msg"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type hookBaton[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	sideEffects []*sideEffect
	chainEvents []IE
	causedBy    E
}

type sideEffect struct {
	msg   o5msg.Message
	delay time.Duration
}

func (td *hookBaton[K, S, ST, SD, E, IE]) ChainEvent(inner IE) {
	td.chainEvents = append(td.chainEvents, inner)
}

func (td *hookBaton[K, S, ST, SD, E, IE]) SideEffect(msg o5msg.Message) {
	td.sideEffects = append(td.sideEffects, &sideEffect{
		msg: msg,
	})
}

func (td *hookBaton[K, S, ST, SD, E, IE]) DelayedSideEffect(msg o5msg.Message, delay time.Duration) {
	dmsg := &sideEffect{
		msg:   msg,
		delay: delay,
	}

	td.sideEffects = append(td.sideEffects, dmsg)
}

func (td *hookBaton[K, S, ST, SD, E, IE]) FullCause() E {
	return td.causedBy
}

func (td *hookBaton[K, S, ST, SD, E, IE]) AsCause() *psm_j5pb.Cause {
	causeMetadata := td.causedBy.PSMMetadata()
	return &psm_j5pb.Cause{
		Type: &psm_j5pb.Cause_PsmEvent{
			PsmEvent: &psm_j5pb.PSMEventCause{
				EventId:      causeMetadata.EventId,
				StateMachine: td.causedBy.PSMKeys().PSMFullName(),
			},
		},
	}
}

type transitionMutation[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	RunMutation(S, E) error
	EventType() string
}

type transitionHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	globalEventHook[K, S, ST, SD, E, IE]
	EventType() string
}

type globalEventHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	RunTransition(context.Context, sqrlx.Transaction, HookBaton[K, S, ST, SD, E, IE], S, E) error
	RunOnFollow() bool
}

type globalStateHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	RunTransition(context.Context, sqrlx.Transaction, HookBaton[K, S, ST, SD, E, IE], S) error
	RunOnFollow() bool
}

type transition[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	fromStatus ST
	toStatus   ST
	noop       bool
	eventType  string
	mutations  []transitionMutation[K, S, ST, SD, E, IE]

	eventHooks []globalEventHook[K, S, ST, SD, E, IE]
	stateHooks []globalStateHook[K, S, ST, SD, E, IE]
}

func (hs *transition[K, S, ST, SD, E, IE]) runMutations(
	ctx context.Context,
	state S,
	event E,
) error {

	log.Debug(ctx, "running transition mutations")
	log.WithFields(ctx, map[string]any{
		"mutationsCount": len(hs.mutations),
	}).Debug("running mutation")

	if hs.noop {
		return nil
	}

	if hs.toStatus != 0 {
		log.WithField(ctx, "toStatus", hs.toStatus).Debug("setting status")
		state.SetStatus(hs.toStatus)
	}

	for _, mutation := range hs.mutations {
		log.WithField(ctx, "mutation", mutation).Debug("running mutation")
		err := mutation.RunMutation(state, event)
		if err != nil {
			return err
		}
	}
	log.Debug(ctx, "mutation complete")
	return nil
}

func (hs *transition[K, S, ST, SD, E, IE]) runHooks(
	ctx context.Context,
	tx sqrlx.Transaction,
	baton HookBaton[K, S, ST, SD, E, IE],
	state S,
	event E,
) error {
	for _, hook := range hs.eventHooks {
		err := hook.RunTransition(ctx, tx, baton, state, event)
		if err != nil {
			return err
		}
	}

	for _, hook := range hs.stateHooks {
		err := hook.RunTransition(ctx, tx, baton, state)
		if err != nil {
			return err
		}

	}
	return nil
}

func (hs *transition[K, S, ST, SD, E, IE]) runFollowerHooks(
	ctx context.Context,
	tx sqrlx.Transaction,
	state S,
	event E,
) error {
	for _, hook := range hs.eventHooks {
		if !hook.RunOnFollow() {
			continue
		}
		err := hook.RunTransition(ctx, tx, nil, state, event)
		if err != nil {
			return err
		}
	}

	for _, hook := range hs.stateHooks {
		if !hook.RunOnFollow() {
			continue
		}
		err := hook.RunTransition(ctx, tx, nil, state)
		if err != nil {
			return err
		}
	}
	return nil
}
