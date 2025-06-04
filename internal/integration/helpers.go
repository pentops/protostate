package integration

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/j5/gen/j5/auth/v1/auth_j5pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"k8s.io/utils/ptr"
)

func silenceLogger() func() {
	defaultLogger := log.DefaultLogger
	log.DefaultLogger = log.NewCallbackLogger(func(level string, msg string, fields []slog.Attr) {
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
		q := sqrl.Select("state").From("foo").Where("foo_id = ?", id)
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

func setupFooListableData(ss *flowtest.Stepper[*testing.T], sm *test_pb.FooPSMDB, tenants []string, count int) map[string][]string {
	ids := make(map[string][]string, len(tenants))

	for ti := range tenants {
		ids[tenants[ti]] = make([]string, 0, count)
		for range count {
			ids[tenants[ti]] = append(ids[tenants[ti]], uuid.NewString())
		}
	}

	ss.Step("Create", func(ctx context.Context, t flowtest.Asserter) {
		ti := 0
		for tenant, fooIDs := range ids {
			tkn := &token{
				claim: &auth_j5pb.Claim{
					TenantType: "tenant",
					TenantId:   tenant,
				},
			}
			ctx = tkn.WithToken(ctx)

			restore := silenceLogger()
			defer restore()

			for ii, fooID := range fooIDs {
				tt := time.Now()

				event := newFooCreatedEvent(fooID, tenants[ti], func(c *test_pb.FooEventType_Created) {
					c.Field = fmt.Sprintf("foo %d at %s (weighted %d, height %d, length %d)", ii, tt.Format(time.RFC3339Nano), (10+ii)*(ti+1), (50-ii)*(ti+1), (ii%2)*(ti+1))
					c.Weight = ptr.To((10 + int64(ii)) * (int64(ti) + 1))
					c.Height = ptr.To((50 - int64(ii)) * (int64(ti) + 1))
					c.Length = ptr.To((int64(ii%2) * (int64(ti) + 1)))
					c.Profiles = []*test_pb.FooProfile{
						{
							Name:  fmt.Sprintf("profile %d", ii),
							Place: int64(ii) + 50,
						},
						{
							Name:  fmt.Sprintf("profile %d", ii),
							Place: int64(ii) + 15,
						},
					}
				})

				stateOut, err := sm.Transition(ctx, event)
				if err != nil {
					t.Fatalf("setup foo: %s (%#v)", err.Error(), event.Keys)
				}
				t.Equal(test_pb.FooStatus_ACTIVE, stateOut.Status)
				t.Equal(tenants[ti], *stateOut.Keys.TenantId)
			}

			ti++
		}
	})

	return ids
}
