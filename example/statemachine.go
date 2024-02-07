package example

import (
	"context"

	"github.com/pentops/protostate/psm"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"github.com/pentops/sqrlx.go/sqrlx"
)

func NewFooStateMachine(db *sqrlx.Wrapper, actorID string) (*testpb.FooPSMDB, error) {
	customTableSpec := testpb.DefaultFooPSMTableSpec

	systemActor, err := psm.NewSystemActor(actorID, &testpb.Actor{
		ActorId: actorID,
	})
	if err != nil {
		return nil, err
	}
	sm, err := testpb.NewFooPSM(testpb.
		DefaultFooPSMConfig().
		WithTableSpec(customTableSpec).
		WithSystemActor(systemActor))
	if err != nil {
		return nil, err
	}

	sm.From(testpb.FooStatus_UNSPECIFIED).
		Do(testpb.FooPSMFunc(func(
			ctx context.Context,
			tb testpb.FooPSMTransitionBaton,
			state *testpb.FooState,
			event *testpb.FooEventType_Created,
		) error {
			state.Status = testpb.FooStatus_ACTIVE
			state.Name = event.Name
			state.Field = event.Field
			state.Description = event.Description
			state.Characteristics = &testpb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}
			state.LastEventId = tb.FullCause().Metadata.EventId
			state.CreatedAt = tb.FullCause().Metadata.Timestamp
			return nil
		}))

	sm.From(testpb.FooStatus_ACTIVE).
		Do(testpb.FooPSMFunc(func(
			ctx context.Context,
			tb testpb.FooPSMTransitionBaton,
			state *testpb.FooState,
			event *testpb.FooEventType_Updated,
		) error {
			state.Field = event.Field
			state.Name = event.Name
			state.Description = event.Description
			state.Characteristics = &testpb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}
			state.LastEventId = tb.FullCause().Metadata.EventId

			if event.Delete {
				tb.ChainDerived(&testpb.FooEventType_Deleted{})
			}
			return nil
		}))

	sm.From(testpb.FooStatus_ACTIVE).
		Do(testpb.FooPSMFunc(func(
			ctx context.Context,
			tb testpb.FooPSMTransitionBaton,
			state *testpb.FooState,
			event *testpb.FooEventType_Deleted,
		) error {
			state.Status = testpb.FooStatus_DELETED
			state.LastEventId = tb.FullCause().Metadata.EventId
			return nil
		}))

	return (*testpb.FooPSMDB)(sm.WithDB(db)), nil
}

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
