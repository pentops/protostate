package integration

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"k8s.io/utils/ptr"
)

func TestMarshaling(t *testing.T) {
	ss, uu := NewFooUniverse(t)
	sm := uu.SM
	db := uu.DB
	queryer := uu.Query
	defer ss.RunSteps(t)

	tenantID := uuid.NewString()

	t.Run("Optional field", func(t *testing.T) {
		fooID := uuid.NewString()

		t.Run("Get with empty", func(t *testing.T) {
			ctx := t.Context()
			event := newFooCreatedEvent(fooID, tenantID, func(c *test_pb.FooEventType_Created) {
				c.Description = ptr.To("")
			})

			_, err := sm.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}

			req := &test_spb.FooGetRequest{
				FooId: fooID,
			}

			res := &test_spb.FooGetResponse{}

			err = queryer.Get(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if res.Foo.Data.Description == nil {
				t.Fatalf("expected description to be non nil")
			}

			if *res.Foo.Data.Description != "" {
				t.Fatalf("expected description to be empty, got %s", *res.Foo.Data.Description)
			}

			stateJSON, err := getRawState(db, fooID)
			if err != nil {
				t.Fatal(err.Error())
			}

			if !strings.Contains(stateJSON, `"description": ""`) {
				t.Fatalf("expected description to be present, but empty: %s", stateJSON)
			}

			eventJSON, err := getRawEvent(db, res.Events[len(res.Events)-1].Metadata.EventId)
			if err != nil {
				t.Fatal(err.Error())
			}

			if !strings.Contains(eventJSON, `"description": ""`) {
				t.Fatalf("expected description to be present, but empty: %s", eventJSON)
			}
		})

		t.Run("Get with non empty", func(t *testing.T) {
			ctx := t.Context()
			event := newFooUpdatedEvent(fooID, tenantID, func(u *test_pb.FooEventType_Updated) {
				u.Description = ptr.To("non blank description")
			})
			_, err := sm.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}

			req := &test_spb.FooGetRequest{
				FooId: fooID,
			}

			res := &test_spb.FooGetResponse{}

			err = queryer.Get(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if res.Foo.Data.Description == nil {
				t.Fatalf("expected description to be non nil")
			}

			if *res.Foo.Data.Description == "" {
				t.Fatalf("expected description to be empty, got %s", *res.Foo.Data.Description)
			}

			stateJSON, err := getRawState(db, fooID)
			if err != nil {
				t.Fatal(err.Error())
			}

			if !strings.Contains(stateJSON, `"description": "non blank description"`) {
				t.Fatalf("expected description to be present, and not empty: %s", stateJSON)
			}

			eventJSON, err := getRawEvent(db, res.Events[len(res.Events)-1].Metadata.EventId)
			if err != nil {
				t.Fatal(err.Error())
			}

			if !strings.Contains(eventJSON, `"description":`) {
				t.Fatalf("expected description to be present: %s", eventJSON)
			}
		})

		t.Run("Get with missing", func(t *testing.T) {
			ctx := t.Context()
			event := newFooUpdatedEvent(fooID, tenantID, func(u *test_pb.FooEventType_Updated) {
				u.Description = nil
			})

			_, err := sm.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}

			req := &test_spb.FooGetRequest{
				FooId: fooID,
			}

			res := &test_spb.FooGetResponse{}

			err = queryer.Get(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if res.Foo.Data.Description != nil {
				t.Fatalf("expected description to be nil")
			}

			stateJSON, err := getRawState(db, fooID)
			if err != nil {
				t.Fatal(err.Error())
			}

			if strings.Contains(stateJSON, `"description":`) {
				t.Fatalf("expected description to not be present: %s", stateJSON)
			}

			eventJSON, err := getRawEvent(db, res.Events[len(res.Events)-1].Metadata.EventId)
			if err != nil {
				t.Fatal(err.Error())
			}

			if strings.Contains(eventJSON, `"description":`) {
				t.Fatalf("expected description to not be present: %s", eventJSON)
			}
		})
	})

	t.Run("Non optional field", func(t *testing.T) {
		ctx := t.Context()
		fooID := uuid.NewString()

		t.Run("Get with empty", func(t *testing.T) {
			event := newFooCreatedEvent(fooID, tenantID, func(c *test_pb.FooEventType_Created) {
				c.Field = ""
				c.Description = nil
			})

			_, err := sm.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}

			req := &test_spb.FooGetRequest{
				FooId: fooID,
			}

			res := &test_spb.FooGetResponse{}

			err = queryer.Get(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if res.Foo.Data.Field != "" {
				t.Fatalf("expected description to be empty")
			}

			stateJSON, err := getRawState(db, fooID)
			if err != nil {
				t.Fatal(err.Error())
			}

			if !strings.Contains(stateJSON, `"field": ""`) {
				t.Fatalf("expected field to be present, but empty: %s", stateJSON)
			}

			eventJSON, err := getRawEvent(db, res.Events[len(res.Events)-1].Metadata.EventId)
			if err != nil {
				t.Fatal(err.Error())
			}

			if !strings.Contains(eventJSON, `"field": ""`) {
				t.Fatalf("expected field to be present, but empty: %s", eventJSON)
			}
		})

		t.Run("Get with non empty", func(t *testing.T) {
			ctx := t.Context()
			event := newFooUpdatedEvent(fooID, tenantID, func(u *test_pb.FooEventType_Updated) {
				u.Field = "non empty"
				u.Description = nil
			})

			_, err := sm.Transition(ctx, event)
			if err != nil {
				t.Fatal(err.Error())
			}

			req := &test_spb.FooGetRequest{
				FooId: fooID,
			}

			res := &test_spb.FooGetResponse{}

			err = queryer.Get(ctx, db, req, res)
			if err != nil {
				t.Fatal(err.Error())
			}

			if res.Foo.Data.Field == "" {
				t.Fatalf("expected description to be non empty")
			}

			stateJSON, err := getRawState(db, fooID)
			if err != nil {
				t.Fatal(err.Error())
			}

			if !strings.Contains(stateJSON, `"field":`) {
				t.Fatalf("expected field to be present: %s", stateJSON)
			}

			eventJSON, err := getRawEvent(db, res.Events[len(res.Events)-1].Metadata.EventId)
			if err != nil {
				t.Fatal(err.Error())
			}

			if !strings.Contains(eventJSON, `"field":`) {
				t.Fatalf("expected field to be present: %s", eventJSON)
			}
		})
	})
}
