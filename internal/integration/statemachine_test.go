package integration

import (
	"context"
	"errors"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/internal/pgstore/pgmigrate"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"github.com/pressly/goose"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"k8s.io/utils/ptr"
)

func TestMermaidPrinter(t *testing.T) {
	fooSM, err := NewFooStateMachine(nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	out, err := fooSM.PrintMermaid()
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(out)

}

func TestStateEntityExtensions(t *testing.T) {
	event := &test_pb.FooEvent{}
	assert.Equal(t, test_pb.FooPSMEventNil, event.PSMEventKey())
	if err := event.SetPSMEvent(&test_pb.FooEventType_Created{}); err != nil {
		t.Fatal(err.Error())
	}
	assert.Equal(t, test_pb.FooPSMEventCreated, event.PSMEventKey())
}

func TestFooStateField(t *testing.T) {
	conn := pgtest.GetTestDB(t)

	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	barSpec, err := test_pb.BarPSMBuilder().BuildQueryTableSpec()
	if err != nil {
		t.Fatal(err.Error())
	}

	if err := pgmigrate.CreateStateMachines(context.Background(), conn,
		sm.StateTableSpec(),
		*barSpec,
	); err != nil {
		t.Fatal(err.Error())
	}

	if err := goose.Up(conn, stage1MigrationsDir); err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestFooStateField")
	defer ss.RunSteps(t)

	fooID := uuid.NewString()
	tenantID := uuid.NewString()

	ss.Step("Create", func(ctx context.Context, t flowtest.Asserter) {
		event := newFooCreatedEvent(fooID, tenantID, nil)
		stateOut, err := sm.Transition(ctx, event)
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Equal(test_pb.FooStatus_ACTIVE, stateOut.Status)
		t.Equal(tenantID, *stateOut.Keys.TenantId)
	})

	ss.Step("Update OK, Same Key", func(ctx context.Context, t flowtest.Asserter) {
		event := newFooUpdatedEvent(fooID, tenantID, func(u *test_pb.FooEventType_Updated) {
			u.Weight = ptr.To(int64(11))
		})
		stateOut, err := sm.Transition(ctx, event)
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Equal(test_pb.FooStatus_ACTIVE, stateOut.Status)
		t.Equal(tenantID, *stateOut.Keys.TenantId)
	})

	ss.Step("Update Not OK, Different key specified", func(ctx context.Context, t flowtest.Asserter) {
		differentTenantId := uuid.NewString()
		event := &test_pb.FooPSMEventSpec{
			EventID: uuid.NewString(),
			Keys: &test_pb.FooKeys{
				FooId:        fooID,
				TenantId:     &differentTenantId,
				MetaTenantId: metaTenant,
			},
			Event: &test_pb.FooEventType_Updated{
				Name:   "foo",
				Field:  "event3",
				Weight: ptr.To(int64(11)),
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

	conn := pgtest.GetTestDB(t, pgtest.WithDir(allMigrationsDir))
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

	if stateOut.GetStatus() != test_pb.BarStatus_ACTIVE {
		t.Fatalf("Expect state ACTIVE, got %s", stateOut.GetStatus().ShortString())
	}
}

func TestStateMachineHook(t *testing.T) {

	flow, uu := NewUniverse(t)
	defer flow.RunSteps(t)

	tenantID := uuid.NewString()
	foo1ID := uuid.NewString()

	flow.Step("Setup", func(ctx context.Context, t flowtest.Asserter) {
		event1 := newFooCreatedEvent(foo1ID, tenantID, nil)

		foo2ID := uuid.NewString()
		event3 := newFooCreatedEvent(foo2ID, tenantID, nil)

		for _, event := range []*test_pb.FooPSMEventSpec{event1, event3} {
			_, err := uu.FooStateMachine.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}
		}
	})

	flow.Step("Summary", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooSummaryRequest{}

		res, err := uu.FooQuery.FooSummary(ctx, req)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if res.CountFoos != 2 {
			t.Fatalf("expected 2 FOOs, got %d", res.CountFoos)
		}
	})

	flow.Step("Mutate", func(ctx context.Context, t flowtest.Asserter) {
		updatedEvent := newFooUpdatedEvent(foo1ID, tenantID, func(u *test_pb.FooEventType_Updated) {
			u.Name = "updated"
			u.Delete = true
		})

		ss, err := uu.FooStateMachine.Transition(ctx, updatedEvent)
		if err != nil {
			t.Fatal(err.Error())
		}

		// Returns the state after the initial transition
		t.Equal(test_pb.FooStatus_ACTIVE, ss.Status)
		t.Equal("updated", ss.Data.Name)

		// The delete=true flag causes a chain event, which sets the status to
		// DELETED

		res2, err := uu.FooQuery.FooGet(ctx, &test_spb.FooGetRequest{
			FooId: foo1ID,
		})
		t.NoError(err)
		t.Equal(test_pb.FooStatus_DELETED, res2.Foo.Status)

	})

	flow.Step("Hook should have pushed to Bar", func(ctx context.Context, t flowtest.Asserter) {

		res, err := uu.BarQuery.BarList(ctx, &test_spb.BarListRequest{})
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))

		if len(res.Bar) != 1 {
			t.Fatalf("expected 1 BAR, got %d", len(res.Bar))
		}

		bar := res.Bar[0]

		if bar.Data.Name != "updated Phoenix" {
			t.Fatalf("expected name updated, got %s", bar.Data.Name)
		}

	})

}

func TestStateMachineIdempotencyInitial(t *testing.T) {
	flow, uu := NewUniverse(t)
	defer flow.RunSteps(t)

	tenantID := uuid.NewString()
	fooID := uuid.NewString()
	event1 := newFooCreatedEvent(fooID, tenantID, nil)

	flow.Step("Create", func(ctx context.Context, t flowtest.Asserter) {

		state, err := uu.FooStateMachine.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}

		if state.GetStatus() != test_pb.FooStatus_ACTIVE {
			t.Fatalf("expected state ACTIVE, got %s", state.GetStatus().ShortString())
		}

	})

	flow.Step("Same Exact Event", func(ctx context.Context, t flowtest.Asserter) {
		// idempotency test
		// event 1 should be idempotent
		state, err := uu.FooStateMachine.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}

		if state.GetStatus() != test_pb.FooStatus_ACTIVE {
			t.Fatalf("expected state ACTIVE, got %s", state.GetStatus().ShortString())
		}

		req := &test_spb.FooEventsRequest{
			FooId: fooID,
		}

		res, err := uu.FooQuery.FooEvents(ctx, req)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Events) != 1 {
			t.Fatalf("expected 1 events, got %d", len(res.Events))
		}

	})

	flow.Step("Different Event Data", func(ctx context.Context, t flowtest.Asserter) {
		// idempotency test
		// event 1 should be idempotent
		event1.Event.(*test_pb.FooEventType_Created).Name = "foo2"
		_, err := uu.FooStateMachine.Transition(ctx, event1)
		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, psm.ErrDuplicateEventID) {
			t.Fatalf("expected duplicate event ID, got %v", err)
		}

	})
}

func TestStateMachineIdempotencySnapshot(t *testing.T) {
	flow, uu := NewUniverse(t)
	defer flow.RunSteps(t)

	tenantID := uuid.NewString()
	fooID := uuid.NewString()
	event1 := newFooCreatedEvent(fooID, tenantID, func(c *test_pb.FooEventType_Created) {
		c.Name = "1"
	})

	flow.Step("Create", func(ctx context.Context, t flowtest.Asserter) {
		state, err := uu.FooStateMachine.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}

		if state.GetStatus() != test_pb.FooStatus_ACTIVE {
			t.Fatalf("expected state ACTIVE, got %s", state.GetStatus().ShortString())
		}

		if state.Data.Name != "1" {
			t.Fatalf("expected state name 1, got %s", state.Data.Name)
		}

	})

	flow.Step("Update", func(ctx context.Context, t flowtest.Asserter) {
		state, err := uu.FooStateMachine.Transition(ctx, newFooUpdatedEvent(fooID, tenantID, func(u *test_pb.FooEventType_Updated) {
			u.Name = "2"
		}))
		if err != nil {
			t.Fatal(err.Error())
		}

		if state.Data.Name != "2" {
			t.Fatalf("expected state name 2, got %s", state.Data.Name)
		}
	})

	flow.Step("Repeat Create Event", func(ctx context.Context, t flowtest.Asserter) {
		state, err := uu.FooStateMachine.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}

		// Should return the state after the initial Create, i.e. name = 1
		if state.Data.Name != "1" {
			t.Fatalf("expected state name 1, got %s", state.Data.Name)
		}

	})

}

func TestFooStateMachine(t *testing.T) {
	flow, uu := NewUniverse(t)
	defer flow.RunSteps(t)

	tenantID := uuid.NewString()

	fooID := uuid.NewString()
	event1 := newFooCreatedEvent(fooID, tenantID, nil)
	event2 := newFooUpdatedEvent(fooID, tenantID, nil)

	foo2ID := uuid.NewString()
	event3 := newFooCreatedEvent(foo2ID, tenantID, nil)
	event4 := newFooUpdatedEvent(foo2ID, tenantID, func(u *test_pb.FooEventType_Updated) {
		u.Delete = true
	})
	statesOut := map[string]*test_pb.FooState{}

	flow.Setup(func(ctx context.Context, t flowtest.Asserter) error {
		for _, event := range []*test_pb.FooPSMEventSpec{event1, event2, event3, event4} {
			stateOut, err := uu.FooStateMachine.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}
			statesOut[event.Keys.FooId] = stateOut
		}

		if statesOut[fooID].GetStatus() != test_pb.FooStatus_ACTIVE {
			t.Fatalf("Expect state ACTIVE, got %s", statesOut[fooID].GetStatus().ShortString())
		}
		return nil
	})

	flow.Step("Get1", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooGetRequest{
			FooId: fooID,
		}

		res, err := uu.FooQuery.FooGet(ctx, req)
		if err != nil {
			t.Fatal(err.Error())
		}

		if !proto.Equal(res.Foo, statesOut[fooID]) {
			t.Fatalf("expected %v, got %v", statesOut[fooID], res.Foo)
		}

		if len(res.Events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(res.Events))
		}
		t.Log(res.Events)
	})

	flow.Step("ListEvents1", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooEventsRequest{
			FooId: fooID,
		}

		res, err := uu.FooQuery.FooEvents(ctx, req)
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

	flow.Step("Get2", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooGetRequest{
			FooId: foo2ID,
		}

		res, err := uu.FooQuery.FooGet(ctx, req)
		if err != nil {
			t.Fatal(err.Error())
		}

		if res.Foo.Status != test_pb.FooStatus_DELETED {
			t.Fatalf("expected state DELETED, got %s - Did the chain run?", res.Foo.Status.ShortString())
		}

		if len(res.Events) != 3 {
			t.Fatalf("expected 3 events, got %d", len(res.Events))
		}
		t.Log(res.Events)

		derivedEvent := res.Events[2]
		if derivedEvent.Metadata == nil {
			t.Fatalf("expected derived event to have metadata")
		}
		if derivedEvent.Metadata.Cause == nil {
			t.Fatalf("expected derived event to have cause")
		}
		t.Log(protojson.Format(derivedEvent.Metadata.Cause))
		causeEvent := derivedEvent.Metadata.Cause.GetPsmEvent()

		if causeEvent == nil {
			t.Fatalf("expected derived event to be caused by a PSM event")
			return
		}
		if causeEvent.EventId != event4.EventID {
			t.Fatalf("expected derived event to be caused by event 4")
		}
	})

	flow.Step("List", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooListRequest{}

		res, err := uu.FooQuery.FooList(ctx, req)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Foo) != 1 {
			t.Fatalf("expected 1 states for default filter (ACTIVE), got %d", len(res.Foo))
		}
	})

}
