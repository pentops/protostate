package integration

import (
	"context"
	"errors"
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

func TestStateMachineHook(t *testing.T) {
	ctx := context.Background()

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm := NewFooTestMachine(t, db)

	controller := NewMiniFooController(db)

	sm.AddHook(testpb.FooPSMGeneralHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		state *testpb.FooState,
		event *testpb.FooEvent,
	) error {
		if state.Characteristics == nil || state.Status != testpb.FooStatus_ACTIVE {
			_, err := tx.Delete(ctx, sq.Delete("foo_cache").Where("id = ?", state.FooId))
			if err != nil {
				return err
			}
			return nil
		}

		_, err := tx.Exec(ctx, sqrlx.Upsert("foo_cache").Key("id", state.FooId).
			Set("weight", state.Characteristics.Weight).
			Set("height", state.Characteristics.Height).
			Set("length", state.Characteristics.Length))
		if err != nil {
			return err
		}

		return nil
	}))

	tenantID := uuid.NewString()
	t.Run("Setup", func(t *testing.T) {
		foo1ID := uuid.NewString()
		event1 := newFooCreatedEvent(foo1ID, tenantID, nil)

		foo2ID := uuid.NewString()
		event3 := newFooCreatedEvent(foo2ID, tenantID, nil)

		for _, event := range []*testpb.FooEvent{event1, event3} {
			_, err := sm.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}
		}

	})

	t.Run("Summary", func(t *testing.T) {
		req := &testpb.FooSummaryRequest{}

		res, err := controller.FooSummary(ctx, req)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if res.CountFoos != 2 {
			t.Fatalf("expected 2 FOOs, got %d", res.CountFoos)
		}
	})
}

func TestStateMachineIdempotencyInitial(t *testing.T) {
	ctx := context.Background()

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm := NewFooTestMachine(t, db)

	tenantID := uuid.NewString()
	fooID := uuid.NewString()
	event1 := newFooCreatedEvent(fooID, tenantID, nil)

	t.Run("Create", func(t *testing.T) {
		state, err := sm.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}

		if state.GetStatus() != testpb.FooStatus_ACTIVE {
			t.Fatalf("expected state ACTIVE, got %s", state.GetStatus().ShortString())
		}

	})

	t.Run("Same Exact Event", func(t *testing.T) {
		// idempotency test
		// event 1 should be idempotent
		state, err := sm.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}

		if state.GetStatus() != testpb.FooStatus_ACTIVE {
			t.Fatalf("expected state ACTIVE, got %s", state.GetStatus().ShortString())
		}

		req := &testpb.ListFooEventsRequest{
			FooId: fooID,
		}
		res := &testpb.ListFooEventsResponse{}

		err = sm.Queryer.ListEvents(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Events) != 1 {
			t.Fatalf("expected 1 events, got %d", len(res.Events))
		}

	})

	t.Run("Different Event Data", func(t *testing.T) {
		// idempotency test
		// event 1 should be idempotent
		event1.Event.Type.(*testpb.FooEventType_Created_).Created.Name = "foo2"
		_, err = sm.Transition(ctx, event1)
		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, psm.ErrDuplicateEventID) {
			t.Fatalf("expected duplicate event ID, got %v", err)
		}

	})
}

func TestStateMachineIdempotencySnapshot(t *testing.T) {
	ctx := context.Background()

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm := NewFooTestMachine(t, db)

	tenantID := uuid.NewString()
	fooID := uuid.NewString()
	event1 := newFooCreatedEvent(fooID, tenantID, func(c *testpb.FooEventType_Created) {
		c.Name = "1"
	})

	t.Run("Create", func(t *testing.T) {
		state, err := sm.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}

		if state.GetStatus() != testpb.FooStatus_ACTIVE {
			t.Fatalf("expected state ACTIVE, got %s", state.GetStatus().ShortString())
		}

		if state.Name != "1" {
			t.Fatalf("expected state name 1, got %s", state.Name)
		}

	})

	t.Run("Update", func(t *testing.T) {
		state, err := sm.Transition(ctx, newFooUpdatedEvent(fooID, tenantID, func(u *testpb.FooEventType_Updated) {
			u.Name = "2"
		}))
		if err != nil {
			t.Fatal(err.Error())
		}

		if state.Name != "2" {
			t.Fatalf("expected state name 2, got %s", state.Name)
		}
	})

	t.Run("Repeat Create Event", func(t *testing.T) {
		state, err := sm.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}

		// Should return the state after the initial Create, i.e. name = 1
		if state.Name != "1" {
			t.Fatalf("expected state name 1, got %s", state.Name)
		}

	})

}

// Test that when a hook returns a duplicated event, the state machine will
// return an error.
func TestStateMachineIdempotencyChained(t *testing.T) {

	ctx := context.Background()

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm := NewFooTestMachine(t, db)

	tenantID := uuid.NewString()
	fooID := uuid.NewString()
	event1 := newFooCreatedEvent(fooID, tenantID, nil)

	// set up a hook which returns this event, the tests can then play with it.
	var autoHookEvent *testpb.FooEvent
	sm.From().Hook(testpb.FooPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		baton testpb.FooPSMHookBaton,
		state *testpb.FooState,
		event *testpb.FooEventType_Updated,
	) error {
		if autoHookEvent == nil {
			return nil
		}
		newEvent := proto.Clone(autoHookEvent).(*testpb.FooEvent)
		autoHookEvent = nil
		baton.ChainEvent(newEvent)
		return nil
	}))

	t.Run("Create", func(t *testing.T) {
		_, err = sm.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}
	})

	updateTo2 := newFooUpdatedEvent(fooID, tenantID, func(u *testpb.FooEventType_Updated) {
		u.Name = "2"
	})

	t.Run("Update, with auto hook", func(t *testing.T) {
		autoHookEvent = updateTo2

		state, err := sm.Transition(ctx, newFooUpdatedEvent(fooID, tenantID, func(u *testpb.FooEventType_Updated) {
			u.Name = "1"
		}))
		if err != nil {
			t.Fatal(err.Error())
		}

		// The return from the first transition should be "1"
		if state.Name != "1" {
			t.Fatalf("expected state name 1, got %s", state.Name)
		}

		// But the state of foo should be "2" from the hook
		sm.AssertFooName(t, fooID, "2")
	})

	t.Run("Update, a new event, but same hook", func(t *testing.T) {
		autoHookEvent = updateTo2 // Same as it was before

		// a brand new event
		_, err := sm.Transition(ctx, newFooUpdatedEvent(fooID, tenantID, func(u *testpb.FooEventType_Updated) {
			u.Name = "3"
		}))
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, psm.ErrDuplicateChainedEventID) {
			t.Fatalf("expected duplicate chained event ID, got %v", err)
		}

		// But the state of foo should still be "2" from the earlier hook
		sm.AssertFooName(t, fooID, "2")
	})

}

func TestFooStateMachine(t *testing.T) {
	ctx := context.Background()

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm := NewFooTestMachine(t, db)

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

	t.Run("Get1", func(t *testing.T) {
		req := &testpb.GetFooRequest{
			FooId: fooID,
		}

		res := &testpb.GetFooResponse{}

		err = sm.Queryer.Get(ctx, db, req, res)
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

		err = sm.Queryer.ListEvents(ctx, db, req, res)
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

		err = sm.Queryer.Get(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if res.State.Status != testpb.FooStatus_DELETED {
			t.Fatalf("expected state DELETED, got %s - Did the chain run?", res.State.Status.ShortString())
		}

		if len(res.Events) != 3 {
			t.Fatalf("expected 3 events, got %d", len(res.Events))
		}
		t.Log(res.Events)

		derivedEvent := res.Events[2]
		if derivedEvent.Metadata.Actor.ActorId != sm.ActorID {
			t.Fatalf("expected derived event to have actor ID %s, got %s", sm.ActorID, derivedEvent.Metadata.Actor.ActorId)
		}
	})

	t.Run("List", func(t *testing.T) {
		req := &testpb.ListFoosRequest{}
		res := &testpb.ListFoosResponse{}

		err = sm.Queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Foos) != 1 {
			t.Fatalf("expected 1 states for default filter (ACTIVE), got %d", len(res.Foos))
		}
	})

}

type MiniFooController struct {
	db *sqrlx.Wrapper
}

func NewMiniFooController(db *sqrlx.Wrapper) *MiniFooController {
	return &MiniFooController{
		db: db,
	}
}

func (c *MiniFooController) FooSummary(ctx context.Context, req *testpb.FooSummaryRequest) (*testpb.FooSummaryResponse, error) {

	res := &testpb.FooSummaryResponse{}

	query := sq.Select("count(id)", "sum(weight)", "sum(height)", "sum(length)").From("foo_cache")
	err := c.db.Transact(ctx, nil, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.QueryRow(ctx, query).Scan(&res.CountFoos, &res.TotalWeight, &res.TotalHeight, &res.TotalLength)
	})
	if err != nil {
		return nil, err
	}
	return res, nil

}
