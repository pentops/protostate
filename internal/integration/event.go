package integration

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/protostate/internal/testproto/gen/testpb"
	"k8s.io/utils/ptr"
)

var metaTenant = uuid.NewString()

func newFooCreatedEvent(fooID, tenantID string, mod func(c *testpb.FooEventType_Created)) *testpb.FooPSMEventSpec {
	weight := int64(10)
	created := &testpb.FooEventType_Created{
		Name:        "foo",
		Field:       fmt.Sprintf("weight: %d", weight),
		Description: ptr.To("creation event for foo: " + fooID),
		Weight:      &weight,
	}

	if mod != nil {
		mod(created)
	}

	return newFooEvent(&testpb.FooKeys{
		FooId:        fooID,
		TenantId:     &tenantID,
		MetaTenantId: metaTenant,
	}, created)
}

func newFooUpdatedEvent(fooID, tenantID string, mod func(u *testpb.FooEventType_Updated)) *testpb.FooPSMEventSpec {
	weight := int64(20)
	updated := &testpb.FooEventType_Updated{
		Name:        "foo",
		Field:       fmt.Sprintf("weight: %d", weight),
		Description: ptr.To("update event for foo: " + fooID),
		Weight:      &weight,
	}

	if mod != nil {
		mod(updated)
	}

	return newFooEvent(&testpb.FooKeys{
		FooId:        fooID,
		TenantId:     &tenantID,
		MetaTenantId: metaTenant,
	}, updated)
}

func newFooEvent(keys *testpb.FooKeys, et testpb.FooPSMEvent) *testpb.FooPSMEventSpec {

	if keys.MetaTenantId == "" {
		panic("metaTenantId is required")
	}
	e := &testpb.FooPSMEventSpec{
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

func newBarCreatedEvent(barID string, mod func(c *testpb.BarEventType_Created)) *testpb.BarPSMEventSpec {
	created := &testpb.BarEventType_Created{
		Name:  "bar",
		Field: "event",
	}

	if mod != nil {
		mod(created)
	}

	return newBarEvent(barID, func(e *testpb.BarPSMEventSpec) {
		e.Event = created
	})
}

var barOtherID = "FCC73FA5-B653-4DCC-AA6F-3548BECB7B2A"

func newBarEvent(barID string, mod func(e *testpb.BarPSMEventSpec)) *testpb.BarPSMEventSpec {
	e := &testpb.BarPSMEventSpec{
		EventID: uuid.NewString(),
		Keys: &testpb.BarKeys{
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
