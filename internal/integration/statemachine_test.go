package integration

import (
	"context"
	"errors"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/flowtest/be"
	"github.com/pentops/golib/gl"
	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/j5/j5types/date_j5t"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"k8s.io/utils/ptr"
)

func TestStateEntityExtensions(t *testing.T) {
	event := &test_pb.FooEvent{}
	assert.Equal(t, test_pb.FooPSMEventNil, event.PSMEventKey())
	if err := event.SetPSMEvent(&test_pb.FooEventType_Created{}); err != nil {
		t.Fatal(err.Error())
	}
	assert.Equal(t, test_pb.FooPSMEventCreated, event.PSMEventKey())
}

func TestKeyMismatch(t *testing.T) {
	ss, uu := NewUniverse(t)
	defer ss.RunSteps(t)

	fooID := uuid.NewString()
	tenantID := uuid.NewString()

	ss.Step("Create", func(ctx context.Context, t flowtest.Asserter) {
		event := &test_pb.FooPSMEventSpec{
			Keys: &test_pb.FooKeys{
				FooId:        fooID,
				TenantId:     &tenantID,
				MetaTenantId: metaTenant,
			},
			Event: &test_pb.FooEventType_Created{
				Name:   "foo",
				Field:  "event3",
				Weight: ptr.To(int64(10)),
			},
			Cause: testCause(),
		}
		stateOut, err := uu.FooStateMachine.Transition(ctx, event)
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Equal(test_pb.FooStatus_ACTIVE, stateOut.Status)
		t.Equal(tenantID, *stateOut.Keys.TenantId)
	})

	ss.Step("Update OK, Same Key", func(ctx context.Context, t flowtest.Asserter) {
		event := &test_pb.FooPSMEventSpec{
			Keys: &test_pb.FooKeys{
				FooId:        fooID,
				TenantId:     &tenantID,
				MetaTenantId: metaTenant,
			},
			Event: &test_pb.FooEventType_Updated{
				Name:   "foo",
				Field:  "event3",
				Weight: ptr.To(int64(11)),
			},
			Cause: testCause(),
		}
		stateOut, err := uu.FooStateMachine.Transition(ctx, event)
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Equal(test_pb.FooStatus_ACTIVE, stateOut.Status)
		t.Equal(tenantID, *stateOut.Keys.TenantId)
	})

	ss.Step("Update OK, Partial Key", func(ctx context.Context, t flowtest.Asserter) {
		event := &test_pb.FooPSMEventSpec{
			Keys: &test_pb.FooKeys{
				FooId:        fooID,
				TenantId:     nil,
				MetaTenantId: metaTenant,
			},
			Event: &test_pb.FooEventType_Updated{
				Name:   "foo",
				Field:  "event3",
				Weight: gl.Ptr(int64(12)),
			},
			Cause: testCause(),
		}
		stateOut, err := uu.FooStateMachine.Transition(ctx, event)
		if err != nil {
			t.Fatal(err.Error())
		}
		be.Equal(test_pb.FooStatus_ACTIVE, stateOut.Status).Run(t, "Status")
		be.Equal(tenantID, *stateOut.Keys.TenantId).Run(t, "TenantId")
		be.Equal(12, stateOut.Data.Characteristics.Weight).Run(t, "Weight")
	})

	ss.Step("Update Not OK, Different key specified", func(ctx context.Context, t flowtest.Asserter) {
		// that it's the TenantID is not important to this test, it's just a key.
		differentTenantId := uuid.NewString()
		event := &test_pb.FooPSMEventSpec{
			Keys: &test_pb.FooKeys{
				FooId:        fooID,
				TenantId:     &differentTenantId,
				MetaTenantId: metaTenant,
			},
			Event: &test_pb.FooEventType_Updated{
				Name:   "foo",
				Field:  "event3",
				Weight: ptr.To(int64(12)),
			},
			Cause: testCause(),
		}
		_, err := uu.FooStateMachine.Transition(ctx, event)
		t.CodeError(err, codes.FailedPrecondition)
	})
}

func TestChain(t *testing.T) {
	flow, uu := NewUniverse(t)
	defer flow.RunSteps(t)

	flow.Setup(func(ctx context.Context, t flowtest.Asserter) error {
		uu.FooStateMachine.From().
			Hook(test_pb.FooPSMLogicHook(func(
				ctx context.Context,
				baton test_pb.FooPSMHookBaton,
				state *test_pb.FooState,
				event *test_pb.FooEventType_Updated,
			) error {
				log.WithFields(ctx, map[string]any{
					"foo_id": state.Keys.FooId,
					"event":  event,
				}).Info("Foo Update Hook")
				if event.Delete {
					baton.ChainEvent(&test_pb.FooEventType_Deleted{})
				}
				return nil
			}))
		return nil
	})

	tenantID := uuid.NewString()
	foo1ID := uuid.NewString()

	flow.Step("Setup", func(ctx context.Context, t flowtest.Asserter) {
		event1 := newFooCreatedEvent(foo1ID, tenantID)
		_, err := uu.FooStateMachine.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
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

		// The delete=true flag causes a chain event.
		// The new status is DELETED
		// That status should not be returned by the initial mutation

		// State after the initial transition
		t.Equal(test_pb.FooStatus_ACTIVE, ss.Status)
		t.Equal("updated", ss.Data.Name)

		// State actually stored after the chain is processed
		res2, err := uu.FooQuery.FooGet(ctx, &test_spb.FooGetRequest{
			FooId: foo1ID,
		})
		t.NoError(err)
		t.Equal(test_pb.FooStatus_DELETED, res2.Foo.Status)

	})

	flow.Step("List Events", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooEventsRequest{
			FooId: foo1ID,
		}

		res, err := uu.FooQuery.FooEvents(ctx, req)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Events) != 3 {
			t.Fatalf("expected 3 events for foo 2, got %d", len(res.Events))
		}
		createEvent := res.Events[2]
		updateEvent := res.Events[1]
		deleteEvent := res.Events[0]

		be.Equal(test_pb.FooPSMEventCreated, createEvent.PSMEventKey()).Run(t, "Create Event")
		be.Equal(test_pb.FooPSMEventUpdated, updateEvent.PSMEventKey()).Run(t, "Update Event")
		be.Equal(test_pb.FooPSMEventDeleted, deleteEvent.PSMEventKey()).Run(t, "Delete Event")

		if deleteEvent.Metadata == nil {
			t.Fatalf("expected derived event to have metadata")
		}
		if deleteEvent.Metadata.Cause == nil {
			t.Fatalf("expected derived event to have cause")
		}
		t.Log(protojson.Format(deleteEvent.Metadata.Cause))
		causeEvent := deleteEvent.Metadata.Cause.GetPsmEvent()

		if causeEvent == nil {
			t.Fatalf("expected derived event to be caused by a PSM event")
			return
		}
		if causeEvent.EventId != updateEvent.Metadata.EventId {
			t.Log(prototext.Format(causeEvent))
			t.Fatalf("expected derived event to be caused by event 4")
		}
	})
}

func TestLink(t *testing.T) {

	flow, uu := NewUniverse(t)
	defer flow.RunSteps(t)

	flow.Setup(func(ctx context.Context, t flowtest.Asserter) error {

		uu.FooStateMachine.From(test_pb.FooStatus_ACTIVE).
			OnEvent(test_pb.FooPSMEventUpdated).
			LinkTo(test_pb.FooPSMLinkHook(uu.BarStateMachine, func(
				ctx context.Context,
				state *test_pb.FooState,
				event *test_pb.FooEventType_Updated,
				cb func(*test_pb.BarKeys, test_pb.BarPSMEvent),
			) error {
				cb(&test_pb.BarKeys{
					BarId:      uuid.NewString(),
					BarOtherId: state.Keys.FooId,
					DateKey:    date_j5t.NewDate(2020, 1, 1),
				}, &test_pb.BarEventType_Created{
					Name:  state.Data.Name + " Phoenix",
					Field: state.Data.Field,
				})
				return nil
			}))
		return nil
	})

	tenantID := uuid.NewString()
	foo1ID := uuid.NewString()

	flow.Step("Setup", func(ctx context.Context, t flowtest.Asserter) {
		event1 := newFooCreatedEvent(foo1ID, tenantID)
		_, err := uu.FooStateMachine.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}
	})

	flow.Step("Mutate", func(ctx context.Context, t flowtest.Asserter) {
		updatedEvent := newFooUpdatedEvent(foo1ID, tenantID, func(u *test_pb.FooEventType_Updated) {
			u.Name = "updated"
		})
		_, err := uu.FooStateMachine.Transition(ctx, updatedEvent)
		if err != nil {
			t.Fatal(err.Error())
		}
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

func TestStateDataHook(t *testing.T) {
	flow, uu := NewUniverse(t)
	defer flow.RunSteps(t)

	flow.Setup(func(ctx context.Context, t flowtest.Asserter) error {

		uu.FooStateMachine.StateDataHook(test_pb.FooPSMGeneralStateDataHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			state *test_pb.FooState,
		) error {

			if state.Data.Characteristics == nil || state.Status != test_pb.FooStatus_ACTIVE {
				_, err := tx.Delete(ctx, sq.Delete("foo_cache").Where("id = ?", state.Keys.FooId))
				if err != nil {
					return err
				}
				return nil
			}

			_, err := tx.Exec(ctx, sqrlx.Upsert("foo_cache").
				Key("id", state.Keys.FooId).
				Set("weight", state.Data.Characteristics.Weight).
				Set("height", state.Data.Characteristics.Height).
				Set("length", state.Data.Characteristics.Length))
			if err != nil {
				return err
			}

			return nil
		}))

		return nil
	})

	tenantID := uuid.NewString()
	foo1ID := uuid.NewString()

	flow.Step("Setup", func(ctx context.Context, t flowtest.Asserter) {
		event1 := newFooCreatedEvent(foo1ID, tenantID, func(c *test_pb.FooEventType_Created) {
			c.Weight = gl.Ptr(int64(100))
		})
		_, err := uu.FooStateMachine.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}
	})

	flow.Step("Query Cache", func(ctx context.Context, t flowtest.Asserter) {
		summary, err := uu.FooQuery.FooSummary(ctx, &test_spb.FooSummaryRequest{})
		t.NoError(err)
		be.Equal(1, summary.CountFoos).Run(t, "Foo Count")
		be.Equal(100, summary.TotalWeight).Run(t, "Weight")
	})
}

func TestTransitionDataHook(t *testing.T) {
	flow, uu := NewUniverse(t)
	defer flow.RunSteps(t)

	flow.Setup(func(ctx context.Context, t flowtest.Asserter) error {

		uu.FooStateMachine.From().
			OnEvent(test_pb.FooPSMEventCreated).
			Hook(test_pb.FooPSMDataHook(func(
				ctx context.Context,
				tx sqrlx.Transaction,
				state *test_pb.FooState,
				event *test_pb.FooEventType_Created,
			) error {

				if state.Data.Characteristics == nil || state.Status != test_pb.FooStatus_ACTIVE {
					_, err := tx.Delete(ctx, sq.Delete("foo_cache").Where("id = ?", state.Keys.FooId))
					if err != nil {
						return err
					}
					return nil
				}

				_, err := tx.Exec(ctx, sqrlx.Upsert("foo_cache").
					Key("id", state.Keys.FooId).
					Set("weight", state.Data.Characteristics.Weight).
					Set("height", state.Data.Characteristics.Height).
					Set("length", state.Data.Characteristics.Length))
				if err != nil {
					return err
				}

				return nil
			}))

		return nil
	})

	tenantID := uuid.NewString()
	foo1ID := uuid.NewString()

	flow.Step("Setup", func(ctx context.Context, t flowtest.Asserter) {
		event1 := newFooCreatedEvent(foo1ID, tenantID, func(c *test_pb.FooEventType_Created) {
			c.Weight = gl.Ptr(int64(100))
		})
		_, err := uu.FooStateMachine.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}
	})

	flow.Step("Query Cache", func(ctx context.Context, t flowtest.Asserter) {
		summary, err := uu.FooQuery.FooSummary(ctx, &test_spb.FooSummaryRequest{})
		t.NoError(err)
		be.Equal(1, summary.CountFoos).Run(t, "Foo Count")
		be.Equal(100, summary.TotalWeight).Run(t, "Weight")
	})
}

func TestStateMachineIdempotencyInitial(t *testing.T) {
	flow, uu := NewUniverse(t)
	defer flow.RunSteps(t)

	tenantID := uuid.NewString()
	fooID := uuid.NewString()
	event1 := newFooCreatedEvent(fooID, tenantID)
	event1.Cause = &psm_j5pb.Cause{
		Type: &psm_j5pb.Cause_ExternalEvent{
			ExternalEvent: &psm_j5pb.ExternalEventCause{
				SystemName: "test",
				ExternalId: gl.Ptr("external-id"),
			},
		},
	}

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

	event1.Cause = &psm_j5pb.Cause{
		Type: &psm_j5pb.Cause_ExternalEvent{
			ExternalEvent: &psm_j5pb.ExternalEventCause{
				SystemName: "test",
				ExternalId: gl.Ptr("external-id"),
			},
		},
	}

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
	event1 := newFooCreatedEvent(fooID, tenantID)
	event2 := newFooUpdatedEvent(fooID, tenantID)

	foo2ID := uuid.NewString()
	event3 := newFooCreatedEvent(foo2ID, tenantID)
	event4 := newFooUpdatedEvent(foo2ID, tenantID)
	event5 := newFooDeletedEvent(foo2ID, tenantID)
	statesOut := map[string]*test_pb.FooState{}

	flow.Step("List Empty", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooListRequest{}

		res, err := uu.FooQuery.FooList(ctx, req)
		if err != nil {
			t.Fatal(err.Error())
		}
		/*
			if res.Foo == nil {
				t.Fatal("expected non-nil response")
			}
		*/
		t.Log(protojson.Format(res))
		if len(res.Foo) != 0 {
			t.Fatalf("expected 0 states")
		}
	})

	flow.Step("Add Events", func(ctx context.Context, t flowtest.Asserter) {
		for _, event := range []*test_pb.FooPSMEventSpec{event1, event2, event3, event4, event5} {
			stateOut, err := uu.FooStateMachine.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}
			statesOut[event.Keys.FooId] = stateOut
		}

		if statesOut[fooID].GetStatus() != test_pb.FooStatus_ACTIVE {
			t.Fatalf("Expect state ACTIVE, got %s", statesOut[fooID].GetStatus().ShortString())
		}
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

	flow.Step("ListEvents2", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooEventsRequest{
			FooId: foo2ID,
		}

		res, err := uu.FooQuery.FooEvents(ctx, req)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Events) != 3 {
			t.Fatalf("expected 3 events for foo 2, got %d", len(res.Events))
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
