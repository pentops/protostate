package publish

import (
	"fmt"

	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func WalkFile(file *protogen.File) ([]*PSMPublishSet, error) {

	sets := make([]*PSMPublishSet, 0)
	for _, service := range file.Services {
		annotation := proto.GetExtension(service.Desc.Options(), messaging_j5pb.E_Service).(*messaging_j5pb.ServiceConfig)
		if annotation == nil {
			continue
		}

		eventAnnotation := annotation.GetEvent()
		if eventAnnotation == nil {
			continue
		}

		parsed, err := parsePublish(service, eventAnnotation)
		if err != nil {
			return nil, fmt.Errorf("parsing publish service %s: %w", service.Desc.FullName(), err)
		}

		sets = append(sets, parsed)

	}
	return sets, nil

}

func parsePublish(service *protogen.Service, eventAnnotation *messaging_j5pb.ServiceConfig_Event) (*PSMPublishSet, error) {
	if eventAnnotation.EntityName == "" {
		return nil, fmt.Errorf("missing entity name in event annotation for service %s", service.Desc.FullName())
	}

	if len(service.Methods) != 1 {
		return nil, fmt.Errorf("publish service %s must have exactly one method", service.Desc.FullName())
	}

	method := service.Methods[0]

	ps := &PSMPublishSet{
		name:           eventAnnotation.EntityName,
		publishMessage: method.Input,
	}
	for _, field := range method.Input.Fields {
		switch field.Desc.Name() {
		case "metadata":
			ps.metadataField = field
		case "keys":
			ps.keyField = field
		case "event":
			ps.eventField = field
		case "data":
			ps.dataField = field
		case "status":
			ps.statusField = field
		}
	}

	if ps.metadataField == nil {
		return nil, fmt.Errorf("missing metadata field in method %s", method.Desc.FullName())
	}
	if ps.metadataField.Desc.Kind() != protoreflect.MessageKind {
		return nil, fmt.Errorf("metadata field must be a message")
	}
	if ps.metadataField.Desc.Message().FullName() != EventPublishMetadata {
		return nil, fmt.Errorf("metadata field must be of type %s", EventPublishMetadata)
	}

	if ps.keyField == nil {
		return nil, fmt.Errorf("missing keys field in method %s", method.Desc.FullName())
	}
	if ps.keyField.Desc.Kind() != protoreflect.MessageKind {
		return nil, fmt.Errorf("keys field must be a message")
	}

	if ps.eventField == nil {
		return nil, fmt.Errorf("missing event field in method %s", method.Desc.FullName())
	}
	if ps.eventField.Desc.Kind() != protoreflect.MessageKind {
		return nil, fmt.Errorf("event field must be a message")
	}

	if ps.dataField == nil {
		return nil, fmt.Errorf("missing data field in method %s", method.Desc.FullName())
	}
	if ps.dataField.Desc.Kind() != protoreflect.MessageKind {
		return nil, fmt.Errorf("data field must be a message")
	}

	if ps.statusField == nil {
		return nil, fmt.Errorf("missing status field in method %s", method.Desc.FullName())
	}
	if ps.statusField.Desc.Kind() != protoreflect.EnumKind {
		return nil, fmt.Errorf("status field must be an enum")
	}

	return ps, nil

	/*
	   j5.state.v1.EventPublishMetadata metadata = 1 [
	   ixc.assets.v1.SecurityKeys keys = 2 [
	   ixc.assets.v1.SecurityEventType event = 3 [
	   ixc.assets.v1.SecurityData data = 4 [
	   ixc.assets.v1.SecurityStatus status = 5 [
	*/
}
