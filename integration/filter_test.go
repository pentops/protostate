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

func TestDefaultFiltering(t *testing.T) {
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db, uuid.NewString())
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestDefaultFiltering")
	defer ss.RunSteps(t)

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	tenants := []string{uuid.NewString()}
	tenantIDs := setupFooListableData(t, ss, sm, tenants, 10)

	t.Run("Default Filters", func(t *testing.T) {
		ss.StepC("Setup Extra Statuses", func(ctx context.Context, t flowtest.Asserter) {
			for _, id := range tenantIDs[tenants[0]][:2] {
				event := newFooUpdatedEvent(id, tenants[0], func(u *testpb.FooEventType_Updated) {
					u.Delete = true
				})

				_, err := sm.Transition(ctx, event)
				if err != nil {
					t.Fatal(err.Error())
				}
			}
		})

		ss.StepC("List Page", func(ctx context.Context, t flowtest.Asserter) {
			req := &testpb.ListFoosRequest{
				Page: &psml_pb.PageRequest{
					PageSize: proto.Int64(10),
				},
			}
			res := &testpb.ListFoosResponse{}

			err = queryer.List(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(res.Foos) != 8 {
				t.Fatalf("expected %d states, got %d", 8, len(res.Foos))
			}

			for ii, state := range res.Foos {
				t.Logf("%d: %s", ii, state.Field)
			}

			for ii, state := range res.Foos {
				if state.Characteristics.Weight != int64(12+ii) {
					t.Fatalf("expected weight %d, got %d", 12+ii, state.Characteristics.Weight)
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
	})
}

func SkipTestDynamicFiltering(t *testing.T) {
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewFooStateMachine(db, uuid.NewString())
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := flowtest.NewStepper[*testing.T]("TestDynamicFiltering")
	defer ss.RunSteps(t)

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	tenants := []string{uuid.NewString()}
	setupFooListableData(t, ss, sm, tenants, 30)

	ss.StepC("List Page", func(ctx context.Context, t flowtest.Asserter) {
		req := &testpb.ListFoosRequest{
			Page: &psml_pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &psml_pb.QueryRequest{
				Filters: []*psml_pb.Filter{
					{
						Field: "characteristics.weight",
						Type: &psml_pb.Filter_Range{
							Range: &psml_pb.Range{
								Min: "12",
								Max: "15",
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

		if len(res.Foos) != int(4) {
			t.Fatalf("expected %d states, got %d", 4, len(res.Foos))
		}

		for ii, state := range res.Foos {
			t.Logf("%d: %s", ii, state.Field)
		}

		for ii, state := range res.Foos {
			if state.Characteristics.Weight != int64(12+ii) {
				t.Fatalf("expected weight %d, got %d", 12+ii, state.Characteristics.Weight)
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
