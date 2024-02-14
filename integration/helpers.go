package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"k8s.io/utils/ptr"
)

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

func setupFooListableData(t *testing.T, ss *flowtest.Stepper[*testing.T], sm *testpb.FooPSMDB, tenants []string, count int) []string {
	ids := make([]string, 0, count)

	ss.StepC("Create", func(ctx context.Context, a flowtest.Asserter) {
		for ti := range tenants {
			tkn := &token{
				tenantID: tenants[ti],
			}
			ctx = tkn.WithToken(ctx)

			restore := silenceLogger()
			defer restore()

			for ii := 0; ii < count; ii++ {
				tt := time.Now()
				fooID := uuid.NewString()
				ids = append(ids, fooID)

				event := newFooCreatedEvent(fooID, tenants[ti], func(c *testpb.FooEventType_Created) {
					c.Field = fmt.Sprintf("foo %d at %s (weighted %d, height %d, length %d)", ii, tt.Format(time.RFC3339Nano), (10+ii)*(ti+1), (50-ii)*(ti+1), (ii%2)*(ti+1))
					c.Weight = ptr.To((10 + int64(ii)) * (int64(ti) + 1))
					c.Height = ptr.To((50 - int64(ii)) * (int64(ti) + 1))
					c.Length = ptr.To((int64(ii%2) * (int64(ti) + 1)))
				})

				stateOut, err := sm.Transition(ctx, event)
				if err != nil {
					t.Fatal(err.Error())
				}
				a.Equal(testpb.FooStatus_ACTIVE, stateOut.Status)
				a.Equal(tenants[ti], *stateOut.TenantId)
			}
		}
	})

	return ids
}
