package integration

import (
	"context"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/internal/testproto/gen/testpb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type Universe struct {
	FooStateMachine *testpb.FooPSMDB
	BarStateMachine *testpb.BarPSMDB

	FooQuery *MiniFooController
}

func NewUniverse(t *testing.T) (*flowtest.Stepper[*testing.T], *Universe) {
	name := t.Name()
	stepper := flowtest.NewStepper[*testing.T](name)
	uu := &Universe{}

	stepper.Setup(func(ctx context.Context, t flowtest.Asserter) error {
		log.DefaultLogger = log.NewCallbackLogger(stepper.LevelLog)
		setupUniverse(ctx, t, uu)
		return nil
	})

	return stepper, uu
}

func setupUniverse(ctx context.Context, t flowtest.Asserter, uu *Universe) {

	conn := pgtest.GetTestDB(t, pgtest.WithDir(allMigrationsDir))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	fooState, err := NewFooStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	barState, err := NewBarStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(fooState.StateTableSpec()), psm.StateQueryOptions{})

	if err != nil {
		t.Fatal(err.Error())
	}

	uu.FooStateMachine = fooState
	uu.BarStateMachine = barState
	uu.FooQuery = NewMiniFooController(db, queryer)
}

type MiniFooController struct {
	db    *sqrlx.Wrapper
	query *testpb.FooPSMQuerySet
}

func NewMiniFooController(db *sqrlx.Wrapper, query *testpb.FooPSMQuerySet) *MiniFooController {
	return &MiniFooController{
		db:    db,
		query: query,
	}
}

func (c *MiniFooController) FooSummary(ctx context.Context, req *testpb.FooSummaryRequest) (*testpb.FooSummaryResponse, error) {

	res := &testpb.FooSummaryResponse{}

	query := sq.Select("count(id)", "sum(weight)", "sum(height)", "sum(length)").From("foo_cache")
	err := c.db.Transact(ctx, nil, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.QueryRow(ctx, query).Scan(&res.CountFoos, &res.TotalWeight, &res.TotalHeight, &res.TotalLength)
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *MiniFooController) GetFoo(ctx context.Context, req *testpb.GetFooRequest) (*testpb.GetFooResponse, error) {
	res := &testpb.GetFooResponse{}
	err := c.query.Get(ctx, c.db, req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *MiniFooController) ListFoos(ctx context.Context, req *testpb.ListFoosRequest) (*testpb.ListFoosResponse, error) {
	res := &testpb.ListFoosResponse{}
	err := c.query.List(ctx, c.db, req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *MiniFooController) ListFooEvents(ctx context.Context, req *testpb.ListFooEventsRequest) (*testpb.ListFooEventsResponse, error) {
	res := &testpb.ListFooEventsResponse{}
	err := c.query.ListEvents(ctx, c.db, req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
