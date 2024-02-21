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
	ChainDerived(IE)
	FullCause() E
}

type TransitionData[E IEvent[IE], IE IInnerEvent] struct {
	sideEffects []outbox.OutboxMessage
	chainEvents []E
	causedBy    E
	err         error

	// returns a new event with metadata derived from the causing event and
	// metadata from the system actor
	deriveEvent func(E, IE) (E, error)
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

func (td *TransitionData[E, IE]) ChainDerived(inner IE) {
	derived, err := td.deriveEvent(td.causedBy, inner)
	if err != nil {
		td.err = err
		return
	}

	td.ChainEvent(derived)
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

	SystemActor SystemActor

	validator *protovalidate.Validator
}

func (ee Eventer[S, ST, E, IE]) FindTransition(state S, event E) (ITransition[S, ST, E, IE], error) {
	for _, search := range ee.Transitions {
		if search.Matches(state, event) {
			return search, nil
		}
	}

	innerEvent := event.UnwrapPSMEvent()
	typeKey := innerEvent.PSMEventKey() // ee.conversions.EventLabel(innerEvent)
	return nil, fmt.Errorf("no transition found for status %s -> %s",
		state.GetStatus().String(),
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
		return fmt.Errorf("validating event %s: %w", outerEvent.ProtoReflect().Descriptor().FullName(), err)
	}

	eventQueue := []E{outerEvent}

	for len(eventQueue) > 0 {
		innerEvent := eventQueue[0]
		eventQueue = eventQueue[1:]

		chained, err := ee.runEvent(ctx, tx, state, innerEvent)
		if err != nil {
			return err
		}
		eventQueue = append(eventQueue, chained...)
	}

	return nil
}

func (ee Eventer[S, ST, E, IE]) runEvent(
	ctx context.Context,
	tx Transaction[S, E],
	state S,
	innerEvent E,
) ([]E, error) {

	baton := &TransitionData[E, IE]{
		causedBy:    innerEvent,
		deriveEvent: ee.deriveEvent,
		err:         nil,
	}

	unwrapped := innerEvent.UnwrapPSMEvent()

	typeKey := unwrapped.PSMEventKey()
	stateBefore := state.GetStatus().String()

	ctx = log.WithFields(ctx, map[string]interface{}{
		"eventType":  typeKey,
		"transition": fmt.Sprintf("%s -> ? : %s", stateBefore, typeKey),
	})

	transition, err := ee.FindTransition(state, innerEvent)
	if err != nil {
		return nil, err
	}

	log.Debug(ctx, "Begin Event")

	if err := transition.RunTransition(ctx, baton, state, unwrapped); err != nil {
		log.WithError(ctx, err).Error("Running Transition")
		return nil, err
	}

	if baton.err != nil {
		return nil, baton.err
	}

	ctx = log.WithFields(ctx, map[string]interface{}{
		"transition": fmt.Sprintf("%s -> %s : %s", stateBefore, state.GetStatus().String(), typeKey),
	})

	log.Info(ctx, "Event Handled")

	if err := tx.StoreEvent(ctx, state, innerEvent); err != nil {
		return nil, err
	}

	for _, se := range baton.sideEffects {
		if err := tx.Outbox(ctx, se); err != nil {
			return nil, fmt.Errorf("side effect outbox: %w", err)
		}
	}

	for _, event := range baton.chainEvents {
		if err := ee.ValidateEvent(event); err != nil {
			return nil, fmt.Errorf("validating chained event: %w", err)
		}
	}
	return baton.chainEvents, nil
}

// deriveEvent returns a new event with metadata derived from the causing
// event and system actor
func (ee Eventer[S, ST, E, IE]) deriveEvent(event E, inner IE) (evt E, err error) {
	if ee.SystemActor == nil {
		err = fmt.Errorf("no system actor defined, cannot derive events")
		return
	}
	eventKey := inner.PSMEventKey()
	derived := ee.conversions.DeriveChainEvent(event, ee.SystemActor, eventKey)
	derived.SetPSMEvent(inner)
	return derived, nil
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
		innerEvent := fullEvent.UnwrapPSMEvent()
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
