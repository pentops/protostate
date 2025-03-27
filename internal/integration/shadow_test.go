package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStateMachineShadow(t *testing.T) {
	flow, uu := NewUniverse(t)
	defer flow.RunSteps(t)

	tenantID := uuid.NewString()
	foo1ID := uuid.NewString()
	foo2ID := uuid.NewString()

	events := []*test_pb.FooEvent{{
		Metadata: &psm_j5pb.EventMetadata{
			EventId:   uuid.NewString(),
			Sequence:  1,
			Cause:     &psm_j5pb.Cause{},
			Timestamp: timestamppb.Now(),
		},
		Keys: &test_pb.FooKeys{
			FooId:        foo1ID,
			TenantId:     &tenantID,
			MetaTenantId: metaTenant,
		},
		Event: &test_pb.FooEventType{
			Type: &test_pb.FooEventType_Created_{
				Created: &test_pb.FooEventType_Created{
					Name: "foo1",
				},
			},
		},
	}, {
		Metadata: &psm_j5pb.EventMetadata{
			EventId:   uuid.NewString(),
			Sequence:  1,
			Cause:     &psm_j5pb.Cause{},
			Timestamp: timestamppb.Now(),
		},
		Keys: &test_pb.FooKeys{
			FooId:        foo2ID,
			TenantId:     &tenantID,
			MetaTenantId: metaTenant,
		},
		Event: &test_pb.FooEventType{
			Type: &test_pb.FooEventType_Created_{
				Created: &test_pb.FooEventType_Created{
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
		res1, err := uu.FooQuery.FooGet(ctx, &test_spb.FooGetRequest{
			FooId: foo1ID,
		})
		t.NoError(err)
		// ACTIVE means the logic hook did not automatically run, which is what we
		// want.
		t.Equal(test_pb.FooStatus_ACTIVE, res1.Foo.Status)
		t.Equal("foo1", res1.Foo.Data.Name)

		res2, err := uu.FooQuery.FooGet(ctx, &test_spb.FooGetRequest{
			FooId: foo2ID,
		})
		t.NoError(err)
		t.Equal(test_pb.FooStatus_ACTIVE, res2.Foo.Status)
		t.Equal("foo2", res2.Foo.Data.Name)

	})
}
