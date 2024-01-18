package main

import (
	"fmt"

	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// attempts to walk through the query methods to find the descriptors for the
// state and event messages.
func deriveStateDescriptorFromQueryDescriptor(src queryServiceDescriptorSet) (*stateEntityDescriptorSet, error) {

	out := &stateEntityDescriptorSet{
		name: src.name,
	}

	if src.getMethod == nil {
		return nil, fmt.Errorf("no get nethod, cannot derive state fields")
	}
	var repeatedEventMessage protoreflect.MessageDescriptor
	getOutFields := src.getMethod.Output().Fields()
	for idx := 0; idx < getOutFields.Len(); idx++ {
		field := getOutFields.Get(idx)
		if field.Kind() != protoreflect.MessageKind {
			continue

		}
		if field.Cardinality() == protoreflect.Repeated {
			if repeatedEventMessage != nil {
				return nil, fmt.Errorf("state get response %s should have exactly one repeated field", src.getMethod.FullName())
			}
			repeatedEventMessage = field.Message()
			continue
		}

		if out.stateMessage != nil {
			return nil, fmt.Errorf("state get response %s should have exactly one field", src.getMethod.FullName())
		}
		out.stateMessage = field.Message()
	}
	if out.stateMessage == nil {
		return nil, fmt.Errorf("state get response should have exactly one field, which is a message")
	}
	out.stateOptions = proto.GetExtension(out.stateMessage.Options(), psm_pb.E_State).(*psm_pb.StateObjectOptions)
	if out.stateOptions == nil {
		// No state, can't add fallbacks.
		return nil, nil
	}

	if repeatedEventMessage != nil {
		eventOptions := proto.GetExtension(repeatedEventMessage.Options(), psm_pb.E_Event).(*psm_pb.EventObjectOptions)
		if eventOptions == nil {
			return nil, nil
		}

		out.eventMessage = repeatedEventMessage
		out.eventOptions = eventOptions

		return out, nil
	}

	if src.listEventsMethod != nil {
		// derive from the list response if available
		listOutFields := src.listEventsMethod.Output().Fields()
		for idx := 0; idx < listOutFields.Len(); idx++ {
			field := listOutFields.Get(idx)
			if field.Kind() != protoreflect.MessageKind {
				continue
			}
			if field.Cardinality() != protoreflect.Repeated {
				continue
			}
			repeatedEventMessage = field.Message()
			break
		}
		if repeatedEventMessage == nil {
			return nil, nil
		}
		eventOptions := proto.GetExtension(repeatedEventMessage.Options(), psm_pb.E_Event).(*psm_pb.EventObjectOptions)
		if eventOptions == nil {
			return nil, nil
		}

		out.eventMessage = repeatedEventMessage
		out.eventOptions = eventOptions

		return out, nil
	}

	// Last ditch effort, try to find the event message in the file
	stateFileMessages := out.stateMessage.ParentFile().Messages()
	for i := 0; i < stateFileMessages.Len(); i++ {
		msg := stateFileMessages.Get(i)
		eventOptions := proto.GetExtension(msg.Options(), psm_pb.E_Event).(*psm_pb.EventObjectOptions)
		if eventOptions == nil {
			continue
		}
		if eventOptions.Name != out.stateOptions.Name {
			continue
		}
		out.eventMessage = msg
		out.eventOptions = eventOptions
		return out, nil
	}

	return nil, nil
}
