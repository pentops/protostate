package integration

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/j5/gen/j5/auth/v1/auth_j5pb"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"google.golang.org/protobuf/proto"
)

func TestSortingWithAuthScope(t *testing.T) {

	ss, uu := NewFooUniverse(t, WithStateQueryOptions(newTokenQueryStateOption()))
	sm := uu.SM
	db := uu.DB
	queryer := uu.Query
	var err error
	defer ss.RunSteps(t)

	tenantID1 := uuid.NewString()
	tenantID2 := uuid.NewString()

	tenants := []string{tenantID1, tenantID2}
	setupFooListableData(ss, sm, tenants, 10)

	tkn := &token{
		claim: &auth_j5pb.Claim{
			TenantType: "tenant",
			TenantId:   tenantID1,
		},
	}

	nextToken := ""
	ss.Step("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Sorts: []*list_j5pb.Sort{
					{Field: "data.characteristics.weight"},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != int(5) {
			t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for ii, state := range res.Foo {
			if state.Data.Characteristics.Weight != int64(10+ii) {
				t.Fatalf("expected weight %d, got %d", 10+ii, state.Data.Characteristics.Weight)
			}

			if *state.Keys.TenantId != tenantID1 {
				t.Fatalf("expected tenant ID %s, got %s", tenantID1, state.Keys.TenantId)
			}
		}

		pageResp := res.Page

		if pageResp.GetNextToken() == "" {
			t.Fatalf("NextToken should not be empty")
		}
		if pageResp.NextToken == nil {
			t.Fatalf("Should not be the final page")
		}

		nextToken = pageResp.GetNextToken()
	})

	ss.Step("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		pageTokenBytes, err := base64.StdEncoding.DecodeString(nextToken)
		if err != nil {
			t.Fatalf("failed to decode next token: %v", err)
		}
		t.Logf("Page Token: %s", string(pageTokenBytes))

		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
				Token:    &nextToken,
			},
			Query: &list_j5pb.QueryRequest{
				Sorts: []*list_j5pb.Sort{
					{Field: "data.characteristics.weight"},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != int(5) {
			t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for ii, state := range res.Foo {
			if state.Data.Characteristics.Weight != int64(15+ii) {
				t.Fatalf("expected weight %d, got %d", 15+ii, state.Data.Characteristics.Weight)
			}

			if *state.Keys.TenantId != tenantID1 {
				t.Fatalf("expected tenant ID %s, got %s", tenantID1, state.Keys.TenantId)
			}
		}

		if res.Page != nil {
			t.Fatalf("page response should be empty")
		}
	})
}

func TestSortingWithAuthNoScope(t *testing.T) {
	ss, uu := NewFooUniverse(t, WithStateQueryOptions(newTokenQueryStateOption()))
	sm := uu.SM
	db := uu.DB
	queryer := uu.Query
	var err error
	defer ss.RunSteps(t)

	tenants := []string{uuid.NewString(), uuid.NewString()}
	setupFooListableData(ss, sm, tenants, 30)

	tkn := &token{
		claim: &auth_j5pb.Claim{
			TenantType: "meta_tenant",
			TenantId:   metaTenant,
		},
	}

	nextToken := ""
	ss.Step("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Sorts: []*list_j5pb.Sort{
					{Field: "data.characteristics.weight"},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != int(5) {
			t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for ii, state := range res.Foo {
			if state.Data.Characteristics.Weight != int64(10+ii) {
				t.Fatalf("expected weight %d, got %d", 10+ii, state.Data.Characteristics.Weight)
			}
		}

		pageResp := res.Page

		if pageResp.GetNextToken() == "" {
			t.Fatalf("NextToken should not be empty")
		}
		if pageResp.NextToken == nil {
			t.Fatalf("Should not be the final page")
		}

		nextToken = pageResp.GetNextToken()
	})

	ss.Step("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
				Token:    &nextToken,
			},
			Query: &list_j5pb.QueryRequest{
				Sorts: []*list_j5pb.Sort{
					{Field: "data.characteristics.weight"},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != int(5) {
			t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for ii, state := range res.Foo {
			if state.Data.Characteristics.Weight != int64(15+ii) {
				t.Fatalf("expected weight %d, got %d", 15+ii, state.Data.Characteristics.Weight)
			}
		}

		pageResp := res.Page

		if pageResp.GetNextToken() == "" {
			t.Fatalf("NextToken should not be empty")
		}
		if pageResp.NextToken == nil {
			t.Fatalf("Should not be the final page")
		}
	})
}

func TestDynamicSorting(t *testing.T) {

	t.Run("Top Level Field", func(t *testing.T) {
		ss, uu := NewFooUniverse(t)
		sm := uu.SM
		db := uu.DB
		queryer := uu.Query
		var err error
		defer ss.RunSteps(t)

		tenants := []string{uuid.NewString()}
		setupFooListableData(ss, sm, tenants, 30)
		nextToken := ""
		ss.Step("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Sorts: []*list_j5pb.Sort{
						{Field: "metadata.createdAt"},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foo) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for ii, state := range res.Foo {
				if state.Data.Characteristics.Weight != int64(10+ii) {
					t.Fatalf("expected weight %d, got %d", 10+ii, state.Data.Characteristics.Weight)
				}
			}

			pageResp := res.Page

			if pageResp.GetNextToken() == "" {
				t.Fatalf("NextToken should not be empty")
			}
			if pageResp.NextToken == nil {
				t.Fatalf("Should not be the final page")
			}

			nextToken = pageResp.GetNextToken()
		})

		ss.Step("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
					Token:    &nextToken,
				},
				Query: &list_j5pb.QueryRequest{
					Sorts: []*list_j5pb.Sort{
						{Field: "metadata.createdAt"},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foo) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for ii, state := range res.Foo {
				if state.Data.Characteristics.Weight != int64(15+ii) {
					t.Fatalf("expected weight %d, got %d", 15+ii, state.Data.Characteristics.Weight)
				}
			}

			pageResp := res.Page

			if pageResp.GetNextToken() == "" {
				t.Fatalf("NextToken should not be empty")
			}
			if pageResp.NextToken == nil {
				t.Fatalf("Should not be the final page")
			}
		})
	})

	t.Run("Nested Field", func(t *testing.T) {
		ss, uu := NewFooUniverse(t)
		sm := uu.SM
		db := uu.DB
		queryer := uu.Query
		var err error
		defer ss.RunSteps(t)

		tenants := []string{uuid.NewString()}
		setupFooListableData(ss, sm, tenants, 30)
		nextToken := ""
		ss.Step("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Sorts: []*list_j5pb.Sort{
						{Field: "data.characteristics.weight"},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foo) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for ii, state := range res.Foo {
				if state.Data.Characteristics.Weight != int64(10+ii) {
					t.Fatalf("expected weight %d, got %d", 10+ii, state.Data.Characteristics.Weight)
				}
			}

			pageResp := res.Page

			if pageResp.GetNextToken() == "" {
				t.Fatalf("NextToken should not be empty")
			}
			if pageResp.NextToken == nil {
				t.Fatalf("Should not be the final page")
			}

			nextToken = pageResp.GetNextToken()
		})

		ss.Step("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
					Token:    &nextToken,
				},
				Query: &list_j5pb.QueryRequest{
					Sorts: []*list_j5pb.Sort{
						{Field: "data.characteristics.weight"},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foo) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for ii, state := range res.Foo {
				if state.Data.Characteristics.Weight != int64(15+ii) {
					t.Fatalf("expected weight %d, got %d", 15+ii, state.Data.Characteristics.Weight)
				}
			}

			pageResp := res.Page

			if pageResp.GetNextToken() == "" {
				t.Fatalf("NextToken should not be empty")
			}
			if pageResp.NextToken == nil {
				t.Fatalf("Should not be the final page")
			}
		})
	})

	t.Run("Multiple Nested Fields", func(t *testing.T) {
		ss, uu := NewFooUniverse(t)
		sm := uu.SM
		db := uu.DB
		queryer := uu.Query
		var err error
		defer ss.RunSteps(t)

		tenants := []string{uuid.NewString()}
		setupFooListableData(ss, sm, tenants, 30)
		nextToken := ""
		ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Sorts: []*list_j5pb.Sort{
						{Field: "data.characteristics.length"},
						{Field: "data.characteristics.weight"},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foo) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			if res.Foo[0].Data.Characteristics.Weight != int64(10) {
				t.Fatalf("expected list to start with weight %d, got %d", 10, res.Foo[0].Data.Characteristics.Weight)
			}

			for _, state := range res.Foo {
				if state.Data.Characteristics.Weight%2 != 0 {
					t.Fatalf("expected even number weight, got %d", state.Data.Characteristics.Weight)
				}
			}

			pageResp := res.Page

			if pageResp.GetNextToken() == "" {
				t.Fatalf("NextToken should not be empty")
			}
			if pageResp.NextToken == nil {
				t.Fatalf("Should not be the final page")
			}

			nextToken = pageResp.GetNextToken()
		})

		ss.Step("List Page 2", func(ctx context.Context, t flowtest.Asserter) {

			pageTokenBytes, err := base64.StdEncoding.DecodeString(nextToken)
			if err != nil {
				t.Fatalf("failed to decode next token: %v", err)
			}
			t.Logf("Page Token: %s", string(pageTokenBytes))

			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
					Token:    &nextToken,
				},
				Query: &list_j5pb.QueryRequest{
					Sorts: []*list_j5pb.Sort{
						{Field: "data.characteristics.length"},
						{Field: "data.characteristics.weight"},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foo) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			if res.Foo[0].Data.Characteristics.Weight != int64(20) {
				t.Fatalf("expected list to start with weight %d, got %d", 20, res.Foo[0].Data.Characteristics.Weight)
			}

			for _, state := range res.Foo {
				if state.Data.Characteristics.Weight%2 != 0 {
					t.Fatalf("expected even number weight, got %d", state.Data.Characteristics.Weight)
				}
			}

			pageResp := res.Page

			if pageResp.GetNextToken() == "" {
				t.Fatalf("NextToken should not be empty")
			}
			if pageResp.NextToken == nil {
				t.Fatalf("Should not be the final page")
			}
		})
	})

	t.Run("Descending", func(t *testing.T) {
		ss, uu := NewFooUniverse(t)
		sm := uu.SM
		db := uu.DB
		queryer := uu.Query
		var err error
		defer ss.RunSteps(t)

		tenants := []string{uuid.NewString()}
		setupFooListableData(ss, sm, tenants, 30)
		nextToken := ""
		ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Sorts: []*list_j5pb.Sort{
						{
							Field:      "data.characteristics.weight",
							Descending: true,
						},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foo) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for ii, state := range res.Foo {
				if state.Data.Characteristics.Weight != int64(39-ii) {
					t.Fatalf("expected weight %d, got %d", 39-ii, state.Data.Characteristics.Weight)
				}
			}

			pageResp := res.Page

			if pageResp.GetNextToken() == "" {
				t.Fatalf("NextToken should not be empty")
			}
			if pageResp.NextToken == nil {
				t.Fatalf("Should not be the final page")
			}

			nextToken = pageResp.GetNextToken()
		})

		ss.Step("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
					Token:    &nextToken,
				},
				Query: &list_j5pb.QueryRequest{
					Sorts: []*list_j5pb.Sort{
						{
							Field:      "data.characteristics.weight",
							Descending: true,
						},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(ctx, db, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foo) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for ii, state := range res.Foo {
				if state.Data.Characteristics.Weight != int64(34-ii) {
					t.Fatalf("expected weight %d, got %d", 34-ii, state.Data.Characteristics.Weight)
				}
			}

			pageResp := res.Page

			if pageResp.GetNextToken() == "" {
				t.Fatalf("NextToken should not be empty")
			}
			if pageResp.NextToken == nil {
				t.Fatalf("Should not be the final page")
			}
		})
	})
}
