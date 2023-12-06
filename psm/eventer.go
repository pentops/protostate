package psm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pentops/log.go/log"
	"github.com/pentops/outbox.pg.go/outbox"
	"gopkg.daemonl.com/sqrlx"
)

type TransitionBaton[E IEvent[IE], IE IInnerEvent] interface {
	SideEffect(outbox.OutboxMessage)
	ChainEvent(IE)
}

type TransitionData[Event any] struct {
	SideEffects []outbox.OutboxMessage
	ChainEvents []Event
}

func (td *TransitionData[Event]) ChainEvent(event Event) {
	td.ChainEvents = append(td.ChainEvents, event)
}

func (td *TransitionData[Event]) SideEffect(msg outbox.OutboxMessage) {
	td.SideEffects = append(td.SideEffects, msg)
}

type Eventer[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event *interface*
] struct {
	WrapEvent   func(context.Context, S, IE) E
	UnwrapEvent func(E) IE
	StateLabel  func(S) string
	EventLabel  func(IE) string

	Transitions []ITransition[S, ST, E, IE]
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

func (ee Eventer[State, Status, WrappedEvent, Event]) Run(
	ctx context.Context,
	tx Transaction[State, WrappedEvent],
	state State,
	outerEvent WrappedEvent,
) error {

	eventQueue := []WrappedEvent{outerEvent}

	for len(eventQueue) > 0 {
		innerEvent := eventQueue[0]
		eventQueue = eventQueue[1:]

		baton := &TransitionData[Event]{}

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

		for _, se := range baton.SideEffects {
			if err := tx.Outbox(ctx, se); err != nil {
				return fmt.Errorf("side effect outbox: %w", err)
			}
		}

		for _, event := range baton.ChainEvents {
			wrappedEvent := ee.WrapEvent(ctx, state, event)
			eventQueue = append(eventQueue, wrappedEvent)
		}
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
	filter     func(E) bool
}

func (ee *Eventer[S, ST, E, IE]) From(states ...ST) EventerTransitionBuilder[S, ST, E, IE] {
	return EventerTransitionBuilder[S, ST, E, IE]{
		eventer:    ee,
		fromStates: states,
	}
}

func (tb EventerTransitionBuilder[S, ST, E, IE]) Where(filter func(event E) bool) EventerTransitionBuilder[S, ST, E, IE] {
	tb.filter = filter
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

var TxOptions = &sqrlx.TxOptions{
	Isolation: sql.LevelReadCommitted,
	Retryable: true,
	ReadOnly:  false,
}
