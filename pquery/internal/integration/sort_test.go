package integration

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/j5/gen/j5/auth/v1/auth_j5pb"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/pquery"
	"google.golang.org/protobuf/proto"
)

func TestSortingWithAuthScope(t *testing.T) {

	uu := NewSchemaUniverse(t)
	ss := NewStepper(t)
	defer ss.RunSteps(t)

	var queryer *pquery.Lister

	tenantID1 := uuid.NewString()
	tenantID2 := uuid.NewString()

	tenants := []string{tenantID1, tenantID2}

	ss.Setup(func(ctx context.Context, t flowtest.Asserter) error {
		queryer = uu.FooLister(t, func(spec *pquery.TableSpec) {
			spec.Auth = tokenAuth()
		})
		uu.SetupFoo(t, 20, func(idx int, foo *TestObject) {
			// recreating the old function logic
			tenantId := idx % len(tenants)
			tenantID := tenants[tenantId]
			ii := idx / len(tenants)
			ti := tenantId

			foo.SetScalar(pquery.JSONPath("tenantId"), tenantID)

			weight := ((10 + int64(ii)) * (int64(ti) + 1))
			height := ((50 - int64(ii)) * (int64(ti) + 1))
			length := (int64(ii%2) * (int64(ti) + 1))

			foo.SetScalar(pquery.JSONPath("data", "characteristics", "weight"), weight)
			foo.SetScalar(pquery.JSONPath("data", "characteristics", "height"), height)
			foo.SetScalar(pquery.JSONPath("data", "characteristics", "length"), length)
			createdAt := time.Now()
			foo.SetScalar(pquery.JSONPath("metadata", "createdAt"), createdAt.Format(time.RFC3339Nano))

			label := fmt.Sprintf("foo %d (weighted %d, height %d, length %d)", ii, weight, height, length)
			foo.SetScalar(pquery.JSONPath("data", "field"), label)

		})
		return nil
	})

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

		err := queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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
				t.Fatalf("%d: expected weight %d, got %d", 10+ii, ii, state.Data.Characteristics.Weight)
			}

			if *state.Keys.TenantId != tenantID1 {
				t.Fatalf("%d: expected tenant ID %s, got %s", ii, tenantID1, state.Keys.TenantId)
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

		err = queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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
				t.Fatalf("%d: expected weight %d, got %d", ii, 15+ii, state.Data.Characteristics.Weight)
			}

			if *state.Keys.TenantId != tenantID1 {
				t.Fatalf("%d: expected tenant ID %s, got %s", ii, tenantID1, state.Keys.TenantId)
			}
		}

		if res.Page != nil {
			t.Fatalf("page response should be empty")
		}
	})
}

func TestDynamicSorting(t *testing.T) {
	uu := NewSchemaUniverse(t)

	queryer := uu.FooLister(t)
	tenantID := uuid.NewString()
	uu.SetupFoo(t, 30, func(ii int, foo *TestObject) {
		weight := (10 + int64(ii))
		height := (50 - int64(ii))
		length := (int64(ii % 2))

		foo.SetScalar(pquery.JSONPath("tenantId"), tenantID)
		foo.SetScalar(pquery.JSONPath("data", "characteristics", "weight"), weight)
		foo.SetScalar(pquery.JSONPath("data", "characteristics", "height"), height)
		foo.SetScalar(pquery.JSONPath("data", "characteristics", "length"), length)
		createdAt := time.Now()
		foo.SetScalar(pquery.JSONPath("metadata", "createdAt"), createdAt.Format(time.RFC3339Nano))

		label := fmt.Sprintf("foo %d at %s (weighted %d, height %d, length %d)", ii, createdAt.Format(time.RFC3339Nano), weight, height, length)
		foo.SetScalar(pquery.JSONPath("data", "field"), label)

		val, err := foo.GetOrCreateValue("data", "profiles")
		if err != nil {
			t.Fatalf("failed to get or create profiles: %v", err)
		}

		objArray, ok := val.AsArrayOfObject()
		if !ok {
			t.Fatalf("expected profiles to be an array of objects, got %T", val)
		}

		profile1, _ := objArray.NewObjectElement()
		profile2, _ := objArray.NewObjectElement()

		if err := SetScalar(profile1, pquery.JSONPath("name"), fmt.Sprintf("profile %d", ii)); err != nil {
			t.Fatalf("failed to set profile name: %v", err)
		}

		if err := SetScalar(profile1, pquery.JSONPath("place"), int64(ii)+50); err != nil {
			t.Fatalf("failed to set profile place: %v", err)
		}

		if err := SetScalar(profile2, pquery.JSONPath("name"), fmt.Sprintf("profile %d", ii)); err != nil {
			t.Fatalf("failed to set profile name: %v", err)
		}

		if err := SetScalar(profile2, pquery.JSONPath("place"), int64(ii)+15); err != nil {
			t.Fatalf("failed to set profile place: %v", err)
		}

		switch ii {
		case 10:
			foo.SetScalar(pquery.JSONPath("data", "shape", "circle", "radius"), int64(10))
		case 11:
			foo.SetScalar(pquery.JSONPath("data", "shape", "square", "side"), int64(10))
		}
	})

	t.Run("Top Level Field", func(t *testing.T) {
		ss := NewStepper(t)
		defer ss.RunSteps(t)
		var err error

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

			err = queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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

			err = queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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
		ss := NewStepper(t)
		defer ss.RunSteps(t)
		var err error

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

			err = queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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

			err = queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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
		ss := NewStepper(t)
		defer ss.RunSteps(t)
		var err error
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

			err = queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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

			err = queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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
		ss := NewStepper(t)
		defer ss.RunSteps(t)
		var err error
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

			err = queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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

			err = queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
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
