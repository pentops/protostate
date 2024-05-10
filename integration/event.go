package integration

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/utils/ptr"
)

func newFooCreatedEvent(fooID, tenantID string, mod func(c *testpb.FooEventType_Created)) *testpb.FooEvent {
	e := newFooEvent(fooID, tenantID, func(e *testpb.FooEvent) {
		weight := int64(10)
		e.Event.Type = &testpb.FooEventType_Created_{
			Created: &testpb.FooEventType_Created{
				Name:        "foo",
				Field:       fmt.Sprintf("weight: %d", weight),
				Description: ptr.To("creation event for foo: " + fooID),
				Weight:      &weight,
			},
		}
	})

	if mod != nil {
		mod(e.Event.GetCreated())
	}

	return e
}

func newFooUpdatedEvent(fooID, tenantID string, mod func(u *testpb.FooEventType_Updated)) *testpb.FooEvent {
	e := newFooEvent(fooID, tenantID, func(e *testpb.FooEvent) {
		weight := int64(20)
		e.Event.Type = &testpb.FooEventType_Updated_{
			Updated: &testpb.FooEventType_Updated{
				Name:        "foo",
				Field:       fmt.Sprintf("weight: %d", weight),
				Description: ptr.To("update event for foo: " + fooID),
				Weight:      &weight,
			},
		}
	})

	if mod != nil {
		mod(e.Event.GetUpdated())
	}

	return e
}

func newFooEvent(fooID, tenantID string, mod func(e *testpb.FooEvent)) *testpb.FooEvent {
	e := &testpb.FooEvent{
		Metadata: &psm_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Keys: &testpb.FooKeys{
			FooId:    fooID,
			TenantId: &tenantID,
		},
		Event: &testpb.FooEventType{},
	}
	mod(e)
	return e
}

func newBarCreatedEvent(barID string, mod func(c *testpb.BarEventType_Created)) *testpb.BarEvent {
	e := newBarEvent(barID, func(e *testpb.BarEvent) {
		e.Event.Type = &testpb.BarEventType_Created_{
			Created: &testpb.BarEventType_Created{
				Name:  "bar",
				Field: "event",
			},
		}
	})

	if mod != nil {
		mod(e.Event.GetCreated())
	}

	return e
}

func newBarEvent(barID string, mod func(e *testpb.BarEvent)) *testpb.BarEvent {
	e := &testpb.BarEvent{
		Metadata: &psm_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Keys: &testpb.BarKeys{
			BarId: barID,
		},
		Event: &testpb.BarEventType{},
	}
	mod(e)
	return e
}
