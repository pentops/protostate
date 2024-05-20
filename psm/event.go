package psm

import (
	"fmt"
	"time"

	"github.com/pentops/protostate/gen/state/v1/psm_pb"
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

	// The cause of the event. Must be set for incomming events.
	Cause *psm_pb.Cause

	// Optional, defaults to the system time (if Zero())
	Timestamp time.Time
}

func (es EventSpec[K, S, ST, SD, E, IE]) validateIncomming() error {

	if !es.Keys.PSMIsSet() {
		return fmt.Errorf("EventSpec.Keys is required")
	}

	if !es.Event.PSMIsSet() {
		return fmt.Errorf("EventSpec.Event must be set")
	}

	if es.Cause == nil {
		return fmt.Errorf("EventSpec.Cause must be set")
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
