package main

import (
	"fmt"

	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// stateEntityDescriptorSet contains the protoreflect descriptors for the state
// entity and its events. It cannot be used to generate code, see
// stateEntityGenerateSet for the wrapped version.
type stateEntityDescriptorSet struct {
	name         string
	stateMessage protoreflect.MessageDescriptor
	stateOptions *psm_pb.StateObjectOptions
	eventMessage protoreflect.MessageDescriptor
	eventOptions *psm_pb.EventObjectOptions
}

type stateEntityGenerateSet struct {
	// name of the state machine
	name string
	// for errors / debugging, includes the proto source name
	fullName string

	stateMessage *protogen.Message
	stateOptions *psm_pb.StateObjectOptions
	eventMessage *protogen.Message
	eventOptions *psm_pb.EventObjectOptions
}

func (gs stateEntityGenerateSet) descriptors() stateEntityDescriptorSet {
	return stateEntityDescriptorSet{
		name:         gs.name,
		stateMessage: gs.stateMessage.Desc,
		stateOptions: gs.stateOptions,
		eventMessage: gs.eventMessage.Desc,
		eventOptions: gs.eventOptions,
	}
}

// descriptorSet contains the protoreflect descriptors for the query service
type queryServiceDescriptorSet struct {
	name             string
	getMethod        protoreflect.MethodDescriptor
	listMethod       protoreflect.MethodDescriptor
	listEventsMethod protoreflect.MethodDescriptor
}

// generateSet contains the protogen wrapper around the descriptors
type queryServiceGenerateSet struct {

	// name of the state machine
	name string
	// for errors / debugging, includes the proto source name
	fullName string

	getMethod        *protogen.Method
	listMethod       *protogen.Method
	listEventsMethod *protogen.Method
}

func (qs queryServiceGenerateSet) validate() error {
	if qs.getMethod == nil {
		return fmt.Errorf("PSM Query '%s' does not have a get method", qs.fullName)
	}

	if qs.listMethod == nil {
		return fmt.Errorf("PSM Qurey '%s' does not have a list method", qs.fullName)
	}

	return nil

}

func (gs queryServiceGenerateSet) descriptors() queryServiceDescriptorSet {
	qds := queryServiceDescriptorSet{
		name:       gs.name,
		getMethod:  gs.getMethod.Desc,
		listMethod: gs.listMethod.Desc,
	}
	if gs.listEventsMethod != nil {
		qds.listEventsMethod = gs.listEventsMethod.Desc
	}
	return qds
}

type mappedSourceFile struct {
	stateSets map[string]*stateEntityGenerateSet
	querySets map[string]*queryServiceGenerateSet
}

func mapSourceFile(file *protogen.File) (*mappedSourceFile, error) {
	source := &mappedSourceFile{
		stateSets: map[string]*stateEntityGenerateSet{},
		querySets: map[string]*queryServiceGenerateSet{},
	}

	for _, service := range file.Services {
		stateQueryAnnotation := proto.GetExtension(service.Desc.Options(), psm_pb.E_StateQuery).(*psm_pb.StateQueryServiceOptions)

		for _, method := range service.Methods {
			methodOpt := proto.GetExtension(method.Desc.Options(), psm_pb.E_StateQueryMethod).(*psm_pb.StateQueryMethodOptions)
			if methodOpt == nil {
				continue
			}
			if methodOpt.Name == "" {
				if stateQueryAnnotation == nil || stateQueryAnnotation.Name == "" {
					return nil, fmt.Errorf("service %s method %s does not have a state query name, and no service default", service.GoName, method.GoName)
				}
				methodOpt.Name = stateQueryAnnotation.Name
			}

			methodSet, ok := source.querySets[methodOpt.Name]
			if !ok {
				methodSet = &queryServiceGenerateSet{
					name:     methodOpt.Name,
					fullName: fmt.Sprintf("%s/%s", service.Desc.FullName(), methodOpt.Name),
				}
				source.querySets[methodOpt.Name] = methodSet
			}

			if methodOpt.Get {
				methodSet.getMethod = method
			} else if methodOpt.List {
				methodSet.listMethod = method
			} else if methodOpt.ListEvents {
				methodSet.listEventsMethod = method
			} else {
				return nil, fmt.Errorf("service %s method %s does not have a state query type", service.GoName, method.GoName)
			}
		}

	}

	for _, message := range file.Messages {
		stateObjectAnnotation, ok := proto.GetExtension(message.Desc.Options(), psm_pb.E_State).(*psm_pb.StateObjectOptions)
		if ok && stateObjectAnnotation != nil {
			stateSet, ok := source.stateSets[stateObjectAnnotation.Name]
			if !ok {
				stateSet = &stateEntityGenerateSet{
					name:     stateObjectAnnotation.Name,
					fullName: fmt.Sprintf("%s/%s", message.Desc.ParentFile().FullName(), stateObjectAnnotation.Name),
				}
				source.stateSets[stateObjectAnnotation.Name] = stateSet
			} else if stateSet.stateMessage != nil || stateSet.stateOptions != nil {
				return nil, fmt.Errorf("duplicate state object name %s", stateObjectAnnotation.Name)
			}

			stateSet.stateMessage = message
			stateSet.stateOptions = stateObjectAnnotation
		}

		eventObjectAnnotation, ok := proto.GetExtension(message.Desc.Options(), psm_pb.E_Event).(*psm_pb.EventObjectOptions)
		if ok && eventObjectAnnotation != nil {
			ss, ok := source.stateSets[eventObjectAnnotation.Name]
			if !ok {
				ss = &stateEntityGenerateSet{
					name:     eventObjectAnnotation.Name,
					fullName: fmt.Sprintf("%s/%s", message.Desc.ParentFile().FullName(), eventObjectAnnotation.Name),
				}
				source.stateSets[eventObjectAnnotation.Name] = ss
			} else if ss.eventMessage != nil || ss.eventOptions != nil {
				return nil, fmt.Errorf("duplicate event object name %s", eventObjectAnnotation.Name)
			}

			ss.eventMessage = message
			ss.eventOptions = eventObjectAnnotation

		}
	}

	return source, nil
}
