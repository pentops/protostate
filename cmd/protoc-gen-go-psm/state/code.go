package state

import (
	"fmt"
	"strings"

	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/protobuf/compiler/protogen"
)

var (
	// all imports from PSM are defined here, i.e. this is the committed PSM interface.
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
	smEventSpec             = smImportPath.Ident("EventSpec")
	smIInnerEvent           = smImportPath.Ident("IInnerEvent")

	psmProtoImportPath     = protogen.GoImportPath("github.com/pentops/protostate/gen/state/v1/psm_pb")
	psmEventMetadataStruct = psmProtoImportPath.Ident("EventMetadata")
	psmStateMetadataStruct = psmProtoImportPath.Ident("StateMetadata")
)

type PSMEntity struct {
	specifiedName string // foo

	namePrefix  string // Foo
	machineName string // FooPSM
	eventName   string // FooPSMEvent

	keyMessage *protogen.Message
	state      *stateEntityState
	event      *stateEntityEvent

	keyFields []keyField
}

type keyField struct {
	*psm_pb.FieldOptions
	field *protogen.Field
}

func (ss PSMEntity) Write(g *protogen.GeneratedFile) {
	g.P("// PSM ", ss.machineName)
	g.P()

	ss.typeAlias(g, "", smStateMachine)
	ss.typeAlias(g, "DB", smDBStateMachine)
	ss.typeAlias(g, "Eventer", smEventer)
	ss.typeAlias(g, "EventSpec", smEventSpec)
	g.P()
	ss.psmEventKey(g)
	g.P()
	ss.implementIKeyset(g)
	g.P()
	ss.implementIState(g)
	g.P()
	ss.implementIEvent(g)
	g.P()
	ss.implementIInnerEvent(g)
	g.P()
	ss.tableSpecAndConfig(g)
	g.P()
	ss.transitionFuncTypes(g)
}

// prints the generic type parameters K, S, ST, E, IE
func (ss PSMEntity) writeBaseTypes(g *protogen.GeneratedFile) {
	g.P("*", ss.keyMessage.GoIdent.GoName, ", // implements psm.IKeyset")
	g.P("*", ss.state.message.GoIdent.GoName, ", // implements psm.IState")
	g.P(ss.state.statusField.Enum.GoIdent.GoName, ", // implements psm.IStatusEnum")
	g.P("*", ss.event.message.GoIdent.GoName, ", // implements psm.IEvent")
	g.P(ss.eventName, ", // implements psm.IInnerEvent")
}

func (ss PSMEntity) writeBaseTypesWithSE(g *protogen.GeneratedFile) {
	ss.writeBaseTypes(g)
	g.P("SE, // Specific event type for the transition")
}

func (ss PSMEntity) typeAlias(g *protogen.GeneratedFile, nameSuffix string, implements protogen.GoIdent) string {
	ident := fmt.Sprintf("%s%s", ss.machineName, nameSuffix)
	g.P("type ", ident, " = ", implements, " [")
	ss.writeBaseTypes(g)
	g.P("]")
	g.P()
	return ident
}

func (ss PSMEntity) psmEventKey(g *protogen.GeneratedFile) {
	g.P()
	g.P("type ", ss.eventName, "Key = string")
	g.P()
	g.P("const (")
	g.P(ss.namePrefix, "PSMEventNil ", ss.eventName, "Key = \"<nil>\"")
	for _, field := range ss.event.eventTypeField.Message.Fields {
		g.P(ss.namePrefix, "PSMEvent", field.GoName, " ", ss.eventName, "Key = \"", field.Desc.Name(), "\"")
	}
	g.P(")")
}

func (ss PSMEntity) implementIPSMMessage(g *protogen.GeneratedFile, msg *protogen.Message) {
	g.P()
	g.P("// PSMIsSet is a helper for != nil, which does not work with generic parameters")
	g.P("func (msg *", msg.GoIdent, ") PSMIsSet() bool {")
	g.P("  return msg != nil")
	g.P("}")
}

// implements psm.IKeyset for the key message
func (ss PSMEntity) implementIKeyset(g *protogen.GeneratedFile) {
	g.P("// EXTEND ", ss.keyMessage.GoIdent, " with the psm.IKeyset interface")
	ss.implementIPSMMessage(g, ss.keyMessage)
	g.P()
	g.P("// PSMFullName returns the full name of state machine with package prefix")
	g.P("func (msg *", ss.keyMessage.GoIdent, ") PSMFullName() string {")
	g.P("  return \"", ss.keyMessage.Desc.ParentFile().Package(), ".", ss.specifiedName, "\"")
	g.P("}")

}

// implements psm.IState for the state message
func (ss PSMEntity) implementIState(g *protogen.GeneratedFile) {
	g.P("// EXTEND ", ss.state.message.GoIdent, " with the psm.IState interface")
	ss.implementIPSMMessage(g, ss.state.message)
	g.P()
	g.P("func (msg *", ss.state.message.GoIdent, ") PSMMetadata() *", psmStateMetadataStruct, " {")
	g.P("  if msg.", ss.state.metadataField.GoName, " == nil {")
	g.P("    msg.", ss.state.metadataField.GoName, " = &", psmStateMetadataStruct, "{}")
	g.P("  }")
	g.P("  return msg.", ss.state.metadataField.GoName)
	g.P("}")
	g.P()
	g.P("func (msg *", ss.state.message.GoIdent, ") PSMKeys() *", ss.keyMessage.GoIdent, " {")
	g.P("  return msg.", ss.state.keyField.GoName)
	g.P("}")
	g.P()
	g.P("func (msg *", ss.state.message.GoIdent, ") SetPSMKeys(inner *", ss.keyMessage.GoIdent, ") {")
	g.P("  msg.", ss.state.keyField.GoName, " = inner")
	g.P("}")
}

// implements psm.IEvent for the event message
func (ss PSMEntity) implementIEvent(g *protogen.GeneratedFile) {
	g.P("// EXTEND ", ss.event.message.GoIdent, " with the psm.IEvent interface")
	g.P()
	ss.implementIPSMMessage(g, ss.event.message)
	g.P()
	g.P("func (msg *", ss.event.message.GoIdent, ") PSMMetadata() *", psmEventMetadataStruct, " {")
	g.P("  if msg.", ss.event.metadataField.GoName, " == nil {")
	g.P("    msg.", ss.event.metadataField.GoName, " = &", psmEventMetadataStruct, "{}")
	g.P("  }")
	g.P("  return msg.", ss.event.metadataField.GoName)
	g.P("}")
	g.P()
	g.P("func (msg *", ss.event.message.GoIdent, ") PSMKeys() *", ss.keyMessage.GoIdent, " {")
	g.P("  return msg.", ss.event.keyField.GoName)
	g.P("}")
	g.P()
	g.P("func (msg *", ss.event.message.GoIdent, ") SetPSMKeys(inner *", ss.keyMessage.GoIdent, ") {")
	g.P("  msg.", ss.event.keyField.GoName, " = inner")
	g.P("}")
	g.P()
	g.P("// PSMEventKey returns the ", ss.eventName, "PSMEventKey for the event, implementing psm.IEvent")
	g.P("func (msg *", ss.event.message.GoIdent, ") PSMEventKey() ", ss.namePrefix, "PSMEventKey {")
	g.P("   tt := msg.UnwrapPSMEvent()")
	g.P("   if tt == nil {")
	g.P("     return ", ss.namePrefix, "PSMEventNil")
	g.P("   }")
	g.P("	return tt.PSMEventKey()")
	g.P("}")
	g.P()
	g.P("// UnwrapPSMEvent implements psm.IEvent, returning the inner event message")
	g.P("func (msg *", ss.event.message.GoIdent, ") UnwrapPSMEvent() ", ss.eventName, " {")
	g.P("   if msg == nil {")
	g.P("     return nil")
	g.P("   }")
	g.P("   if msg.", ss.event.eventTypeField.GoName, " == nil {")
	g.P("     return nil")
	g.P("   }")
	g.P("	switch v := msg.", ss.event.eventTypeField.GoName, ".Type.(type) {")
	for _, field := range ss.event.eventTypeField.Message.Fields {
		g.P("	case *", field.GoIdent, ":")
		g.P("		return v.", field.GoName)
	}
	g.P("	default:")
	g.P("		return nil")
	g.P("	}")
	g.P("}")
	g.P()
	g.P("// SetPSMEvent sets the inner event message from a concrete type, implementing psm.IEvent")
	g.P("func (msg *", ss.event.message.GoIdent, ") SetPSMEvent(inner ", ss.eventName, ") error {")
	g.P("  if msg.", ss.event.eventTypeField.GoName, " == nil {")
	g.P("    msg.", ss.event.eventTypeField.GoName, " = &", ss.event.eventTypeField.Message.GoIdent, "{}")
	g.P("  }")
	g.P("  switch v := inner.(type) {")
	for _, field := range ss.event.eventTypeField.Message.Fields {
		g.P("	case *", field.Message.GoIdent, ":")
		g.P("		msg.", ss.event.eventTypeField.GoName, ".Type = &", field.GoIdent, "{", field.GoName, ": v}")
	}
	g.P("	default:")
	g.P("   return ", protogen.GoImportPath("fmt").Ident("Errorf"), "(\"invalid type %T for ", ss.event.eventTypeField.Message.GoIdent, "\", v)")
	g.P("	}")
	g.P("	return nil")
	g.P("}")
	g.P()

}

func (ss PSMEntity) implementIInnerEvent(g *protogen.GeneratedFile) {
	g.P()
	g.P("type ", ss.eventName, " interface {")
	g.P(smIInnerEvent)
	// already implied by the interface, but more specific.
	g.P("PSMEventKey() ", ss.eventName, "Key")
	g.P("}")
	g.P()
	for _, field := range ss.event.eventTypeField.Message.Fields {
		g.P("// EXTEND ", field.Message.GoIdent, " with the ", ss.eventName, " interface")
		ss.implementIPSMMessage(g, field.Message)
		g.P()
		g.P("func (*", field.Message.GoIdent, ") PSMEventKey() ", ss.eventName, "Key  {")
		g.P("		return ", ss.namePrefix, "PSMEvent", field.GoName)
		g.P("}")
		g.P()
	}

}

func (ss PSMEntity) transitionFuncTypes(g *protogen.GeneratedFile) {

	// FooPSMFunc - This is for backwards compatibility
	transitionBatonType := ss.typeAlias(g, "TransitionBaton", smTransitionBaton)
	g.P("func ", ss.machineName,
		"Func[SE ", ss.eventName, "]",
		"(cb func(",
		protogen.GoImportPath("context").Ident("Context"), ", ",
		transitionBatonType, ", *",
		ss.state.message.GoIdent, ", SE) error) ", smCombinedFunc, "[")
	ss.writeBaseTypesWithSE(g)
	g.P("] {")
	g.P("return ", smCombinedFunc, "[")
	ss.writeBaseTypesWithSE(g)
	g.P("](cb)")
	g.P("}")

	// FooPSMTransition
	g.P("func ", ss.machineName,
		"Transition[SE ", ss.eventName, "]",
		"(cb func(",
		protogen.GoImportPath("context").Ident("Context"), ", ",
		"*", ss.state.message.GoIdent, ", SE) error) ", smTransitionFunc, "[")
	ss.writeBaseTypesWithSE(g)
	g.P("] {")
	g.P("return ", smTransitionFunc, "[")
	ss.writeBaseTypesWithSE(g)
	g.P("](cb)")
	g.P("}")

	// FooPSMHook
	hookBatonType := ss.typeAlias(g, "HookBaton", smStateHookBaton)
	g.P("func ", ss.machineName,
		"Hook[SE ", ss.eventName, "]",
		"(cb func(",
		protogen.GoImportPath("context").Ident("Context"), ", ",
		protogen.GoImportPath("github.com/pentops/sqrlx.go/sqrlx").Ident("Transaction"), ", ",
		hookBatonType, ", *",
		ss.state.message.GoIdent, ", SE) error) ", smHookFunc, "[")
	ss.writeBaseTypesWithSE(g)
	g.P("] {")
	g.P("return ", smHookFunc, "[")
	ss.writeBaseTypesWithSE(g)
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
	ss.writeBaseTypes(g)
	g.P("] {")
	g.P("return ", smGeneralHookFunc, "[")
	ss.writeBaseTypes(g)
	g.P("](cb)")
	g.P("}")
}

func (ss PSMEntity) tableSpecAndConfig(g *protogen.GeneratedFile) {

	FooPSMTableSpec := ss.typeAlias(g, "TableSpec", smPSMTableSpec)
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
	g.P("  },")
	g.P("  EventPrimaryKey: func(id string, keys *", ss.keyMessage.GoIdent, ") (map[string]interface{}, error) {")
	g.P("    return map[string]interface{}{")
	g.P("      \"id\": id,")
	g.P("    }, nil")
	g.P("  },")
	g.P("  PrimaryKey: func(keys *", ss.keyMessage.GoIdent, ") (map[string]interface{}, error) {")
	g.P("    return map[string]interface{}{")
	for _, field := range ss.keyFields {
		if !field.PrimaryKey {
			continue
		}
		// stripping the prefix foo_ from the name in the event. In the DB, we
		// expect the primary key to be called just id, so foo_id -> id
		keyName := strings.TrimPrefix(string(field.field.Desc.Name()), ss.specifiedName+"_")
		g.P("      \"", keyName, "\": keys.", field.field.GoName, ",")
	}
	g.P("    }, nil")
	g.P("  },")

	g.P("}")

	g.P("func Default", ss.machineName, "Config() *", smStateMachineConfig, "[")
	ss.writeBaseTypes(g)
	g.P("] {")
	g.P("return ", smNewStateMachineConfig, "[")
	ss.writeBaseTypes(g)
	g.P("](", DefaultFooPSMTableSpec, ")")
	g.P("}")
	g.P()

	g.P("func New", ss.machineName, "(config *", smStateMachineConfig, "[")
	ss.writeBaseTypes(g)
	g.P("]) (*", ss.machineName, ", error) {")
	g.P("return ", smNewStateMachine, "[")
	ss.writeBaseTypes(g)
	g.P("](config)")
	g.P("}")
	g.P()

}
