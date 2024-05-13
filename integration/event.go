package integration

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"k8s.io/utils/ptr"
)

func newFooCreatedEvent(fooID, tenantID string, mod func(c *testpb.FooEventType_Created)) *testpb.FooPSMEventSpec {
	weight := int64(10)
	created := &testpb.FooEventType_Created{
		Name:        "foo",
		Field:       fmt.Sprintf("weight: %d", weight),
		Description: ptr.To("creation event for foo: " + fooID),
		Weight:      &weight,
	}
	e := newFooEvent(fooID, tenantID, func(e *testpb.FooPSMEventSpec) {
		e.Event = created
	})

	if mod != nil {
		mod(created)
	}

	return e
}

func newFooUpdatedEvent(fooID, tenantID string, mod func(u *testpb.FooEventType_Updated)) *testpb.FooPSMEventSpec {
	weight := int64(20)
	updated := &testpb.FooEventType_Updated{
		Name:        "foo",
		Field:       fmt.Sprintf("weight: %d", weight),
		Description: ptr.To("update event for foo: " + fooID),
		Weight:      &weight,
	}
	e := newFooEvent(fooID, tenantID, func(e *testpb.FooPSMEventSpec) {
		e.Event = updated
	})

	if mod != nil {
		mod(updated)
	}

	return e
}

func newFooEvent(fooID, tenantID string, mod func(e *testpb.FooPSMEventSpec)) *testpb.FooPSMEventSpec {
	e := &testpb.FooPSMEventSpec{
		EventID: uuid.NewString(),
		Keys: &testpb.FooKeys{
			FooId:    fooID,
			TenantId: &tenantID,
		},
	}
	mod(e)
	return e
}

func newBarCreatedEvent(barID string, mod func(c *testpb.BarEventType_Created)) *testpb.BarPSMEventSpec {
	created := &testpb.BarEventType_Created{
		Name:  "bar",
		Field: "event",
	}
	e := newBarEvent(barID, func(e *testpb.BarPSMEventSpec) {
		e.Event = created
	})

	if mod != nil {
		mod(created)
	}

	return e
}

func newBarEvent(barID string, mod func(e *testpb.BarPSMEventSpec)) *testpb.BarPSMEventSpec {
	e := &testpb.BarPSMEventSpec{
		EventID: uuid.NewString(),
		Keys: &testpb.BarKeys{
			BarId: barID,
		},
	}
	mod(e)
	return e
}
