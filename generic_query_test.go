package genericstate

import (
	"context"
	"database/sql"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/genericstate/testproto/gen/testpb"
	"github.com/pentops/pgtest.go/pgtest"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.daemonl.com/sqrlx"
)

func TestStateQuery(t *testing.T) {
	ctx := context.Background()

	conn := pgtest.GetTestDB(t, pgtest.WithDir("./testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	tenantID := uuid.NewString()

	testFoo := &testpb.FooState{
		FooId:    uuid.NewString(),
		TenantId: tenantID,
	}

	foo2 := &testpb.FooState{
		FooId:    uuid.NewString(),
		TenantId: tenantID,
	}

	event1 := &testpb.FooEvent{
		EventId: uuid.NewString(),
		Text:    "event1",
	}
	event2 := &testpb.FooEvent{
		EventId: uuid.NewString(),
		Text:    "event2",
	}

	if err := db.Transact(ctx, &sqrlx.TxOptions{
		ReadOnly:  false,
		Isolation: sql.LevelReadCommitted,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {

		for _, foo := range []*testpb.FooState{testFoo, foo2} {
			asBytes, err := protojson.Marshal(foo)
			if err != nil {
				return err
			}
			if _, err := tx.Insert(ctx, sq.
				Insert("foo").
				Columns("id", "tenant_id", "state").
				Values(foo.FooId, foo.TenantId, asBytes)); err != nil {
				return err
			}
		}

		for _, event := range []*testpb.FooEvent{event1, event2} {
			asBytes, err := protojson.Marshal(event)
			if err != nil {
				return err
			}

			if _, err := tx.Insert(ctx, sq.
				Insert("foo_event").
				Columns("id", "foo_id", "data").
				Values(event.EventId, testFoo.FooId, asBytes)); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatal(err.Error())
	}

	config := StateQuerySpec{
		TableName:              "foo",
		DataColumn:             "state",
		PrimaryKeyColumn:       "id",
		PrimaryKeyRequestField: protoreflect.Name("foo_id"),

		Events: &GetJoinSpec{
			TableName:        "foo_event",
			DataColumn:       "data",
			FieldInParent:    "events",
			ForeignKeyColumn: "foo_id",
		},

		Auth: AuthProviderFunc(func(ctx context.Context) (map[string]interface{}, error) {
			return map[string]interface{}{
				"tenant_id": testFoo.TenantId,
			}, nil

		}),

		Get: &MethodDescriptor{
			Request:  (&testpb.GetFooRequest{}).ProtoReflect().Descriptor(),
			Response: (&testpb.GetFooResponse{}).ProtoReflect().Descriptor(),
		},

		List: &MethodDescriptor{
			Request:  (&testpb.ListFoosRequest{}).ProtoReflect().Descriptor(),
			Response: (&testpb.ListFoosResponse{}).ProtoReflect().Descriptor(),
		},

		ListEvents: &MethodDescriptor{
			Request:  (&testpb.ListFooEventsRequest{}).ProtoReflect().Descriptor(),
			Response: (&testpb.ListFooEventsResponse{}).ProtoReflect().Descriptor(),
		},
	}

	queryer, err := NewStateQuery(config)
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Run("Get", func(t *testing.T) {

		req := &testpb.GetFooRequest{
			FooId: testFoo.FooId,
		}

		res := &testpb.GetFooResponse{}

		err = queryer.Get(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		if !proto.Equal(res.State, testFoo) {
			t.Fatalf("expected %v, got %v", testFoo, res.State)
		}

		if len(res.Events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(res.Events))
		}
		t.Log(res.Events)
	})

	t.Run("List", func(t *testing.T) {

		req := &testpb.ListFoosRequest{}
		res := &testpb.ListFoosResponse{}

		err = queryer.List(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Foos) != 2 {
			t.Fatalf("expected 2 states, got %d", len(res.Foos))
		}

	})

	t.Run("ListEvents", func(t *testing.T) {

		req := &testpb.ListFooEventsRequest{}
		res := &testpb.ListFooEventsResponse{}

		err = queryer.ListEvents(ctx, db, req, res)
		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log(protojson.Format(res))
		if len(res.Events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(res.Events))
		}
	})

	// TODO: Test the auth filters

}
