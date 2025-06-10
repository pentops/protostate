package integration

import (
	"context"
	"testing"

	"github.com/pentops/flowtest"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/internal/pgstore/pgmigrate"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/pquery"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type Universe struct {
	SM    *test_pb.FooPSMDB
	Query *test_spb.FooPSMQuerySet
	DB    sqrlx.Transactor
}

type universeSpec struct {
	opts psm.StateQueryOptions
}

type universeOption func(*universeSpec)

func WithStateQueryOptions(opts psm.StateQueryOptions) universeOption {
	return func(us *universeSpec) {
		us.opts = opts
	}
}

func NewFooUniverse(t *testing.T, opts ...universeOption) (*flowtest.Stepper[*testing.T], *Universe) {
	t.Helper()

	spec := &universeSpec{
		opts: psm.StateQueryOptions{},
	}

	for _, opt := range opts {
		opt(spec)
	}

	smR, err := NewFooStateMachine()
	if err != nil {
		t.Fatal(err.Error())
	}

	conn := pgtest.GetTestDB(t, pgtest.WithSchemaName("query_test"))
	db := sqrlx.NewPostgres(conn)

	specs := []psm.QueryTableSpec{
		smR.StateTableSpec(),
	}

	if err := pgmigrate.CreateStateMachines(context.Background(), conn, specs...); err != nil {
		t.Fatal(err.Error())
	}

	if err := pgmigrate.AddIndexes(context.Background(), conn, specs...); err != nil {
		t.Fatal(err.Error())
	}

	sm := smR.WithDB(db)

	ss := flowtest.NewStepper[*testing.T](t.Name())
	defer ss.RunSteps(t)

	queryer, err := test_spb.NewFooPSMQuerySet(test_spb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), spec.opts)
	if err != nil {
		t.Fatal(err.Error())
	}
	queryer.SetQueryLogger(testLogger(t))

	return ss, &Universe{
		DB:    db,
		SM:    sm,
		Query: queryer,
	}
}

func (uu *Universe) ListFoo(t flowtest.Asserter, req *test_spb.FooListRequest) *test_spb.FooListResponse {
	t.Helper()
	ctx := context.Background()

	resp := &test_spb.FooListResponse{}
	err := uu.Query.List(ctx, uu.DB, req, resp)
	if err != nil {
		t.Fatal(err.Error())
	}

	return resp
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
