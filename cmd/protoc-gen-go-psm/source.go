package main

import (
	"fmt"

	"github.com/pentops/protostate/cmd/protoc-gen-go-psm/query"
	"github.com/pentops/protostate/cmd/protoc-gen-go-psm/state"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
)

// generateSet contains the protogen wrapper around the descriptors

type mappedSourceFile struct {
	stateSets map[string]*state.StateEntityGenerateSet
	querySets map[string]*query.QueryServiceGenerateSet
}

func mapSourceFile(file *protogen.File) (*mappedSourceFile, error) {

	stateSets := state.NewStateEntities()

	source := &mappedSourceFile{
		querySets: map[string]*query.QueryServiceGenerateSet{},
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
				methodSet = query.NewQueryServiceGenerateSet(methodOpt.Name, service.Desc.FullName())
				source.querySets[methodOpt.Name] = methodSet
			}
			if err := methodSet.AddMethod(method, methodOpt); err != nil {
				return nil, fmt.Errorf("adding method %s to %s: %w", method.Desc.Name(), service.Desc.FullName(), err)
			}
		}
	}

	for _, message := range file.Messages {
		if err := stateSets.CheckMessage(message); err != nil {
			return nil, err
		}
	}

	source.stateSets = stateSets.StateMachines

	return source, nil
}
