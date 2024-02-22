package integration

import (
	"context"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/utils/ptr"
)

func TestStateEntityExtensions(t *testing.T) {
	event := &testpb.FooEvent{}
	assert.Equal(t, testpb.FooPSMEventNil, event.PSMEventKey())
	event.SetPSMEvent(&testpb.FooEventType_Created{})
	assert.Equal(t, testpb.FooPSMEventCreated, event.Event.PSMEventKey())
	assert.Equal(t, testpb.FooPSMEventCreated, event.PSMEventKey())
}

func TestFooStateField(t *testing.T) {
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db, uuid.NewString())
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestFooStateField")
	defer ss.RunSteps(t)

	fooID := uuid.NewString()
	tenantID := uuid.NewString()

	ss.StepC("Create", func(ctx context.Context, a flowtest.Asserter) {
		event := newFooCreatedEvent(fooID, tenantID, nil)
		stateOut, err := sm.Transition(ctx, event)
		if err != nil {
			t.Fatal(err.Error())
		}
		a.Equal(testpb.FooStatus_ACTIVE, stateOut.Status)
		a.Equal(tenantID, *stateOut.TenantId)
	})

	ss.StepC("Update OK, Same Key", func(ctx context.Context, a flowtest.Asserter) {
		event := newFooUpdatedEvent(fooID, tenantID, func(u *testpb.FooEventType_Updated) {
			u.Weight = ptr.To(int64(11))
		})
		stateOut, err := sm.Transition(ctx, event)
		if err != nil {
			t.Fatal(err.Error())
		}
		a.Equal(testpb.FooStatus_ACTIVE, stateOut.Status)
		a.Equal(tenantID, *stateOut.TenantId)
	})

	ss.StepC("Update Not OK, Different key specified", func(ctx context.Context, a flowtest.Asserter) {
		differentTenantId := uuid.NewString()
		event := &testpb.FooEvent{
			Metadata: &testpb.Metadata{
				EventId:   uuid.NewString(),
				Timestamp: timestamppb.Now(),
				Actor: &testpb.Actor{
					ActorId: uuid.NewString(),
				},
			},
			FooId:    fooID,
			TenantId: &differentTenantId,
			Event: &testpb.FooEventType{
				Type: &testpb.FooEventType_Updated_{
					Updated: &testpb.FooEventType_Updated{
						Name:   "foo",
						Field:  "event3",
						Weight: ptr.To(int64(11)),
					},
				},
			},
		}
		_, err := sm.Transition(ctx, event)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestBarStateMachine(t *testing.T) {
	ctx := context.Background()

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewBarStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	barID := uuid.NewString()
	event := newBarCreatedEvent(barID, nil)

	stateOut, err := sm.Transition(ctx, event)
	if err != nil {
		t.Fatal(err.Error())
	}

	if stateOut.GetStatus() != testpb.BarStatus_ACTIVE {
		t.Fatalf("Expect state ACTIVE, got %s", stateOut.GetStatus().ShortString())
	}
}

func TestFooStateMachine(t *testing.T) {
	ctx := context.Background()

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	actorID := uuid.NewString()

	sm, err := NewFooStateMachine(db, actorID)
	if err != nil {
		t.Fatal(err.Error())
	}

	tenantID := uuid.NewString()

	fooID := uuid.NewString()
	event1 := newFooCreatedEvent(fooID, tenantID, nil)
	event2 := newFooUpdatedEvent(fooID, tenantID, nil)

	foo2ID := uuid.NewString()
	event3 := newFooCreatedEvent(foo2ID, tenantID, nil)
	event4 := newFooUpdatedEvent(foo2ID, tenantID, func(u *testpb.FooEventType_Updated) {
		u.Delete = true
	})

	statesOut := map[string]*testpb.FooState{}
	for _, event := range []*testpb.FooEvent{event1, event2, event3, event4} {
		stateOut, err := sm.Transition(ctx, event)
		if err != nil {
			t.Fatal(err.Error())
		}
		statesOut[event.FooId] = stateOut
	}

	if statesOut[fooID].GetStatus() != testpb.FooStatus_ACTIVE {
		t.Fatalf("Expect state ACTIVE, got %s", statesOut[fooID].GetStatus().ShortString())
	}

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Run("List", func(t *testing.T) {
		req := &testpb.ListFoosRequest{}
		res := &testpb.ListFoosResponse{}

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Foos) != 1 {
			t.Fatalf("expected 1 states, got %d", len(res.Foos))
		}
	})

	t.Run("Get1", func(t *testing.T) {
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

	t.Run("ListEvents1", func(t *testing.T) {
		req := &testpb.ListFooEventsRequest{
			FooId: fooID,
		}
		res := &testpb.ListFooEventsResponse{}

		err = queryer.ListEvents(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Events) != 2 {
			t.Fatalf("expected 2 events for foo 1, got %d", len(res.Events))
		}

		for idx, event := range res.Events {
			// events are returned in reverse order
			expect := uint64(len(res.Events) - idx - 1)
			if event.Metadata.Sequence != expect {
				t.Fatalf("expected sequence %d (idx %d), got %d", expect, idx, event.Metadata.Sequence)
			}
		}
	})

	t.Run("Get2", func(t *testing.T) {
		req := &testpb.GetFooRequest{
			FooId: foo2ID,
		}

		res := &testpb.GetFooResponse{}

		err = queryer.Get(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if !proto.Equal(res.State, statesOut[foo2ID]) {
			t.Fatalf("expected %v, got %v", statesOut[foo2ID], res.State)
		}

		if len(res.Events) != 3 {
			t.Fatalf("expected 3 events, got %d", len(res.Events))
		}
		t.Log(res.Events)

		derivedEvent := res.Events[2]
		if derivedEvent.Metadata.Actor.ActorId != actorID {
			t.Fatalf("expected derived event to have actor ID %s, got %s", actorID, derivedEvent.Metadata.Actor.ActorId)
		}
	})
}
