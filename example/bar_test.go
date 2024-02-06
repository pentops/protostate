package example

import (
	"context"
	"testing"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"github.com/pentops/sqrlx.go/sqrlx"
)

func NewBarStateMachine(db *sqrlx.Wrapper) (*testpb.BarPSMDB, error) {

	config := testpb.DefaultBarPSMConfig().
		WithTableSpec(testpb.BarPSMTableSpec{
			StateTable: "bar",
			EventTable: "bar_event",
			PrimaryKey: func(event *testpb.BarEvent) (map[string]interface{}, error) {
				return map[string]interface{}{
					"id": event.BarId,
				}, nil
			},
			EventColumns: func(event *testpb.BarEvent) (map[string]interface{}, error) {
				return map[string]interface{}{
					"bar_id":    event.BarId,
					"id":        event.Metadata.EventId,
					"timestamp": event.Metadata.Timestamp,
					"data":      event,
				}, nil
			},
		})

	sm, err := testpb.NewBarPSM(config)
	if err != nil {
		return nil, err
	}

	sm.From(testpb.BarStatus_UNSPECIFIED).
		Where(func(event testpb.BarPSMEvent) bool {
			return true
		}).
		Do(testpb.BarPSMFunc(func(
			ctx context.Context,
			tb testpb.BarPSMTransitionBaton,
			state *testpb.BarState,
			event *testpb.BarEventType_Created,
		) error {
			state.Status = testpb.BarStatus_ACTIVE
			state.Name = event.Name
			state.Field = event.Field
			return nil
		}))

	return (*testpb.BarPSMDB)(sm.WithDB(db)), nil
}

func TestBarStateMachine(t *testing.T) {
	ctx := context.Background()

	conn := pgtest.GetTestDB(t, pgtest.WithDir("../testproto/db"))
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		t.Fatal(err.Error())
	}

	sm, err := NewBarStateMachine(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	barID := uuid.NewString()
	event := newBarCreatedEvent(barID, nil)

	stateOut, err := sm.Transition(ctx, event)
	if err != nil {
		t.Fatal(err.Error())
	}

	if stateOut.GetStatus() != testpb.BarStatus_ACTIVE {
		t.Fatalf("Expect state ACTIVE, got %s", stateOut.GetStatus().ShortString())
	}

}
