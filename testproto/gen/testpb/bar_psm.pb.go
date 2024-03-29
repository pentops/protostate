// Code generated by protoc-gen-go-psm. DO NOT EDIT.

package testpb

import (
	context "context"
	fmt "fmt"
	psm "github.com/pentops/protostate/psm"
	proto "google.golang.org/protobuf/proto"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// StateObjectOptions: BarPSM
type BarPSM = psm.StateMachine[
	*BarState,
	BarStatus,
	*BarEvent,
	BarPSMEvent,
]

type BarPSMDB = psm.DBStateMachine[
	*BarState,
	BarStatus,
	*BarEvent,
	BarPSMEvent,
]

type BarPSMEventer = psm.Eventer[
	*BarState,
	BarStatus,
	*BarEvent,
	BarPSMEvent,
]

func DefaultBarPSMConfig() *psm.StateMachineConfig[
	*BarState,
	BarStatus,
	*BarEvent,
	BarPSMEvent,
] {
	return psm.NewStateMachineConfig[
		*BarState,
		BarStatus,
		*BarEvent,
		BarPSMEvent,
	](BarPSMConverter{}, DefaultBarPSMTableSpec)
}

func NewBarPSM(config *psm.StateMachineConfig[
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
	](config)
}

type BarPSMTableSpec = psm.PSMTableSpec[
	*BarState,
	BarStatus,
	*BarEvent,
	BarPSMEvent,
]

var DefaultBarPSMTableSpec = BarPSMTableSpec{
	State: psm.TableSpec[*BarState]{
		TableName:  "bar",
		DataColumn: "state",
		StoreExtraColumns: func(state *BarState) (map[string]interface{}, error) {
			return map[string]interface{}{}, nil
		},
		PKFieldPaths: []string{
			"bar_id",
		},
	},
	Event: psm.TableSpec[*BarEvent]{
		TableName:  "bar_event",
		DataColumn: "data",
		StoreExtraColumns: func(event *BarEvent) (map[string]interface{}, error) {
			metadata := event.Metadata
			return map[string]interface{}{
				"id":        metadata.EventId,
				"timestamp": metadata.Timestamp,
				"bar_id":    event.BarId,
			}, nil
		},
		PKFieldPaths: []string{
			"metadata.event_id",
		},
	},
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

type BarPSMEventKey = string

const (
	BarPSMEventNil     BarPSMEventKey = "<nil>"
	BarPSMEventCreated BarPSMEventKey = "created"
	BarPSMEventUpdated BarPSMEventKey = "updated"
	BarPSMEventDeleted BarPSMEventKey = "deleted"
)

type BarPSMEvent interface {
	proto.Message
	PSMEventKey() BarPSMEventKey
}

type BarPSMConverter struct{}

func (c BarPSMConverter) EmptyState(e *BarEvent) *BarState {
	return &BarState{
		BarId: e.BarId,
	}
}

func (c BarPSMConverter) DeriveChainEvent(e *BarEvent, systemActor psm.SystemActor, eventKey string) *BarEvent {
	metadata := &StrangeMetadata{
		EventId:   systemActor.NewEventID(e.Metadata.EventId, eventKey),
		Timestamp: timestamppb.Now(),
	}
	return &BarEvent{
		Metadata: metadata,
		BarId:    e.BarId,
	}
}

func (c BarPSMConverter) CheckStateKeys(s *BarState, e *BarEvent) error {
	if s.BarId != e.BarId {
		return fmt.Errorf("state field 'BarId' %q does not match event field %q", s.BarId, e.BarId)
	}
	return nil
}

func (etw *BarEventType) UnwrapPSMEvent() BarPSMEvent {
	if etw == nil {
		return nil
	}
	switch v := etw.Type.(type) {
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
func (etw *BarEventType) PSMEventKey() BarPSMEventKey {
	tt := etw.UnwrapPSMEvent()
	if tt == nil {
		return BarPSMEventNil
	}
	return tt.PSMEventKey()
}
func (etw *BarEventType) SetPSMEvent(inner BarPSMEvent) {
	switch v := inner.(type) {
	case *BarEventType_Created:
		etw.Type = &BarEventType_Created_{Created: v}
	case *BarEventType_Updated:
		etw.Type = &BarEventType_Updated_{Updated: v}
	case *BarEventType_Deleted:
		etw.Type = &BarEventType_Deleted_{Deleted: v}
	default:
		panic("invalid type")
	}
}

func (ee *BarEvent) PSMEventKey() BarPSMEventKey {
	return ee.Event.PSMEventKey()
}

func (ee *BarEvent) UnwrapPSMEvent() BarPSMEvent {
	return ee.Event.UnwrapPSMEvent()
}

func (ee *BarEvent) SetPSMEvent(inner BarPSMEvent) {
	if ee.Event == nil {
		ee.Event = &BarEventType{}
	}
	ee.Event.SetPSMEvent(inner)
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

// State Query Service for %sBar
// QuerySet is the query set for the Bar service.

type BarPSMQuerySet = psm.StateQuerySet[
	*GetBarRequest,
	*GetBarResponse,
	*ListBarsRequest,
	*ListBarsResponse,
	proto.Message,
	proto.Message,
]

func NewBarPSMQuerySet(
	smSpec psm.QuerySpec[
		*GetBarRequest,
		*GetBarResponse,
		*ListBarsRequest,
		*ListBarsResponse,
		proto.Message,
		proto.Message,
	],
	options psm.StateQueryOptions,
) (*BarPSMQuerySet, error) {
	return psm.BuildStateQuerySet[
		*GetBarRequest,
		*GetBarResponse,
		*ListBarsRequest,
		*ListBarsResponse,
		proto.Message,
		proto.Message,
	](smSpec, options)
}

type BarPSMQuerySpec = psm.QuerySpec[
	*GetBarRequest,
	*GetBarResponse,
	*ListBarsRequest,
	*ListBarsResponse,
	proto.Message,
	proto.Message,
]

func DefaultBarPSMQuerySpec(tableSpec psm.QueryTableSpec) BarPSMQuerySpec {
	return psm.QuerySpec[
		*GetBarRequest,
		*GetBarResponse,
		*ListBarsRequest,
		*ListBarsResponse,
		proto.Message,
		proto.Message,
	]{
		QueryTableSpec: tableSpec,
		ListRequestFilter: func(req *ListBarsRequest) (map[string]interface{}, error) {
			filter := map[string]interface{}{}
			if req.TenantId != nil {
				filter["tenant_id"] = *req.TenantId
			}
			return filter, nil
		},
	}
}
