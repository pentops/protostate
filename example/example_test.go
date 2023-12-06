package example

import (
	"context"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.daemonl.com/sqrlx"
)

type FooStateMachine struct {
	*psm.StateMachine[
		*testpb.FooState,
		testpb.FooStatus,
		*testpb.FooEvent,
		testpb.FooPSMEvent,
	]
}

func NewFooStateMachine(db *sqrlx.Wrapper) (*FooStateMachine, error) {

	sm, err := psm.NewStateMachine[
		*testpb.FooState,
		testpb.FooStatus,
		*testpb.FooEvent,
		testpb.FooPSMEvent,
	](db, testpb.FooPSMConverter{
		NewMetadata: func(context.Context) *testpb.Metadata {
			return &testpb.Metadata{
				EventId:   uuid.NewString(),
				Timestamp: timestamppb.Now(),
			}
		},
		ExtractMetadata: func(event *testpb.Metadata) *psm.Metadata {
			return &psm.Metadata{
				EventID:   event.EventId,
				Timestamp: event.Timestamp.AsTime(),
			}
		},
	}, testpb.FooPSMSpec{
		StateTable: "foo",
		EventTable: "foo_event",
		PrimaryKey: func(event *testpb.FooEvent) map[string]interface{} {
			return map[string]interface{}{
				"id": event.FooId,
			}
		},
		EventForeignKey: func(event *testpb.FooEvent) map[string]interface{} {
			return map[string]interface{}{
				"foo_id": event.FooId,
			}
		},
	})
	if err != nil {
		return nil, err
	}

	sm.Register(psm.NewTransition[
		*testpb.FooState,
		testpb.FooStatus,
		*testpb.FooEvent,
		testpb.FooPSMEvent,
		*testpb.FooEventType_Created,
	]([]testpb.FooStatus{
		testpb.FooStatus_UNSPECIFIED,
	}, func(
		ctx context.Context,
		tb testpb.FooPSMTransitionBaton,
		state *testpb.FooState,
		event *testpb.FooEventType_Created,
	) error {
		state.Status = testpb.FooStatus_ACTIVE
		state.Name = event.Name
		state.Field = event.Field

		return nil
	}))

	return &FooStateMachine{
		StateMachine: sm,
	}, nil
}

func TestStateMachine(t *testing.T) {
	ctx := context.Background()

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	fooID := uuid.NewString()
	event1 := &testpb.FooEvent{
		Metadata: &testpb.Metadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
			Actor: &testpb.Actor{
				ActorId: uuid.NewString(),
			},
		},
		FooId: fooID,
		Event: &testpb.FooEventType{
			Type: &testpb.FooEventType_Created_{
				Created: &testpb.FooEventType_Created{
					Name:  "foo",
					Field: "event1",
				},
			},
		},
	}

	stateOut, err := sm.Transition(ctx, event1)
	if err != nil {
		t.Fatal(err.Error())
	}

	if stateOut.GetStatus() != testpb.FooStatus_ACTIVE {
		t.Fatalf("Expect state ACTIVE, got %s", stateOut.GetStatus().ShortString())
	}

}
