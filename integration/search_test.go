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

func TestDynamicSearching(t *testing.T) {
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db, uuid.NewString())
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestDynamicSearching")
	defer ss.RunSteps(t)

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	tenants := []string{uuid.NewString()}
	setupFooListableData(t, ss, sm, tenants, 30)

	t.Run("Simple Search Field", func(t *testing.T) {
		ss.StepC("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &psml_pb.QueryRequest{
					Searches: []*psml_pb.Search{
						{
							Field: "field",
							Value: "weighted 30",
						},
					},
				},
			}
			res := &testpb.ListFoosResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foos) != 1 {
				t.Fatalf("expected %d states, got %d", 1, len(res.Foos))
			}

			for ii, state := range res.Foos {
				t.Logf("%d: %s", ii, state.Field)
			}

			for ii, state := range res.Foos {
				if state.Characteristics.Weight != int64(30-ii) {
					t.Fatalf("expected weight %d, got %d", 30-ii, state.Characteristics.Weight)
				}
			}

			if res.Page != nil {
				t.Fatalf("page response should be empty")
			}
		})
	})

	/*
		t.Run("Complex Search Field", func(t *testing.T) {
			nextToken := ""
			ss.StepC("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
				req := &testpb.ListFoosRequest{
					Page: &psml_pb.PageRequest{
						PageSize: proto.Int64(5),
					},
					Query: &psml_pb.QueryRequest{
						Filters: []*psml_pb.Filter{
							{
								Type: &psml_pb.Filter_Field{
									Field: &psml_pb.Field{
										Name: "profiles.place",
										Type: &psml_pb.Field_Range{
											Range: &psml_pb.Range{
												Min: "15",
												Max: "21",
											},
										},
									},
								},
							},
						},
					},
				}
				res := &testpb.ListFoosResponse{}

				err = queryer.List(ctx, db, req, res)
				if err != nil {
					t.Fatal(err.Error())
				}

				for ii, state := range res.Foos {
					t.Logf("%d: %s", ii, state.Profiles)
				}

				if len(res.Foos) != 5 {
					t.Fatalf("expected %d states, got %d", 5, len(res.Foos))
				}

				for _, state := range res.Foos {
					matched := false
					for _, profile := range state.Profiles {
						if profile.Place >= 17 && profile.Place <= 21 {
							matched = true
							break
						}
					}

					if !matched {
						t.Fatalf("expected at least one profile to match the filter")
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
						Filters: []*psml_pb.Filter{
							{
								Type: &psml_pb.Filter_Field{
									Field: &psml_pb.Field{
										Name: "profiles.place",
										Type: &psml_pb.Field_Range{
											Range: &psml_pb.Range{
												Min: "15",
												Max: "21",
											},
										},
									},
								},
							},
						},
					},
				}
				res := &testpb.ListFoosResponse{}

				err = queryer.List(ctx, db, req, res)
				if err != nil {
					t.Fatal(err.Error())
				}

				for ii, state := range res.Foos {
					t.Logf("%d: %s", ii, state.Profiles)
				}

				if len(res.Foos) != 2 {
					t.Fatalf("expected %d states, got %d", 2, len(res.Foos))
				}

				for _, state := range res.Foos {
					matched := false
					for _, profile := range state.Profiles {
						if profile.Place >= 15 && profile.Place <= 16 {
							matched = true
							break
						}
					}

					if !matched {
						t.Fatalf("expected at least one profile to match the filter")
					}
				}
			})
		})
	*/
}