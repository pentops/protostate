package psm

import (
	"context"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/outbox.pg.go/outbox"
)

type TransitionBaton[E IEvent[IE], IE IInnerEvent] interface {
	SideEffect(outbox.OutboxMessage)
	ChainEvent(E)
	FullCause() E
}

type TransitionData[E IEvent[IE], IE IInnerEvent] struct {
	sideEffects []outbox.OutboxMessage
	chainEvents []E
	causedBy    E
}

func (td *TransitionData[E, IE]) ChainEvent(event E) {
	td.chainEvents = append(td.chainEvents, event)
}

func (td *TransitionData[E, IE]) SideEffect(msg outbox.OutboxMessage) {
	td.sideEffects = append(td.sideEffects, msg)
}

func (td *TransitionData[E, IE]) FullCause() E {
	return td.causedBy
}

type Eventer[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event *interface*
] struct {
	UnwrapEvent func(E) IE
	StateLabel  func(S) string
	EventLabel  func(IE) string

	Transitions []ITransition[S, ST, E, IE]

	validator *protovalidate.Validator
}

func (ee Eventer[S, ST, E, IE]) FindTransition(state S, event E) (ITransition[S, ST, E, IE], error) {
	for _, search := range ee.Transitions {
		if search.Matches(state, event) {
			return search, nil
		}
	}

	innerEvent := ee.UnwrapEvent(event)
	typeKey := ee.EventLabel(innerEvent)
	return nil, fmt.Errorf("no transition found for status %s -> %s",
		ee.StateLabel(state),
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

func (ee Eventer[S, ST, E, IE]) Run(
	ctx context.Context,
	tx Transaction[S, E],
	state S,
	outerEvent E,
) error {
	if err := ee.ValidateEvent(outerEvent); err != nil {
		return fmt.Errorf("validating event: %w", err)
	}

	eventQueue := []E{outerEvent}

	for len(eventQueue) > 0 {
		innerEvent := eventQueue[0]
		eventQueue = eventQueue[1:]

		baton := &TransitionData[E, IE]{
			causedBy: innerEvent,
		}

		unwrapped := ee.UnwrapEvent(innerEvent)

		typeKey := ee.EventLabel(unwrapped)
		stateBefore := ee.StateLabel(state)

		ctx = log.WithFields(ctx, map[string]interface{}{
			"eventType":  typeKey,
			"transition": fmt.Sprintf("%s -> ? : %s", stateBefore, typeKey),
		})

		transition, err := ee.FindTransition(state, innerEvent)
		if err != nil {
			return err
		}

		log.Debug(ctx, "Begin Event")

		if err := transition.RunTransition(ctx, baton, state, unwrapped); err != nil {
			log.WithError(ctx, err).Error("Running Transition")
			return err
		}

		ctx = log.WithFields(ctx, map[string]interface{}{
			"transition": fmt.Sprintf("%s -> %s : %s", stateBefore, ee.StateLabel(state), typeKey),
		})

		log.Info(ctx, "Event Handled")

		if err := tx.StoreEvent(ctx, state, innerEvent); err != nil {
			return err
		}

		for _, se := range baton.sideEffects {
			if err := tx.Outbox(ctx, se); err != nil {
				return fmt.Errorf("side effect outbox: %w", err)
			}
		}

		for _, event := range baton.chainEvents {
			if err := ee.ValidateEvent(event); err != nil {
				return fmt.Errorf("validating chained event: %w", err)
			}
		}
		eventQueue = append(eventQueue, baton.chainEvents...)
	}

	return nil
}

type ITransition[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
] interface {
	Matches(S, E) bool
	RunTransition(context.Context, TransitionBaton[E, IE], S, IE) error
}

func (ee *Eventer[S, ST, E, IE]) Register(transition ITransition[S, ST, E, IE]) {
	ee.Transitions = append(ee.Transitions, transition)
}

type EventerTransitionBuilder[
	S IState[ST],
	ST IStatusEnum,
	E IEvent[IE],
	IE IInnerEvent,
] struct {
	eventer    *Eventer[S, ST, E, IE]
	fromStates []ST
	filters    []func(E) bool
}

func (ee *Eventer[S, ST, E, IE]) From(states ...ST) EventerTransitionBuilder[S, ST, E, IE] {
	return EventerTransitionBuilder[S, ST, E, IE]{
		eventer:    ee,
		fromStates: states,
	}
}

func (tb EventerTransitionBuilder[S, ST, E, IE]) Where(filter func(event IE) bool) EventerTransitionBuilder[S, ST, E, IE] {
	innerFilter := func(fullEvent E) bool {
		innerEvent := tb.eventer.UnwrapEvent(fullEvent)
		return filter(innerEvent)
	}
	tb.filters = append(tb.filters, innerFilter)
	return tb
}

func (tb EventerTransitionBuilder[S, ST, E, IE]) Do(
	transition TypedTransitionHandler[S, ST, E, IE],
) EventerTransitionBuilder[S, ST, E, IE] {

	tb.eventer.Register(&TypedTransition[S, ST, E, IE]{
		handler:     transition,
		fromStatus:  tb.fromStates,
		eventFilter: tb.filter,
	})

	return tb
}

func (tb EventerTransitionBuilder[S, ST, E, IE]) filter(event E) bool {
	for _, filter := range tb.filters {
		if !filter(event) {
			return false
		}
	}
	return true
}
