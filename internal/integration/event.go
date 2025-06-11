package integration

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"

	"k8s.io/utils/ptr"
)

var metaTenant = uuid.NewString()

func newFooCreatedEvent(fooID, tenantID string, mod ...func(c *test_pb.FooEventType_Created)) *test_pb.FooPSMEventSpec {
	weight := int64(10)
	created := &test_pb.FooEventType_Created{
		Name:        "foo",
		Field:       fmt.Sprintf("weight: %d", weight),
		Description: ptr.To("creation event for foo: " + fooID),
		Weight:      &weight,
	}

	for _, m := range mod {
		m(created)
	}

	return newFooEvent(&test_pb.FooKeys{
		FooId:        fooID,
		TenantId:     &tenantID,
		MetaTenantId: metaTenant,
	}, created)
}

func newFooUpdatedEvent(fooID, tenantID string, mod ...func(u *test_pb.FooEventType_Updated)) *test_pb.FooPSMEventSpec {
	weight := int64(20)
	updated := &test_pb.FooEventType_Updated{
		Name:        "foo",
		Field:       fmt.Sprintf("weight: %d", weight),
		Description: ptr.To("update event for foo: " + fooID),
		Weight:      &weight,
	}

	for _, m := range mod {
		m(updated)
	}

	return newFooEvent(&test_pb.FooKeys{
		FooId:        fooID,
		TenantId:     &tenantID,
		MetaTenantId: metaTenant,
	}, updated)
}

func newFooDeletedEvent(fooID, tenantID string) *test_pb.FooPSMEventSpec {
	return newFooEvent(&test_pb.FooKeys{
		FooId:        fooID,
		TenantId:     &tenantID,
		MetaTenantId: metaTenant,
	}, &test_pb.FooEventType_Deleted{})
}

func newFooEvent(keys *test_pb.FooKeys, et test_pb.FooPSMEvent) *test_pb.FooPSMEventSpec {

	if keys.MetaTenantId == "" {
		panic("metaTenantId is required")
	}
	e := &test_pb.FooPSMEventSpec{
		Keys:  keys,
		Cause: testCause(),
		Event: et,
	}

	return e
}

func testCause() *psm_j5pb.Cause {
	return &psm_j5pb.Cause{
		Type: &psm_j5pb.Cause_ExternalEvent{
			ExternalEvent: &psm_j5pb.ExternalEventCause{
				SystemName: "test",
			},
		},
	}
}
