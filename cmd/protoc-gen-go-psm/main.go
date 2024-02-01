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
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/protostate/pquery"
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
	convertedStateSets := make(map[string]*queryPkFields)
	for _, stateSet := range sourceMap.stateSets {
		converted, err := buildStateSet(stateSet)
		if err != nil {
			gen.Error(err)
			continue
		}
		convertedStateSets[stateSet.name] = converted.eventDescriptors.stateDescriptorInfo()
		if err := addStateSet(g, converted); err != nil {
			gen.Error(err)
		}
	}

	for _, querySrc := range sourceMap.querySets {
		if err := querySrc.validate(); err != nil {
			gen.Error(err)
			continue
		}
		stateSet, ok := convertedStateSets[querySrc.name]
		if !ok {
			// attempt to derive by linking to other files. This state spec
			// should not be generated here
			stateSource, err := deriveStateDescriptorFromQueryDescriptor(querySrc.descriptors())
			if err != nil {
				gen.Error(err)
				return nil
			}

			if stateSource != nil {
				eventFields, err := buildEventFieldDescriptors(*stateSource)
				if err != nil {
					gen.Error(err)
					return nil
				}

				stateDescriptor := eventFields.stateDescriptorInfo()
				stateSet = stateDescriptor
			}
		}

		querySet := &querySet{
			specifiedName:    querySrc.name,
			getMethod:        querySrc.getMethod,
			listMethod:       querySrc.listMethod,
			listEventsMethod: querySrc.listEventsMethod,
		}

		if err := addStateQueryService(g, querySet, stateSet); err != nil {
			gen.Error(err)
		}
	}

	return g
}

func addStateQueryService(g *protogen.GeneratedFile, qs *querySet, ss *queryPkFields) error {

	if qs.getMethod == nil && qs.listMethod == nil && qs.listEventsMethod == nil {
		// if none are set, that's ok, but any of them set requires that at
		// least get and list are set
		return nil
	}

	g.P("// State Query Service for %s", qs.specifiedName)

	if qs.getMethod == nil {
		return fmt.Errorf("service %s does not have a get method", qs.specifiedName)
	}

	if qs.listMethod == nil {
		return fmt.Errorf("service %s does not have a list method", qs.specifiedName)
	}

	var statePkFields []string
	if ss != nil {
		statePkFields = ss.statePkFields
	}

	listReflectionSet, err := pquery.BuildListReflection(qs.listMethod.Input.Desc, qs.listMethod.Output.Desc, pquery.WithTieBreakerFields(statePkFields...))
	if err != nil {
		return fmt.Errorf("query service %s is not compatible with PSM: %w", qs.listMethod.Desc.FullName(), err)
	}

	goServiceName := stateGoName(qs.specifiedName)

	ww := PSMQuerySet{
		GoServiceName: goServiceName,
		GetREQ:        qs.getMethod.Input.GoIdent,
		GetRES:        qs.getMethod.Output.GoIdent,
		ListREQ:       qs.listMethod.Input.GoIdent,
		ListRES:       qs.listMethod.Output.GoIdent,
	}

	for _, field := range listReflectionSet.RequestFilterFields {
		genField := mapGenField(qs.listMethod.Input, field)
		ww.ListRequestFilter = append(ww.ListRequestFilter, ListFilterField{
			DBName:   string(field.Name()),
			Getter:   genField.GoName,
			Optional: field.HasOptionalKeyword(),
		})
	}

	if qs.listEventsMethod != nil {
		var fallbackPkFields []string
		if ss != nil {
			fallbackPkFields = ss.eventPkFields
		}
		listEventsReflectionSet, err := pquery.BuildListReflection(qs.listEventsMethod.Input.Desc, qs.listEventsMethod.Output.Desc, pquery.WithTieBreakerFields(fallbackPkFields...))
		if err != nil {
			return fmt.Errorf("query service %s is not compatible with PSM: %w", qs.listMethod.Desc.FullName(), err)
		}
		ww.ListEventsREQ = &qs.listEventsMethod.Input.GoIdent
		ww.ListEventsRES = &qs.listEventsMethod.Output.GoIdent
		for _, field := range listEventsReflectionSet.RequestFilterFields {
			genField := mapGenField(qs.listEventsMethod.Input, field)
			ww.ListEventsRequestFilter = append(ww.ListEventsRequestFilter, ListFilterField{
				DBName:   string(field.Name()),
				Getter:   genField.GoName,
				Optional: field.HasOptionalKeyword(),
			})
		}
	}

	if err := ww.Write(g); err != nil {
		return err
	}

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

	eventDescriptors eventFieldDescriptors
	eventFieldGenerators
}

type querySet struct {
	specifiedName    string
	getMethod        *protogen.Method
	listMethod       *protogen.Method
	listEventsMethod *protogen.Method
}

type eventStateFieldMap struct {
	eventField protoreflect.FieldDescriptor
	stateField protoreflect.FieldDescriptor
	isKey      bool
}

type eventStateFieldGenMap struct {
	eventField *protogen.Field
	stateField *protogen.Field
	isKey      bool
}

func mapGenField(parent *protogen.Message, field protoreflect.FieldDescriptor) *protogen.Field {
	for _, f := range parent.Fields {
		if f.Desc.FullName() == field.FullName() {
			return f
		}
	}
	panic(fmt.Sprintf("field %s not found in parent %s", field.FullName(), parent.Desc.FullName()))
}

type queryPkFields struct {
	statePkFields []string
	eventPkFields []string
}

type eventFieldDescriptors struct {
	// fields in which are the same in the state and the event, e.g. the PK
	eventStateKeyFields []eventStateFieldMap

	// field in the root of the outer event for the inner event oneof wrapper
	// is always a Message
	eventTypeField protoreflect.FieldDescriptor

	// field in the root of the outer event for the event metadata
	metadataField protoreflect.FieldDescriptor

	eventActorField     protoreflect.FieldDescriptor
	eventTimestampField protoreflect.FieldDescriptor
	eventIDField        protoreflect.FieldDescriptor
}

func (desc eventFieldDescriptors) validate() error {

	if desc.metadataField == nil {
		return fmt.Errorf("event object missing metadata field")
	}

	if desc.eventIDField == nil {
		return fmt.Errorf("event object missing id field")
	}

	if desc.eventTimestampField == nil {
		return fmt.Errorf("event object missing timestamp field")
	}

	// the oneof wrapper
	if desc.eventTypeField == nil {
		return fmt.Errorf("event object missing eventType field")
	}

	if desc.eventTypeField.Kind() != protoreflect.MessageKind {
		return fmt.Errorf("event object event type field is not a message")
	}

	return nil
}

func (ss eventFieldDescriptors) stateDescriptorInfo() *queryPkFields {
	out := &queryPkFields{}
	out.statePkFields = make([]string, len(ss.eventStateKeyFields))
	for i, field := range ss.eventStateKeyFields {
		out.statePkFields[i] = string(field.stateField.Name())
	}

	fallbackPkFields := make([]string, 0, 1)
	pkName := fmt.Sprintf("%s.%s", ss.metadataField.Name(), ss.eventIDField.Name())
	fallbackPkFields = append(fallbackPkFields, pkName)
	out.eventPkFields = fallbackPkFields

	return out
}

type eventFieldGenerators struct {
	eventStateKeyFields []eventStateFieldGenMap
	eventTypeField      *protogen.Field
	metadataField       *protogen.Field
	eventIDField        *protogen.Field
	eventTimestampField *protogen.Field
	eventActorField     *protogen.Field
}

func mapEventFieldGenMap(in eventFieldDescriptors, stateSrc *stateEntityGenerateSet) eventFieldGenerators {
	out := eventFieldGenerators{
		eventStateKeyFields: make([]eventStateFieldGenMap, len(in.eventStateKeyFields)),
	}
	for i, field := range in.eventStateKeyFields {
		eventField := mapGenField(stateSrc.eventMessage, field.eventField)
		stateField := mapGenField(stateSrc.stateMessage, field.stateField)
		out.eventStateKeyFields[i] = eventStateFieldGenMap{
			eventField: eventField,
			stateField: stateField,
			isKey:      field.isKey,
		}
	}
	out.metadataField = mapGenField(stateSrc.eventMessage, in.metadataField)
	out.eventIDField = mapGenField(out.metadataField.Message, in.eventIDField)
	out.eventTimestampField = mapGenField(out.metadataField.Message, in.eventTimestampField)
	if in.eventActorField != nil {
		out.eventActorField = mapGenField(out.metadataField.Message, in.eventActorField)
	}

	out.eventTypeField = mapGenField(stateSrc.eventMessage, in.eventTypeField)
	return out
}

func buildEventFieldDescriptors(src stateEntityDescriptorSet) (*eventFieldDescriptors, error) {
	out := eventFieldDescriptors{}
	eventFields := src.eventMessage.Fields()
	for idx := 0; idx < eventFields.Len(); idx++ {
		field := eventFields.Get(idx)
		fieldOpt := proto.GetExtension(field.Options(), psm_pb.E_EventField).(*psm_pb.EventField)
		if fieldOpt == nil {
			continue
		}

		if fieldOpt.EventType {
			if field.Kind() != protoreflect.MessageKind {
				return nil, fmt.Errorf("event object event type field is not a message")
			}
			out.eventTypeField = field
		} else if fieldOpt.Metadata {
			if field.Kind() != protoreflect.MessageKind {
				return nil, fmt.Errorf("event object %s metadata field is not a message", src.eventOptions.Name)
			}
			out.metadataField = field
		} else if fieldOpt.StateKey || fieldOpt.StateField {

			desc := field
			if desc.IsList() || desc.IsMap() || desc.Kind() == protoreflect.MessageKind {
				return nil, fmt.Errorf("event object %s state key field %s is not a scalar", src.eventOptions.Name, field.Name())
			}

			matchingStateField := src.stateMessage.Fields().ByName(field.Name())
			if matchingStateField == nil {
				return nil, fmt.Errorf("event object %s state key field %s does not exist in state object %s", src.eventOptions.Name, field.Name(), src.stateOptions.Name)
			}

			if matchingStateField.Kind() != field.Kind() {
				return nil, fmt.Errorf("event object %s state key field %s is not the same type as state object %s", src.eventOptions.Name, field.Name(), src.stateOptions.Name)
			}

			if matchingStateField.HasOptionalKeyword() {
				if !field.HasOptionalKeyword() {
					return nil, fmt.Errorf("event object %s state key field %s is marked optional, but state object %s is not", src.eventOptions.Name, field.Name(), src.stateOptions.Name)
				}
			} else if field.HasOptionalKeyword() {
				return nil, fmt.Errorf("event object %s state key field %s is not marked optional, but state object %s is", src.eventOptions.Name, field.Name(), src.stateOptions.Name)
			}

			out.eventStateKeyFields = append(out.eventStateKeyFields, eventStateFieldMap{
				eventField: field,
				stateField: matchingStateField,
				isKey:      fieldOpt.StateKey,
			})

		}

	}

	metadataFields := out.metadataField.Message().Fields()
	for idx := 0; idx < metadataFields.Len(); idx++ {
		field := metadataFields.Get(idx)
		switch field.Name() {
		case "actor":
			if field.Kind() != protoreflect.MessageKind {
				break // This will not match
			}
			out.eventActorField = field
		case "timestamp":
			if field.Kind() != protoreflect.MessageKind || field.Message().FullName() != protoreflect.FullName("google.protobuf.Timestamp") {
				break // This will not match
			}
			out.eventTimestampField = field

		case "event_id", "message_id", "id":
			if field.Kind() != protoreflect.StringKind {
				break // This will not match
			}
			out.eventIDField = field
		}
	}

	if err := out.validate(); err != nil {
		return nil, fmt.Errorf("event object %s: %w", src.eventOptions.Name, err)
	}

	return &out, nil
}

func buildStateSet(src *stateEntityGenerateSet) (*stateSet, error) {

	if src.stateMessage == nil || src.stateOptions == nil {
		return nil, fmt.Errorf("state object '%s' does not have a State message (message with (psm.state.v1.state).name = '%s')", src.fullName, src.name)
	}

	if src.eventMessage == nil || src.eventOptions == nil {
		return nil, fmt.Errorf("state object '%s' does not have an Event message (message with (psm.state.v1.event).name = '%s')", src.fullName, src.name)
	}

	ss := &stateSet{
		stateMessage: src.stateMessage,
		eventMessage: src.eventMessage,
	}

	ss.specifiedName = src.stateOptions.Name
	ss.namePrefix = stateGoName(ss.specifiedName)
	ss.machineName = ss.namePrefix + "PSM"
	ss.eventName = ss.namePrefix + "PSMEvent"

	for _, field := range ss.stateMessage.Fields {
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

	if src.eventMessage == nil {
		return nil, fmt.Errorf("state object %s does not have an event object", src.stateOptions.Name)
	}

	stateDescriptors := src.descriptors()

	eventDescriptors, err := buildEventFieldDescriptors(stateDescriptors)
	if err != nil {
		return nil, err
	}
	ss.eventDescriptors = *eventDescriptors

	ss.eventFieldGenerators = mapEventFieldGenMap(*eventDescriptors, src)

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
	g.P("type ", ss.machineName, "DB = ", sm.Ident("DBStateMachine"), "[")
	printTypes()
	g.P("]")
	g.P()

	FooPSM := ss.machineName
	FooPSMConverter := ss.machineName + "Converter"
	FooPSMTableSpec := ss.machineName + "TableSpec"
	DefaultFooPSMTableSpec := "Default" + ss.machineName + "TableSpec"

	// func NewFooPSM(db psm.Transactor, options ...FooPSMOption) (*FooPSM, error) {
	g.P("func Default", ss.machineName, "Config() *", sm.Ident("StateMachineConfig"), "[")
	printTypes()
	g.P("] {")
	g.P("return ", sm.Ident("NewStateMachineConfig"), "[")
	printTypes()
	g.P("](", FooPSMConverter, "{}, ", DefaultFooPSMTableSpec, ")")
	g.P("}")
	g.P()

	// func NewFooPSM(db psm.Transactor, options ...FooPSMOption) (*FooPSM, error) {
	g.P("func New", ss.machineName, "(config *", sm.Ident("StateMachineConfig"), "[")
	printTypes()
	g.P("]) (*", FooPSM, ", error) {")
	g.P("return ", sm.Ident("NewStateMachine"), "[")
	printTypes()
	g.P("](config)")
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

	var eventTypeField *protogen.Field
	for _, field := range ss.eventMessage.Fields {
		if field.Desc.Name() == ss.eventTypeField.Desc.Name() {
			eventTypeField = field
		}
	}

	g.P()
	g.P("type ", ss.eventName, "Key string")
	g.P()
	g.P("const (")
	g.P(ss.namePrefix, "PSMEventNil ", ss.eventName, "Key = \"<nil>\"")
	for _, field := range eventTypeField.Message.Fields {
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

	g.P("func (ee *", eventTypeField.Message.GoIdent, ") UnwrapPSMEvent() ", ss.eventName, " {")
	g.P("   if ee == nil {")
	g.P("     return nil")
	g.P("   }")
	g.P("	switch v := ee.Type.(type) {")
	for _, field := range eventTypeField.Message.Fields {
		g.P("	case *", field.GoIdent, ":")
		g.P("		return v.", field.GoName)
	}
	g.P("	default:")
	g.P("		return nil")
	g.P("	}")
	g.P("}")

	g.P("func (ee *", eventTypeField.Message.GoIdent, ") PSMEventKey() ", ss.namePrefix, "PSMEventKey {")
	g.P("   tt := ee.UnwrapPSMEvent()")
	g.P("   if tt == nil {")
	g.P("     return ", ss.namePrefix, "PSMEventNil")
	g.P("   }")
	g.P("	return tt.PSMEventKey()")
	g.P("}")

	g.P("func (ee *", ss.eventMessage.GoIdent, ") PSMEventKey() ", ss.namePrefix, "PSMEventKey {")
	g.P("	return ee.", eventTypeField.GoName, ".PSMEventKey()")
	g.P("}")

	g.P("func (ee *", ss.eventMessage.GoIdent, ") UnwrapPSMEvent() ", ss.eventName, " {")
	g.P("   return ee.", eventTypeField.GoName, ".UnwrapPSMEvent()")
	g.P("}")

	g.P("func (ee *", ss.eventMessage.GoIdent, ") SetPSMEvent(inner ", ss.eventName, ") {")
	g.P("  if ee.", eventTypeField.GoName, " == nil {")
	g.P("    ee.", eventTypeField.GoName, " = &", eventTypeField.Message.GoIdent, "{}")
	g.P("  }")
	g.P("  switch v := inner.(type) {")
	for _, field := range eventTypeField.Message.Fields {
		g.P("	case *", field.Message.GoIdent, ":")
		g.P("		ee.", eventTypeField.GoName, ".Type = &", field.GoIdent, "{", field.GoName, ": v}")
	}
	g.P("	default:")
	g.P("		panic(\"invalid type\")")
	g.P("	}")
	g.P("}")

	for _, field := range eventTypeField.Message.Fields {
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

	// Look for a 'well formed' metadata message which has:
	// 'actor' as a message
	// 'timestamp' as google.protobuf.Timestamp
	// 'event_id', 'message_id' or 'id' as a string
	// If all the above match, we can automate the ExtractMetadata function

	g.P("var ", DefaultFooPSMTableSpec, " = ", FooPSMTableSpec, " {")
	g.P("  StateTable: \"", ss.specifiedName, "\",")
	g.P("  EventTable: \"", ss.specifiedName, "_event\",")
	g.P("  PrimaryKey: func(event *", ss.eventMessage.GoIdent, ") (map[string]interface{}, error) {")
	g.P("    return map[string]interface{}{")
	for _, field := range ss.eventFieldGenerators.eventStateKeyFields {
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
	g.P("  StateColumns: func(state *", ss.stateMessage.GoIdent, ") (map[string]interface{}, error) {")
	g.P("    return map[string]interface{}{")

	for _, field := range ss.eventFieldGenerators.eventStateKeyFields {
		if field.isKey {
			continue
		}
		// stripping the prefix foo_ from the name in the event. In the DB, we
		// expect the primary key to be called just id, so foo_id -> id
		keyName := strings.TrimPrefix(string(field.stateField.Desc.Name()), ss.specifiedName+"_")
		g.P("      \"", keyName, "\": state.", field.eventField.GoName, ",")
	}
	g.P("    }, nil")
	g.P("  },")

	g.P("  EventColumns: func(event *", ss.eventMessage.GoIdent, ") (map[string]interface{}, error) {")
	g.P("    metadata := event.", ss.metadataField.GoName)
	g.P("    return map[string]interface{}{")
	g.P("      \"id\": metadata.", ss.eventIDField.GoName, ",")
	g.P("      \"timestamp\": metadata.", ss.eventTimestampField.GoName, ",")
	if ss.eventActorField != nil {
		g.P("      \"actor\": metadata.", ss.eventActorField.GoName, ",")
	}
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
	g.P("  EventPrimaryKeyFieldPaths: []string{")
	g.P("    \"", ss.metadataField.Desc.Name(), ".", ss.eventIDField.Desc.Name(), "\",")
	g.P("  },")
	g.P("  StatePrimaryKeyFieldPaths: []string{")
	for _, field := range ss.eventStateKeyFields {
		if !field.isKey {
			continue
		}
		g.P("\"", field.stateField.Desc.Name(), "\",")
	}
	g.P("},")

	g.P("}")

	return nil
}

func addTypeConverter(g *protogen.GeneratedFile, ss *stateSet) error {
	sm := protogen.GoImportPath("github.com/pentops/protostate/psm")
	timestamppb := protogen.GoImportPath("google.golang.org/protobuf/types/known/timestamppb")

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
	g.P()
	g.P("func (c ", ss.machineName, "Converter) DeriveChainEvent(e *", ss.eventMessage.GoIdent, ", systemActor ", sm.Ident("SystemActor"), ", eventKey string) *", ss.eventMessage.GoIdent, " {")
	g.P("  metadata := &", ss.metadataField.Message.GoIdent, "{")
	g.P("  ", ss.eventIDField.GoName, ": systemActor.NewEventID(e.", ss.metadataField.GoName, ".", ss.eventIDField.GoName, ", eventKey),")
	g.P("  ", ss.eventTimestampField.GoName, ": ", timestamppb.Ident("Now()"), ",")
	g.P("}")

	if ss.eventActorField != nil {
		g.P("actorProto := systemActor.ActorProto()")
		g.P("refl := metadata.ProtoReflect()")
		g.P("refl.Set(refl.Descriptor().Fields().ByName(\"", ss.eventActorField.Desc.Name(), "\"), actorProto)")
	}

	g.P("return &", ss.eventMessage.GoIdent, "{")
	g.P(ss.metadataField.GoName, ": metadata,")
	for _, field := range ss.eventStateKeyFields {
		g.P(field.eventField.GoName, ": e.", field.eventField.GoName, ",")
	}
	g.P("}")
	g.P("}")
	g.P()

	g.P("func (c ", ss.machineName, "Converter) CheckStateKeys(s *", ss.stateMessage.GoIdent, ", e *", ss.eventMessage.GoIdent, ") error {")
	for _, field := range ss.eventStateKeyFields {
		stateField := field.stateField
		eventField := field.eventField
		if stateField.Desc.HasOptionalKeyword() {
			g.P("if s.", eventField.GoName, " == nil {")
			g.P("  if e.", eventField.GoName, " != nil {")
			g.P("    return ", protogen.GoImportPath("fmt").Ident("Errorf"), "(\"state field '", stateField.GoName, "' is nil, but event field is not (%q)\", *e.", eventField.GoName, ")")
			g.P("  }")
			g.P("} else if e.", eventField.GoName, " == nil {")
			g.P("  return ", protogen.GoImportPath("fmt").Ident("Errorf"), "(\"state field '", stateField.GoName, "' is not nil (%q), but event field is\", *e.", stateField.GoName, ")")
			g.P("} else if *s.", stateField.GoName, " != *e.", eventField.GoName, " {")
			g.P("  return ", protogen.GoImportPath("fmt").Ident("Errorf"), "(\"state field '", stateField.GoName, "' %q does not match event field %q\", *s.", stateField.GoName, ", *e.", eventField.GoName, ")")
			g.P("}")
		} else {
			g.P("if s.", stateField.GoName, " != e.", eventField.GoName, " {")
			g.P("return ", protogen.GoImportPath("fmt").Ident("Errorf"), "(\"state field '", stateField.GoName, "' %q does not match event field %q\", s.", stateField.GoName, ", e.", eventField.GoName, ")")
			g.P("}")
		}
	}
	g.P("return nil")
	g.P("}")

	g.P()

	return nil
}
