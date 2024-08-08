package psm

import (
	"fmt"
	"time"

	"github.com/pentops/j5/gen/j5/auth/v1/auth_j5pb"
	"github.com/pentops/j5/gen/psm/state/v1/psm_pb"
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

	// EventID must be set for incomming events.
	EventID string

	// The inner PSM Event type. Must be set for incomming events.
	Event IE

	// The cause of the event, Cause or Action must be set for incomming events.
	Cause *psm_pb.Cause

	// The authenticated action cause for the event. Cause or Action must be set
	// for incomming events.
	Action *auth_j5pb.Action

	// Optional, defaults to the system time (if Zero())
	Timestamp time.Time
}

func (es EventSpec[K, S, ST, SD, E, IE]) validateAndPrepare() error {

	if !es.Keys.PSMIsSet() {
		return fmt.Errorf("EventSpec.Keys is required")
	}

	if !es.Event.PSMIsSet() {
		return fmt.Errorf("EventSpec.Event must be set")
	}
	if es.Cause != nil && es.Action != nil {
		return fmt.Errorf("EventSpec.Cause and EventSpec.Action are mutually exclusive")
	}

	if es.Cause == nil {
		if es.Action == nil {
			return fmt.Errorf("EventSpec.Cause or EventSpec.Action must be set")
		}
		es.Cause = &psm_pb.Cause{
			Type: &psm_pb.Cause_Command{
				Command: es.Action,
			},
		}
	}

	// check that the cause type is supported.
	switch es.Cause.Type.(type) {
	case *psm_pb.Cause_PsmEvent, *psm_pb.Cause_Command, *psm_pb.Cause_ExternalEvent, *psm_pb.Cause_Reply:
		// All OK
	default:
		return fmt.Errorf("EventSpec.Cause.Source must be set")
	}

	return nil
}
