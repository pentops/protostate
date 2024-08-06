package integration

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"

	"k8s.io/utils/ptr"
)

var metaTenant = uuid.NewString()

func newFooCreatedEvent(fooID, tenantID string, mod func(c *test_pb.FooEventType_Created)) *test_pb.FooPSMEventSpec {
	weight := int64(10)
	created := &test_pb.FooEventType_Created{
		Name:        "foo",
		Field:       fmt.Sprintf("weight: %d", weight),
		Description: ptr.To("creation event for foo: " + fooID),
		Weight:      &weight,
	}

	if mod != nil {
		mod(created)
	}

	return newFooEvent(&test_pb.FooKeys{
		FooId:        fooID,
		TenantId:     &tenantID,
		MetaTenantId: metaTenant,
	}, created)
}

func newFooUpdatedEvent(fooID, tenantID string, mod func(u *test_pb.FooEventType_Updated)) *test_pb.FooPSMEventSpec {
	weight := int64(20)
	updated := &test_pb.FooEventType_Updated{
		Name:        "foo",
		Field:       fmt.Sprintf("weight: %d", weight),
		Description: ptr.To("update event for foo: " + fooID),
		Weight:      &weight,
	}

	if mod != nil {
		mod(updated)
	}

	return newFooEvent(&test_pb.FooKeys{
		FooId:        fooID,
		TenantId:     &tenantID,
		MetaTenantId: metaTenant,
	}, updated)
}

func newFooEvent(keys *test_pb.FooKeys, et test_pb.FooPSMEvent) *test_pb.FooPSMEventSpec {

	if keys.MetaTenantId == "" {
		panic("metaTenantId is required")
	}
	e := &test_pb.FooPSMEventSpec{
		EventID: uuid.NewString(),
		Keys:    keys,
		Cause: &psm_pb.Cause{
			Type: &psm_pb.Cause_ExternalEvent{
				ExternalEvent: &psm_pb.ExternalEventCause{
					SystemName: "a",
					EventName:  "b",
				},
			},
		},
		Event: et,
	}

	return e
}

func newBarCreatedEvent(barID string, mod func(c *test_pb.BarEventType_Created)) *test_pb.BarPSMEventSpec {
	created := &test_pb.BarEventType_Created{
		Name:  "bar",
		Field: "event",
	}

	if mod != nil {
		mod(created)
	}

	return newBarEvent(barID, func(e *test_pb.BarPSMEventSpec) {
		e.Event = created
	})
}

var barOtherID = "FCC73FA5-B653-4DCC-AA6F-3548BECB7B2A"

func newBarEvent(barID string, mod func(e *test_pb.BarPSMEventSpec)) *test_pb.BarPSMEventSpec {
	e := &test_pb.BarPSMEventSpec{
		EventID: uuid.NewString(),
		Keys: &test_pb.BarKeys{
			BarId:      barID,
			BarOtherId: barOtherID,
		},
		Cause: &psm_pb.Cause{
			Type: &psm_pb.Cause_ExternalEvent{
				ExternalEvent: &psm_pb.ExternalEventCause{
					SystemName: "a",
					EventName:  "b",
				},
			},
		},
	}
	mod(e)
	return e
}
