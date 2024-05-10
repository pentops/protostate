package main

import (
	"strings"

	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/protobuf/compiler/protogen"
)

type PSMEntity struct {
	specifiedName string

	namePrefix  string
	machineName string

	// Name of the inner event interface
	eventName string

	keyMessage *protogen.Message
	state      *stateEntityState
	event      *stateEntityEvent

	//eventFields eventFieldGenerators
	//stateFields stateFieldGenerators

	keyFields []keyField
}

type keyField struct {
	*psm_pb.FieldOptions
	field *protogen.Field
}

func (ss PSMEntity) write(g *protogen.GeneratedFile) error {
	g.P("// StateObjectOptions: ", ss.machineName)

	return ss.addStateSet(g)
}

func (ss PSMEntity) printTypes(g *protogen.GeneratedFile) {
	g.P("*", ss.keyMessage.GoIdent, ",")
	g.P("*", ss.state.message.GoIdent, ",")
	g.P(ss.state.statusField.Enum.GoIdent.GoName, ",")
	g.P("*", ss.event.message.GoIdent, ",")
	g.P(ss.eventName, ",")
}

func (ss PSMEntity) typeAlias(g *protogen.GeneratedFile, nameSuffix string, implements protogen.GoIdent) {
	g.P("type ", ss.machineName, nameSuffix, " = ", implements, " [")
	ss.printTypes(g)
	g.P("]")
	g.P()
}

func (ss PSMEntity) addStateSet(g *protogen.GeneratedFile) error {

	ss.typeAlias(g, "", smStateMachine)
	ss.typeAlias(g, "DB", smDBStateMachine)
	ss.typeAlias(g, "Eventer", smEventer)

	DefaultFooPSMTableSpec := "Default" + ss.machineName + "TableSpec"

	g.P("func Default", ss.machineName, "Config() *", smStateMachineConfig, "[")
	ss.printTypes(g)
	g.P("] {")
	g.P("return ", smNewStateMachineConfig, "[")
	ss.printTypes(g)
	g.P("](", DefaultFooPSMTableSpec, ")")
	g.P("}")
	g.P()

	g.P("func New", ss.machineName, "(config *", smStateMachineConfig, "[")
	ss.printTypes(g)
	g.P("]) (*", ss.machineName, ", error) {")
	g.P("return ", smNewStateMachine, "[")
	ss.printTypes(g)
	g.P("](config)")
	g.P("}")
	g.P()

	ss.typeAlias(g, "TableSpec", smPSMTableSpec)

	if err := ss.addDefaultTableSpec(g); err != nil {
		return err
	}

	g.P()

	ss.typeAlias(g, "TransitionBaton", smTransitionBaton)
	ss.typeAlias(g, "HookBaton", smStateHookBaton)

	// FooPSMFunc - This is for backwards compatibility
	g.P("func ", ss.machineName,
		"Func[SE ", ss.eventName, "]",
		"(cb func(",
		protogen.GoImportPath("context").Ident("Context"), ", ",
		ss.machineName, "TransitionBaton, *",
		ss.state.message.GoIdent, ", SE) error) ", smCombinedFunc, "[")
	ss.printTypes(g)
	g.P("SE,")
	g.P("] {")
	g.P("return ", smCombinedFunc, "[")
	ss.printTypes(g)
	g.P("SE,")
	g.P("](cb)")
	g.P("}")

	// FooPSMTransition
	g.P("func ", ss.machineName,
		"Transition[SE ", ss.eventName, "]",
		"(cb func(",
		protogen.GoImportPath("context").Ident("Context"), ", ",
		"*", ss.state.message.GoIdent, ", SE) error) ", smTransitionFunc, "[")
	ss.printTypes(g)
	g.P("SE,")
	g.P("] {")
	g.P("return ", smTransitionFunc, "[")
	ss.printTypes(g)
	g.P("SE,")
	g.P("](cb)")
	g.P("}")

	// FooPSMHook
	g.P("func ", ss.machineName,
		"Hook[SE ", ss.eventName, "]",
		"(cb func(",
		protogen.GoImportPath("context").Ident("Context"), ", ",
		protogen.GoImportPath("github.com/pentops/sqrlx.go/sqrlx").Ident("Transaction"), ", ",
		ss.machineName, "HookBaton, *",
		ss.state.message.GoIdent, ", SE) error) ", smHookFunc, "[")
	ss.printTypes(g)
	g.P("SE,")
	g.P("] {")
	g.P("return ", smHookFunc, "[")
	ss.printTypes(g)
	g.P("SE,")
	g.P("](cb)")
	g.P("}")

	// FooPSMGenericHook
	g.P("func ", ss.machineName,
		"GeneralHook",
		"(cb func(",
		protogen.GoImportPath("context").Ident("Context"), ", ",
		protogen.GoImportPath("github.com/pentops/sqrlx.go/sqrlx").Ident("Transaction"), ", ",
		"*", ss.state.message.GoIdent, ", ",
		"*", ss.event.message.GoIdent, ") error) ", smGeneralHookFunc, "[")
	ss.printTypes(g)
	g.P("] {")
	g.P("return ", smGeneralHookFunc, "[")
	ss.printTypes(g)
	g.P("](cb)")
	g.P("}")

	g.P()
	g.P("type ", ss.eventName, "Key = string")
	g.P()
	g.P("const (")
	g.P(ss.namePrefix, "PSMEventNil ", ss.eventName, "Key = \"<nil>\"")
	for _, field := range ss.event.eventTypeField.Message.Fields {
		g.P(ss.namePrefix, "PSMEvent", field.GoName, " ", ss.eventName, "Key = \"", field.Desc.Name(), "\"")
	}
	g.P(")")
	g.P()
	g.P("type ", ss.eventName, " interface {")
	g.P(protogen.GoImportPath("google.golang.org/protobuf/proto").Ident("Message"))
	g.P("PSMEventKey() ", ss.eventName, "Key")
	g.P("}")
	g.P()
	g.P()
	if err := ss.eventTypeMethods(g); err != nil {
		return err
	}
	g.P()
	if err := ss.eventAliasMethods(g); err != nil {
		return err
	}
	g.P()

	for _, field := range ss.event.eventTypeField.Message.Fields {
		g.P("func (*", field.Message.GoIdent, ") PSMEventKey() ", ss.eventName, "Key  {")
		g.P("		return ", ss.namePrefix, "PSMEvent", field.GoName)
		g.P("}")
	}

	if err := ss.metadataMethods(g); err != nil {
		return err
	}

	if err := ss.keyMethods(g); err != nil {
		return err
	}

	return nil

}

func (ss PSMEntity) metadataMethods(g *protogen.GeneratedFile) error {
	g.P("func (ee *", ss.event.message.GoIdent, ") PSMMetadata() *", psmEventMetadataStruct, " {")
	g.P("  if ee.", ss.event.metadataField.GoName, " == nil {")
	g.P("    ee.", ss.event.metadataField.GoName, " = &", psmEventMetadataStruct, "{}")
	g.P("  }")
	g.P("  return ee.", ss.event.metadataField.GoName)
	g.P("}")
	g.P()
	g.P("func (st *", ss.state.message.GoIdent, ") PSMMetadata() *", psmStateMetadataStruct, " {")
	g.P("  if st.", ss.state.metadataField.GoName, " == nil {")
	g.P("    st.", ss.state.metadataField.GoName, " = &", psmStateMetadataStruct, "{}")
	g.P("  }")
	g.P("  return st.", ss.state.metadataField.GoName)
	g.P("}")
	g.P()
	return nil
}

func (ss PSMEntity) keyMethods(g *protogen.GeneratedFile) error {
	g.P("func (ee *", ss.event.message.GoIdent, ") PSMKeys() *", ss.keyMessage.GoIdent, " {")
	g.P("  return ee.", ss.event.keyField.GoName)
	g.P("}")
	g.P()
	g.P("func (ee *", ss.event.message.GoIdent, ") SetPSMKeys(inner *", ss.keyMessage.GoIdent, ") {")
	g.P("  ee.", ss.event.keyField.GoName, " = inner")
	g.P("}")
	g.P()
	g.P("func (st *", ss.state.message.GoIdent, ") PSMKeys() *", ss.keyMessage.GoIdent, " {")
	g.P("  return st.", ss.state.keyField.GoName)
	g.P("}")
	g.P()
	g.P("func (st *", ss.state.message.GoIdent, ") SetPSMKeys(inner *", ss.keyMessage.GoIdent, ") {")
	g.P("  st.", ss.state.keyField.GoName, " = inner")
	g.P("}")
	g.P()
	return nil
}

func (ss PSMEntity) eventTypeMethods(g *protogen.GeneratedFile) error {

	g.P("func (etw *", ss.event.eventTypeField.Message.GoIdent, ") UnwrapPSMEvent() ", ss.eventName, " {")
	g.P("   if etw == nil {")
	g.P("     return nil")
	g.P("   }")
	g.P("	switch v := etw.Type.(type) {")
	for _, field := range ss.event.eventTypeField.Message.Fields {
		g.P("	case *", field.GoIdent, ":")
		g.P("		return v.", field.GoName)
	}
	g.P("	default:")
	g.P("		return nil")
	g.P("	}")
	g.P("}")

	g.P("func (etw *", ss.event.eventTypeField.Message.GoIdent, ") PSMEventKey() ", ss.namePrefix, "PSMEventKey {")
	g.P("   tt := etw.UnwrapPSMEvent()")
	g.P("   if tt == nil {")
	g.P("     return ", ss.namePrefix, "PSMEventNil")
	g.P("   }")
	g.P("	return tt.PSMEventKey()")
	g.P("}")

	g.P("func (etw *", ss.event.eventTypeField.Message.GoIdent, ") SetPSMEvent(inner ", ss.eventName, ") {")
	g.P("  switch v := inner.(type) {")
	for _, field := range ss.event.eventTypeField.Message.Fields {
		g.P("	case *", field.Message.GoIdent, ":")
		g.P("		etw.Type = &", field.GoIdent, "{", field.GoName, ": v}")
	}
	g.P("	default:")
	g.P("		panic(\"invalid type\")")
	g.P("	}")
	g.P("}")

	return nil
}

// alias methods from the outer event to the event type wrapper
func (ss PSMEntity) eventAliasMethods(g *protogen.GeneratedFile) error {

	g.P("func (ee *", ss.event.message.GoIdent, ") PSMEventKey() ", ss.namePrefix, "PSMEventKey {")
	g.P("	return ee.", ss.event.eventTypeField.GoName, ".PSMEventKey()")
	g.P("}")
	g.P()
	g.P("func (ee *", ss.event.message.GoIdent, ") UnwrapPSMEvent() ", ss.eventName, " {")
	g.P("   return ee.", ss.event.eventTypeField.GoName, ".UnwrapPSMEvent()")
	g.P("}")
	g.P()
	g.P("func (ee *", ss.event.message.GoIdent, ") SetPSMEvent(inner ", ss.eventName, ") {")
	g.P("  if ee.", ss.event.eventTypeField.GoName, " == nil {")
	g.P("    ee.", ss.event.eventTypeField.GoName, " = &", ss.event.eventTypeField.Message.GoIdent, "{}")
	g.P("  }")
	g.P("  ee.", ss.event.eventTypeField.GoName, ".SetPSMEvent(inner)")
	g.P("}")

	return nil
}

func (ss PSMEntity) addDefaultTableSpec(g *protogen.GeneratedFile) error {
	FooPSMTableSpec := ss.machineName + "TableSpec"
	DefaultFooPSMTableSpec := "Default" + ss.machineName + "TableSpec"

	/* Let's make some assumptions! */

	// Look for a 'well formed' metadata message which has:
	// 'actor' as a message
	// 'timestamp' as google.protobuf.Timestamp
	// 'event_id', 'message_id' or 'id' as a string
	// If all the above match, we can automate the ExtractMetadata function

	g.P("var ", DefaultFooPSMTableSpec, " = ", FooPSMTableSpec, " {")
	g.P("  State: ", smTableSpec, "[*", ss.state.message.GoIdent, "] {")
	g.P("    TableName: \"", ss.specifiedName, "\",")
	g.P("    DataColumn: \"state\",")
	g.P("    StoreExtraColumns: func(state *", ss.state.message.GoIdent, ") (map[string]interface{}, error) {")
	g.P("      return map[string]interface{}{")

	// Assume the NON KEY fields are stored as columns in the state table
	for _, field := range ss.keyFields {
		if field.PrimaryKey {
			continue
		}
		// stripping the prefix foo_ from the name in the event. In the DB, we
		// expect the primary key to be called just id, so foo_id -> id
		keyName := strings.TrimPrefix(string(field.field.Desc.Name()), ss.specifiedName+"_")
		g.P("      \"", keyName, "\": state.", ss.state.keyField.GoName, ".", field.field.GoName, ",")
	}
	g.P("      }, nil")
	g.P("    },")
	g.P("    PKFieldPaths: []string{")
	for _, field := range ss.keyFields {
		if !field.PrimaryKey {
			continue
		}
		g.P("\"", ss.state.keyField.Desc.Name(), ".", field.field.Desc.Name(), "\",")
	}
	g.P("    },")
	g.P("  },")

	g.P("  Event: ", smTableSpec, "[*", ss.event.message.GoIdent, "] {")
	g.P("  TableName: \"", ss.specifiedName, "_event\",")
	g.P("    DataColumn: \"data\",")
	g.P("    StoreExtraColumns: func(event *", ss.event.message.GoIdent, ") (map[string]interface{}, error) {")
	g.P("      metadata := event.", ss.event.metadataField.GoName)
	g.P("      return map[string]interface{}{")
	g.P("        \"id\": metadata.EventId,")
	g.P("        \"timestamp\": metadata.Timestamp,")
	g.P("        \"cause\": metadata.Cause,")
	g.P("        \"sequence\": metadata.Sequence,")
	// Assumes that all key fields should be stored by default
	for _, field := range ss.keyFields {
		keyName := string(field.field.Desc.Name())
		g.P("      \"", keyName, "\": event.", ss.event.keyField.GoName, ".", field.field.GoName, ",")
	}
	g.P("      }, nil")
	g.P("    },")
	g.P("    PKFieldPaths: []string{")
	g.P("      \"", ss.event.metadataField.Desc.Name(), ".EventId\",")
	g.P("    },")
	g.P("    PK: func(event *", ss.event.message.GoIdent, ") (map[string]interface{}, error) {")
	g.P("      return map[string]interface{}{")
	g.P("        \"id\": event.", ss.event.metadataField.GoName, ".EventId,")
	g.P("      }, nil")
	g.P("    },")
	g.P("  },")
	g.P("  PrimaryKey: func(event *", ss.event.message.GoIdent, ") (map[string]interface{}, error) {")
	g.P("    return map[string]interface{}{")
	for _, field := range ss.keyFields {
		if !field.PrimaryKey {
			continue
		}
		// stripping the prefix foo_ from the name in the event. In the DB, we
		// expect the primary key to be called just id, so foo_id -> id
		keyName := strings.TrimPrefix(string(field.field.Desc.Name()), ss.specifiedName+"_")
		g.P("      \"", keyName, "\": event.", ss.event.keyField.GoName, ".", field.field.GoName, ",")
	}
	g.P("    }, nil")
	g.P("  },")

	g.P("}")

	return nil
}
