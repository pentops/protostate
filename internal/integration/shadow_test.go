package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/protostate/internal/testproto/gen/testpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStateMachineShadow(t *testing.T) {
	flow, uu := NewUniverse(t)
	defer flow.RunSteps(t)

	tenantID := uuid.NewString()
	foo1ID := uuid.NewString()
	foo2ID := uuid.NewString()

	events := []*testpb.FooEvent{{
		Metadata: &psm_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Sequence:  1,
			Cause:     &psm_pb.Cause{},
			Timestamp: timestamppb.Now(),
		},
		Keys: &testpb.FooKeys{
			FooId:    foo1ID,
			TenantId: &tenantID,
		},
		Event: &testpb.FooEventType{
			Type: &testpb.FooEventType_Created_{
				Created: &testpb.FooEventType_Created{
					Name: "foo1",
				},
			},
		},
	}, {
		Metadata: &psm_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Sequence:  1,
			Cause:     &psm_pb.Cause{},
			Timestamp: timestamppb.Now(),
		},
		Keys: &testpb.FooKeys{
			FooId:    foo2ID,
			TenantId: &tenantID,
		},
		Event: &testpb.FooEventType{
			Type: &testpb.FooEventType_Created_{
				Created: &testpb.FooEventType_Created{
					Name: "foo2",
				},
			},
		},
	}}

	flow.Step("Load", func(ctx context.Context, t flowtest.Asserter) {
		for _, event := range events {
			if err := uu.FooStateMachine.FollowEvent(ctx, event); err != nil {
				t.Fatalf("failed to load events: %v", err)
			}
		}
	})

	flow.Step("Check", func(ctx context.Context, t flowtest.Asserter) {
		res1, err := uu.FooQuery.GetFoo(ctx, &testpb.GetFooRequest{
			FooId: foo1ID,
		})
		t.NoError(err)
		// ACTIVE means the logic hook did not automatically run, which is what we
		// want.
		t.Equal(testpb.FooStatus_ACTIVE, res1.State.Status)
		t.Equal("foo1", res1.State.Data.Name)

		res2, err := uu.FooQuery.GetFoo(ctx, &testpb.GetFooRequest{
			FooId: foo2ID,
		})
		t.NoError(err)
		t.Equal(testpb.FooStatus_ACTIVE, res2.State.Status)
		t.Equal("foo2", res2.State.Data.Name)

	})
}
