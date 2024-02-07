package integration

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
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

func TestPagination(t *testing.T) {
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

			event := newFooCreatedEvent(fooID, tenantID, func(c *testpb.FooEventType_Created) {
				c.Field = fmt.Sprintf("foo %d at %s", ii, tt.Format(time.RFC3339Nano))
				c.Weight = ptr.To(10 + int64(ii))
			})

			stateOut, err := sm.Transition(ctx, event)
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

func TestEventPagination(t *testing.T) {
	// Foo event default sort is deeply nested. This tests that the nested filter
	// works on pagination

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db, uuid.NewString())
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

		event := newFooCreatedEvent(fooID, tenantID, nil)
		_, err := sm.Transition(ctx, event)
		if err != nil {
			t.Fatal(err.Error())
		}

		for ii := 0; ii < 30; ii++ {
			tt := time.Now()
			event := newFooUpdatedEvent(fooID, tenantID, func(u *testpb.FooEventType_Updated) {
				u.Field = fmt.Sprintf("foo %d at %s", ii, tt.Format(time.RFC3339Nano))
				u.Weight = ptr.To(11 + int64(ii))
			})
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

func TestPageSize(t *testing.T) {
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db, uuid.NewString())
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
							Name:   "foo",
							Field:  fmt.Sprintf("foo %d at %s", ii, tt.Format(time.RFC3339Nano)),
							Weight: ptr.To(10 + int64(ii)),
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
