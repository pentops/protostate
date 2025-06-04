package dataload

import (
	"context"
	"fmt"
	"time"

	"buf.build/go/protovalidate"
	"buf.build/go/protoyaml"

	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/timestamppb"

	yaml "gopkg.in/yaml.v3"
)

type DataFile struct {
	Events []EventYaml `yaml:"events"`
}

type EventYaml struct {
	Type  string    `yaml:"type,omitempty"`
	Keys  yaml.Node `yaml:"keys"`
	Event yaml.Node `yaml:"event"`
}

func resolveAliases(node *yaml.Node) error {
	if node.Alias != nil {
		node.Value = node.Alias.Value
		node.Kind = node.Alias.Kind
		node.Alias = nil
	}

	for _, child := range node.Content {
		if err := resolveAliases(child); err != nil {
			return err
		}
	}

	return nil
}

func simplerMap(node *yaml.Node) (map[string]*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("node is not a mapping")
	}

	pairs := map[string]*yaml.Node{}
	for idx := 0; idx < len(node.Content); idx += 2 {
		key, value := node.Content[idx], node.Content[idx+1]
		if key.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("key is not a scalar")
		}
		pairs[key.Value] = value
	}

	return pairs, nil
}

func DecodeInitData(ctx context.Context, dataBytes []byte) ([]proto.Message, error) {
	dataFile := &yaml.Node{}
	if err := yaml.Unmarshal(dataBytes, dataFile); err != nil {
		return nil, err
	}

	if err := resolveAliases(dataFile); err != nil {
		return nil, err
	}

	rootPairs, err := simplerMap(dataFile.Content[0])
	if err != nil {
		return nil, err
	}

	eventNodes, ok := rootPairs["events"]
	if !ok {
		return nil, fmt.Errorf("no events node")
	}

	lastEventTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	nextTime := func() *timestamppb.Timestamp {
		tt := lastEventTime.Add(time.Second)
		lastEventTime = tt
		return timestamppb.New(tt)
	}

	validate, err := protovalidate.New()
	if err != nil {
		return nil, err
	}

	events := make([]proto.Message, 0, len(eventNodes.Content))

	for idx, node := range eventNodes.Content {
		asMap, err := simplerMap(node)
		if err != nil {
			return nil, fmt.Errorf("event %d at line %d: %w", idx, node.Line, err)
		}

		keysNode, ok := asMap["keys"]
		if !ok {
			return nil, fmt.Errorf("event %d at line %d: no keys node", idx, node.Line)
		}

		eventNode, ok := asMap["event"]
		if !ok {
			return nil, fmt.Errorf("event %d at line %d: no event node", idx, node.Line)
		}

		typeNode, ok := asMap["type"]
		if !ok {
			return nil, fmt.Errorf("event %d at line %d: no type node", idx, node.Line)
		}

		eventType := typeNode.Value

		wrapperNode := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "keys",
			}, keysNode, {
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "event",
			}, eventNode},
		}

		metadata := &psm_j5pb.EventMetadata{
			EventId:   uuid.New().String(),
			Timestamp: nextTime(),
		}

		rawData, err := yaml.Marshal(wrapperNode)
		if err != nil {
			return nil, fmt.Errorf("re-marshal data: %w", err)
		}

		msgType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(eventType))
		if err != nil {
			return nil, fmt.Errorf("find message type %s: %w", eventType, err)
		}

		msg := msgType.New()

		evt := msg.Interface()

		if err := protoyaml.Unmarshal(rawData, evt); err != nil {
			return nil, fmt.Errorf("unmarshal protoyaml: %w", err)
		}

		metadataField := msg.Descriptor().Fields().ByName("metadata")
		if metadataField == nil {
			return nil, fmt.Errorf("event %s has no metadata field", msg.Descriptor().FullName())
		}
		msg.Set(metadataField, protoreflect.ValueOfMessage(metadata.ProtoReflect()))

		if err := validate.Validate(evt); err != nil {
			return nil, fmt.Errorf("event %d at line %d: %w", idx, node.Line, err)
		}

		events = append(events, evt)
	}

	return events, nil
}
