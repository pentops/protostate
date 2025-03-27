// Code generated by protoc-gen-go-o5-messaging. DO NOT EDIT.
// versions:
// - protoc-gen-go-o5-messaging 0.0.0
// source: test/v1/topic/bar.p.j5s.proto

package test_tpb

import (
	context "context"
	messaging_pb "github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_pb"
	o5msg "github.com/pentops/o5-messaging/o5msg"
)

// Service: BarPublishTopic
// Method: BarEvent

func (msg *BarEventMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "test.v1.topic.BarPublishTopic",
		GrpcMethod:       "BarEvent",
		Headers:          map[string]string{},
		DestinationTopic: "bar_publish",
	}
	header.Extension = &messaging_pb.Message_Event_{
		Event: &messaging_pb.Message_Event{
			EntityName: "test.v1.Bar",
		},
	}
	return header
}

type BarPublishTopicTxSender[C any] struct {
	sender o5msg.TxSender[C]
}

func NewBarPublishTopicTxSender[C any](sender o5msg.TxSender[C]) *BarPublishTopicTxSender[C] {
	sender.Register(o5msg.TopicDescriptor{
		Service: "test.v1.topic.BarPublishTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "BarEvent",
				Message: (*BarEventMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &BarPublishTopicTxSender[C]{sender: sender}
}

type BarPublishTopicCollector[C any] struct {
	collector o5msg.Collector[C]
}

func NewBarPublishTopicCollector[C any](collector o5msg.Collector[C]) *BarPublishTopicCollector[C] {
	collector.Register(o5msg.TopicDescriptor{
		Service: "test.v1.topic.BarPublishTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "BarEvent",
				Message: (*BarEventMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &BarPublishTopicCollector[C]{collector: collector}
}

type BarPublishTopicPublisher struct {
	publisher o5msg.Publisher
}

func NewBarPublishTopicPublisher(publisher o5msg.Publisher) *BarPublishTopicPublisher {
	publisher.Register(o5msg.TopicDescriptor{
		Service: "test.v1.topic.BarPublishTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "BarEvent",
				Message: (*BarEventMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &BarPublishTopicPublisher{publisher: publisher}
}

// Method: BarEvent

func (send BarPublishTopicTxSender[C]) BarEvent(ctx context.Context, sendContext C, msg *BarEventMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect BarPublishTopicCollector[C]) BarEvent(sendContext C, msg *BarEventMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish BarPublishTopicPublisher) BarEvent(ctx context.Context, msg *BarEventMessage) error {
	return publish.publisher.Publish(ctx, msg)
}
