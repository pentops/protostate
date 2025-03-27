package integration

import (
	"context"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/internal/pgstore/pgmigrate"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"github.com/pressly/goose"
	"google.golang.org/protobuf/proto"
)

func TestDynamicSearching(t *testing.T) {
	conn := pgtest.GetTestDB(t)
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	barSpec, err := psm.BuildQueryTableSpec(
		(&test_pb.BarState{}).ProtoReflect().Descriptor(),
		(&test_pb.BarEvent{}).ProtoReflect().Descriptor(),
	)
	if err != nil {
		t.Fatal(err)
	}

	stateSpecs := []psm.QueryTableSpec{
		sm.StateTableSpec(),
		barSpec,
	}

	if err := pgmigrate.CreateStateMachines(context.Background(), conn, stateSpecs...); err != nil {
		t.Fatal(err.Error())
	}

	if err := pgmigrate.AddIndexes(context.Background(), conn, stateSpecs...); err != nil {
		t.Fatal(err.Error())
	}

	if err := goose.Up(conn, stage2MigrationsDir); err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestDynamicSearching")
	defer ss.RunSteps(t)

	queryer, err := test_spb.NewFooPSMQuerySet(test_spb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	tenants := []string{uuid.NewString()}
	setupFooListableData(ss, sm, tenants, 30)

	t.Run("Simple Search Field", func(t *testing.T) {
		ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Searches: []*list_j5pb.Search{
						{
							Field: "data.field",
							Value: "weighted 30",
						},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foo) != 1 {
				t.Fatalf("expected %d states, got %d", 1, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for ii, state := range res.Foo {
				if state.Data.Characteristics.Weight != int64(30-ii) {
					t.Fatalf("expected weight %d, got %d", 30-ii, state.Data.Characteristics.Weight)
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
			ss.Step("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
				req := &test_spb.FooListRequest{
					Page: &list_j5pb.PageRequest{
						PageSize: proto.Int64(5),
					},
					Query: &list_j5pb.QueryRequest{
						Filters: []*list_j5pb.Filter{
							{
								Type: &list_j5pb.Filter_Field{
									Field: &list_j5pb.Field{
										Name: "profiles.place",
										Type: &list_j5pb.Field_Range{
											Range: &list_j5pb.Range{
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
				res := &test_spb.FooListResponse{}

				err = queryer.List(ctx, db, req, res)
				if err != nil {
					t.Fatal(err.Error())
				}

				for ii, state := range res.Foo {
					t.Logf("%d: %s", ii, state.Profiles)
				}

				if len(res.Foo) != 5 {
					t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
				}

				for _, state := range res.Foo {
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

			ss.Step("List Page 2", func(ctx context.Context, t flowtest.Asserter) {
				req := &test_spb.FooListRequest{
					Page: &list_j5pb.PageRequest{
						PageSize: proto.Int64(5),
						Token:    &nextToken,
					},
					Query: &list_j5pb.QueryRequest{
						Filters: []*list_j5pb.Filter{
							{
								Type: &list_j5pb.Filter_Field{
									Field: &list_j5pb.Field{
										Name: "profiles.place",
										Type: &list_j5pb.Field_Range{
											Range: &list_j5pb.Range{
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
				res := &test_spb.FooListResponse{}

				err = queryer.List(ctx, db, req, res)
				if err != nil {
					t.Fatal(err.Error())
				}

				for ii, state := range res.Foo {
					t.Logf("%d: %s", ii, state.Profiles)
				}

				if len(res.Foo) != 2 {
					t.Fatalf("expected %d states, got %d", 2, len(res.Foo))
				}

				for _, state := range res.Foo {
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
