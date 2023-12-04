package psm

import (
	"context"
	"fmt"
)

type Transition[
	State IState[Status],
	Status IStatusEnum,
	InnerEventType any,
	InnerEvent any,
] struct {
	fromStatus  []Status
	eventFilter func(InnerEvent) bool
	transition  func(context.Context, TransitionBaton[InnerEventType], State, InnerEvent) error
}

func NewTransition[
	State IState[Status],
	Status IStatusEnum,
	InnerEventType any,
	InnerEvent any,
](
	fromStatus []Status,
	transition func(context.Context, TransitionBaton[InnerEventType], State, InnerEvent) error,
) *Transition[State, Status, InnerEventType, InnerEvent] {
	return &Transition[State, Status, InnerEventType, InnerEvent]{
		fromStatus: fromStatus,
		transition: transition,
	}
}

func (ts *Transition[State, Status, InnerEventType, InnerEvent]) WithEventFilter(
	eventFilter func(InnerEvent) bool,
) *Transition[State, Status, InnerEventType, InnerEvent] {
	ts.eventFilter = eventFilter
	return ts
}

func (ts Transition[State, Status, InnerEventType, InnerEvent]) RunTransition(
	ctx context.Context,
	tb TransitionBaton[InnerEventType],
	state State,
	event InnerEventType) error {
	asType, ok := any(event).(InnerEvent)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}

	return ts.transition(ctx, tb, state, asType)
}

func (ts Transition[State, Status, InnerEventType, InnerEvent]) Matches(state State, event InnerEventType) bool {
	asType, ok := any(event).(InnerEvent)
	if !ok {
		return false
	}
	didMatch := false

	if ts.fromStatus != nil {
		currentStatus := state.GetStatus()
		for _, fromStatus := range ts.fromStatus {
			if fromStatus == currentStatus {
				didMatch = true
				break
			}
		}
		if !didMatch {
			return false
		}
	}

	if ts.eventFilter != nil && !ts.eventFilter(asType) {
		return false
	}
	return true
}
