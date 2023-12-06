package example

import (
	"context"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/pquery"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

	sm.From(testpb.FooStatus_ACTIVE).
		Do(testpb.FooPSMFunc(func(
			ctx context.Context,
			tb testpb.FooPSMTransitionBaton,
			state *testpb.FooState,
			event *testpb.FooEventType_Updated,
		) error {
			state.Field = event.Field
			state.Name = event.Name
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

	event2 := &testpb.FooEvent{
		Metadata: &testpb.Metadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
			Actor: &testpb.Actor{
				ActorId: uuid.NewString(),
			},
		},
		FooId: fooID,
		Event: &testpb.FooEventType{
			Type: &testpb.FooEventType_Updated_{
				Updated: &testpb.FooEventType_Updated{
					Name:  "foo",
					Field: "event2",
				},
			},
		},
	}

	foo2ID := uuid.NewString()
	event3 := &testpb.FooEvent{
		Metadata: &testpb.Metadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
			Actor: &testpb.Actor{
				ActorId: uuid.NewString(),
			},
		},
		FooId: foo2ID,
		Event: &testpb.FooEventType{
			Type: &testpb.FooEventType_Created_{
				Created: &testpb.FooEventType_Created{
					Name:  "foo2",
					Field: "event3",
				},
			},
		},
	}

	statesOut := map[string]*testpb.FooState{}
	for _, event := range []*testpb.FooEvent{event1, event2, event3} {
		stateOut, err := sm.Transition(ctx, event)
		if err != nil {
			t.Fatal(err.Error())
		}
		statesOut[event.FooId] = stateOut
	}

	if statesOut[fooID].GetStatus() != testpb.FooStatus_ACTIVE {
		t.Fatalf("Expect state ACTIVE, got %s", statesOut[fooID].GetStatus().ShortString())
	}

	queryer, err := pquery.FromStateMachine(sm, pquery.StateSpec[
		*testpb.GetFooRequest,
		*testpb.GetFooResponse,
		*testpb.ListFoosRequest,
		*testpb.ListFoosResponse,
		*testpb.ListFooEventsRequest,
		*testpb.ListFooEventsResponse,
	]{})
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Run("Get", func(t *testing.T) {

		req := &testpb.GetFooRequest{
			FooId: fooID,
		}

		res := &testpb.GetFooResponse{}

		err = queryer.Get(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if !proto.Equal(res.State, statesOut[fooID]) {
			t.Fatalf("expected %v, got %v", statesOut[fooID], res.State)
		}

		if len(res.Events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(res.Events))
		}
		t.Log(res.Events)
	})

	t.Run("List", func(t *testing.T) {

		req := &testpb.ListFoosRequest{}
		res := &testpb.ListFoosResponse{}

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Foos) != 2 {
			t.Fatalf("expected 2 states, got %d", len(res.Foos))
		}

	})

	t.Run("ListEvents", func(t *testing.T) {

		req := &testpb.ListFooEventsRequest{}
		res := &testpb.ListFooEventsResponse{}

		err = queryer.ListEvents(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Events) != 3 {
			t.Fatalf("expected 3 events, got %d", len(res.Events))
		}
	})
}
