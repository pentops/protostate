package psm

import (
	"context"
	"errors"
	"fmt"

	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-messaging/outbox"
	"github.com/pentops/sqrlx.go/sqrlx"
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
	state S,
	event E,
	captureState captureStateType,
) (*S, error) {

	if err := sm.validateEvent(event); err != nil {
		return nil, err
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

	err = sm.storeAfterMutation(ctx, tx, state, event)
	if err != nil {
		return nil, fmt.Errorf("after transition from %s on %s: %w",
			transition.fromStatus.ShortString(),
			transition.eventType,
			err)
	}

	var returnState *S
	switch captureState {
	case captureInitialState:
		rsVal := state.Clone().(S) //proto.Clone(state).(S)
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
		err = sm.protoValidator.Validate(se.msg)
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

	chain := []*EventSpec[K, S, ST, SD, E, IE]{}
	for _, chained := range baton.chainEvents {
		derived, err := sm.deriveEvent(event, chained)
		if err != nil {
			return nil, fmt.Errorf("derive chained: %w", err)
		}
		chain = append(chain, derived)
	}

	if err := sm.eventsMustBeUnique(ctx, tx, chain...); err != nil {
		if errors.Is(err, ErrDuplicateEventID) {
			return nil, ErrDuplicateChainedEventID
		}
		return nil, err
	}

	log.Info(ctx, "Event Complete")

	var captureIntermediate = captureNoState
	if captureState == captureFinalState {
		captureIntermediate = captureFinalState
	}

	for _, chainedEvent := range chain {
		prepared, err := sm.prepareEvent(state, chainedEvent)
		if err != nil {
			return nil, fmt.Errorf("prepare event: %w", err)
		}

		stateAfterRun, err := sm.runEvent(ctx, tx, state, prepared, captureIntermediate)
		if err != nil {
			return nil, fmt.Errorf("chained event: %s: %w", chainedEvent.Event.PSMEventKey(), err)
		}
		if captureState == captureFinalState {
			returnState = stateAfterRun
		}
	}

	return returnState, nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) followEvent(ctx context.Context, tx sqrlx.Transaction, state S, event E) error {

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
		return err
	}

	if err := transition.runMutations(ctx, state, event); err != nil {
		return fmt.Errorf("run transition: %w", err)
	}

	err = sm.storeAfterMutation(ctx, tx, state, event)
	if err != nil {
		return fmt.Errorf("after transition from %s on %s: %w",
			transition.fromStatus.ShortString(),
			transition.eventType,
			err)
	}

	if err := transition.runFollowerHooks(ctx, tx, state, event); err != nil {
		return fmt.Errorf("run transition hooks: %w", err)
	}

	return nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) storeAfterMutation(
	ctx context.Context,
	tx sqrlx.Transaction,
	state S,
	event E,
) error {

	if state.GetStatus() == 0 {
		return fmt.Errorf("state machine transitioned to zero status")
	}

	err := sm.validator.Validate(state.J5Reflect())
	if err != nil {
		return err
	}

	if err := sm.store(ctx, tx, state, event); err != nil {
		return err
	}

	return nil

}
