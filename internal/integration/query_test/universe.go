package integration

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/flowtest"
	"github.com/pentops/golib/gl"
	"github.com/pentops/j5/lib/id62"
	"github.com/pentops/j5/lib/j5reflect"
	"github.com/pentops/j5/lib/j5schema"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/internal/dbconvert"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/pquery"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
)

func NewStepper(t *testing.T) *flowtest.Stepper[*testing.T] {
	return flowtest.NewStepper[*testing.T](t.Name())
}

type SchemaUniverse struct {
	DB sqrlx.Transactor

	conn *sql.DB
}

func NewSchemaUniverse(t *testing.T, opts ...universeOption) *SchemaUniverse {
	t.Helper()

	spec := &universeSpec{
		opts: psm.StateQueryOptions{},
	}

	for _, opt := range opts {
		opt(spec)
	}

	conn := pgtest.GetTestDB(t, pgtest.WithSchemaName("query_test"))
	db := sqrlx.NewPostgres(conn)

	/*
		smR, err := NewFooStateMachine()
		if err != nil {
			t.Fatal(err.Error())
		}

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

		queryer, err := test_spb.NewFooPSMQuerySet(test_spb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), spec.opts)
		if err != nil {
			t.Fatal(err.Error())
		}
		queryer.SetQueryLogger(testLogger(t))
	*/
	return &SchemaUniverse{
		DB:   db,
		conn: conn,
	}
}

func (uu *SchemaUniverse) Migrate(t flowtest.TB, commands ...string) {
	for _, cmd := range commands {
		if _, err := uu.conn.Exec(cmd); err != nil {
			t.Fatal(err.Error())
		}
	}
}

func SetScalar(obj j5reflect.Object, fieldPath pquery.JSONPathSpec, value any) error {
	field, err := obj.GetOrCreateValue(fieldPath...)
	if err != nil {
		return err
	}
	scalar, ok := field.AsScalar()
	if !ok {
		return fmt.Errorf("field %s is not a scalar", strings.Join(fieldPath, "."))
	}
	if err := scalar.SetGoValue(value); err != nil {
		return err
	}
	return nil
}

type TestObject struct {
	j5reflect.Object
	t flowtest.TB
}

func (to *TestObject) SetScalar(fieldPath pquery.JSONPathSpec, value any) {
	to.t.Helper()
	if err := SetScalar(to.Object, fieldPath, value); err != nil {
		to.t.Fatal(err.Error())
	}
}

func (uu *SchemaUniverse) SetupFoo(t flowtest.TB, count int, callback ...func(int, *TestObject)) {
	uu.Migrate(t, `
		CREATE TABLE foo (
		  foo_id char(36) NOT NULL,
		  state jsonb NOT NULL
	  )`)
	if err := uu.DB.Transact(t.Context(), &sqrlx.TxOptions{
		Isolation: sql.LevelDefault,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		for ii := range count {
			id := id62.NewString()
			foo := &test_pb.FooState{}
			fooRefl := foo.J5Object()
			if err := SetScalar(fooRefl, pquery.JSONPath("fooId"), id); err != nil {
				return err
			}
			if err := SetScalar(fooRefl, pquery.JSONPath("data", "field"), fmt.Sprintf("Foo %d", ii)); err != nil {
				return err
			}

			if err := SetScalar(fooRefl, pquery.JSONPath("status"), "ACTIVE"); err != nil {
				return err
			}

			for _, cb := range callback {
				cb(ii, &TestObject{t: t, Object: fooRefl})
			}

			fooJSON, err := dbconvert.MarshalJ5(fooRefl)
			if err != nil {
				return fmt.Errorf("marshal: %w", err)
			}

			_, err = tx.Insert(ctx, sq.Insert("foo").Columns("foo_id", "state").Values(id, fooJSON))
			if err != nil {
				return err
			}

		}
		return nil
	}); err != nil {
		t.Fatal(err.Error())
	}

}

func (uu *SchemaUniverse) FooLister(t flowtest.TB) *pquery.Lister {
	requestSchema, ok := (&test_spb.FooListRequest{}).J5Object().RootSchema()
	if !ok {
		t.Fatal("failed to get request schema")
	}
	responseSchema, ok := (&test_spb.FooListResponse{}).J5Object().RootSchema()
	if !ok {
		t.Fatal("failed to get response schema")
	}

	method := &j5schema.MethodSchema{
		Request:  requestSchema.(*j5schema.ObjectSchema),
		Response: responseSchema.(*j5schema.ObjectSchema),
	}
	listSpec := pquery.ListSpec{
		TableSpec: pquery.TableSpec{
			TableName:  "foo",
			DataColumn: "state",
			FallbackSortColumns: []pquery.ProtoField{
				pquery.NewJSONField("fooId", gl.Ptr("foo_id")),
			},
		},
		Method: method,
	}
	var err error
	queryer, err := pquery.NewLister(listSpec)
	if err != nil {
		t.Fatalf("failed to create queryer: %w", err)
	}
	queryer.SetQueryLogger(testLogger(t))
	return queryer

}
