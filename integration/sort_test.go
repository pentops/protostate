package integration

import (
	"context"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/gen/list/v1/psml_pb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
)

func TestSortingWithAuthScope(t *testing.T) {
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db, uuid.NewString())
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestFooSortingWithAuth")
	defer ss.RunSteps(t)

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), newTokenQueryStateOption())
	if err != nil {
		t.Fatal(err.Error())
	}

	tenantID1 := uuid.NewString()
	tenantID2 := uuid.NewString()

	tenants := []string{tenantID1, tenantID2}
	setupFooListableData(t, ss, sm, tenants, 10)

	tkn := &token{
		tenantID: tenantID1,
	}

	nextToken := ""
	ss.StepC("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		req := &testpb.ListFoosRequest{
			Page: &psml_pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &psml_pb.QueryRequest{
				Sorts: []*psml_pb.Sort{
					{Field: "characteristics.weight"},
				},
			},
		}
		res := &testpb.ListFoosResponse{}

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foos) != int(5) {
			t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
		}

		for ii, state := range res.Foos {
			t.Logf("%d: %s", ii, state.Field)
		}

		for ii, state := range res.Foos {
			if state.Characteristics.Weight != int64(10+ii) {
				t.Fatalf("expected weight %d, got %d", 10+ii, state.Characteristics.Weight)
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

	ss.StepC("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		req := &testpb.ListFoosRequest{
			Page: &psml_pb.PageRequest{
				PageSize: proto.Int64(5),
				Token:    &nextToken,
			},
			Query: &psml_pb.QueryRequest{
				Sorts: []*psml_pb.Sort{
					{Field: "characteristics.weight"},
				},
			},
		}
		res := &testpb.ListFoosResponse{}

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foos) != int(5) {
			t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
		}

		for ii, state := range res.Foos {
			t.Logf("%d: %s", ii, state.Field)
		}

		for ii, state := range res.Foos {
			if state.Characteristics.Weight != int64(15+ii) {
				t.Fatalf("expected weight %d, got %d", 15+ii, state.Characteristics.Weight)
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
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db, uuid.NewString())
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestFooSortingWithAuth")
	defer ss.RunSteps(t)

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), newTokenQueryStateOption())
	if err != nil {
		t.Fatal(err.Error())
	}

	tenants := []string{uuid.NewString(), uuid.NewString()}
	setupFooListableData(t, ss, sm, tenants, 30)

	tkn := &token{
		tenantID: "",
	}

	nextToken := ""
	ss.StepC("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		req := &testpb.ListFoosRequest{
			Page: &psml_pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &psml_pb.QueryRequest{
				Sorts: []*psml_pb.Sort{
					{Field: "characteristics.weight"},
				},
			},
		}
		res := &testpb.ListFoosResponse{}

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foos) != int(5) {
			t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
		}

		for ii, state := range res.Foos {
			t.Logf("%d: %s", ii, state.Field)
		}

		for ii, state := range res.Foos {
			if state.Characteristics.Weight != int64(10+ii) {
				t.Fatalf("expected weight %d, got %d", 10+ii, state.Characteristics.Weight)
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

	ss.StepC("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		req := &testpb.ListFoosRequest{
			Page: &psml_pb.PageRequest{
				PageSize: proto.Int64(5),
				Token:    &nextToken,
			},
			Query: &psml_pb.QueryRequest{
				Sorts: []*psml_pb.Sort{
					{Field: "characteristics.weight"},
				},
			},
		}
		res := &testpb.ListFoosResponse{}

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foos) != int(5) {
			t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
		}

		for ii, state := range res.Foos {
			t.Logf("%d: %s", ii, state.Field)
		}

		for ii, state := range res.Foos {
			if state.Characteristics.Weight != int64(15+ii) {
				t.Fatalf("expected weight %d, got %d", 15+ii, state.Characteristics.Weight)
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
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db, uuid.NewString())
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestDynamicSorting")
	defer ss.RunSteps(t)

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	tenants := []string{uuid.NewString()}
	setupFooListableData(t, ss, sm, tenants, 30)

	t.Run("Top Level Field", func(t *testing.T) {
		nextToken := ""
		ss.StepC("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &psml_pb.QueryRequest{
					Sorts: []*psml_pb.Sort{
						{Field: "createdAt"},
					},
				},
			}
			res := &testpb.ListFoosResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foos) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
			}

			for ii, state := range res.Foos {
				t.Logf("%d: %s", ii, state.Field)
			}

			for ii, state := range res.Foos {
				if state.Characteristics.Weight != int64(10+ii) {
					t.Fatalf("expected weight %d, got %d", 10+ii, state.Characteristics.Weight)
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

		ss.StepC("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
					Token:    &nextToken,
				},
				Query: &psml_pb.QueryRequest{
					Sorts: []*psml_pb.Sort{
						{Field: "createdAt"},
					},
				},
			}
			res := &testpb.ListFoosResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foos) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
			}

			for ii, state := range res.Foos {
				t.Logf("%d: %s", ii, state.Field)
			}

			for ii, state := range res.Foos {
				if state.Characteristics.Weight != int64(15+ii) {
					t.Fatalf("expected weight %d, got %d", 15+ii, state.Characteristics.Weight)
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
		nextToken := ""
		ss.StepC("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &psml_pb.QueryRequest{
					Sorts: []*psml_pb.Sort{
						{Field: "characteristics.weight"},
					},
				},
			}
			res := &testpb.ListFoosResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foos) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
			}

			for ii, state := range res.Foos {
				t.Logf("%d: %s", ii, state.Field)
			}

			for ii, state := range res.Foos {
				if state.Characteristics.Weight != int64(10+ii) {
					t.Fatalf("expected weight %d, got %d", 10+ii, state.Characteristics.Weight)
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

		ss.StepC("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
					Token:    &nextToken,
				},
				Query: &psml_pb.QueryRequest{
					Sorts: []*psml_pb.Sort{
						{Field: "characteristics.weight"},
					},
				},
			}
			res := &testpb.ListFoosResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foos) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
			}

			for ii, state := range res.Foos {
				t.Logf("%d: %s", ii, state.Field)
			}

			for ii, state := range res.Foos {
				if state.Characteristics.Weight != int64(15+ii) {
					t.Fatalf("expected weight %d, got %d", 15+ii, state.Characteristics.Weight)
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
		nextToken := ""
		ss.StepC("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &psml_pb.QueryRequest{
					Sorts: []*psml_pb.Sort{
						{Field: "characteristics.length"},
						{Field: "characteristics.weight"},
					},
				},
			}
			res := &testpb.ListFoosResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foos) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
			}

			for ii, state := range res.Foos {
				t.Logf("%d: %s", ii, state.Field)
			}

			if res.Foos[0].Characteristics.Weight != int64(10) {
				t.Fatalf("expected list to start with weight %d, got %d", 10, res.Foos[0].Characteristics.Weight)
			}

			for _, state := range res.Foos {
				if state.Characteristics.Weight%2 != 0 {
					t.Fatalf("expected even number weight, got %d", state.Characteristics.Weight)
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

		ss.StepC("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
					Token:    &nextToken,
				},
				Query: &psml_pb.QueryRequest{
					Sorts: []*psml_pb.Sort{
						{Field: "characteristics.length"},
						{Field: "characteristics.weight"},
					},
				},
			}
			res := &testpb.ListFoosResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foos) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
			}

			for ii, state := range res.Foos {
				t.Logf("%d: %s", ii, state.Field)
			}

			if res.Foos[0].Characteristics.Weight != int64(20) {
				t.Fatalf("expected list to start with weight %d, got %d", 20, res.Foos[0].Characteristics.Weight)
			}

			for _, state := range res.Foos {
				if state.Characteristics.Weight%2 != 0 {
					t.Fatalf("expected even number weight, got %d", state.Characteristics.Weight)
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
		nextToken := ""
		ss.StepC("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &psml_pb.QueryRequest{
					Sorts: []*psml_pb.Sort{
						{
							Field:      "characteristics.weight",
							Descending: true,
						},
					},
				},
			}
			res := &testpb.ListFoosResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foos) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
			}

			for ii, state := range res.Foos {
				t.Logf("%d: %s", ii, state.Field)
			}

			for ii, state := range res.Foos {
				if state.Characteristics.Weight != int64(39-ii) {
					t.Fatalf("expected weight %d, got %d", 39-ii, state.Characteristics.Weight)
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

		ss.StepC("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
					Token:    &nextToken,
				},
				Query: &psml_pb.QueryRequest{
					Sorts: []*psml_pb.Sort{
						{
							Field:      "characteristics.weight",
							Descending: true,
						},
					},
				},
			}
			res := &testpb.ListFoosResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foos) != int(5) {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
			}

			for ii, state := range res.Foos {
				t.Logf("%d: %s", ii, state.Field)
			}

			for ii, state := range res.Foos {
				if state.Characteristics.Weight != int64(34-ii) {
					t.Fatalf("expected weight %d, got %d", 34-ii, state.Characteristics.Weight)
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
