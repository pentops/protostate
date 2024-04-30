package psm

import (
	"context"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type ITransition[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
] interface {
	Matches(ST, E) bool
	RunTransition(context.Context, S, E) error
}

type IStateHook[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
] interface {
	Matches(ST, E) bool
	RunStateHook(context.Context, sqrlx.Transaction, StateHookBaton[E, IE], S, E) error
}

// Eventer is the inner state machine, independent of storage.
type Eventer[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event *interface*
] struct {
	conversions EventTypeConverter[S, ST, E, IE]

	Transitions []ITransition[S, ST, E, IE]

	validator *protovalidate.Validator
}

func (ee Eventer[S, ST, E, IE]) FindTransition(status ST, event E) (ITransition[S, ST, E, IE], error) {
	for _, search := range ee.Transitions {
		if search.Matches(status, event) {
			return search, nil
		}
	}

	innerEvent := event.UnwrapPSMEvent()
	typeKey := innerEvent.PSMEventKey() // ee.conversions.EventLabel(innerEvent)
	return nil, fmt.Errorf("no transition found for status %s -> %s",
		status.String(),
		typeKey,
	)
}

func (ee *Eventer[S, ST, E, IE]) ValidateEvent(event E) error {
	if ee.validator == nil {
		v, err := protovalidate.New()
		if err != nil {
			fmt.Println("failed to initialize validator:", err)
		}
		ee.validator = v
	}

	return ee.validator.Validate(event)
}

func (ee Eventer[S, ST, E, IE]) RunEvent(
	ctx context.Context,
	state S,
	innerEvent E,
) error {

	if err := ee.ValidateEvent(innerEvent); err != nil {
		return fmt.Errorf("validating event %s: %w", innerEvent.ProtoReflect().Descriptor().FullName(), err)
	}

	unwrapped := innerEvent.UnwrapPSMEvent()

	typeKey := unwrapped.PSMEventKey()
	stateBefore := state.GetStatus()

	ctx = log.WithFields(ctx, map[string]interface{}{
		"eventType":  typeKey,
		"transition": fmt.Sprintf("%s -> ? : %s", stateBefore.ShortString(), typeKey),
	})

	log.Debug(ctx, "Begin Event")

	transition, err := ee.FindTransition(stateBefore, innerEvent)
	if err != nil {
		return fmt.Errorf("find transition: %w", err)
	}

	if err := transition.RunTransition(ctx, state, innerEvent); err != nil {
		log.WithError(ctx, err).Error("Running Transition")
		return fmt.Errorf("run transition: %w", err)
	}

	ctx = log.WithFields(ctx, map[string]interface{}{
		"transition": fmt.Sprintf("%s -> %s : %s", stateBefore, state.GetStatus().String(), typeKey),
	})

	log.Info(ctx, "Event Handled")

	return nil
}

func (ee *Eventer[S, ST, E, IE]) Register(transition ITransition[S, ST, E, IE]) {
	ee.Transitions = append(ee.Transitions, transition)
}
