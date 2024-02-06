package example

import (
	"context"
	"fmt"

	"github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/utils/ptr"
)

func newFooCreatedEvent(fooID, tenantID string, mod func(c *testpb.FooEventType_Created)) *testpb.FooEvent {
	e := newFooEvent(fooID, tenantID, func(e *testpb.FooEvent) {
		weight := int64(10)
		e.Event.Type = &testpb.FooEventType_Created_{
			Created: &testpb.FooEventType_Created{
				Name:        "foo",
				Field:       fmt.Sprintf("weight: %d", weight),
				Description: ptr.To("creation event for foo: " + fooID),
				Weight:      &weight,
			},
		}
	})

	if mod != nil {
		mod(e.Event.GetCreated())
	}

	return e
}

func newFooUpdatedEvent(fooID, tenantID string, mod func(u *testpb.FooEventType_Updated)) *testpb.FooEvent {
	e := newFooEvent(fooID, tenantID, func(e *testpb.FooEvent) {
		weight := int64(20)
		e.Event.Type = &testpb.FooEventType_Updated_{
			Updated: &testpb.FooEventType_Updated{
				Name:        "foo",
				Field:       fmt.Sprintf("weight: %d", weight),
				Description: ptr.To("update event for foo: " + fooID),
				Weight:      &weight,
			},
		}
	})

	if mod != nil {
		mod(e.Event.GetUpdated())
	}

	return e
}

func newFooEvent(fooID, tenantID string, mod func(e *testpb.FooEvent)) *testpb.FooEvent {
	e := &testpb.FooEvent{
		Metadata: &testpb.Metadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
			Actor: &testpb.Actor{
				ActorId: uuid.NewString(),
			},
		},
		FooId:    fooID,
		TenantId: &tenantID,
		Event:    &testpb.FooEventType{},
	}
	mod(e)
	return e
}

func newBarCreatedEvent(barID string, mod func(c *testpb.BarEventType_Created)) *testpb.BarEvent {
	e := newBarEvent(barID, func(e *testpb.BarEvent) {
		e.Event.Type = &testpb.BarEventType_Created_{
			Created: &testpb.BarEventType_Created{
				Name:  "bar",
				Field: "event",
			},
		}
	})

	if mod != nil {
		mod(e.Event.GetCreated())
	}

	return e
}

func newBarEvent(barID string, mod func(e *testpb.BarEvent)) *testpb.BarEvent {
	e := &testpb.BarEvent{
		Metadata: &testpb.StrangeMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		BarId: barID,
		Event: &testpb.BarEventType{},
	}
	mod(e)
	return e
}

type tokenCtxKey struct{}

type token struct {
	tenantID string
}

func (t *token) WithToken(ctx context.Context) context.Context {
	return context.WithValue(ctx, tokenCtxKey{}, t)
}

func TokenFromCtx(ctx context.Context) (*token, error) {
	token, exists := ctx.Value(tokenCtxKey{}).(*token)
	if !exists {
		return nil, fmt.Errorf("no token found")
	}

	return token, nil
}

func silenceLogger() func() {
	defaultLogger := log.DefaultLogger
	log.DefaultLogger = log.NewCallbackLogger(func(level string, msg string, fields map[string]interface{}) {
	})
	return func() {
		log.DefaultLogger = defaultLogger
	}
}

func printQuery(t flowtest.TB, query *sqrl.SelectBuilder) {
	stmt, args, err := query.ToSql()
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(stmt, args)
}

func getRawState(db *sqrlx.Wrapper, id string) (string, error) {
	var state []byte
	err := db.Transact(context.Background(), nil, func(ctx context.Context, tx sqrlx.Transaction) error {
		q := sqrl.Select("state").From("foo").Where("id = ?", id)
		err := tx.QueryRow(ctx, q).Scan(&state)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return string(state), nil
}

func getRawEvent(db *sqrlx.Wrapper, id string) (string, error) {
	var data []byte
	err := db.Transact(context.Background(), nil, func(ctx context.Context, tx sqrlx.Transaction) error {
		q := sqrl.Select("data").From("foo_event").Where("id = ?", id)
		err := tx.QueryRow(ctx, q).Scan(&data)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return string(data), nil
}
