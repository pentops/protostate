package integration

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"k8s.io/utils/ptr"
)

func NewFooStateMachine() (*test_pb.FooPSM, error) {
	sm, err := test_pb.FooPSMBuilder().BuildStateMachine()
	if err != nil {
		return nil, err
	}

	sm.From(0).
		OnEvent(test_pb.FooPSMEventCreated).
		SetStatus(test_pb.FooStatus_ACTIVE).
		Mutate(test_pb.FooPSMMutation(func(
			data *test_pb.FooData,
			event *test_pb.FooEventType_Created,
		) error {
			data.Name = event.Name
			data.Field = event.Field
			data.Description = event.Description
			data.Characteristics = &test_pb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}
			data.Profiles = event.Profiles
			return nil
		}))

	// Testing Mutate() without OnEvent, the callback implies the event type.
	sm.From(test_pb.FooStatus_ACTIVE).
		Mutate(test_pb.FooPSMMutation(func(
			data *test_pb.FooData,
			event *test_pb.FooEventType_Updated,
		) error {
			data.Field = event.Field
			data.Name = event.Name
			data.Description = event.Description
			data.Characteristics = &test_pb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}

			return nil
		}))

	sm.From(test_pb.FooStatus_ACTIVE).
		OnEvent(test_pb.FooPSMEventDeleted).
		SetStatus(test_pb.FooStatus_DELETED)

	return sm, nil

}

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
func newFooDeletedEvent(fooID, tenantID string, mod ...func(u *test_pb.FooEventType_Deleted)) *test_pb.FooPSMEventSpec {
	deleted := &test_pb.FooEventType_Deleted{}

	for _, m := range mod {
		m(deleted)
	}

	return newFooEvent(&test_pb.FooKeys{
		FooId:        fooID,
		TenantId:     &tenantID,
		MetaTenantId: metaTenant,
	}, deleted)
}

func newFooEvent(keys *test_pb.FooKeys, et test_pb.FooPSMEvent) *test_pb.FooPSMEventSpec {

	if keys.MetaTenantId == "" {
		panic("metaTenantId is required")
	}
	e := &test_pb.FooPSMEventSpec{
		Keys: keys,
		Cause: &psm_j5pb.Cause{
			Type: &psm_j5pb.Cause_ExternalEvent{
				ExternalEvent: &psm_j5pb.ExternalEventCause{
					SystemName: "a",
					EventName:  "b",
				},
			},
		},
		Event: et,
	}

	return e
}
