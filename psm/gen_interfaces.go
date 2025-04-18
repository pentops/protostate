package psm

import (
	"context"
	"fmt"
	"time"

	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/o5-messaging/o5msg"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
)

/*
# Generic Type Parameter Sets

Two sets of generic type sets exist:

`K S ST SD E IE`
`K S ST SD E IE SE`

Both share the same types, as follows, and defined below

### `K IKeyset`
### `S IState[K, ST, SD]`
### `ST IStatusEnum`
### `SD IStateData`
### `SD IStateData,
E IEvent[K, S, ST, SD, IE]`
### `IE IInnerEvent`
### `SE IInnerEvent`

The Specific single typed event *struct* which is the specific event for the transition.
SE implements the same interface of IE.
e.g. *testpb.FooPSMEvent_Created, the concrete proto message which implements testpb.FooPSMEvent


The state machine deals with the first shorter chain, as it deals with all events.
Transitions deal with a single specific event type, so have the extra SE parameter.

K, S, ST, E, and IE are set to one single type for the entire state machine
SE is set to a single type for each transition.
*/

// IGenericProtoMessage is the base extensions shared by all message entities in the PSM generated code
type IPSMMessage interface {
	proto.Message
	PSMIsSet() bool
}

// IStatusEnum is enum representing the named state of the entity.
// e.g. *testpb.FooStatus (int32)
type IStatusEnum interface {
	~int32
	ShortString() string
	String() string
}

type IKeyset interface {
	IPSMMessage
	PSMFullName() string
	PSMKeyValues() (map[string]string, error) // map of column_name to UUID string
}

// IState[K, ST, SD]is the main State Entity e.g. *testpb.FooState
type IState[K IKeyset, ST IStatusEnum, SD IStateData] interface {
	IPSMMessage
	GetStatus() ST
	SetStatus(ST)
	PSMMetadata() *psm_j5pb.StateMetadata
	PSMKeys() K
	SetPSMKeys(K)
	PSMData() SD
}

// IStateData is the Data Entity e.g. *testpb.FooData
type IStateData interface {
	IPSMMessage
}

// IEvent is the Event Wrapper, the top level which has metadata, foreign keys to the state, and the event itself.
// e.g. *testpb.FooEvent, the concrete proto message
type IEvent[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	Inner any,
] interface {
	proto.Message
	UnwrapPSMEvent() Inner
	SetPSMEvent(Inner) error
	PSMKeys() K
	SetPSMKeys(K)
	PSMMetadata() *psm_j5pb.EventMetadata
	PSMIsSet() bool
}

// IInnerEvent is the typed event *interface* which is the set of all possible events for the state machine
// e.g. testpb.FooPSMEvent interface - this is generated by the protoc plugin in _psm.pb.go
// It is set at compile time specifically to the interface type.
type IInnerEvent interface {
	IPSMMessage
	PSMEventKey() string
}

// HookBaton is sent with each logic transition hook to collect chain events and
// side effects
type HookBaton[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	SideEffect(o5msg.Message)
	DelayedSideEffect(o5msg.Message, time.Duration)
	ChainEvent(IE)
	FullCause() E
	AsCause() *psm_j5pb.Cause
}

type Publisher interface {
	Publish(o5msg.Message)
}

type transitionMutationWrapper[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	TransitionMutation(SD, IE) error
	EventType() string
}

// TransitionMutation runs at the start of a transition to merge the event
// information into the state data object. The state object is mutable in this
// context.
type TransitionMutation[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
	SE IInnerEvent,
] func(SD, SE) error

func (f TransitionMutation[K, S, ST, SD, E, IE, SE]) TransitionMutation(stateData SD, event IE) error {
	asType, ok := any(event).(SE)
	if !ok {
		name := event.ProtoReflect().Descriptor().FullName()
		return fmt.Errorf("unexpected event type in transition: %s [IE] does not match [SE] (%T)", name, new(SE))
	}

	return f(stateData, asType)
}

func (f TransitionMutation[K, S, ST, SD, E, IE, SE]) EventType() string {
	return (*new(SE)).PSMEventKey()
}

type transitionLogicHookWrapper[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	TransitionLogicHook(context.Context, HookBaton[K, S, ST, SD, E, IE], S, IE) error
	EventType() string
}

// TransitionLogicHook Executes after the mutation is complete. This hook
// can trigger side effects, including chained events, which are additional
// events processed by the state machine. Use this for Business Logic which
// determines the 'next step' in processing.
type TransitionLogicHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
	SE IInnerEvent,
] func(context.Context, HookBaton[K, S, ST, SD, E, IE], S, SE) error

func (f TransitionLogicHook[K, S, ST, SD, E, IE, SE]) TransitionLogicHook(ctx context.Context, baton HookBaton[K, S, ST, SD, E, IE], state S, event IE) error {
	asType, ok := any(event).(SE)
	if !ok {
		name := event.ProtoReflect().Descriptor().FullName()
		return fmt.Errorf("unexpected event type in transition: %s [IE] does not match [SE] (%T)", name, new(SE))
	}

	return f(ctx, baton, state, asType)
}

func (f TransitionLogicHook[K, S, ST, SD, E, IE, SE]) EventType() string {
	return (*new(SE)).PSMEventKey()
}

type transitionDataHookWrapper[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	TransitionDataHook(context.Context, sqrlx.Transaction, S, IE) error
	EventType() string
}

// TransitionDataHook runs after the mutations, and can be used to update data
// in tables which are not controlled as the state machine, e.g. for
// pre-calculating fields for performance reasons. Use of this hook prevents
// (future) transaction optimizations, as the transaction state
// when the function is called must needs to match the processing state, but
// only for this single transition, unlike the GeneralEventDataHook.
type TransitionDataHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
	SE IInnerEvent,
] func(context.Context, sqrlx.Transaction, S, SE) error

func (f TransitionDataHook[K, S, ST, SD, E, IE, SE]) TransitionDataHook(ctx context.Context, tx sqrlx.Transaction, state S, event IE) error {
	asType, ok := any(event).(SE)
	if !ok {
		name := event.ProtoReflect().Descriptor().FullName()
		return fmt.Errorf("unexpected event type in transition: %s [IE] does not match [SE] (%T)", name, new(SE))
	}

	return f(ctx, tx, state, asType)
}

func (f TransitionDataHook[K, S, ST, SD, E, IE, SE]) EventType() string {
	return (*new(SE)).PSMEventKey()
}

type generalLogicHookWrapper[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	GeneralLogicHook(context.Context, HookBaton[K, S, ST, SD, E, IE], S, E) error
}

// GeneralLogicHook runs once per transition at the state-machine level
// regardless of which transition / event is being processed. It runs exactly
// once per transition, with the state object in the final state after the
// transition but prior to processing any further events. Chained events are
// added to the *end* of the event queue for the transaction, and side effects
// are published (as always) when the transaction is committed. The function
// MUST be pure, i.e. It MUST NOT produce any side-effects outside of the
// HookBaton, and MUST NOT modify the
// state.
type GeneralLogicHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] func(context.Context, HookBaton[K, S, ST, SD, E, IE], S, E) error

func (f GeneralLogicHook[K, S, ST, SD, E, IE]) GeneralLogicHook(ctx context.Context, baton HookBaton[K, S, ST, SD, E, IE], state S, event E) error {
	return f(ctx, baton, state, event)
}

type generalStateDataWrapper[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	GeneralStateDataHook(context.Context, sqrlx.Transaction, S) error
}

// GeneralStateDataHook runs at the state-machine level regardless of which
// transition / event is being processed. It runs at-least once before
// committing a database transaction after multiple transitions are complete.
// This hook has access only to the final state after the transitions and is
// used to update other tables based on the resulting state. It MUST be
// idempotent, it may be called after injecting externally-held state data.
type GeneralStateDataHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] func(context.Context, sqrlx.Transaction, S) error

func (f GeneralStateDataHook[K, S, ST, SD, E, IE]) GeneralStateDataHook(ctx context.Context, tx sqrlx.Transaction, state S) error {
	return f(ctx, tx, state)
}

type generalEventDataHookWrapper[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	GeneralEventDataHook(context.Context, sqrlx.Transaction, S, E) error
}

// GeneralEventDataHook runs after each transition at the state-machine level regardless of which
// transition / event is being processed. It runs exactly once per transition,
// before any other events are processed. The presence of this hook type
// prevents (future) transaction optimizations, so should be used sparingly.
type GeneralEventDataHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] func(context.Context, sqrlx.Transaction, S, E) error

func (f GeneralEventDataHook[K, S, ST, SD, E, IE]) GeneralEventDataHook(ctx context.Context, tx sqrlx.Transaction, state S, event E) error {
	return f(ctx, tx, state, event)
}

// EventPublishHook runs for each transition, at least once before committing a
// database transaction after multiple transitions are complete. It should
// publish a derived version of the event using the publisher.
type EventPublishHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] func(context.Context, Publisher, S, E) error

// UpsertPublishHook runs at least once after a set of transitions for an
// entity, it should be used to publish an 'upsert' message of the current
// state. The last call to for a set of transitions will be the final state. Use
// the state metadata's last modified for upsert time.
type UpsertPublishHook[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
] func(context.Context, Publisher, S) error

type LinkHook[
	K IKeyset, // Source Machine Keyset
	S IState[K, ST, SD], // Source Machine State
	ST IStatusEnum, // Source Machine Status Enum
	SD IStateData, // Source Machine State Data
	E IEvent[K, S, ST, SD, IE], // Source Machine Event
	IE IInnerEvent, // Source Machine Inner Event
	SE IInnerEvent, // Source Machine Specific Event

	DK IKeyset, // Destination Keyset
	DIE IInnerEvent, // Destination Inner Event
] struct {
	Destination LinkDestination[DK, DIE]
	Derive      func(context.Context, S, SE, func(DK, DIE)) error
}

func (lh LinkHook[K, S, ST, SD, E, IE, SE, DK, DIE]) RunLinkTransition(ctx context.Context, tx sqrlx.Transaction, state S, event E) error {
	asType, ok := any(event.UnwrapPSMEvent()).(SE)
	if !ok {
		return fmt.Errorf("unexpected event type in transition: %T [IE] does not match [SE] (%T)", any(event), new(SE))
	}

	type matchedEvent struct {
		Key   DK
		Inner DIE
	}

	events := make([]matchedEvent, 0, 1)
	err := lh.Derive(ctx, state, asType, func(key DK, event DIE) {
		events = append(events, matchedEvent{Key: key, Inner: event})
	})
	if err != nil {
		return err
	}

	cause := &psm_j5pb.Cause{
		Type: &psm_j5pb.Cause_PsmEvent{
			PsmEvent: &psm_j5pb.PSMEventCause{
				EventId:      event.PSMMetadata().EventId,
				StateMachine: (*new(K)).PSMFullName(),
			},
		},
	}

	for _, chained := range events {
		destKeys := chained.Key
		destEvent := chained.Inner
		if err := lh.Destination.transitionFromLink(ctx, tx, cause, destKeys, destEvent); err != nil {
			return err
		}
	}
	return nil
}

func (lh LinkHook[K, S, ST, SD, E, IE, SE, DK, DIE]) EventType() string {
	return (*new(SE)).PSMEventKey()
}

type LinkDestination[
	DK IKeyset,
	DIE IInnerEvent,
] interface {
	transitionFromLink(
		ctx context.Context,
		tx sqrlx.Transaction,
		cause *psm_j5pb.Cause,
		destKeys DK,
		destEvent DIE,
	) error
}

type transitionLink[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] interface {
	RunLinkTransition(context.Context, sqrlx.Transaction, S, E) error
	EventType() string
}
