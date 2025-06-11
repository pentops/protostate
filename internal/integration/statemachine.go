package integration

import (
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type StateMachines struct {
	Foo *test_pb.FooPSMDB
	Bar *test_pb.BarPSMDB
}

func BuildStateMachines(db *sqrlx.Wrapper) (*StateMachines, error) {

	foo, err := NewFooStateMachine()
	if err != nil {
		return nil, err
	}

	bar, err := NewBarStateMachine()
	if err != nil {
		return nil, err
	}

	return &StateMachines{
		Foo: foo.WithDB(db),
		Bar: bar.WithDB(db),
	}, nil

}

func NewFooStateMachine() (*test_pb.FooPSM, error) {
	sm, err := test_pb.FooPSMBuilder().BuildStateMachine()
	if err != nil {
		return nil, err
	}

	sm.From(0).
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

	sm.From(test_pb.FooStatus_ACTIVE).
		OnEvent(test_pb.FooPSMEventDeleted).
		SetStatus(test_pb.FooStatus_DELETED)

	return sm, nil

}

func NewBarStateMachine() (*test_pb.BarPSM, error) {
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

	return sm, nil
}
