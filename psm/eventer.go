package psm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pentops/log.go/log"
	"github.com/pentops/outbox.pg.go/outbox"
	"google.golang.org/protobuf/proto"
	"gopkg.daemonl.com/sqrlx"
)

type ITransition[State proto.Message, Event any] interface {
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

type IStatusEnum interface {
	~int32
	ShortString() string
}

type IState[Status IStatusEnum] interface {
	proto.Message
	GetStatus() Status
}

type IEvent[Inner any] interface {
	proto.Message
}

type IInnerEvent interface {
	proto.Message
}

type Eventer[
	State IState[Status], // Outer State Entity
	Status IStatusEnum, // Status Enum in State Entity
	Event IEvent[InnerEvent], // Event Wrapper, with IDs and Metadata
	InnerEvent any, // Inner Event, the typed event
] struct {
	WrapEvent   func(context.Context, State, InnerEvent) Event
	UnwrapEvent func(Event) InnerEvent
	StateLabel  func(State) string
	EventLabel  func(InnerEvent) string

	Transitions []ITransition[State, InnerEvent]
}

func (ee Eventer[State, Status, Event, Inner]) FindTransition(state State, event Inner) (ITransition[State, Inner], error) {
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

func (ee *Eventer[State, Status, Event, Inner]) Register(transition ITransition[State, Inner]) {
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
