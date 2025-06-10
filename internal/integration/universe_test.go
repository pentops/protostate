package integration

import (
	"context"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/internal/pgstore/pgmigrate"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type Universe struct {
	FooStateMachine *test_pb.FooPSMDB
	BarStateMachine *test_pb.BarPSMDB

	FooQuery *MiniFooController
	BarQuery *MiniBarController
}

func NewUniverse(t *testing.T) (*flowtest.Stepper[*testing.T], *Universe) {
	name := t.Name()
	stepper := flowtest.NewStepper[*testing.T](name)
	uu := &Universe{}

	stepper.Setup(func(ctx context.Context, t flowtest.Asserter) error {
		log.DefaultLogger = log.NewCallbackLogger(stepper.LevelLog)
		setupUniverse(t, uu)
		return nil
	})

	return stepper, uu
}

func setupUniverse(t flowtest.Asserter, uu *Universe) {
	conn := pgtest.GetTestDB(t)
	_, err := conn.Exec(`
	CREATE TABLE foo_cache (
		id uuid PRIMARY KEY,
		weight int NOT NULL,
		height int NOT NULL,
		length int NOT NULL
	);`)
	t.NoError(err)
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := BuildStateMachines(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	fooQuery, err := test_spb.NewFooPSMQuerySet(test_spb.DefaultFooPSMQuerySpec(sm.Foo.StateTableSpec()), psm.StateQueryOptions{})

	if err != nil {
		t.Fatal(err.Error())
	}

	barQuery, err := test_spb.NewBarPSMQuerySet(test_spb.DefaultBarPSMQuerySpec(sm.Bar.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	uu.FooStateMachine = sm.Foo
	uu.BarStateMachine = sm.Bar
	uu.FooQuery = NewMiniFooController(db, fooQuery)
	uu.BarQuery = NewMiniBarController(db, barQuery)

	if err := pgmigrate.CreateStateMachines(context.Background(), conn,
		sm.Foo.StateTableSpec(),
		sm.Bar.StateTableSpec(),
	); err != nil {
		t.Fatal(err.Error())
	}
}

type MiniFooController struct {
	db    *sqrlx.Wrapper
	query *test_spb.FooPSMQuerySet
}

func NewMiniFooController(db *sqrlx.Wrapper, query *test_spb.FooPSMQuerySet) *MiniFooController {
	return &MiniFooController{
		db:    db,
		query: query,
	}
}

func (c *MiniFooController) FooSummary(ctx context.Context, req *test_spb.FooSummaryRequest) (*test_spb.FooSummaryResponse, error) {

	res := &test_spb.FooSummaryResponse{}

	query := sq.Select("count(id)", "sum(weight)", "sum(height)", "sum(length)").From("foo_cache")
	err := c.db.Transact(ctx, nil, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.QueryRow(ctx, query).Scan(&res.CountFoos, &res.TotalWeight, &res.TotalHeight, &res.TotalLength)
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *MiniFooController) FooGet(ctx context.Context, req *test_spb.FooGetRequest) (*test_spb.FooGetResponse, error) {
	res := &test_spb.FooGetResponse{}
	err := c.query.Get(ctx, c.db, req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *MiniFooController) FooList(ctx context.Context, req *test_spb.FooListRequest) (*test_spb.FooListResponse, error) {
	res := &test_spb.FooListResponse{}
	err := c.query.List(ctx, c.db, req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *MiniFooController) FooEvents(ctx context.Context, req *test_spb.FooEventsRequest) (*test_spb.FooEventsResponse, error) {
	res := &test_spb.FooEventsResponse{}
	err := c.query.ListEvents(ctx, c.db, req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

type MiniBarController struct {
	db    *sqrlx.Wrapper
	query *test_spb.BarPSMQuerySet
}

func NewMiniBarController(db *sqrlx.Wrapper, query *test_spb.BarPSMQuerySet) *MiniBarController {
	return &MiniBarController{
		db:    db,
		query: query,
	}
}

func (b *MiniBarController) BarGet(ctx context.Context, req *test_spb.BarGetRequest) (*test_spb.BarGetResponse, error) {
	res := &test_spb.BarGetResponse{}
	err := b.query.Get(ctx, b.db, req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (b *MiniBarController) BarList(ctx context.Context, req *test_spb.BarListRequest) (*test_spb.BarListResponse, error) {
	res := &test_spb.BarListResponse{}
	err := b.query.List(ctx, b.db, req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (b *MiniBarController) BarEvents(ctx context.Context, req *test_spb.BarEventsRequest) (*test_spb.BarEventsResponse, error) {
	res := &test_spb.BarEventsResponse{}
	err := b.query.ListEvents(ctx, b.db, req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
