package psm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pentops/log.go/log"
	"github.com/pentops/outbox.pg.go/outbox"
	"gopkg.daemonl.com/sqrlx"
)

type TransitionBaton[OuterEvent IEvent[NextInnerEvent], NextInnerEvent any] interface {
	SideEffect(outbox.OutboxMessage)
	ChainEvent(NextInnerEvent)
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
	E IEvent[E], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event *interface*
] struct {
	WrapEvent   func(context.Context, S, IE) E
	UnwrapEvent func(E) IE
	StateLabel  func(S) string
	EventLabel  func(IE) string

	Transitions []ITransition[S, ST, E, IE]
}

func (ee Eventer[S, ST, E, IE]) FindTransition(state S, event IE) (ITransition[S, ST, E, IE], error) {
	for _, search := range ee.Transitions {
		if search.Matches(state, event) {
			return search, nil
		}
	}
	typeKey := ee.EventLabel(event)
	return nil, fmt.Errorf("no transition found for status %s -> %s",
		ee.StateLabel(state),
		typeKey,
	)
}

func (ee *Eventer[S, ST, E, IE]) Register(transition ITransition[S, ST, E, IE]) {
	ee.Transitions = append(ee.Transitions, transition)
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

		transition, err := ee.FindTransition(state, unwrapped)
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

var TxOptions = &sqrlx.TxOptions{
	Isolation: sql.LevelReadCommitted,
	Retryable: true,
	ReadOnly:  false,
}
