package example

import (
	"context"
	"testing"
	"time"

	"github.com/elgris/sqrl"
	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/gen/list/v1/psml_pb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewFooStateMachine(db *sqrlx.Wrapper) (*testpb.FooPSM, error) {

	sm, err := testpb.NewFooPSM(db)
	if err != nil {
		return nil, err
	}

	sm.From(testpb.FooStatus_UNSPECIFIED).
		Do(testpb.FooPSMFunc(func(
			ctx context.Context,
			tb testpb.FooPSMTransitionBaton,
			state *testpb.FooState,
			event *testpb.FooEventType_Created,
		) error {
			state.Status = testpb.FooStatus_ACTIVE
			state.Name = event.Name
			state.Field = event.Field
			state.LastEventId = tb.FullCause().Metadata.EventId
			state.CreatedAt = tb.FullCause().Metadata.Timestamp
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
			state.LastEventId = tb.FullCause().Metadata.EventId
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

	queryer, err := testpb.NewFooPSMQuerySet(sm.GetQuerySpec(), psm.StateQueryOptions{})
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

		if res.State.LastEventId != event2.Metadata.EventId {
			t.Fatalf("event ID not passed, want %s, got %s", event2.Metadata.EventId, res.State.LastEventId)
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

func silenceLogger() func() {
	defaultLogger := log.DefaultLogger
	log.DefaultLogger = log.NewCallbackLogger(func(level string, msg string, fields map[string]interface{}) {
	})
	return func() {
		log.DefaultLogger = defaultLogger
	}
}

func TestFooPagination(t *testing.T) {

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestFooStateField")
	defer ss.RunSteps(t)

	queryer, err := testpb.NewFooPSMQuerySet(sm.GetQuerySpec(), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	ss.StepC("Create", func(ctx context.Context, a flowtest.Asserter) {
		tenantID := uuid.NewString()

		restore := silenceLogger()
		defer restore()

		tt := time.Now()

		for ii := 0; ii < 30; ii++ {
			tt = tt.Add(time.Second)
			fooID := uuid.NewString()

			event1 := &testpb.FooEvent{
				Metadata: &testpb.Metadata{
					EventId:   uuid.NewString(),
					Timestamp: timestamppb.New(tt),
					Actor: &testpb.Actor{
						ActorId: uuid.NewString(),
					},
				},
				TenantId: &tenantID,
				FooId:    fooID,
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
			a.Equal(testpb.FooStatus_ACTIVE, stateOut.Status)
			a.Equal(tenantID, *stateOut.TenantId)
		}
	})

	var pageResp *psml_pb.PageResponse

	ss.StepC("List Page 1", func(ctx context.Context, t flowtest.Asserter) {

		req := &testpb.ListFoosRequest{}
		res := &testpb.ListFoosResponse{}

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foos) != 20 {
			t.Fatalf("expected 20 states, got %d", len(res.Foos))
		}

		pageResp = res.Page

		if pageResp.GetNextToken() == "" {
			t.Fatalf("NextToken should not be empty")
		}
		if pageResp.NextToken == nil {
			t.Fatalf("Should not be the final page")
		}

	})

	ss.StepC("List Page 2", func(ctx context.Context, t flowtest.Asserter) {

		req := &testpb.ListFoosRequest{
			Page: &psml_pb.PageRequest{
				Token: pageResp.NextToken,
			},
		}
		res := &testpb.ListFoosResponse{}

		query, err := queryer.MainLister.BuildQuery(ctx, req.ProtoReflect(), res.ProtoReflect())
		if err != nil {
			t.Fatal(err.Error())
		}
		printQuery(t, query)

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foos) != 10 {
			t.Fatalf("expected 10 states, got %d", len(res.Foos))
		}

	})

}

func printQuery(t flowtest.TB, query *sqrl.SelectBuilder) {
	stmt, args, err := query.ToSql()
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(stmt, args)
}

func TestFooStateField(t *testing.T) {

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestFooStateField")
	defer ss.RunSteps(t)

	fooID := uuid.NewString()
	tenantID := uuid.NewString()

	event1 := &testpb.FooEvent{
		Metadata: &testpb.Metadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
			Actor: &testpb.Actor{
				ActorId: uuid.NewString(),
			},
		},
		TenantId: &tenantID,
		FooId:    fooID,
		Event: &testpb.FooEventType{
			Type: &testpb.FooEventType_Created_{
				Created: &testpb.FooEventType_Created{
					Name:  "foo",
					Field: "event1",
				},
			},
		},
	}

	ss.StepC("Create", func(ctx context.Context, a flowtest.Asserter) {
		stateOut, err := sm.Transition(ctx, event1)
		if err != nil {
			t.Fatal(err.Error())
		}
		a.Equal(testpb.FooStatus_ACTIVE, stateOut.Status)
		a.Equal(tenantID, *stateOut.TenantId)
	})

	ss.StepC("Update OK, Same Key", func(ctx context.Context, a flowtest.Asserter) {
		event := &testpb.FooEvent{
			Metadata: &testpb.Metadata{
				EventId:   uuid.NewString(),
				Timestamp: timestamppb.Now(),
				Actor: &testpb.Actor{
					ActorId: uuid.NewString(),
				},
			},
			FooId:    fooID,
			TenantId: &tenantID,
			Event: &testpb.FooEventType{
				Type: &testpb.FooEventType_Updated_{
					Updated: &testpb.FooEventType_Updated{
						Name:  "foo",
						Field: "event2",
					},
				},
			},
		}
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
						Name:  "foo",
						Field: "event3",
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
