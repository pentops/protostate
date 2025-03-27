package integration

import (
	"context"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type StateMachines struct {
	Foo *test_pb.FooPSMDB
	Bar *test_pb.BarPSMDB
}

func BuildStateMachines(db *sqrlx.Wrapper) (*StateMachines, error) {

	foo, err := NewFooStateMachine(db)
	if err != nil {
		return nil, err
	}

	bar, err := NewBarStateMachine(db)
	if err != nil {
		return nil, err
	}

	foo.From(test_pb.FooStatus_ACTIVE).
		OnEvent(test_pb.FooPSMEventDeleted).
		LinkTo(test_pb.FooPSMLinkHook(bar, func(
			ctx context.Context,
			state *test_pb.FooState,
			event *test_pb.FooEventType_Deleted,
			cb func(*test_pb.BarKeys, test_pb.BarPSMEvent),
		) error {
			cb(&test_pb.BarKeys{
				BarId:      uuid.NewString(),
				BarOtherId: state.Keys.FooId,
			}, &test_pb.BarEventType_Created{
				Name:  state.Data.Name + " Phoenix",
				Field: state.Data.Field,
			})
			return nil
		}))

	return &StateMachines{
		Foo: foo,
		Bar: bar,
	}, nil

}

func NewFooStateMachine(db *sqrlx.Wrapper) (*test_pb.FooPSMDB, error) {
	sm, err := test_pb.FooPSMBuilder().BuildStateMachine()
	if err != nil {
		return nil, err
	}

	sm.StateDataHook(test_pb.FooPSMGeneralStateDataHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		state *test_pb.FooState,
	) error {

		if state.Data.Characteristics == nil || state.Status != test_pb.FooStatus_ACTIVE {
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

	sm.From(test_pb.FooStatus_UNSPECIFIED).
		OnEvent(test_pb.FooPSMEventCreated).
		SetStatus(test_pb.FooStatus_ACTIVE).
		Mutate(test_pb.FooPSMMutation(func(
			data *test_pb.FooData,
			event *test_pb.FooEventType_Created,
		) error {
			data.Name = event.Name
			data.Field = event.Field
			data.Description = event.Description
			data.Characteristics = &test_pb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}
			data.Profiles = event.Profiles
			return nil
		}))

	sm.From(test_pb.FooStatus_UNSPECIFIED).
		OnEvent(test_pb.FooPSMEventCreated).
		SetStatus(test_pb.FooStatus_ACTIVE).
		Mutate(test_pb.FooPSMMutation(func(
			data *test_pb.FooData,
			event *test_pb.FooEventType_Created,
		) error {
			data.Name = event.Name
			data.Field = event.Field
			data.Description = event.Description
			data.Characteristics = &test_pb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}
			data.Profiles = event.Profiles
			return nil
		}))

	// Testing Mutate() without OnEvent, the callback implies the event type.
	sm.From(test_pb.FooStatus_ACTIVE).
		Mutate(test_pb.FooPSMMutation(func(
			data *test_pb.FooData,
			event *test_pb.FooEventType_Updated,
		) error {
			data.Field = event.Field
			data.Name = event.Name
			data.Description = event.Description
			data.Characteristics = &test_pb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}

			return nil
		}))

	sm.From().
		//OnEvent(test_pb.FooPSMEventUpdated).
		LogicHook(test_pb.FooPSMLogicHook(func(
			ctx context.Context,
			baton test_pb.FooPSMHookBaton,
			state *test_pb.FooState,
			event *test_pb.FooEventType_Updated,
		) error {
			log.WithFields(ctx, map[string]interface{}{
				"foo_id": state.Keys.FooId,
				"event":  event,
			}).Info("Foo Update Hook")
			if event.Delete {
				baton.ChainEvent(&test_pb.FooEventType_Deleted{})
			}
			return nil
		}))

	sm.From(test_pb.FooStatus_ACTIVE).
		OnEvent(test_pb.FooPSMEventDeleted).
		SetStatus(test_pb.FooStatus_DELETED)

	return (*test_pb.FooPSMDB)(sm.WithDB(db)), nil
}

func NewBarStateMachine(db *sqrlx.Wrapper) (*test_pb.BarPSMDB, error) {
	sm, err := test_pb.BarPSMBuilder().BuildStateMachine()
	if err != nil {
		return nil, err
	}

	sm.From(0).
		OnEvent(test_pb.BarPSMEventCreated).
		SetStatus(test_pb.BarStatus_ACTIVE).
		Mutate(test_pb.BarPSMMutation(func(
			data *test_pb.BarData,
			event *test_pb.BarEventType_Created,
		) error {
			data.Name = event.Name
			data.Field = event.Field
			return nil
		}))

	return (*test_pb.BarPSMDB)(sm.WithDB(db)), nil
}
