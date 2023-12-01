package sm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pentops/log.go/log"
	"github.com/pentops/outbox.pg.go/outbox"
	"google.golang.org/protobuf/proto"
	"gopkg.daemonl.com/sqrlx"
)

type Transition[State proto.Message, Event any] interface {
	Matches(State, Event) bool
	RunTransition(context.Context, TransitionBaton[Event], State, Event) error
}

type TransitionBaton[Event any] interface {
	SideEffect(outbox.OutboxMessage)
	ChainEvent(Event)
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

type Eventer[WrappedEvent proto.Message, Event any, State proto.Message] struct {
	WrapEvent   func(State, Event) WrappedEvent
	UnwrapEvent func(WrappedEvent) Event
	StateLabel  func(State) string
	EventLabel  func(Event) string

	Transitions []Transition[State, Event]
}

func (ee Eventer[WrappedEvent, Event, State]) findTransition(state State, event Event) (Transition[State, Event], error) {
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

type Transaction[State proto.Message, WrappedEvent proto.Message] interface {
	StoreEvent(context.Context, State, WrappedEvent) error
	Outbox(context.Context, outbox.OutboxMessage) error
}

type SqrlxTransaction[State proto.Message, WrappedEvent proto.Message] struct {
	sqrlx.Transaction
	callback func(context.Context, sqrlx.Transaction, State, WrappedEvent) error
}

func (st *SqrlxTransaction[State, WrappedEvent]) StoreEvent(ctx context.Context, state State, event WrappedEvent) error {

	return st.callback(ctx, st.Transaction, state, event)
}

func (st *SqrlxTransaction[State, WrappedEvent]) Outbox(ctx context.Context, msg outbox.OutboxMessage) error {

	return outbox.Send(ctx, st.Transaction, msg)
}

func NewSqrlxTransaction[State proto.Message, WrappedEvent proto.Message](
	tx sqrlx.Transaction,
	callback func(context.Context, sqrlx.Transaction, State, WrappedEvent) error,
) *SqrlxTransaction[State, WrappedEvent] {
	return &SqrlxTransaction[State, WrappedEvent]{
		Transaction: tx,
		callback:    callback,
	}
}

func (ee *Eventer[WrappedEvent, Event, State]) Run(
	ctx context.Context,
	tx Transaction[State, WrappedEvent],
	state State,
	outerEvent WrappedEvent) error {

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

		transition, err := ee.findTransition(state, unwrapped)
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
			wrappedEvent := ee.WrapEvent(state, event)
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
