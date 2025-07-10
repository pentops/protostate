package psm

import (
	"fmt"
	"time"

	"github.com/pentops/j5/gen/j5/auth/v1/auth_j5pb"
	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
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

	// EventID is optional and will be set by the state machine if empty
	EventID string

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
	case *psm_j5pb.Cause_PsmEvent,
		*psm_j5pb.Cause_Command,
		*psm_j5pb.Cause_ExternalEvent,
		*psm_j5pb.Cause_Message,
		*psm_j5pb.Cause_Init:
		// All OK
	default:
		return fmt.Errorf("EventSpec.Cause.Source must be set")
	}

	return nil
}
