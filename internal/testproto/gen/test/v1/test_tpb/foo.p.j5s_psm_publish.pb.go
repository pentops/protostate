// Code generated by protoc-gen-go-psm. DO NOT EDIT.

package test_tpb

import (
	context "context"
	test_pb "github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	psm "github.com/pentops/protostate/psm"
)

// Publish Toipc for test.v1.Foo
func PublishFoo() psm.GeneralEventHook[
	*test_pb.FooKeys,    // implements psm.IKeyset
	*test_pb.FooState,   // implements psm.IState
	test_pb.FooStatus,   // implements psm.IStatusEnum
	*test_pb.FooData,    // implements psm.IStateData
	*test_pb.FooEvent,   // implements psm.IEvent
	test_pb.FooPSMEvent, // implements psm.IInnerEvent
] {
	return test_pb.FooPSMEventPublishHook(func(
		ctx context.Context,
		publisher psm.Publisher,
		state *test_pb.FooState,
		event *test_pb.FooEvent,
	) error {
		publisher.Publish(&FooEventMessage{
			Metadata: event.EventPublishMetadata(),
			Keys:     event.Keys,
			Event:    event.Event,
			Data:     state.Data,
			Status:   state.Status,
		})
		return nil
	})
}
