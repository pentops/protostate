package integration

import (
	"context"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/internal/testproto/gen/testpb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
)

func NewFooStateMachine(db *sqrlx.Wrapper) (*testpb.FooPSMDB, error) {
	actorID := uuid.NewString()
	systemActor, err := psm.NewSystemActor(actorID)
	if err != nil {
		return nil, err
	}
	sm, err := testpb.FooPSMBuilder().SystemActor(systemActor).BuildStateMachine()
	if err != nil {
		return nil, err
	}

	sm.StateDataHook(testpb.FooPSMGeneralStateDataHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		state *testpb.FooState,
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
		Mutate(testpb.FooPSMMutation(func(
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

	sm.From(testpb.FooStatus_UNSPECIFIED).
		OnEvent(testpb.FooPSMEventCreated).
		SetStatus(testpb.FooStatus_ACTIVE).
		Mutate(testpb.FooPSMMutation(func(
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

	// Testing Mutate() without OnEvent, the callback implies the event type.
	sm.From(testpb.FooStatus_ACTIVE).
		Mutate(testpb.FooPSMMutation(func(
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
		//OnEvent(testpb.FooPSMEventUpdated).
		LogicHook(testpb.FooPSMLogicHook(func(
			ctx context.Context,
			baton testpb.FooPSMHookBaton,
			state *testpb.FooState,
			event *testpb.FooEventType_Updated,
		) error {
			log.WithFields(ctx, map[string]interface{}{
				"foo_id": state.Keys.FooId,
				"event":  event,
			}).Info("Foo Update Hook")
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
	sm, err := testpb.BarPSMBuilder().BuildStateMachine()
	if err != nil {
		return nil, err
	}

	sm.From(0).
		OnEvent(testpb.BarPSMEventCreated).
		SetStatus(testpb.BarStatus_ACTIVE).
		Mutate(testpb.BarPSMMutation(func(
			data *testpb.BarStateData,
			event *testpb.BarEventType_Created,
		) error {
			data.Name = event.Name
			data.Field = event.Field
			return nil
		}))

	return (*testpb.BarPSMDB)(sm.WithDB(db)), nil
}
