// Code generated by protoc-gen-go-psm. DO NOT EDIT.

package test_pb

import (
	context "context"
	fmt "fmt"
	psm_j5pb "github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	psm "github.com/pentops/protostate/psm"
	sqrlx "github.com/pentops/sqrlx.go/sqrlx"
)

// PSM BarPSM

type BarPSM = psm.StateMachine[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
]

type BarPSMDB = psm.DBStateMachine[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
]

type BarPSMEventSpec = psm.EventSpec[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
]

type BarPSMHookBaton = psm.HookBaton[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
]

type BarPSMFullBaton = psm.CallbackBaton[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
]

type BarPSMEventKey = string

const (
	BarPSMEventNil     BarPSMEventKey = "<nil>"
	BarPSMEventCreated BarPSMEventKey = "created"
	BarPSMEventUpdated BarPSMEventKey = "updated"
	BarPSMEventDeleted BarPSMEventKey = "deleted"
)

// EXTEND BarKeys with the psm.IKeyset interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *BarKeys) PSMIsSet() bool {
	return msg != nil
}

// PSMFullName returns the full name of state machine with package prefix
func (msg *BarKeys) PSMFullName() string {
	return "test.v1.bar"
}
func (msg *BarKeys) PSMKeyValues() (map[string]any, error) {
	keyset := map[string]any{
		"bar_id":       msg.BarId,
		"bar_other_id": msg.BarOtherId,
		"date_key":     msg.DateKey,
	}
	return keyset, nil
}

// EXTEND BarState with the psm.IState interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *BarState) PSMIsSet() bool {
	return msg != nil
}

func (msg *BarState) PSMMetadata() *psm_j5pb.StateMetadata {
	if msg.Metadata == nil {
		msg.Metadata = &psm_j5pb.StateMetadata{}
	}
	return msg.Metadata
}

func (msg *BarState) PSMKeys() *BarKeys {
	return msg.Keys
}

func (msg *BarState) SetStatus(status BarStatus) {
	msg.Status = status
}

func (msg *BarState) SetPSMKeys(inner *BarKeys) {
	msg.Keys = inner
}

func (msg *BarState) PSMData() *BarData {
	if msg.Data == nil {
		msg.Data = &BarData{}
	}
	return msg.Data
}

// EXTEND BarData with the psm.IStateData interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *BarData) PSMIsSet() bool {
	return msg != nil
}

// EXTEND BarEvent with the psm.IEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *BarEvent) PSMIsSet() bool {
	return msg != nil
}

func (msg *BarEvent) PSMMetadata() *psm_j5pb.EventMetadata {
	if msg.Metadata == nil {
		msg.Metadata = &psm_j5pb.EventMetadata{}
	}
	return msg.Metadata
}

func (msg *BarEvent) PSMKeys() *BarKeys {
	return msg.Keys
}

func (msg *BarEvent) SetPSMKeys(inner *BarKeys) {
	msg.Keys = inner
}

// PSMEventKey returns the BarPSMEventPSMEventKey for the event, implementing psm.IEvent
func (msg *BarEvent) PSMEventKey() BarPSMEventKey {
	tt := msg.UnwrapPSMEvent()
	if tt == nil {
		return BarPSMEventNil
	}
	return tt.PSMEventKey()
}

// UnwrapPSMEvent implements psm.IEvent, returning the inner event message
func (msg *BarEvent) UnwrapPSMEvent() BarPSMEvent {
	if msg == nil {
		return nil
	}
	if msg.Event == nil {
		return nil
	}
	switch v := msg.Event.Type.(type) {
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

// SetPSMEvent sets the inner event message from a concrete type, implementing psm.IEvent
func (msg *BarEvent) SetPSMEvent(inner BarPSMEvent) error {
	if msg.Event == nil {
		msg.Event = &BarEventType{}
	}
	switch v := inner.(type) {
	case *BarEventType_Created:
		msg.Event.Type = &BarEventType_Created_{Created: v}
	case *BarEventType_Updated:
		msg.Event.Type = &BarEventType_Updated_{Updated: v}
	case *BarEventType_Deleted:
		msg.Event.Type = &BarEventType_Deleted_{Deleted: v}
	default:
		return fmt.Errorf("invalid type %T for BarEventType", v)
	}
	return nil
}

type BarPSMEvent interface {
	psm.IInnerEvent
	PSMEventKey() BarPSMEventKey
}

// EXTEND BarEventType_Created with the BarPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *BarEventType_Created) PSMIsSet() bool {
	return msg != nil
}

func (*BarEventType_Created) PSMEventKey() BarPSMEventKey {
	return BarPSMEventCreated
}

// EXTEND BarEventType_Updated with the BarPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *BarEventType_Updated) PSMIsSet() bool {
	return msg != nil
}

func (*BarEventType_Updated) PSMEventKey() BarPSMEventKey {
	return BarPSMEventUpdated
}

// EXTEND BarEventType_Deleted with the BarPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *BarEventType_Deleted) PSMIsSet() bool {
	return msg != nil
}

func (*BarEventType_Deleted) PSMEventKey() BarPSMEventKey {
	return BarPSMEventDeleted
}

func BarPSMBuilder() *psm.StateMachineConfig[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
] {
	return &psm.StateMachineConfig[
		*BarKeys,    // implements psm.IKeyset
		*BarState,   // implements psm.IState
		BarStatus,   // implements psm.IStatusEnum
		*BarData,    // implements psm.IStateData
		*BarEvent,   // implements psm.IEvent
		BarPSMEvent, // implements psm.IInnerEvent
	]{}
}

// BarPSMMutation runs at the start of a transition to merge the event information into the state data object. The state object is mutable in this context.
func BarPSMMutation[SE BarPSMEvent](cb func(*BarData, SE) error) psm.TransitionMutation[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
	SE,          // Specific event type for the transition
] {
	return psm.TransitionMutation[
		*BarKeys,    // implements psm.IKeyset
		*BarState,   // implements psm.IState
		BarStatus,   // implements psm.IStatusEnum
		*BarData,    // implements psm.IStateData
		*BarEvent,   // implements psm.IEvent
		BarPSMEvent, // implements psm.IInnerEvent
		SE,          // Specific event type for the transition
	](cb)
}

// BarPSMLogicHook runs after the mutation is complete. This hook can trigger side effects, including chained events, which are additional events processed by the state machine. Use this for Business Logic which determines the 'next step' in processing.
func BarPSMLogicHook[
	SE BarPSMEvent,
](
	cb func(
		context.Context,
		BarPSMHookBaton,
		*BarState,
		SE,
	) error) psm.TransitionHook[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
] {
	eventType := (*new(SE)).PSMEventKey()
	return psm.TransitionHook[
		*BarKeys,    // implements psm.IKeyset
		*BarState,   // implements psm.IState
		BarStatus,   // implements psm.IStatusEnum
		*BarData,    // implements psm.IStateData
		*BarEvent,   // implements psm.IEvent
		BarPSMEvent, // implements psm.IInnerEvent
	]{
		Callback: func(ctx context.Context, tx sqrlx.Transaction, baton BarPSMFullBaton, state *BarState, event *BarEvent) error {
			asType, ok := any(event.UnwrapPSMEvent()).(SE)
			if !ok {
				name := event.ProtoReflect().Descriptor().FullName()
				return fmt.Errorf("unexpected event type in transition: %s [IE] does not match [SE] (%T)", name, new(SE))
			}
			return cb(ctx, baton, state, asType)
		},
		EventType:   eventType,
		RunOnFollow: false,
	}
}

// BarPSMDataHook runs after the mutations, and can be used to update data in tables which are not controlled as the state machine, e.g. for pre-calculating fields for performance reasons. Use of this hook prevents (future) transaction optimizations, as the transaction state when the function is called must needs to match the processing state, but only for this single transition, unlike the GeneralEventDataHook.
func BarPSMDataHook[
	SE BarPSMEvent,
](
	cb func(
		context.Context,
		sqrlx.Transaction,
		*BarState,
		SE,
	) error) psm.TransitionHook[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
] {
	eventType := (*new(SE)).PSMEventKey()
	return psm.TransitionHook[
		*BarKeys,    // implements psm.IKeyset
		*BarState,   // implements psm.IState
		BarStatus,   // implements psm.IStatusEnum
		*BarData,    // implements psm.IStateData
		*BarEvent,   // implements psm.IEvent
		BarPSMEvent, // implements psm.IInnerEvent
	]{
		Callback: func(ctx context.Context, tx sqrlx.Transaction, baton BarPSMFullBaton, state *BarState, event *BarEvent) error {
			asType, ok := any(event.UnwrapPSMEvent()).(SE)
			if !ok {
				name := event.ProtoReflect().Descriptor().FullName()
				return fmt.Errorf("unexpected event type in transition: %s [IE] does not match [SE] (%T)", name, new(SE))
			}
			return cb(ctx, tx, state, asType)
		},
		EventType:   eventType,
		RunOnFollow: true,
	}
}

// BarPSMLinkHook runs after the mutation and logic hook, and can be used to link the state machine to other state machines in the same database transaction
func BarPSMLinkHook[
	SE BarPSMEvent,
	DK psm.IKeyset,
	DIE psm.IInnerEvent,
](
	linkDestination psm.LinkDestination[DK, DIE],
	cb func(
		context.Context,
		*BarState,
		SE,
		func(DK, DIE),
	) error) psm.TransitionHook[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
] {
	eventType := (*new(SE)).PSMEventKey()
	wrapped := func(ctx context.Context, tx sqrlx.Transaction, state *BarState, event SE, add func(DK, DIE)) error {
		return cb(ctx, state, event, add)
	}
	return psm.TransitionHook[
		*BarKeys,    // implements psm.IKeyset
		*BarState,   // implements psm.IState
		BarStatus,   // implements psm.IStatusEnum
		*BarData,    // implements psm.IStateData
		*BarEvent,   // implements psm.IEvent
		BarPSMEvent, // implements psm.IInnerEvent
	]{
		Callback: func(ctx context.Context, tx sqrlx.Transaction, baton BarPSMFullBaton, state *BarState, event *BarEvent) error {
			return psm.RunLinkHook(ctx, linkDestination, wrapped, tx, state, event)
		},
		EventType:   eventType,
		RunOnFollow: false,
	}
}

// BarPSMLinkDBHook like LinkHook, but has access to the current transaction for reads only (not enforced), use in place of controller logic to look up existing state.
func BarPSMLinkDBHook[
	SE BarPSMEvent,
	DK psm.IKeyset,
	DIE psm.IInnerEvent,
](
	linkDestination psm.LinkDestination[DK, DIE],
	cb func(
		context.Context,
		sqrlx.Transaction,
		*BarState,
		SE,
		func(DK, DIE),
	) error) psm.TransitionHook[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
] {
	eventType := (*new(SE)).PSMEventKey()
	return psm.TransitionHook[
		*BarKeys,    // implements psm.IKeyset
		*BarState,   // implements psm.IState
		BarStatus,   // implements psm.IStatusEnum
		*BarData,    // implements psm.IStateData
		*BarEvent,   // implements psm.IEvent
		BarPSMEvent, // implements psm.IInnerEvent
	]{
		Callback: func(ctx context.Context, tx sqrlx.Transaction, baton BarPSMFullBaton, state *BarState, event *BarEvent) error {
			return psm.RunLinkHook(ctx, linkDestination, cb, tx, state, event)
		},
		EventType:   eventType,
		RunOnFollow: false,
	}
}

// BarPSMGeneralLogicHook runs once per transition at the state-machine level regardless of which transition / event is being processed. It runs exactly once per transition, with the state object in the final state after the transition but prior to processing any further events. Chained events are added to the *end* of the event queue for the transaction, and side effects are published (as always) when the transaction is committed. The function MUST be pure, i.e. It MUST NOT produce any side-effects outside of the HookBaton, and MUST NOT modify the state.
func BarPSMGeneralLogicHook(
	cb func(
		context.Context,
		BarPSMHookBaton,
		*BarState,
		*BarEvent,
	) error) psm.GeneralEventHook[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
] {
	return psm.GeneralEventHook[
		*BarKeys,    // implements psm.IKeyset
		*BarState,   // implements psm.IState
		BarStatus,   // implements psm.IStatusEnum
		*BarData,    // implements psm.IStateData
		*BarEvent,   // implements psm.IEvent
		BarPSMEvent, // implements psm.IInnerEvent
	]{
		Callback: func(
			ctx context.Context,
			tx sqrlx.Transaction,
			baton BarPSMFullBaton,
			state *BarState,
			event *BarEvent,
		) error {
			return cb(ctx, baton, state, event)
		},
		RunOnFollow: false,
	}
}

// BarPSMGeneralStateDataHook runs at the state-machine level regardless of which transition / event is being processed. It runs at-least once before committing a database transaction after multiple transitions are complete. This hook has access only to the final state after the transitions and is used to update other tables based on the resulting state. It MUST be idempotent, it may be called after injecting externally-held state data.
func BarPSMGeneralStateDataHook(
	cb func(
		context.Context,
		sqrlx.Transaction,
		*BarState,
	) error) psm.GeneralStateHook[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
] {
	return psm.GeneralStateHook[
		*BarKeys,    // implements psm.IKeyset
		*BarState,   // implements psm.IState
		BarStatus,   // implements psm.IStatusEnum
		*BarData,    // implements psm.IStateData
		*BarEvent,   // implements psm.IEvent
		BarPSMEvent, // implements psm.IInnerEvent
	]{
		Callback: func(
			ctx context.Context,
			tx sqrlx.Transaction,
			baton BarPSMFullBaton,
			state *BarState,
		) error {
			return cb(ctx, tx, state)
		},
		RunOnFollow: true,
	}
}

// BarPSMGeneralEventDataHook runs after each transition at the state-machine level regardless of which transition / event is being processed. It runs exactly once per transition, before any other events are processed. The presence of this hook type prevents (future) transaction optimizations, so should be used sparingly.
func BarPSMGeneralEventDataHook(
	cb func(
		context.Context,
		sqrlx.Transaction,
		*BarState,
		*BarEvent,
	) error) psm.GeneralEventHook[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
] {
	return psm.GeneralEventHook[
		*BarKeys,    // implements psm.IKeyset
		*BarState,   // implements psm.IState
		BarStatus,   // implements psm.IStatusEnum
		*BarData,    // implements psm.IStateData
		*BarEvent,   // implements psm.IEvent
		BarPSMEvent, // implements psm.IInnerEvent
	]{
		Callback: func(
			ctx context.Context,
			tx sqrlx.Transaction,
			baton BarPSMFullBaton,
			state *BarState,
			event *BarEvent,
		) error {
			return cb(ctx, tx, state, event)
		},
		RunOnFollow: true,
	}
}

// BarPSMEventPublishHook  EventPublishHook runs for each transition, at least once before committing a database transaction after multiple transitions are complete. It should publish a derived version of the event using the publisher.
func BarPSMEventPublishHook(
	cb func(
		context.Context,
		psm.Publisher,
		*BarState,
		*BarEvent,
	) error) psm.GeneralEventHook[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
] {
	return psm.GeneralEventHook[
		*BarKeys,    // implements psm.IKeyset
		*BarState,   // implements psm.IState
		BarStatus,   // implements psm.IStatusEnum
		*BarData,    // implements psm.IStateData
		*BarEvent,   // implements psm.IEvent
		BarPSMEvent, // implements psm.IInnerEvent
	]{
		Callback: func(
			ctx context.Context,
			tx sqrlx.Transaction,
			baton BarPSMFullBaton,
			state *BarState,
			event *BarEvent,
		) error {
			return cb(ctx, baton, state, event)
		},
		RunOnFollow: false,
	}
}

// BarPSMUpsertPublishHook runs for each transition, at least once before committing a database transaction after multiple transitions are complete. It should publish a derived version of the event using the publisher.
func BarPSMUpsertPublishHook(
	cb func(
		context.Context,
		psm.Publisher,
		*BarState,
	) error) psm.GeneralStateHook[
	*BarKeys,    // implements psm.IKeyset
	*BarState,   // implements psm.IState
	BarStatus,   // implements psm.IStatusEnum
	*BarData,    // implements psm.IStateData
	*BarEvent,   // implements psm.IEvent
	BarPSMEvent, // implements psm.IInnerEvent
] {
	return psm.GeneralStateHook[
		*BarKeys,    // implements psm.IKeyset
		*BarState,   // implements psm.IState
		BarStatus,   // implements psm.IStatusEnum
		*BarData,    // implements psm.IStateData
		*BarEvent,   // implements psm.IEvent
		BarPSMEvent, // implements psm.IInnerEvent
	]{
		Callback: func(
			ctx context.Context,
			tx sqrlx.Transaction,
			baton BarPSMFullBaton,
			state *BarState,
		) error {
			return cb(ctx, baton, state)
		},
		RunOnFollow: false,
	}
}

func (event *BarEvent) EventPublishMetadata() *psm_j5pb.EventPublishMetadata {
	tenantKeys := make([]*psm_j5pb.EventTenant, 0)
	return &psm_j5pb.EventPublishMetadata{
		EventId:   event.Metadata.EventId,
		Sequence:  event.Metadata.Sequence,
		Timestamp: event.Metadata.Timestamp,
		Cause:     event.Metadata.Cause,
		Auth: &psm_j5pb.PublishAuth{
			TenantKeys: tenantKeys,
		},
	}
}
