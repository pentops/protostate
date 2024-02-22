package main

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type metadataDescriptor struct {
	id        protoreflect.FieldDescriptor
	timestamp protoreflect.FieldDescriptor
	sequence  protoreflect.FieldDescriptor
	actor     protoreflect.FieldDescriptor
}

func metadataFields(md protoreflect.MessageDescriptor) metadataDescriptor {
	out := metadataDescriptor{}
	for idx := 0; idx < md.Fields().Len(); idx++ {
		field := md.Fields().Get(idx)
		switch field.Name() {
		case "actor":
			if field.Kind() != protoreflect.MessageKind {
				break // This will not match
			}
			out.actor = field
		case "timestamp":
			if field.Kind() != protoreflect.MessageKind || field.Message().FullName() != protoreflect.FullName("google.protobuf.Timestamp") {
				break // This will not match
			}
			out.timestamp = field

		case "event_id", "message_id", "id":
			if field.Kind() != protoreflect.StringKind {
				break // This will not match
			}
			out.id = field

		case "sequence":
			if field.Kind() != protoreflect.Uint64Kind {
				break // This will not match
			}
			out.sequence = field
		}
	}
	return out
}

type metadataGenerators struct {
	id        *protogen.Field
	timestamp *protogen.Field
	sequence  *protogen.Field
	actor     *protogen.Field
}

func (md metadataDescriptor) toGenerators(mdMsg *protogen.Message) metadataGenerators {
	return metadataGenerators{
		actor:     mapGenField(mdMsg, md.actor),
		timestamp: mapGenField(mdMsg, md.timestamp),
		id:        mapGenField(mdMsg, md.id),
		sequence:  mapGenField(mdMsg, md.sequence),
	}
}
