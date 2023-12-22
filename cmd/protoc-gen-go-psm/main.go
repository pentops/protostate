package main

import (
	"flag"
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/iancoleman/strcase"
	"github.com/pentops/protostate/gen/v1/psm_pb"
)

const version = "1.0"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("protoc-gen-go-psm %v\n", version)
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

type buildingStateSet struct {
	stateMessage *protogen.Message
	stateOptions *psm_pb.StateObjectOptions
	eventMessage *protogen.Message
	eventOptions *psm_pb.EventObjectOptions
}

type buildingStateQueryService struct {
	name string

	getMethod        *protogen.Method
	listMethod       *protogen.Method
	listEventsMethod *protogen.Method
}

func stateGoName(specified string) string {
	// return the exported go name for the string, which is provided as _ case
	// separated
	return strcase.ToCamel(specified)
}

// generateFile generates a _psm.pb.go
func generateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {

	stateServices := map[string]*buildingStateQueryService{}
	for _, service := range file.Services {
		stateQueryAnnotation := proto.GetExtension(service.Desc.Options(), psm_pb.E_StateQuery).(*psm_pb.StateQueryServiceOptions)

		for _, method := range service.Methods {
			methodOpt := proto.GetExtension(method.Desc.Options(), psm_pb.E_StateQueryMethod).(*psm_pb.StateQueryMethodOptions)
			if methodOpt == nil {
				continue
			}
			if methodOpt.Name == "" {
				if stateQueryAnnotation == nil || stateQueryAnnotation.Name == "" {
					gen.Error(fmt.Errorf("service %s method %s does not have a state query name, and no service default", service.GoName, method.GoName))
					return nil
				}
				methodOpt.Name = stateQueryAnnotation.Name
			}

			methodSet, ok := stateServices[methodOpt.Name]
			if !ok {
				methodSet = &buildingStateQueryService{
					name: methodOpt.Name,
				}
				stateServices[methodOpt.Name] = methodSet
			}

			if methodOpt.Get {
				methodSet.getMethod = method
			} else if methodOpt.List {
				methodSet.listMethod = method
			} else if methodOpt.ListEvents {
				methodSet.listEventsMethod = method
			} else {
				gen.Error(fmt.Errorf("service %s method %s does not have a state query type", service.GoName, method.GoName))
				return nil
			}
		}

	}

	stateSets := make(map[string]*buildingStateSet)

	for _, message := range file.Messages {
		stateObjectAnnotation, ok := proto.GetExtension(message.Desc.Options(), psm_pb.E_State).(*psm_pb.StateObjectOptions)
		if ok && stateObjectAnnotation != nil {
			stateSet, ok := stateSets[stateObjectAnnotation.Name]
			if !ok {
				stateSet = &buildingStateSet{}
				stateSets[stateObjectAnnotation.Name] = stateSet
			} else if stateSet.stateMessage != nil || stateSet.stateOptions != nil {
				gen.Error(fmt.Errorf("duplicate state object name %s", stateObjectAnnotation.Name))
				continue
			}

			stateSet.stateMessage = message
			stateSet.stateOptions = stateObjectAnnotation
		}

		eventObjectAnnotation, ok := proto.GetExtension(message.Desc.Options(), psm_pb.E_Event).(*psm_pb.EventObjectOptions)
		if ok && eventObjectAnnotation != nil {
			ss, ok := stateSets[eventObjectAnnotation.Name]
			if !ok {
				ss = &buildingStateSet{}
				stateSets[eventObjectAnnotation.Name] = ss
			} else if ss.eventMessage != nil || ss.eventOptions != nil {
				gen.Error(fmt.Errorf("duplicate event object name %s", eventObjectAnnotation.Name))
				continue
			}

			ss.eventMessage = message
			ss.eventOptions = eventObjectAnnotation

		}
	}

	if len(stateSets) == 0 && len(stateServices) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_psm.pb.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	g.P("// Code generated by protoc-gen-go-psm. DO NOT EDIT.")
	g.P()
	g.P("package ", file.GoPackageName)
	g.P()

	for _, stateService := range stateServices {
		if err := addStateQueryService(g, *stateService); err != nil {
			gen.Error(err)
		}
	}

	for _, stateSet := range stateSets {
		converted, err := buildStateSet(stateSet)
		if err != nil {
			gen.Error(err)
			continue
		}
		if err := addStateSet(g, converted); err != nil {
			gen.Error(err)
		}
	}
	return g
}

func addStateQueryService(g *protogen.GeneratedFile, service buildingStateQueryService) error {

	sm := protogen.GoImportPath("github.com/pentops/protostate/psm")
	protoPackage := protogen.GoImportPath("google.golang.org/protobuf/proto")

	g.P("// State Query Service for %s", service.name)

	if service.getMethod == nil {
		return fmt.Errorf("service %s does not have a get method", service.name)
	}

	if service.listMethod == nil {
		return fmt.Errorf("service %s does not have a list method", service.name)
	}

	goServiceName := stateGoName(service.name)

	g.P("type ", goServiceName, "PSMStateQuerySet = ", sm.Ident("StateQuerySet"), "[")
	g.P("*", service.getMethod.Input.GoIdent, ",")
	g.P("*", service.getMethod.Output.GoIdent, ",")
	g.P("*", service.listMethod.Input.GoIdent, ",")
	g.P("*", service.listMethod.Output.GoIdent, ",")
	if service.listEventsMethod != nil {
		g.P("*", service.listEventsMethod.Input.GoIdent, ",")
		g.P("*", service.listEventsMethod.Output.GoIdent, ",")
	} else {
		g.P(protoPackage.Ident("Message"), ",")
		g.P(protoPackage.Ident("Message"), ",")
	}
	g.P("]")
	g.P()
	g.P("type ", goServiceName, "PSMStateQuerySpec = ", sm.Ident("StateQuerySpec"), "[")
	g.P("*", service.getMethod.Input.GoIdent, ",")
	g.P("*", service.getMethod.Output.GoIdent, ",")
	g.P("*", service.listMethod.Input.GoIdent, ",")
	g.P("*", service.listMethod.Output.GoIdent, ",")
	if service.listEventsMethod != nil {
		g.P("*", service.listEventsMethod.Input.GoIdent, ",")
		g.P("*", service.listEventsMethod.Output.GoIdent, ",")
	} else {
		g.P(protoPackage.Ident("Message"), ",")
		g.P(protoPackage.Ident("Message"), ",")
	}
	g.P("]")
	g.P()

	return nil
}

type stateSet struct {
	specifiedName string

	namePrefix  string
	machineName string

	stateMessage *protogen.Message
	eventMessage *protogen.Message

	// Name of the inner event interface
	eventName string

	// In the Status message, this is the enum for the simplified status
	// is always an Enum
	statusFieldInState *protogen.Field

	// fields in which are the same in the state and the event, e.g. the PK
	eventStateKeyFields []eventStateFieldMap

	// field in the root of the outer event for the inner event oneof wrapper
	// is always a Message
	eventTypeField *protogen.Field

	// field in the root of the outer event for the event metadata
	metadataField *protogen.Field
}

type eventStateFieldMap struct {
	eventField *protogen.Field
	stateField *protogen.Field
	isKey      bool
}

func buildStateSet(src *buildingStateSet) (*stateSet, error) {
	ss := &stateSet{
		stateMessage: src.stateMessage,
		eventMessage: src.eventMessage,
	}
	ss.specifiedName = src.stateOptions.Name
	ss.namePrefix = stateGoName(ss.specifiedName)
	ss.machineName = ss.namePrefix + "PSM"
	ss.eventName = ss.namePrefix + "PSMEvent"

	for _, field := range src.stateMessage.Fields {
		fr := field.Desc
		if fr.Name() == "status" {
			ss.statusFieldInState = field
		}
	}

	if ss.statusFieldInState == nil {
		return nil, fmt.Errorf("state object %s does not have a 'status' field", src.stateOptions.Name)
	}

	if ss.statusFieldInState.Enum == nil {
		return nil, fmt.Errorf("state object %s state field is not an enum", src.stateOptions.Name)
	}

	for _, field := range src.eventMessage.Fields {
		fieldOpt := proto.GetExtension(field.Desc.Options(), psm_pb.E_EventField).(*psm_pb.EventField)
		if fieldOpt == nil {
			continue
		}

		if fieldOpt.EventType {
			if field.Message == nil {
				return nil, fmt.Errorf("event object %s event type field is not a message", src.eventOptions.Name)
			}
			ss.eventTypeField = field
		} else if fieldOpt.Metadata {
			ss.metadataField = field
		} else if fieldOpt.StateKey || fieldOpt.StateField {

			desc := field.Desc
			if desc.IsList() || desc.IsMap() || desc.Kind() == protoreflect.MessageKind {
				return nil, fmt.Errorf("event object %s state key field %s is not a scalar", src.eventOptions.Name, field.Desc.Name())
			}

			var matchingStateField *protogen.Field
			for _, stateField := range src.stateMessage.Fields {
				if stateField.Desc.Name() == field.Desc.Name() {
					matchingStateField = stateField
					break
				}
			}
			if matchingStateField == nil {
				return nil, fmt.Errorf("event object %s state key field %s does not exist in state object %s", src.eventOptions.Name, field.Desc.Name(), src.stateOptions.Name)
			}

			if matchingStateField.Desc.Kind() != field.Desc.Kind() {
				return nil, fmt.Errorf("event object %s state key field %s is not the same type as state object %s", src.eventOptions.Name, field.Desc.Name(), src.stateOptions.Name)
			}

			if matchingStateField.Desc.HasOptionalKeyword() {
				if !field.Desc.HasOptionalKeyword() {
					return nil, fmt.Errorf("event object %s state key field %s is marked optional, but state object %s is not", src.eventOptions.Name, field.Desc.Name(), src.stateOptions.Name)
				}
			} else if field.Desc.HasOptionalKeyword() {
				return nil, fmt.Errorf("event object %s state key field %s is not marked optional, but state object %s is", src.eventOptions.Name, field.Desc.Name(), src.stateOptions.Name)
			}

			ss.eventStateKeyFields = append(ss.eventStateKeyFields, eventStateFieldMap{
				eventField: field,
				stateField: matchingStateField,
				isKey:      fieldOpt.StateKey,
			})

		}

	}

	// the oneof wrapper
	if ss.eventTypeField == nil {
		return nil, fmt.Errorf("event object %s does not have an event type field", src.eventOptions.Name)
	}

	if ss.eventTypeField.Message == nil {
		return nil, fmt.Errorf("event object %s event type field is not a message", src.eventOptions.Name)
	}

	return ss, nil
}

func addStateSet(g *protogen.GeneratedFile, ss *stateSet) error {

	g.P("// StateObjectOptions: ", ss.machineName)

	sm := protogen.GoImportPath("github.com/pentops/protostate/psm")

	printTypes := func() {
		g.P("*", ss.stateMessage.GoIdent, ",")
		g.P(ss.statusFieldInState.Enum.GoIdent.GoName, ",")
		g.P("*", ss.eventMessage.GoIdent, ",")
		g.P(ss.eventName, ",")
	}

	g.P("type ", ss.machineName, "Eventer = ", sm.Ident("Eventer"), "[")
	printTypes()
	g.P("]")
	g.P()
	g.P("type ", ss.machineName, " = ", sm.Ident("StateMachine"), "[")
	printTypes()
	g.P("]")

	g.P()

	FooPSM := ss.machineName
	FooPSMConverter := ss.machineName + "Converter"
	FooPSMTableSpec := ss.machineName + "TableSpec"
	DefaultFooPSMTableSpec := "Default" + ss.machineName + "TableSpec"

	// func NewFooPSM(db psm.Transactor, options ...FooPSMOption) (*FooPSM, error) {
	g.P("func New", ss.machineName, "(db ", sm.Ident("Transactor"), ", options ...", sm.Ident("StateMachineOption"), "[")
	printTypes()
	g.P("]) (*", FooPSM, ", error) {")
	g.P("return ", sm.Ident("NewStateMachine"), "[")
	printTypes()
	g.P("](db, ", FooPSMConverter, "{}, ", DefaultFooPSMTableSpec, ", options...)")
	g.P("}")
	g.P()

	g.P("type ", FooPSMTableSpec, " = ", sm.Ident("TableSpec"), "[")
	printTypes()
	g.P("]")
	g.P()

	if err := addDefaultTableSpec(g, ss); err != nil {
		return err
	}

	g.P()

	g.P("type ", ss.machineName, "TransitionBaton = ", sm.Ident("TransitionBaton"), "[*", ss.eventMessage.GoIdent, ", ", ss.eventName, "]")

	g.P()
	g.P("func ", ss.machineName,
		"Func[SE ", ss.eventName, "]",
		"(cb func(",
		protogen.GoImportPath("context").Ident("Context"), ", ",
		ss.machineName, "TransitionBaton, *",
		ss.stateMessage.GoIdent, ", SE) error) ", sm.Ident("TransitionFunc"), "[")
	printTypes()
	g.P("SE,")
	g.P("] {")
	g.P("return ", sm.Ident("TransitionFunc"), "[")
	printTypes()
	g.P("SE,")
	g.P("](cb)")
	g.P("}")

	g.P()
	g.P("type ", ss.eventName, "Key string")
	g.P()
	g.P("const (")
	for _, field := range ss.eventTypeField.Message.Fields {
		g.P(ss.namePrefix, "PSMEvent", field.GoName, " ", ss.eventName, "Key = \"", field.Desc.Name(), "\"")
	}
	g.P(")")
	g.P()
	g.P("type ", ss.eventName, " interface {")
	g.P(protogen.GoImportPath("google.golang.org/protobuf/proto").Ident("Message"))
	g.P("PSMEventKey() ", ss.eventName, "Key")
	g.P("}")

	// Converting types
	if err := addTypeConverter(g, ss); err != nil {
		return err
	}

	g.P("func (ee *", ss.eventTypeField.Message.GoIdent, ") UnwrapPSMEvent() ", ss.eventName, " {")
	g.P("	switch v := ee.Type.(type) {")
	for _, field := range ss.eventTypeField.Message.Fields {
		g.P("	case *", field.GoIdent, ":")
		g.P("		return v.", field.GoName)
	}
	g.P("	default:")
	g.P("		return nil")
	g.P("	}")
	g.P("}")

	g.P("func (ee *", ss.eventTypeField.Message.GoIdent, ") PSMEventKey() ", ss.namePrefix, "PSMEventKey {")
	g.P("	tt := ee.UnwrapPSMEvent()")
	g.P("   if tt == nil {")
	g.P("     return \"<nil>\"")
	g.P("   }")
	g.P("	return tt.PSMEventKey()")
	g.P("}")

	g.P("func (ee *", ss.eventMessage.GoIdent, ") UnwrapPSMEvent() ", ss.eventName, " {")
	g.P("   return ee.", ss.eventTypeField.GoName, ".UnwrapPSMEvent()")
	g.P("}")

	g.P("func (ee *", ss.eventMessage.GoIdent, ") SetPSMEvent(inner ", ss.eventName, ") {")
	g.P("	switch v := inner.(type) {")
	for _, field := range ss.eventTypeField.Message.Fields {
		g.P("	case *", field.Message.GoIdent, ":")
		g.P("		ee.", ss.eventTypeField.GoName, ".Type = &", field.GoIdent, "{", field.GoName, ": v}")
	}
	g.P("	default:")
	g.P("		panic(\"invalid type\")")
	g.P("	}")
	g.P("}")

	for _, field := range ss.eventTypeField.Message.Fields {
		g.P("func (*", field.Message.GoIdent, ") PSMEventKey() ", ss.eventName, "Key  {")
		g.P("		return ", ss.namePrefix, "PSMEvent", field.GoName)
		g.P("}")
	}
	g.P()

	return nil

}

func addDefaultTableSpec(g *protogen.GeneratedFile, ss *stateSet) error {
	FooPSMTableSpec := ss.machineName + "TableSpec"
	DefaultFooPSMTableSpec := "Default" + ss.machineName + "TableSpec"

	/* Let's make some assumptions! */

	metadataMessage := ss.metadataField.Message

	// Look for a 'well formed' metadata message which has:
	// 'actor' as a message
	// 'timestamp' as google.protobuf.Timestamp
	// 'event_id', 'message_id' or 'id' as a string
	// If all the above match, we can automate the ExtractMetadata function
	var actorField *protogen.Field
	var timestampField *protogen.Field
	var eventIDField *protogen.Field
	for _, field := range metadataMessage.Fields {
		switch field.Desc.Name() {
		case "actor":
			if field.Message == nil {
				break // This will not match
			}
			actorField = field
		case "timestamp":
			if field.Message == nil || field.Message.Desc.FullName() != "google.protobuf.Timestamp" {
				break // This will not match
			}
			timestampField = field

		case "event_id", "message_id", "id":
			if field.Desc.Kind() != protoreflect.StringKind {
				break // This will not match
			}
			eventIDField = field
		}
	}

	canAutomateMetadata := actorField != nil && timestampField != nil && eventIDField != nil

	g.P("var ", DefaultFooPSMTableSpec, " = ", FooPSMTableSpec, " {")
	g.P("  StateTable: \"", ss.specifiedName, "\",")
	g.P("  EventTable: \"", ss.specifiedName, "_event\",")
	g.P("  PrimaryKey: func(event *", ss.eventMessage.GoIdent, ") (map[string]interface{}, error) {")
	g.P("    return map[string]interface{}{")
	for _, field := range ss.eventStateKeyFields {
		if !field.isKey {
			continue
		}
		// stripping the prefix foo_ from the name in the event. In the DB, we
		// expect the primary key to be called just id, so foo_id -> id
		keyName := strings.TrimPrefix(string(field.stateField.Desc.Name()), ss.specifiedName+"_")
		g.P("      \"", keyName, "\": event.", field.eventField.GoName, ",")
	}
	g.P("    }, nil")
	g.P("  },")

	if canAutomateMetadata {
		g.P("  EventColumns: func(event *", ss.eventMessage.GoIdent, ") (map[string]interface{}, error) {")
		g.P("    metadata := event.", ss.metadataField.GoName)
		g.P("    return map[string]interface{}{")
		g.P("      \"id\": metadata.", eventIDField.GoName, ",")
		g.P("      \"timestamp\": metadata.", timestampField.GoName, ",")
		g.P("      \"actor\": metadata.", actorField.GoName, ",")
		// Assumes that all fields in the event marked as state key should be
		// directly written to the table. If not, they should not be in the
		// event, i.e. if they are derivable from the state, rather than
		// identifying the state, there is no need to copy them to the event.
		for _, field := range ss.eventStateKeyFields {
			keyName := string(field.stateField.Desc.Name())
			g.P("      \"", keyName, "\": event.", field.eventField.GoName, ",")
		}
		g.P("    }, nil")
		g.P("  },")
	}

	g.P("}")

	return nil
}

func addTypeConverter(g *protogen.GeneratedFile, ss *stateSet) error {

	g.P("type ", ss.machineName, "Converter struct {}")
	g.P()
	g.P("func (c ", ss.machineName, "Converter) Unwrap(e *", ss.eventMessage.GoIdent, ") ", ss.eventName, " {")
	g.P("return e.UnwrapPSMEvent()")
	g.P("}")
	g.P()
	g.P("func (c ", ss.machineName, "Converter) StateLabel(s *", ss.stateMessage.GoIdent, ") string {")
	g.P("return s.Status.String()")
	g.P("}")
	g.P()
	g.P("func (c ", ss.machineName, "Converter) EventLabel(e ", ss.eventName, ") string {")
	g.P("return string(e.PSMEventKey())")
	g.P("}")
	g.P()
	g.P("func (c ", ss.machineName, "Converter) EmptyState(e *", ss.eventMessage.GoIdent, ") *", ss.stateMessage.GoIdent, " {")
	g.P("return &", ss.stateMessage.GoIdent, "{")
	for _, field := range ss.eventStateKeyFields {
		g.P(field.stateField.GoName, ": e.", field.eventField.GoName, ",")
	}
	g.P("}")
	g.P("}")
	g.P("func (c ", ss.machineName, "Converter) CheckStateKeys(s *", ss.stateMessage.GoIdent, ", e *", ss.eventMessage.GoIdent, ") error {")
	for _, field := range ss.eventStateKeyFields {
		if field.stateField.Desc.HasOptionalKeyword() {
			g.P("if s.", field.eventField.GoName, " == nil {")
			g.P("  if e.", field.eventField.GoName, " != nil {")
			g.P("    return ", protogen.GoImportPath("fmt").Ident("Errorf"), "(\"state field '", field.stateField.GoName, "' is nil, but event field is not (%q)\", *e.", field.eventField.GoName, ")")
			g.P("  }")
			g.P("} else if e.", field.eventField.GoName, " == nil {")
			g.P("  return ", protogen.GoImportPath("fmt").Ident("Errorf"), "(\"state field '", field.stateField.GoName, "' is not nil (%q), but event field is\", *e.", field.stateField.GoName, ")")
			g.P("} else if *s.", field.stateField.GoName, " != *e.", field.eventField.GoName, " {")
			g.P("  return ", protogen.GoImportPath("fmt").Ident("Errorf"), "(\"state field '", field.stateField.GoName, "' %q does not match event field %q\", *s.", field.stateField.GoName, ", *e.", field.eventField.GoName, ")")
			g.P("}")
		} else {
			g.P("if s.", field.stateField.GoName, " != e.", field.eventField.GoName, " {")
			g.P("return ", protogen.GoImportPath("fmt").Ident("Errorf"), "(\"state field '", field.stateField.GoName, "' %q does not match event field %q\", s.", field.stateField.GoName, ", e.", field.eventField.GoName, ")")
			g.P("}")
		}
	}
	g.P("return nil")
	g.P("}")

	g.P()

	return nil
}
