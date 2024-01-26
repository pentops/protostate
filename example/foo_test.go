package example

import (
	"context"
	"encoding/base64"
	"fmt"
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
	"k8s.io/utils/ptr"
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
			state.Description = event.Description
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
			state.Description = event.Description
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
					Name:        "foo",
					Field:       "event1",
					Description: ptr.To("event1"),
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
					Name:        "foo",
					Field:       "event2",
					Description: ptr.To("event2"),
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
					Name:        "foo2",
					Field:       "event3",
					Description: ptr.To("event3"),
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
		if len(res.Foos) != 2 {
			t.Fatalf("expected 2 states, got %d", len(res.Foos))
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
	})
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

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	ss.StepC("Create", func(ctx context.Context, a flowtest.Asserter) {
		tenantID := uuid.NewString()

		restore := silenceLogger()
		defer restore()

		for ii := 0; ii < 30; ii++ {
			tt := time.Now()
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
							Field: fmt.Sprintf("foo %d at %s", ii, tt.Format(time.RFC3339Nano)),
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

		for ii, state := range res.Foos {
			t.Logf("%d: %s", ii, state.Field)
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

		for ii, state := range res.Foos {
			t.Logf("%d: %s", ii, state.Field)
		}

		if len(res.Foos) != 10 {
			t.Fatalf("expected 10 states, got %d", len(res.Foos))
		}

	})
}

func TestFooEventPagination(t *testing.T) {
	// Foo event default sort is deeply nested. This tests that the nested filter
	// works on pagination

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T](t.Name())
	defer ss.RunSteps(t)

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	fooID := uuid.NewString()
	ss.StepC("CreateEvents", func(ctx context.Context, a flowtest.Asserter) {
		tenantID := uuid.NewString()

		restore := silenceLogger()
		defer restore()

		foo1Create := &testpb.FooEvent{
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
						Name: "foo",
					},
				},
			},
		}

		_, err := sm.Transition(ctx, foo1Create)
		if err != nil {
			t.Fatal(err.Error())
		}

		for ii := 0; ii < 30; ii++ {
			tt := time.Now()
			event := &testpb.FooEvent{
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
					Type: &testpb.FooEventType_Updated_{
						Updated: &testpb.FooEventType_Updated{
							Name:  "foo",
							Field: fmt.Sprintf("foo %d at %s", ii, tt.Format(time.RFC3339Nano)),
						},
					},
				},
			}
			t.Logf("Entering foo %d TS: %d, ID: %s", ii, event.Metadata.Timestamp.AsTime().Round(time.Microsecond).UnixMicro(), event.Metadata.EventId)
			_, err := sm.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}
		}
	})

	var pageResp *psml_pb.PageResponse

	ss.StepC("List Page 1", func(ctx context.Context, t flowtest.Asserter) {

		req := &testpb.ListFooEventsRequest{
			FooId: fooID,
		}
		res := &testpb.ListFooEventsResponse{}

		err = queryer.ListEvents(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Events) != 20 {
			t.Fatalf("expected 20 states, got %d", len(res.Events))
		}

		for ii, evt := range res.Events {
			tsv := evt.Metadata.Timestamp.AsTime().Round(time.Microsecond).UnixMicro()
			switch et := evt.Event.Type.(type) {
			case *testpb.FooEventType_Created_:
				t.Logf("%03d: Create %s - %d", ii, et.Created.Field, tsv)
			case *testpb.FooEventType_Updated_:
				t.Logf("%03d: Update %s - %d", ii, et.Updated.Field, tsv)
			default:
				t.Fatalf("unexpected event type %T", et)
			}
		}

		pageResp = res.Page

		if pageResp.GetNextToken() == "" {
			t.Fatalf("NextToken should not be empty")
		}
		if pageResp.NextToken == nil {
			t.Fatalf("Should not be the final page")
		}

		rowBytes, err := base64.StdEncoding.DecodeString(*pageResp.NextToken)
		if err != nil {
			t.Fatal(err.Error())
		}

		msg := &testpb.FooEvent{}
		if err := proto.Unmarshal(rowBytes, msg); err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(msg))
		t.Logf("Token entry, TS: %d, ID: %s", msg.Metadata.Timestamp.AsTime().Round(time.Microsecond).UnixMicro(), msg.Metadata.EventId)

	})

	ss.StepC("List Page 2", func(ctx context.Context, t flowtest.Asserter) {

		req := &testpb.ListFooEventsRequest{
			Page: &psml_pb.PageRequest{
				Token: pageResp.NextToken,
			},
			FooId: fooID,
		}
		res := &testpb.ListFooEventsResponse{}

		query, err := queryer.EventLister.BuildQuery(ctx, req.ProtoReflect(), res.ProtoReflect())
		if err != nil {
			t.Fatal(err.Error())
		}
		printQuery(t, query)

		err = queryer.ListEvents(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		for ii, evt := range res.Events {
			switch et := evt.Event.Type.(type) {
			case *testpb.FooEventType_Created_:
				t.Logf("%d: Create %s", ii, et.Created.Field)
			case *testpb.FooEventType_Updated_:
				t.Logf("%d: Update %s", ii, et.Updated.Field)
			default:
				t.Fatalf("unexpected event type %T", et)
			}
		}

		if len(res.Events) != 11 {
			t.Fatalf("expected 10 states, got %d", len(res.Events))
		}

	})
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

func TestFooRequestPageSize(t *testing.T) {
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestFooRequestPageSize")
	defer ss.RunSteps(t)

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	ss.StepC("Create", func(ctx context.Context, a flowtest.Asserter) {
		tenantID := uuid.NewString()

		restore := silenceLogger()
		defer restore()

		for ii := 0; ii < 30; ii++ {
			tt := time.Now()
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
							Field: fmt.Sprintf("foo %d at %s", ii, tt.Format(time.RFC3339Nano)),
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

	ss.StepC("List Page (default)", func(ctx context.Context, t flowtest.Asserter) {
		req := &testpb.ListFoosRequest{}
		res := &testpb.ListFoosResponse{}

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foos) != int(20) {
			t.Fatalf("expected %d states, got %d", 20, len(res.Foos))
		}

		for ii, state := range res.Foos {
			t.Logf("%d: %s", ii, state.Field)
		}

		pageResp = res.Page

		if pageResp.GetNextToken() == "" {
			t.Fatalf("NextToken should not be empty")
		}
		if pageResp.NextToken == nil {
			t.Fatalf("Should not be the final page")
		}
	})

	ss.StepC("List Page", func(ctx context.Context, t flowtest.Asserter) {
		pageSize := int64(5)
		req := &testpb.ListFoosRequest{
			Page: &psml_pb.PageRequest{
				PageSize: &pageSize,
			},
		}
		res := &testpb.ListFoosResponse{}

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foos) != int(pageSize) {
			t.Fatalf("expected %d states, got %d", pageSize, len(res.Foos))
		}

		for ii, state := range res.Foos {
			t.Logf("%d: %s", ii, state.Field)
		}

		pageResp = res.Page

		if pageResp.GetNextToken() == "" {
			t.Fatalf("NextToken should not be empty")
		}
		if pageResp.NextToken == nil {
			t.Fatalf("Should not be the final page")
		}
	})

	ss.StepC("List Page (exceeding)", func(ctx context.Context, t flowtest.Asserter) {
		pageSize := int64(50)
		req := &testpb.ListFoosRequest{
			Page: &psml_pb.PageRequest{
				PageSize: &pageSize,
			},
		}
		res := &testpb.ListFoosResponse{}

		err = queryer.List(ctx, db, req, res)
		if err == nil {
			t.Fatal("expected error")
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

func printQuery(t flowtest.TB, query *sqrl.SelectBuilder) {
	stmt, args, err := query.ToSql()
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(stmt, args)
}
