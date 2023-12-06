package example

import (
	"context"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.daemonl.com/sqrlx"
)

func NewFooStateMachine(db *sqrlx.Wrapper) (*testpb.FooPSM, error) {

	sm, err := testpb.NewFooPSM(db)
	if err != nil {
		return nil, err
	}

	sm.From(testpb.FooStatus_UNSPECIFIED).
		Where(func(event *testpb.FooEvent) bool {
			return true
		}).
		Do(testpb.FooPSMFunc(func(
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

	return (*testpb.FooPSM)(sm), nil
}

func TestFooStateMachine(t *testing.T) {
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
