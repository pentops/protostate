package psm

import (
	"context"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ITransition[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	Matches(ST, E) bool
	RunTransition(context.Context, S, E) error
}

type IStateHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	Matches(ST, E) bool
	RunStateHook(context.Context, sqrlx.Transaction, HookBaton[K, S, ST, SD, E, IE], S, E) error
}

type eventerCallback[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] func(ctx context.Context, statusBefore ST, mutableState S, event E) error

// Eventer is the inner state machine, independent of storage.
type Eventer[
	K IKeyset,
	S IState[K, ST, SD], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	SD IStateData,
	E IEvent[K, S, ST, SD, IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event *interface*
] struct {
	Transitions []ITransition[K, S, ST, SD, E, IE]
	validator   *protovalidate.Validator
}

func (ee Eventer[K, S, ST, SD, E, IE]) FindTransition(status ST, event E) (ITransition[K, S, ST, SD, E, IE], error) {
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

func (ee *Eventer[K, S, ST, SD, E, IE]) ValidateEvent(event E) error {
	if ee.validator == nil {
		v, err := protovalidate.New()
		if err != nil {
			fmt.Println("failed to initialize validator:", err)
		}
		ee.validator = v
	}

	return ee.validator.Validate(event)
}

func (ee Eventer[K, S, ST, SD, E, IE]) RunEvent(
	ctx context.Context,
	state S,
	spec *EventSpec[K, S, ST, SD, E, IE],
	callbacks ...eventerCallback[K, S, ST, SD, E, IE],
) error {

	innerEvent, err := ee.prepareEvent(state, spec)
	if err != nil {
		return fmt.Errorf("prepare event: %w", err)
	}

	if err := ee.ValidateEvent(innerEvent); err != nil {
		return fmt.Errorf("validating event %s: %w", innerEvent.ProtoReflect().Descriptor().FullName(), err)
	}

	unwrapped := innerEvent.UnwrapPSMEvent()

	typeKey := unwrapped.PSMEventKey()
	stateBefore := state.GetStatus()

	ctx = log.WithFields(ctx, map[string]interface{}{
		"eventType":    typeKey,
		"stateMachine": state.PSMKeys().PSMFullName(),
		"transition": map[string]interface{}{
			"from":  stateBefore.ShortString(),
			"event": typeKey,
		},
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
		"transition": map[string]string{
			"from":  stateBefore.ShortString(),
			"to":    state.GetStatus().ShortString(),
			"event": typeKey,
		},
	})

	if state.GetStatus() == 0 {
		return fmt.Errorf("state machine transitioned to zero status")
	}

	log.Info(ctx, "Event Handled")

	for _, callback := range callbacks {
		if err := callback(ctx, stateBefore, state, innerEvent); err != nil {
			return err
		}
	}

	return nil
}

func (ee *Eventer[K, S, ST, SD, E, IE]) Register(transition ITransition[K, S, ST, SD, E, IE]) {
	ee.Transitions = append(ee.Transitions, transition)
}

func (ee *Eventer[K, S, ST, SD, E, IE]) prepareEvent(state S, spec *EventSpec[K, S, ST, SD, E, IE]) (built E, err error) {

	built = (*new(E)).ProtoReflect().New().Interface().(E)
	if err := built.SetPSMEvent(spec.Event); err != nil {
		return built, fmt.Errorf("set event: %w", err)
	}
	built.SetPSMKeys(spec.Keys)

	eventMeta := built.PSMMetadata()
	eventMeta.EventId = spec.EventID
	eventMeta.Timestamp = timestamppb.Now()
	eventMeta.Cause = spec.Cause

	stateMeta := state.PSMMetadata()

	eventMeta.Sequence = 0
	if state.GetStatus() == 0 {
		eventMeta.Sequence = 0
		stateMeta.CreatedAt = eventMeta.Timestamp
		stateMeta.UpdatedAt = eventMeta.Timestamp
	} else {
		eventMeta.Sequence = stateMeta.LastSequence + 1
		stateMeta.LastSequence = eventMeta.Sequence
		stateMeta.UpdatedAt = eventMeta.Timestamp
		stateMeta.CreatedAt = eventMeta.Timestamp
	}
	return
}
