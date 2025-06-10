package psm

import (
	"crypto/sha1"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/pentops/j5/gen/j5/auth/v1/auth_j5pb"
	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/j5/lib/id62"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventSpec[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	// Keys must be set, to identify the state machine.
	Keys K

	// The inner PSM Event type. Must be set for incoming events.
	Event IE

	// The cause of the event, Cause, Action or Message must be set for incoming events.
	Cause *psm_j5pb.Cause

	// The authenticated action cause for the event. Cause, Action or Message must be set
	// for incoming events.
	Action *auth_j5pb.Action

	// The message cause for the event. Cause, Action or Message must be set for
	// incoming events.
	Message *messaging_j5pb.MessageCause

	// Optional, defaults to the system time (if Zero())
	Timestamp time.Time
}

func (es *EventSpec[K, S, ST, SD, E, IE]) validateAndPrepare() error {

	if !es.Keys.PSMIsSet() {
		return fmt.Errorf("EventSpec.Keys is required")
	}

	if !es.Event.PSMIsSet() {
		return fmt.Errorf("EventSpec.Event must be set")
	}
	if es.Cause != nil && es.Action != nil {
		return fmt.Errorf("EventSpec.Cause and EventSpec.Action are mutually exclusive")
	}

	if es.Action != nil {
		if es.Cause != nil {
			return fmt.Errorf("EventSpec.Cause, EventSpec.Action and EventSpec.Message are mutually exclusive")
		}
		es.Cause = &psm_j5pb.Cause{
			Type: &psm_j5pb.Cause_Command{
				Command: es.Action,
			},
		}
	}
	if es.Message != nil {
		if es.Cause != nil {
			return fmt.Errorf("EventSpec.Cause, EventSpec.Action and EventSpec.Message are mutually exclusive")
		}
		es.Cause = &psm_j5pb.Cause{
			Type: &psm_j5pb.Cause_Message{
				Message: es.Message,
			},
		}
	}

	if es.Cause == nil {
		return fmt.Errorf("EventSpec.Cause, EventSpec.Action or EventSpec.Message must be set")
	}

	// check that the cause type is supported.
	switch es.Cause.Type.(type) {
	case *psm_j5pb.Cause_PsmEvent:
	case *psm_j5pb.Cause_Command:
	case *psm_j5pb.Cause_ExternalEvent:
	case *psm_j5pb.Cause_Message:

	default:
		return fmt.Errorf("EventSpec.Cause.Source must be set")
	}

	return nil
}

func base62String(id []byte) string {
	var i big.Int
	i.SetBytes(id)
	str := i.Text(62)
	return fmt.Sprintf("%022s", str)
}

// hashString generates a sha1 hash from the input strings.
func hashString(input ...string) (string, error) {
	if len(input) == 0 {
		return "", fmt.Errorf("hashString requires at least one input string")
	}
	fullInput := strings.Join(input, "")
	sum := sha1.Sum([]byte(fullInput))
	return base62String(sum[:]), nil
}

func eventIdempotencyKey(event looseEvent) (string, error) {
	eventFullType := string(event.ProtoReflect().Descriptor().FullName())

	switch cause := event.PSMMetadata().Cause.Type.(type) {
	case *psm_j5pb.Cause_PsmEvent:
		return hashString(
			eventFullType,
			cause.PsmEvent.EventId,
		)

	case *psm_j5pb.Cause_Command:
		providedKey := cause.Command.IdempotencyKey
		if providedKey == "" {
			return event.PSMMetadata().GetEventId(), nil
		}
		return hashString(
			eventFullType,
			cause.Command.Actor.Claim.TenantId,
			providedKey,
		)

	case *psm_j5pb.Cause_ExternalEvent:
		if cause.ExternalEvent.ExternalId != nil {
			return hashString(
				eventFullType,
				*cause.ExternalEvent.ExternalId,
			)
		} else {
			return event.PSMMetadata().GetEventId(), nil
		}

	case *psm_j5pb.Cause_Message:
		return hashString(
			eventFullType,
			cause.Message.MessageId,
		)

	case *psm_j5pb.Cause_Init:
		return event.PSMMetadata().GetEventId(), nil

	default:
		return "", fmt.Errorf("EventSpec.Cause.Source must be set")
	}
}

type preparedEvent[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
] struct {
	event          E
	state          S
	idempotencyKey string
}

func prepareFollowEvent[
	K IKeyset,
	S IState[K, ST, SD],
	ST IStatusEnum,
	SD IStateData,
	E IEvent[K, S, ST, SD, IE],
	IE IInnerEvent,
](event E, state S) (built preparedEvent[K, S, ST, SD, E, IE], err error) {
	idempotencyKey, err := eventIdempotencyKey(event)
	if err != nil {
		return built, err
	}
	return preparedEvent[K, S, ST, SD, E, IE]{
		event:          event,
		state:          state,
		idempotencyKey: idempotencyKey,
	}, nil
}

func (es *EventSpec[K, S, ST, SD, E, IE]) buildWrapper(state S) (built preparedEvent[K, S, ST, SD, E, IE], err error) {

	evt := (*new(E)).ProtoReflect().New().Interface().(E)
	if err := evt.SetPSMEvent(es.Event); err != nil {
		return built, fmt.Errorf("set event: %w", err)
	}
	evt.SetPSMKeys(es.Keys)

	eventMeta := evt.PSMMetadata()
	eventMeta.EventId = id62.NewString()
	eventMeta.Timestamp = timestamppb.Now()
	eventMeta.Cause = es.Cause

	incrementEventSequence(state, eventMeta)

	built.event = evt
	built.state = state

	idempotencyKey, err := eventIdempotencyKey(evt)
	if err != nil {
		return built, err
	}
	built.idempotencyKey = idempotencyKey

	return

}

func incrementEventSequence[K IKeyset, S IState[K, ST, SD], ST IStatusEnum, SD IStateData](state S, eventMeta *psm_j5pb.EventMetadata) {
	stateMeta := state.PSMMetadata()

	eventMeta.Sequence = 0
	if state.GetStatus() == 0 {
		eventMeta.Sequence = 0
		stateMeta.CreatedAt = eventMeta.Timestamp
		stateMeta.UpdatedAt = eventMeta.Timestamp
	} else {
		eventMeta.Sequence = stateMeta.LastSequence + 1
		stateMeta.LastSequence = eventMeta.Sequence
		stateMeta.UpdatedAt = eventMeta.Timestamp
	}
}
