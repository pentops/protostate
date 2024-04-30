package integration

import (
	"context"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"github.com/pentops/sqrlx.go/sqrlx"
)

func NewFooStateMachine(db *sqrlx.Wrapper, actorID string) (*testpb.FooPSMDB, error) {
	systemActor, err := psm.NewSystemActor(actorID, &testpb.Actor{
		ActorId: actorID,
	})
	if err != nil {
		return nil, err
	}
	sm, err := testpb.NewFooPSM(testpb.
		DefaultFooPSMConfig().
		StoreEventStateSnapshot().
		SystemActor(systemActor))
	if err != nil {
		return nil, err
	}

	sm.AddHook(testpb.FooPSMGeneralHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		state *testpb.FooState,
		event *testpb.FooEvent,
	) error {

		if state.Characteristics == nil || state.Status != testpb.FooStatus_ACTIVE {
			_, err := tx.Delete(ctx, sq.Delete("foo_cache").Where("id = ?", state.FooId))
			if err != nil {
				return err
			}
			return nil
		}

		_, err := tx.Exec(ctx, sqrlx.Upsert("foo_cache").Key("id", state.FooId).
			Set("weight", state.Characteristics.Weight).
			Set("height", state.Characteristics.Height).
			Set("length", state.Characteristics.Length))
		if err != nil {
			return err
		}

		return nil
	}))

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
			state.CreatedAt = tb.FullCause().Metadata.Timestamp
			state.Profiles = event.Profiles
			return nil
		}))

	sm.From(testpb.FooStatus_ACTIVE).
		Transition(testpb.FooPSMTransition(func(
			ctx context.Context,
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

			return nil
		}))

	sm.From().
		Hook(testpb.FooPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			baton testpb.FooPSMHookBaton,
			state *testpb.FooState,
			event *testpb.FooEventType_Updated,
		) error {
			if event.Delete {
				baton.ChainDerived(&testpb.FooEventType_Deleted{})
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
			return nil
		}))

	return (*testpb.FooPSMDB)(sm.WithDB(db)), nil
}

func NewBarStateMachine(db *sqrlx.Wrapper) (*testpb.BarPSMDB, error) {
	sm, err := testpb.DefaultBarPSMConfig().
		StateTableName("bar").
		EventTableName("bar_event").
		StoreExtraEventColumns(func(event *testpb.BarEvent) (map[string]interface{}, error) {
			return map[string]interface{}{
				"bar_id":    event.BarId,
				"id":        event.Metadata.EventId,
				"timestamp": event.Metadata.Timestamp,
			}, nil
		}).
		PrimaryKey(func(event *testpb.BarEvent) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id": event.BarId,
			}, nil
		}).NewStateMachine()
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
