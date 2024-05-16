package integration

import (
	"context"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"github.com/pentops/sqrlx.go/sqrlx"
)

func NewFooStateMachine(db *sqrlx.Wrapper, actorID string) (*testpb.FooPSMDB, error) {
	systemActor, err := psm.NewSystemActor(actorID)
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

		if state.Data.Characteristics == nil || state.Status != testpb.FooStatus_ACTIVE {
			_, err := tx.Delete(ctx, sq.Delete("foo_cache").Where("id = ?", state.Keys.FooId))
			if err != nil {
				return err
			}
			return nil
		}

		_, err := tx.Exec(ctx, sqrlx.Upsert("foo_cache").Key("id", state.Keys.FooId).
			Set("weight", state.Data.Characteristics.Weight).
			Set("height", state.Data.Characteristics.Height).
			Set("length", state.Data.Characteristics.Length))
		if err != nil {
			return err
		}

		return nil
	}))

	sm.From(testpb.FooStatus_UNSPECIFIED).
		OnEvent(testpb.FooPSMEventCreated).
		SetStatus(testpb.FooStatus_ACTIVE).
		Mutate(testpb.FooPSMTransition(func(
			ctx context.Context,
			data *testpb.FooStateData,
			event *testpb.FooEventType_Created,
		) error {
			data.Name = event.Name
			data.Field = event.Field
			data.Description = event.Description
			data.Characteristics = &testpb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}
			data.Profiles = event.Profiles
			return nil
		})).
		Hook(testpb.FooPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			baton testpb.FooPSMHookBaton,
			state *testpb.FooState,
			event *testpb.FooEventType_Created,
		) error {
			return nil
		}))

	sm.From(testpb.FooStatus_UNSPECIFIED).
		OnEvent(testpb.FooPSMEventCreated).
		SetStatus(testpb.FooStatus_ACTIVE).
		Mutate(testpb.FooPSMTransition(func(
			ctx context.Context,
			data *testpb.FooStateData,
			event *testpb.FooEventType_Created,
		) error {
			data.Name = event.Name
			data.Field = event.Field
			data.Description = event.Description
			data.Characteristics = &testpb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}
			data.Profiles = event.Profiles
			return nil
		}))

	sm.From(testpb.FooStatus_ACTIVE).
		OnEvent(testpb.FooPSMEventUpdated).
		Mutate(testpb.FooPSMTransition(func(
			ctx context.Context,
			data *testpb.FooStateData,
			event *testpb.FooEventType_Updated,
		) error {
			data.Field = event.Field
			data.Name = event.Name
			data.Description = event.Description
			data.Characteristics = &testpb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}

			return nil
		}))

	sm.From().
		OnEvent(testpb.FooPSMEventUpdated).
		Hook(testpb.FooPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			baton testpb.FooPSMHookBaton,
			state *testpb.FooState,
			event *testpb.FooEventType_Updated,
		) error {
			if event.Delete {
				baton.ChainEvent(&testpb.FooEventType_Deleted{})
			}
			return nil
		}))

	sm.From(testpb.FooStatus_ACTIVE).
		OnEvent(testpb.FooPSMEventDeleted).
		SetStatus(testpb.FooStatus_DELETED)

	return (*testpb.FooPSMDB)(sm.WithDB(db)), nil
}

func NewBarStateMachine(db *sqrlx.Wrapper) (*testpb.BarPSMDB, error) {
	sm, err := testpb.DefaultBarPSMConfig().
		StateTableName("bar").
		EventTableName("bar_event").
		StoreExtraEventColumns(func(event *testpb.BarEvent) (map[string]interface{}, error) {
			return map[string]interface{}{
				"bar_id":    event.Keys.BarId,
				"id":        event.Metadata.EventId,
				"timestamp": event.Metadata.Timestamp,
			}, nil
		}).
		PrimaryKey(func(keys *testpb.BarKeys) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id": keys.BarId,
			}, nil
		}).NewStateMachine()
	if err != nil {
		return nil, err
	}

	sm.From(testpb.BarStatus_UNSPECIFIED).
		OnEvent(testpb.BarPSMEventCreated).
		SetStatus(testpb.BarStatus_ACTIVE).
		Mutate(testpb.BarPSMTransition(func(
			ctx context.Context,
			data *testpb.BarStateData,
			event *testpb.BarEventType_Created,
		) error {
			data.Name = event.Name
			data.Field = event.Field
			return nil
		}))

	return (*testpb.BarPSMDB)(sm.WithDB(db)), nil
}
