// Code generated by protoc-gen-go-psm. DO NOT EDIT.

package testpb

import (
	context "context"
	fmt "fmt"
	psm "github.com/pentops/protostate/psm"
	proto "google.golang.org/protobuf/proto"
)

// State Query Service for %sbar
type BarPSMStateQuerySet = psm.StateQuerySet[
	*GetBarRequest,
	*GetBarResponse,
	*ListBarsRequest,
	*ListBarsResponse,
	proto.Message,
	proto.Message,
]

type BarPSMStateQuerySpec = psm.StateQuerySpec[
	*GetBarRequest,
	*GetBarResponse,
	*ListBarsRequest,
	*ListBarsResponse,
	proto.Message,
	proto.Message,
]

// StateObjectOptions: BarPSM
type BarPSMEventer = psm.Eventer[
	*BarState,
	BarStatus,
	*BarEvent,
	BarPSMEvent,
]

type BarPSM = psm.StateMachine[
	*BarState,
	BarStatus,
	*BarEvent,
	BarPSMEvent,
]

func NewBarPSM(db psm.Transactor, options ...psm.StateMachineOption[
	*BarState,
	BarStatus,
	*BarEvent,
	BarPSMEvent,
]) (*BarPSM, error) {
	return psm.NewStateMachine[
		*BarState,
		BarStatus,
		*BarEvent,
		BarPSMEvent,
	](db, BarPSMConverter{}, DefaultBarPSMTableSpec, options...)
}

type BarPSMTableSpec = psm.TableSpec[
	*BarState,
	BarStatus,
	*BarEvent,
	BarPSMEvent,
]

var DefaultBarPSMTableSpec = BarPSMTableSpec{
	StateTable: "bar",
	EventTable: "bar_event",
	PrimaryKey: func(event *BarEvent) (map[string]interface{}, error) {
		return map[string]interface{}{
			"id": event.BarId,
		}, nil
	},
}

type BarPSMTransitionBaton = psm.TransitionBaton[*BarEvent, BarPSMEvent]

func BarPSMFunc[SE BarPSMEvent](cb func(context.Context, BarPSMTransitionBaton, *BarState, SE) error) psm.TransitionFunc[
	*BarState,
	BarStatus,
	*BarEvent,
	BarPSMEvent,
	SE,
] {
	return psm.TransitionFunc[
		*BarState,
		BarStatus,
		*BarEvent,
		BarPSMEvent,
		SE,
	](cb)
}

type BarPSMEventKey string

const (
	BarPSMEventCreated BarPSMEventKey = "created"
	BarPSMEventUpdated BarPSMEventKey = "updated"
	BarPSMEventDeleted BarPSMEventKey = "deleted"
)

type BarPSMEvent interface {
	proto.Message
	PSMEventKey() BarPSMEventKey
}
type BarPSMConverter struct{}

func (c BarPSMConverter) Unwrap(e *BarEvent) BarPSMEvent {
	return e.UnwrapPSMEvent()
}

func (c BarPSMConverter) StateLabel(s *BarState) string {
	return s.Status.String()
}

func (c BarPSMConverter) EventLabel(e BarPSMEvent) string {
	return string(e.PSMEventKey())
}

func (c BarPSMConverter) EmptyState(e *BarEvent) *BarState {
	return &BarState{
		BarId: e.BarId,
	}
}
func (c BarPSMConverter) CheckStateKeys(s *BarState, e *BarEvent) error {
	if s.BarId != e.BarId {
		return fmt.Errorf("state field 'BarId' %q does not match event field %q", s.BarId, e.BarId)
	}
	return nil
}

func (ee *BarEventType) UnwrapPSMEvent() BarPSMEvent {
	switch v := ee.Type.(type) {
	case *BarEventType_Created_:
		return v.Created
	case *BarEventType_Updated_:
		return v.Updated
	case *BarEventType_Deleted_:
		return v.Deleted
	default:
		return nil
	}
}
func (ee *BarEventType) PSMEventKey() BarPSMEventKey {
	tt := ee.UnwrapPSMEvent()
	if tt == nil {
		return "<nil>"
	}
	return tt.PSMEventKey()
}
func (ee *BarEvent) UnwrapPSMEvent() BarPSMEvent {
	return ee.Event.UnwrapPSMEvent()
}
func (ee *BarEvent) SetPSMEvent(inner BarPSMEvent) {
	switch v := inner.(type) {
	case *BarEventType_Created:
		ee.Event.Type = &BarEventType_Created_{Created: v}
	case *BarEventType_Updated:
		ee.Event.Type = &BarEventType_Updated_{Updated: v}
	case *BarEventType_Deleted:
		ee.Event.Type = &BarEventType_Deleted_{Deleted: v}
	default:
		panic("invalid type")
	}
}
func (*BarEventType_Created) PSMEventKey() BarPSMEventKey {
	return BarPSMEventCreated
}
func (*BarEventType_Updated) PSMEventKey() BarPSMEventKey {
	return BarPSMEventUpdated
}
func (*BarEventType_Deleted) PSMEventKey() BarPSMEventKey {
	return BarPSMEventDeleted
}
