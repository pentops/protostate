package integration

import (
	"context"
	"testing"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/j5/gen/j5/auth/v1/auth_j5pb"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/pquery"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
)

func TestDefaultFiltering(t *testing.T) {
	conn := pgtest.GetTestDB(t, pgtest.WithDir(allMigrationsDir))
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

	queryer, err := test_spb.NewFooPSMQuerySet(test_spb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	tenants := []string{uuid.NewString()}
	tenantIDs := setupFooListableData(ss, sm, tenants, 10)

	ss.Step("Setup Extra Statuses", func(ctx context.Context, t flowtest.Asserter) {
		for _, id := range tenantIDs[tenants[0]][:2] {
			event := newFooUpdatedEvent(id, tenants[0], func(u *test_pb.FooEventType_Updated) {
				u.Delete = true
			})

			_, err := sm.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}
		}
	})

	ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(10),
			},
		}
		res := &test_spb.FooListResponse{}

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != 8 {
			t.Fatalf("expected %d states, got %d", 8, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for _, state := range res.Foo {
			if state.Status != test_pb.FooStatus_ACTIVE {
				t.Fatalf("expected status %s, got %s", test_pb.FooStatus_ACTIVE, state.Status)
			}
		}

		if res.Page != nil {
			t.Fatalf("page response should be empty")
		}
	})
}

func testLogger(t *testing.T) pquery.QueryLogger {
	return func(query sqrlx.Sqlizer) {
		queryString, args, err := query.ToSql()
		if err != nil {
			t.Logf("Query Error: %s", err.Error())
			return
		}
		t.Logf("Query %s; ARGS %#v", queryString, args)
	}
}

func TestFilteringWithAuthScope(t *testing.T) {
	conn := pgtest.GetTestDB(t, pgtest.WithDir(allMigrationsDir))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestFooFilteringWithAuth")
	defer ss.RunSteps(t)

	queryer, err := test_spb.NewFooPSMQuerySet(test_spb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), newTokenQueryStateOption())
	if err != nil {
		t.Fatal(err.Error())
	}
	queryer.SetQueryLogger(testLogger(t))

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

	ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								Name: "data.characteristics.weight",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Range{
										Range: &list_j5pb.Range{
											Min: "12",
											Max: "15",
										},
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

		if len(res.Foo) != 4 {
			t.Fatalf("expected %d states, got %d", 4, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s (%s)", ii, state.Data.Field, state.Metadata.CreatedAt.AsTime().Format(time.RFC3339Nano))
		}

		for ii, state := range res.Foo {
			if *state.Keys.TenantId != tenantID1 {
				t.Fatalf("expected tenant ID %s, got %s", tenantID1, state.Keys.TenantId)
			}
			if state.Data.Characteristics.Weight != int64(15-ii) {
				t.Fatalf("expected weight %d, got %d", 15-ii, state.Data.Characteristics.Weight)
			}

		}

		if res.Page != nil {
			t.Fatalf("page response should be empty")
		}
	})
}

func TestDynamicFiltering(t *testing.T) {
	conn := pgtest.GetTestDB(t, pgtest.WithDir(allMigrationsDir))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestDynamicFiltering")
	defer ss.RunSteps(t)

	queryer, err := test_spb.NewFooPSMQuerySet(test_spb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}
	queryer.SetQueryLogger(testLogger(t))

	tenants := []string{uuid.NewString()}
	ids := setupFooListableData(ss, sm, tenants, 60)

	t.Run("Single Range Filter", func(t *testing.T) {
		ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "data.characteristics.weight",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Range{
											Range: &list_j5pb.Range{
												Min: "12",
												Max: "15",
											},
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

			if len(res.Foo) != 4 {
				t.Fatalf("expected %d states, got %d", 4, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for ii, state := range res.Foo {
				if state.Data.Characteristics.Weight != int64(15-ii) {
					t.Fatalf("expected weight %d, got %d", 15-ii, state.Data.Characteristics.Weight)
				}
			}

			if res.Page != nil {
				t.Fatalf("page response should be empty")
			}
		})
	})

	t.Run("Min Range Filter", func(t *testing.T) {
		ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "data.characteristics.weight",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Range{
											Range: &list_j5pb.Range{
												Min: "12",
											},
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

			if len(res.Foo) != 5 {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for _, state := range res.Foo {
				if state.Data.Characteristics.Weight < int64(12) {
					t.Fatalf("expected weights greater than or equal to %d, got %d", 12, state.Data.Characteristics.Weight)
				}
			}
		})
	})

	t.Run("Max Range Filter", func(t *testing.T) {
		ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "data.characteristics.weight",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Range{
											Range: &list_j5pb.Range{
												Max: "15",
											},
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

			if len(res.Foo) != 5 {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for _, state := range res.Foo {
				if state.Data.Characteristics.Weight > int64(15) {
					t.Fatalf("expected weight less than or equal to %d, got %d", 15, state.Data.Characteristics.Weight)
				}
			}
		})
	})

	t.Run("Multi Range Filter", func(t *testing.T) {
		nextToken := ""
		ss.Step("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(10),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Or{
								Or: &list_j5pb.Or{
									Filters: []*list_j5pb.Filter{
										{
											Type: &list_j5pb.Filter_Field{
												Field: &list_j5pb.Field{
													Name: "data.characteristics.weight",
													Type: &list_j5pb.FieldType{
														Type: &list_j5pb.FieldType_Range{
															Range: &list_j5pb.Range{
																Min: "12",
																Max: "20",
															},
														},
													},
												},
											},
										},
										{
											Type: &list_j5pb.Filter_Field{
												Field: &list_j5pb.Field{
													Name: "data.characteristics.height",
													Type: &list_j5pb.FieldType{
														Type: &list_j5pb.FieldType_Range{
															Range: &list_j5pb.Range{
																Min: "16",
																Max: "18",
															},
														},
													},
												},
											},
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
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			if len(res.Foo) != 10 {
				t.Fatalf("expected %d states, got %d", 10, len(res.Foo))
			}

			for ii, state := range res.Foo[:3] {
				if state.Data.Characteristics.Weight != int64(44-ii) {
					t.Fatalf("expected weight %d, got %d", 44-ii, state.Data.Characteristics.Weight)
				}
			}

			for ii, state := range res.Foo[3:] {
				if state.Data.Characteristics.Weight != int64(20-ii) {
					t.Fatalf("expected weight %d, got %d", 20-ii, state.Data.Characteristics.Weight)
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
					PageSize: proto.Int64(10),
					Token:    &nextToken,
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Or{
								Or: &list_j5pb.Or{
									Filters: []*list_j5pb.Filter{
										{
											Type: &list_j5pb.Filter_Field{
												Field: &list_j5pb.Field{
													Name: "data.characteristics.weight",
													Type: &list_j5pb.FieldType{
														Type: &list_j5pb.FieldType_Range{
															Range: &list_j5pb.Range{
																Min: "12",
																Max: "20",
															},
														},
													},
												},
											},
										},
										{
											Type: &list_j5pb.Filter_Field{
												Field: &list_j5pb.Field{
													Name: "data.characteristics.height",
													Type: &list_j5pb.FieldType{
														Type: &list_j5pb.FieldType_Range{
															Range: &list_j5pb.Range{
																Min: "16",
																Max: "18",
															},
														},
													},
												},
											},
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
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			if len(res.Foo) != 2 {
				t.Fatalf("expected %d states, got %d", 2, len(res.Foo))
			}

			for ii, state := range res.Foo {
				if state.Data.Characteristics.Weight != int64(13-ii) {
					t.Fatalf("expected weight %d, got %d", 13-ii, state.Data.Characteristics.Weight)
				}
			}

			if res.Page != nil {
				t.Fatalf("page response should be empty")
			}
		})
	})

	t.Run("Flattened filterable fields", func(t *testing.T) {
		ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "tenantId",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Value{
											Value: tenants[0],
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
				t.Fatal(err)
			}

			if len(res.Foo) == 0 {
				t.Fatalf("expected to receive results filtered by tenantId, but got none")
			}

			for _, foo := range res.Foo {
				if *foo.Keys.TenantId != tenants[0] {
					t.Fatalf("expected tenantId %s, got %s", tenants[0], *foo.Keys.TenantId)
				}
			}
		})
	})

	t.Run("Non filterable fields", func(t *testing.T) {
		ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "foo_id",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Value{
											Value: "d34d826f-afe3-410d-8326-4e9af3f09467",
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
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	})

	t.Run("Enum values", func(t *testing.T) {
		ss.Step("List Page short enum name", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "status",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Value{
											Value: "active",
										},
									},
								},
							},
						},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err := queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err)
			}

			if len(res.Foo) != 5 {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for _, state := range res.Foo {
				if state.Status != test_pb.FooStatus_ACTIVE {
					t.Fatalf("expected status %s, got %s", test_pb.FooStatus_ACTIVE, state.Status)
				}
			}
		})

		ss.Step("List Page full enum name", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "status",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Value{
											Value: "FOO_STATUS_ACTIVE",
										},
									},
								},
							},
						},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err := queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err)
			}

			if len(res.Foo) != 5 {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			for _, state := range res.Foo {
				if state.Status != test_pb.FooStatus_ACTIVE {
					t.Fatalf("expected status %s, got %s", test_pb.FooStatus_ACTIVE, state.Status)
				}
			}
		})

		ss.Step("List Page bad enum name", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "status",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Value{
											Value: "FOO_STATUS_UNUSED",
										},
									},
								},
							},
						},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err := queryer.List(ctx, db, req, res)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("Single Complex Range Filter", func(t *testing.T) {
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
									Name: "data.profiles.place",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Range{
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
				},
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Profiles)
			}

			if len(res.Foo) != 5 {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for _, state := range res.Foo {
				matched := false
				for _, profile := range state.Data.Profiles {
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
									Name: "data.profiles.place",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Range{
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
				},
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Profiles)
			}

			if len(res.Foo) != 2 {
				t.Fatalf("expected %d states, got %d", 2, len(res.Foo))
			}

			for _, state := range res.Foo {
				matched := false
				for _, profile := range state.Data.Profiles {
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

	t.Run("Oneof filter", func(t *testing.T) {
		ss.Step("List Page (created)", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooEventsRequest{
				FooId: ids[tenants[0]][0],
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "event.type",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Value{
											Value: "created",
										},
									},
								},
							},
						},
					},
				},
			}
			res := &test_spb.FooEventsResponse{}

			err := queryer.EventLister.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err)
			}

			if len(res.Events) != 1 {
				t.Fatalf("expected %d states, got %d", 1, len(res.Events))
			}

			for ii, event := range res.Events {
				switch event.Event.Type.(type) {
				case *test_pb.FooEventType_Created_:
				default:
					t.Fatalf("expected event to be of type %T, got %T", &test_pb.FooEventType_Created_{}, event.Event.Type)
				}

				t.Logf("%d: %s", ii, event.Event)
			}
		})

		ss.Step("List Page (deleted)", func(ctx context.Context, t flowtest.Asserter) {
			id := ids[tenants[0]][0]

			event := newFooUpdatedEvent(id, tenants[0], func(u *test_pb.FooEventType_Updated) {
				u.Delete = true
			})

			_, err := sm.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}

			req := &test_spb.FooEventsRequest{
				FooId: id,
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "event.type",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Value{
											Value: "deleted",
										},
									},
								},
							},
						},
					},
				},
			}
			res := &test_spb.FooEventsResponse{}

			err = queryer.EventLister.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err)
			}

			if len(res.Events) != 1 {
				t.Fatalf("expected %d states, got %d", 1, len(res.Events))
			}

			for ii, event := range res.Events {
				switch event.Event.Type.(type) {
				case *test_pb.FooEventType_Deleted_:
				default:
					t.Fatalf("expected event to be of type %T, got %T", &test_pb.FooEventType_Deleted_{}, event.Event.Type)
				}

				t.Logf("%d: %s", ii, event.Event)
			}
		})

		ss.Step("List Page bad name", func(ctx context.Context, t flowtest.Asserter) {
			req := &test_spb.FooEventsRequest{
				FooId: ids[tenants[0]][0],
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "event.type",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Value{
											Value: "damaged",
										},
									},
								},
							},
						},
					},
				},
			}
			res := &test_spb.FooEventsResponse{}

			err := queryer.EventLister.List(ctx, db, req, res)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}
