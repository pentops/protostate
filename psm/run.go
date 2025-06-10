package psm

import (
	"context"
	"fmt"

	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-messaging/outbox"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
)

type captureStateType int

const (
	captureInitialState captureStateType = iota
	captureFinalState
	captureNoState
)

func (sm *StateMachine[K, S, ST, SD, E, IE]) runEvent(
	ctx context.Context,
	tx sqrlx.Transaction,
	bb preparedEvent[K, S, ST, SD, E, IE],
	captureState captureStateType,
) (*S, error) {
	event := bb.event
	state := bb.state

	if err := sm.validator.Validate(event); err != nil {
		return nil, fmt.Errorf("validating event %s: %w", event.ProtoReflect().Descriptor().FullName(), err)
	}

	typeKey := event.UnwrapPSMEvent().PSMEventKey()
	statusBefore := state.GetStatus()

	ctx = log.WithFields(ctx, map[string]any{
		"stateMachine": state.PSMKeys().PSMFullName(),
		"transition": map[string]any{
			"eventId": event.PSMMetadata().EventId,
			"from":    statusBefore.ShortString(),
			"event":   typeKey,
		},
	})

	transition, err := sm.buildTransition(statusBefore, typeKey)
	if err != nil {
		return nil, err
	}

	if err := transition.runMutations(ctx, state, event); err != nil {
		return nil, fmt.Errorf("run transition: %w", err)
	}

	err = sm.storeStateAndEvent(ctx, tx, bb)
	if err != nil {
		return nil, fmt.Errorf("after transition from %s on %s: %w",
			transition.fromStatus.ShortString(),
			transition.eventType,
			err)
	}

	var returnState *S
	switch captureState {
	case captureInitialState:
		rsVal := proto.Clone(state).(S)
		returnState = &rsVal
	case captureFinalState:
		returnState = &state
	}

	baton := &hookBaton[K, S, ST, SD, E, IE]{
		causedBy: event,
	}

	if err := transition.runHooks(ctx, tx, baton, state, event); err != nil {
		return nil, fmt.Errorf("run transition hooks: %w", err)
	}

	for _, se := range baton.sideEffects {
		err = sm.validator.Validate(se.msg)
		if err != nil {
			return nil, fmt.Errorf("validate side effect: %s %w", se.msg.ProtoReflect().Descriptor().FullName(), err)
		}

		if se.delay == 0 {
			err = outbox.DefaultSender.Send(ctx, tx, se.msg)
			if err != nil {
				return nil, fmt.Errorf("side effect outbox: %w", err)
			}
		} else {
			err = outbox.DefaultSender.SendDelayed(ctx, tx, se.delay, se.msg)
			if err != nil {
				return nil, fmt.Errorf("delayed side effect outbox: %w", err)
			}
		}
	}

	log.Info(ctx, "Event Complete")

	var captureIntermediate = captureNoState
	if captureState == captureFinalState {
		captureIntermediate = captureFinalState
	}

	for _, chained := range baton.chainEvents {
		derived, err := sm.deriveEvent(event, chained)
		if err != nil {
			return nil, fmt.Errorf("derive chained: %w", err)
		}

		prepared, err := derived.buildWrapper(state)
		if err != nil {
			return nil, fmt.Errorf("prepare event: %w", err)
		}

		stateAfterRun, err := sm.runEvent(ctx, tx, prepared, captureIntermediate)
		if err != nil {
			return nil, fmt.Errorf("chained event: %s: %w", derived.Event.PSMEventKey(), err)
		}
		if captureState == captureFinalState {
			returnState = stateAfterRun
		}
	}

	return returnState, nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) followEvent(ctx context.Context, tx sqrlx.Transaction, se preparedEvent[K, S, ST, SD, E, IE]) error {

	typeKey := se.event.UnwrapPSMEvent().PSMEventKey()
	statusBefore := se.state.GetStatus()

	ctx = log.WithFields(ctx, map[string]any{
		"stateMachine": se.state.PSMKeys().PSMFullName(),
		"transition": map[string]any{
			"eventId": se.event.PSMMetadata().EventId,
			"from":    statusBefore.ShortString(),
			"event":   typeKey,
		},
	})

	transition, err := sm.buildTransition(statusBefore, typeKey)
	if err != nil {
		return err
	}

	if err := transition.runMutations(ctx, se.state, se.event); err != nil {
		return fmt.Errorf("run transition: %w", err)
	}

	err = sm.storeStateAndEvent(ctx, tx, se)
	if err != nil {
		return fmt.Errorf("after transition from %s on %s: %w",
			transition.fromStatus.ShortString(),
			transition.eventType,
			err)
	}

	if err := transition.runFollowerHooks(ctx, tx, se.state, se.event); err != nil {
		return fmt.Errorf("run transition hooks: %w", err)
	}

	return nil
}
