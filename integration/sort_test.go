package integration

import (
	"context"
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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/utils/ptr"
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

	tkn := &token{
		tenantID: tenantID1,
	}

	ss.StepC("Create", func(ctx context.Context, a flowtest.Asserter) {
		tenants := []string{tenantID1, tenantID2}

		for ti := range tenants {
			tkn := &token{
				tenantID: tenants[ti],
			}
			ctx = tkn.WithToken(ctx)

			restore := silenceLogger()
			defer restore()

			for ii := 0; ii < 10; ii++ {
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
					TenantId: &tenants[ti],
					FooId:    fooID,
					Event: &testpb.FooEventType{
						Type: &testpb.FooEventType_Created_{
							Created: &testpb.FooEventType_Created{
								Name:   "foo",
								Field:  fmt.Sprintf("foo %d at %s (weighted %d, height %d, length %d)", ii, tt.Format(time.RFC3339Nano), (10+ii)*(ti+1), (50-ii)*(ti+1), (ii%2)*(ti+1)),
								Weight: ptr.To((10 + int64(ii)) * (int64(ti) + 1)),
								Height: ptr.To((50 - int64(ii)) * (int64(ti) + 1)),
								Length: ptr.To((int64(ii%2) * (int64(ti) + 1))),
							},
						},
					},
				}
				stateOut, err := sm.Transition(ctx, event1)
				if err != nil {
					t.Fatal(err.Error())
				}
				a.Equal(testpb.FooStatus_ACTIVE, stateOut.Status)
				a.Equal(tenants[ti], *stateOut.TenantId)
			}
		}
	})

	nextToken := ""
	ss.StepC("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		req := &testpb.ListFoosRequest{
			Page: &psml_pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &psml_pb.QueryRequest{
				Sort: []*psml_pb.Sort{
					{
						Field: "characteristics.weight",
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
			if state.Characteristics.Weight != int64(10+ii) {
				t.Fatalf("expected weight %d, got %d", 10+ii, state.Characteristics.Weight)
			}

			if *state.TenantId != tenantID1 {
				t.Fatalf("expected tenant ID %s, got %s", tenantID1, state.TenantId)
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
				Sort: []*psml_pb.Sort{
					{
						Field: "characteristics.weight",
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
			if state.Characteristics.Weight != int64(15+ii) {
				t.Fatalf("expected weight %d, got %d", 15+ii, state.Characteristics.Weight)
			}

			if *state.TenantId != tenantID1 {
				t.Fatalf("expected tenant ID %s, got %s", tenantID1, state.TenantId)
			}
		}

		pageResp := res.Page

		if pageResp.GetNextToken() != "" {
			t.Fatalf("NextToken should be empty")
		}
		if pageResp.NextToken != nil {
			t.Fatalf("Should be the final page")
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

	tkn := &token{
		tenantID: "",
	}

	ss.StepC("Create", func(ctx context.Context, a flowtest.Asserter) {
		tenants := []string{uuid.NewString(), uuid.NewString()}

		for ti := range tenants {
			tkn := &token{
				tenantID: tenants[ti],
			}
			ctx = tkn.WithToken(ctx)

			restore := silenceLogger()
			defer restore()

			for ii := 0; ii < 10; ii++ {
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
					TenantId: &tenants[ti],
					FooId:    fooID,
					Event: &testpb.FooEventType{
						Type: &testpb.FooEventType_Created_{
							Created: &testpb.FooEventType_Created{
								Name:   "foo",
								Field:  fmt.Sprintf("foo %d at %s (weighted %d, height %d, length %d)", ii, tt.Format(time.RFC3339Nano), (10+ii)*(ti+1), (50-ii)*(ti+1), (ii%2)*(ti+1)),
								Weight: ptr.To((10 + int64(ii)) * (int64(ti) + 1)),
								Height: ptr.To((50 - int64(ii)) * (int64(ti) + 1)),
								Length: ptr.To((int64(ii%2) * (int64(ti) + 1))),
							},
						},
					},
				}
				stateOut, err := sm.Transition(ctx, event1)
				if err != nil {
					t.Fatal(err.Error())
				}
				a.Equal(testpb.FooStatus_ACTIVE, stateOut.Status)
				a.Equal(tenants[ti], *stateOut.TenantId)
			}
		}
	})

	nextToken := ""
	ss.StepC("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		req := &testpb.ListFoosRequest{
			Page: &psml_pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &psml_pb.QueryRequest{
				Sort: []*psml_pb.Sort{
					{
						Field: "characteristics.weight",
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
				Sort: []*psml_pb.Sort{
					{
						Field: "characteristics.weight",
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

func TestFooDynamicSorting(t *testing.T) {
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db, uuid.NewString())
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestFooDynamicSorting")
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
							Field:  fmt.Sprintf("foo %d at %s (weighted %d, height %d, length %d)", ii, tt.Format(time.RFC3339Nano), 10+ii, 50-ii, ii%2),
							Weight: ptr.To(10 + int64(ii)),
							Height: ptr.To(50 - int64(ii)),
							Length: ptr.To(int64(ii % 2)),
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

	{
		nextToken := ""
		ss.StepC("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &psml_pb.QueryRequest{
					Sort: []*psml_pb.Sort{
						{
							Field: "createdAt",
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
					Sort: []*psml_pb.Sort{
						{
							Field: "createdAt",
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

	{
		nextToken := ""
		ss.StepC("List Page 1", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &psml_pb.QueryRequest{
					Sort: []*psml_pb.Sort{
						{
							Field: "characteristics.weight",
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
					Sort: []*psml_pb.Sort{
						{
							Field: "characteristics.weight",
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

	{
		nextToken := ""
		ss.StepC("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &psml_pb.QueryRequest{
					Sort: []*psml_pb.Sort{
						{
							Field: "characteristics.length",
						},
						{
							Field: "characteristics.weight",
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
					Sort: []*psml_pb.Sort{
						{
							Field: "characteristics.length",
						},
						{
							Field: "characteristics.weight",
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
	}
}
