package integration

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/psm"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"k8s.io/utils/ptr"
)

func TestPagination(t *testing.T) {
	ss, uu := NewFooUniverse(t)
	sm := uu.SM
	defer ss.RunSteps(t)

	queryer, err := test_spb.NewFooPSMQuerySet(test_spb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	ss.Step("Create", func(ctx context.Context, t flowtest.Asserter) {
		tenantID := uuid.NewString()

		restore := silenceLogger()
		defer restore()

		for ii := range 30 {
			tt := time.Now()
			fooID := uuid.NewString()

			event := newFooCreatedEvent(fooID, tenantID, func(c *test_pb.FooEventType_Created) {
				c.Field = fmt.Sprintf("foo %d at %s", ii, tt.Format(time.RFC3339Nano))
				c.Weight = ptr.To(10 + int64(ii))
			})

			stateOut, err := sm.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}
			t.Equal(test_pb.FooStatus_ACTIVE, stateOut.Status)
			t.Equal(tenantID, *stateOut.Keys.TenantId)
		}
	})

	var pageResp *list_j5pb.PageResponse

	ss.Step("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
		res := uu.ListFoo(t, &test_spb.FooListRequest{})

		if len(res.Foo) != 20 {
			t.Fatalf("expected 20 states, got %d", len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		pageResp = res.Page

		if pageResp.GetNextToken() == "" {
			t.Fatalf("NextToken should not be empty")
		}
		if pageResp.NextToken == nil {
			t.Fatalf("Should not be the final page")
		}
	})

	ss.Step("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				Token: pageResp.NextToken,
			},
		}
		res := &test_spb.FooListResponse{}

		query, err := queryer.MainLister.BuildQuery(ctx, req.ProtoReflect(), res.ProtoReflect())
		if err != nil {
			t.Fatal(err.Error())
		}
		printQuery(t, query)

		err = queryer.List(ctx, uu.DB, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		if len(res.Foo) != 10 {
			t.Fatalf("expected 10 states, got %d", len(res.Foo))
		}
	})
}

func TestEventPagination(t *testing.T) {
	// Foo event default sort is deeply nested. This tests that the nested filter
	// works on pagination
	ss, uu := NewFooUniverse(t)
	sm := uu.SM
	db := uu.DB
	defer ss.RunSteps(t)

	queryer, err := test_spb.NewFooPSMQuerySet(test_spb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	fooID := uuid.NewString()
	ss.Step("CreateEvents", func(ctx context.Context, t flowtest.Asserter) {
		tenantID := uuid.NewString()

		restore := silenceLogger()
		defer restore()

		event := newFooCreatedEvent(fooID, tenantID)
		_, err := sm.Transition(ctx, event)
		if err != nil {
			t.Fatal(err.Error())
		}

		for ii := range 30 {
			tt := time.Now()
			event := newFooUpdatedEvent(fooID, tenantID, func(u *test_pb.FooEventType_Updated) {
				u.Field = fmt.Sprintf("foo %d at %s", ii, tt.Format(time.RFC3339Nano))
				u.Weight = ptr.To(11 + int64(ii))
			})
			_, err := sm.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}
		}
	})

	var pageResp *list_j5pb.PageResponse

	ss.Step("List Page 1", func(ctx context.Context, t flowtest.Asserter) {

		req := &test_spb.FooEventsRequest{
			FooId: fooID,
		}
		res := &test_spb.FooEventsResponse{}

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
			case *test_pb.FooEventType_Created_:
				t.Logf("%03d: Create %s - %d", ii, et.Created.Field, tsv)
			case *test_pb.FooEventType_Updated_:
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

		msg := &test_pb.FooEvent{}
		if err := proto.Unmarshal(rowBytes, msg); err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(msg))
		t.Logf("Token entry, TS: %d, ID: %s", msg.Metadata.Timestamp.AsTime().Round(time.Microsecond).UnixMicro(), msg.Metadata.EventId)

	})

	ss.Step("List Page 2", func(ctx context.Context, t flowtest.Asserter) {

		req := &test_spb.FooEventsRequest{
			Page: &list_j5pb.PageRequest{
				Token: pageResp.NextToken,
			},
			FooId: fooID,
		}
		res := &test_spb.FooEventsResponse{}

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
			case *test_pb.FooEventType_Created_:
				t.Logf("%d: Create %s", ii, et.Created.Field)
			case *test_pb.FooEventType_Updated_:
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

	ss, uu := NewFooUniverse(t)
	sm := uu.SM
	db := uu.DB
	queryer := uu.Query
	defer ss.RunSteps(t)

	ss.Step("Create", func(ctx context.Context, t flowtest.Asserter) {
		tenantID := uuid.NewString()

		restore := silenceLogger()
		defer restore()

		for ii := range 30 {
			tt := time.Now()
			fooID := uuid.NewString()

			event1 := newFooEvent(&test_pb.FooKeys{
				TenantId:     &tenantID,
				FooId:        fooID,
				MetaTenantId: metaTenant,
			}, &test_pb.FooEventType_Created{
				Name:   "foo",
				Field:  fmt.Sprintf("foo %d at %s", ii, tt.Format(time.RFC3339Nano)),
				Weight: ptr.To(10 + int64(ii)),
			},
			)
			stateOut, err := sm.Transition(ctx, event1)
			if err != nil {
				t.Fatal(err.Error())
			}
			t.Equal(test_pb.FooStatus_ACTIVE, stateOut.Status)
			t.Equal(tenantID, *stateOut.Keys.TenantId)
		}
	})

	var pageResp *list_j5pb.PageResponse

	ss.Step("List Page (default)", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooListRequest{}
		res := &test_spb.FooListResponse{}

		err := queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != int(20) {
			t.Fatalf("expected %d states, got %d", 20, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		pageResp = res.Page

		if pageResp.GetNextToken() == "" {
			t.Fatalf("NextToken should not be empty")
		}
		if pageResp.NextToken == nil {
			t.Fatalf("Should not be the final page")
		}
	})

	ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
		pageSize := int64(5)
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: &pageSize,
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != int(pageSize) {
			t.Fatalf("expected %d states, got %d", pageSize, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		pageResp = res.Page

		if pageResp.GetNextToken() == "" {
			t.Fatalf("NextToken should not be empty")
		}
		if pageResp.NextToken == nil {
			t.Fatalf("Should not be the final page")
		}
	})

	ss.Step("List Page (exceeding)", func(ctx context.Context, t flowtest.Asserter) {
		pageSize := int64(50)
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: &pageSize,
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(ctx, db, req, res)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
