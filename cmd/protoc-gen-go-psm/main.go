package main

import (
	"flag"
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/iancoleman/strcase"
)

var Version = "1.0"

var (
	smImportPath            = protogen.GoImportPath("github.com/pentops/protostate/psm")
	smEventer               = smImportPath.Ident("Eventer")
	smStateMachine          = smImportPath.Ident("StateMachine")
	smDBStateMachine        = smImportPath.Ident("DBStateMachine")
	smTableSpec             = smImportPath.Ident("TableSpec")
	smPSMTableSpec          = smImportPath.Ident("PSMTableSpec")
	smStateMachineConfig    = smImportPath.Ident("StateMachineConfig")
	smNewStateMachineConfig = smImportPath.Ident("NewStateMachineConfig")
	smNewStateMachine       = smImportPath.Ident("NewStateMachine")
	smTransitionBaton       = smImportPath.Ident("TransitionBaton")
	smStateHookBaton        = smImportPath.Ident("StateHookBaton")
	smTransitionFunc        = smImportPath.Ident("PSMTransitionFunc")
	smHookFunc              = smImportPath.Ident("PSMHookFunc")
	smGeneralHookFunc       = smImportPath.Ident("GeneralStateHook")
	smCombinedFunc          = smImportPath.Ident("PSMCombinedFunc")
	smSystemActor           = smImportPath.Ident("SystemActor")

	psmProtoImportPath     = protogen.GoImportPath("github.com/pentops/protostate/gen/state/v1/psm_pb")
	psmEventMetadataStruct = psmProtoImportPath.Ident("EventMetadata")
	psmStateMetadataStruct = psmProtoImportPath.Ident("StateMetadata")

	eventMetadataProtoName = protoreflect.FullName("psm.state.v1.EventMetadata")
	stateMetadataProtoName = protoreflect.FullName("psm.state.v1.StateMetadata")
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("protoc-gen-go-psm %v\n", Version)
		return
	}

	var flags flag.FlagSet

	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			generateFile(gen, f)
		}
		return nil
	})
}

func stateGoName(specified string) string {
	// return the exported go name for the string, which is provided as _ case
	// separated
	return strcase.ToCamel(specified)
}

// generateFile generates a _psm.pb.go
func generateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {

	sourceMap, err := mapSourceFile(file)
	if err != nil {
		gen.Error(err)
		return nil
	}

	if len(sourceMap.stateSets) == 0 && len(sourceMap.querySets) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_psm.pb.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	g.P("// Code generated by protoc-gen-go-psm. DO NOT EDIT.")
	g.P()
	g.P("package ", file.GoPackageName)
	g.P()

	// Build the state sets from the source file
	for _, stateSet := range sourceMap.stateSets {
		converted, err := buildStateSet(*stateSet)
		if err != nil {
			gen.Error(err)
			continue
		}

		if err := converted.write(g); err != nil {
			gen.Error(err)
		}
	}

	for _, querySrc := range sourceMap.querySets {
		converted, err := buildQuerySet(*querySrc)
		if err != nil {
			gen.Error(err)
			continue

		}
		if err := converted.write(g); err != nil {
			gen.Error(err)
		}

	}

	return g
}

func mapGenField(parent *protogen.Message, field protoreflect.FieldDescriptor) *protogen.Field {
	if field == nil {
		return nil
	}
	for _, f := range parent.Fields {
		if f.Desc.FullName() == field.FullName() {
			return f
		}
	}
	panic(fmt.Sprintf("field %s not found in parent %s", field.FullName(), parent.Desc.FullName()))
}

func fieldByName(msg *protogen.Message, name protoreflect.Name) *protogen.Field {
	for _, field := range msg.Fields {
		if field.Desc.Name() == name {
			return field
		}
	}
	return nil
}

func descriptorTypesMatch(a, b protoreflect.FieldDescriptor) error {
	if a.Kind() != b.Kind() {
		return fmt.Errorf("different kind")
	}

	if a.HasOptionalKeyword() != b.HasOptionalKeyword() {
		return fmt.Errorf("different optional")
	}

	if a.Cardinality() != b.Cardinality() {
		return fmt.Errorf("different cardinality")
	}

	return nil
}
