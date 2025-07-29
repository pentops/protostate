package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/pquery"
)

func TestPagination(t *testing.T) {
	uu := NewSchemaUniverse(t)
	var queryer *pquery.Lister

	ss := NewStepper(t)
	defer ss.RunSteps(t)

	ss.Setup(func(ctx context.Context, t flowtest.Asserter) error {
		tenantID := uuid.NewString()

		uu.SetupFoo(t, 30, func(ii int, foo *TestObject) {

			foo.SetScalar(pquery.JSONPath("tenantId"), tenantID)

			weight := (10 + int64(ii))

			foo.SetScalar(pquery.JSONPath("data", "characteristics", "weight"), weight)
			createdAt := time.Now()
			foo.SetScalar(pquery.JSONPath("metadata", "createdAt"), createdAt)
			foo.SetScalar(pquery.JSONPath("data", "field"), fmt.Sprintf("foo %d at %s", ii, createdAt.Format(time.RFC3339Nano)))
		})
		queryer = uu.FooLister(t)
		return nil

	})

	queryer = uu.FooLister(t)
	var pageResp *list_j5pb.PageResponse

	ss.Step("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooListRequest{}
		res := &test_spb.FooListResponse{}
		if err := queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object()); err != nil {
			t.Fatal(err.Error())
		}

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

		query, err := queryer.BuildQuery(ctx, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err.Error())
		}
		printQuery(t, query)

		err = queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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

	ss.Step("List Page - Short", func(ctx context.Context, t flowtest.Asserter) {
		pageSize := int64(5)
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: &pageSize,
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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

	ss.Step("List Page - exceeding", func(ctx context.Context, t flowtest.Asserter) {
		pageSize := int64(50)
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: &pageSize,
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
