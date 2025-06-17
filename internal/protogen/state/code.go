package state

import (
	"fmt"

	"github.com/pentops/j5/gen/j5/ext/v1/ext_j5pb"
	"github.com/pentops/j5/gen/j5/schema/v1/schema_j5pb"
	"github.com/pentops/protostate/psm"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	// all imports from PSM are defined here, i.e. this is the committed PSM interface.
	smImportPath         = protogen.GoImportPath("github.com/pentops/protostate/psm")
	smStateMachine       = smImportPath.Ident("StateMachine")
	smDBStateMachine     = smImportPath.Ident("DBStateMachine")
	smStateMachineConfig = smImportPath.Ident("StateMachineConfig")
	smStateHookBaton     = smImportPath.Ident("HookBaton")

	smTransitionMutation = smImportPath.Ident("TransitionMutation")
	smTransitionHook     = smImportPath.Ident("TransitionHook")
	smGeneralEventHook   = smImportPath.Ident("GeneralEventHook")
	smGeneralStateHook   = smImportPath.Ident("GeneralStateHook")
	smFullBaton          = smImportPath.Ident("CallbackBaton")

	smLinkDestination = smImportPath.Ident("LinkDestination")
	smRunLinkHook     = smImportPath.Ident("RunLinkHook")
	smPublisher       = smImportPath.Ident("Publisher")

	smEventSpec   = smImportPath.Ident("EventSpec")
	smIInnerEvent = smImportPath.Ident("IInnerEvent")
	smIKeyset     = smImportPath.Ident("IKeyset")

	psmProtoImportPath            = protogen.GoImportPath("github.com/pentops/j5/gen/j5/state/v1/psm_j5pb")
	psmEventMetadataStruct        = psmProtoImportPath.Ident("EventMetadata")
	psmStateMetadataStruct        = psmProtoImportPath.Ident("StateMetadata")
	psmEventPublishMetadataStruct = psmProtoImportPath.Ident("EventPublishMetadata")

	contextContext   = protogen.GoImportPath("context").Ident("Context")
	sqrlxTransaction = protogen.GoImportPath("github.com/pentops/sqrlx.go/sqrlx").Ident("Transaction")
)

type PSMEntity struct {
	specifiedName string // foo

	namePrefix  string // Foo
	machineName string // FooPSM
	eventName   string // FooPSMEvent

	keyMessage *protogen.Message
	state      *stateEntityState
	event      *stateEntityEvent

	tableMap *psm.TableMap
	alias    struct {
		hookBaton     string
		callbackBaton string
	}
}

func (ss PSMEntity) Write(g *protogen.GeneratedFile) {
	g.P("// PSM ", ss.machineName)
	g.P()

	ss.typeAlias(g, "", smStateMachine)
	ss.typeAlias(g, "DB", smDBStateMachine)
	ss.typeAlias(g, "EventSpec", smEventSpec)
	ss.alias.hookBaton = ss.typeAlias(g, "HookBaton", smStateHookBaton)
	ss.alias.callbackBaton = ss.typeAlias(g, "FullBaton", smFullBaton)
	g.P()
	ss.psmEventKey(g)
	g.P()
	ss.implementIKeyset(g)
	g.P()
	ss.implementIState(g)
	g.P()
	ss.implementIStateData(g)
	g.P()
	ss.implementIEvent(g)
	g.P()
	ss.implementIInnerEvent(g)
	g.P()
	ss.tableSpecAndConfig(g)
	g.P()
	ss.transitionFuncTypes(g)
	g.P()
	ss.eventPublishMetadata(g)
}

// prints the generic type parameters K, S, ST, SD, E, IE
func (ss PSMEntity) writeBaseTypes(g *protogen.GeneratedFile) {
	ss.writeBaseTypesNoEvent(g)
	g.P("*", ss.event.message.GoIdent.GoName, ", // implements psm.IEvent")
	g.P(ss.eventName, ", // implements psm.IInnerEvent")
}

func (ss PSMEntity) writeBaseTypesNoEvent(g *protogen.GeneratedFile) {
	g.P("*", ss.keyMessage.GoIdent.GoName, ", // implements psm.IKeyset")
	g.P("*", ss.state.message.GoIdent.GoName, ", // implements psm.IState")
	g.P(ss.state.statusField.Enum.GoIdent.GoName, ", // implements psm.IStatusEnum")
	g.P("*", ss.state.dataField.Message.GoIdent.GoName, ", // implements psm.IStateData")
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

	keyColumns := ss.tableMap.KeyColumns

	g.P("func (msg *", ss.keyMessage.GoIdent, ") PSMKeyValues() (map[string]any, error) {")
	g.P("  keyset := map[string]any{")
	for _, columnSpec := range keyColumns {
		if !columnSpec.Primary {
			continue
		}
		field := fieldByDesc(ss.keyMessage.Fields, columnSpec.ProtoName)
		g.P("      \"", columnSpec.ColumnName, "\": msg.", field.GoName, ",")
	}
	g.P("  }")
	for _, columnSpec := range keyColumns {
		if columnSpec.Primary {
			continue
		}
		field := fieldByDesc(ss.keyMessage.Fields, columnSpec.ProtoName)
		if columnSpec.ExplicitlyOptional {
			g.P("if msg.", field.GoName, " != nil {")
			g.P("  keyset[\"", columnSpec.ColumnName, "\"] = *msg.", field.GoName)
			g.P("}")
		} else {
			g.P("if msg.", field.GoName, " != \"\" {")
			g.P("  keyset[\"", columnSpec.ColumnName, "\"] = msg.", field.GoName)
			g.P("}")
		}
	}
	g.P("  return keyset, nil")
	g.P("}")

}

// implements psm.IState for the state message
func (ss PSMEntity) implementIState(g *protogen.GeneratedFile) {
	stateMessage := ss.state.message
	g.P("// EXTEND ", stateMessage.GoIdent, " with the psm.IState interface")
	ss.implementIPSMMessage(g, stateMessage)
	g.P()
	g.P("func (msg *", stateMessage.GoIdent, ") PSMMetadata() *", psmStateMetadataStruct, " {")
	g.P("  if msg.", ss.state.metadataField.GoName, " == nil {")
	g.P("    msg.", ss.state.metadataField.GoName, " = &", psmStateMetadataStruct, "{}")
	g.P("  }")
	g.P("  return msg.", ss.state.metadataField.GoName)
	g.P("}")
	g.P()
	g.P("func (msg *", stateMessage.GoIdent, ") PSMKeys() *", ss.keyMessage.GoIdent, " {")
	g.P("  return msg.", ss.state.keyField.GoName)
	g.P("}")
	g.P()
	g.P("func (msg *", stateMessage.GoIdent, ") SetStatus(status ", ss.state.statusField.Enum.GoIdent, ") {")
	g.P("  msg.", ss.state.statusField.GoName, " = status")
	g.P("}")
	g.P()
	g.P("func (msg *", stateMessage.GoIdent, ") SetPSMKeys(inner *", ss.keyMessage.GoIdent, ") {")
	g.P("  msg.", ss.state.keyField.GoName, " = inner")
	g.P("}")
	g.P()
	g.P("func (msg *", stateMessage.GoIdent, ") PSMData() *", ss.state.dataField.Message.GoIdent, " {")
	g.P("  if msg.", ss.state.dataField.GoName, " == nil {")
	g.P("    msg.", ss.state.dataField.GoName, " = &", ss.state.dataField.Message.GoIdent, "{}")
	g.P("  }")
	g.P("  return msg.", ss.state.dataField.GoName)
	g.P("}")
}

// implements psm.IStateData for the key message
func (ss PSMEntity) implementIStateData(g *protogen.GeneratedFile) {
	dataMessage := ss.state.dataField.Message
	g.P("// EXTEND ", dataMessage.GoIdent, " with the psm.IStateData interface")
	ss.implementIPSMMessage(g, dataMessage)
}

// implements psm.IEvent for the event message
func (ss PSMEntity) implementIEvent(g *protogen.GeneratedFile) {
	eventMessage := ss.event.message
	g.P("// EXTEND ", eventMessage.GoIdent, " with the psm.IEvent interface")
	g.P()
	ss.implementIPSMMessage(g, eventMessage)
	g.P()
	g.P("func (msg *", eventMessage.GoIdent, ") PSMMetadata() *", psmEventMetadataStruct, " {")
	g.P("  if msg.", ss.event.metadataField.GoName, " == nil {")
	g.P("    msg.", ss.event.metadataField.GoName, " = &", psmEventMetadataStruct, "{}")
	g.P("  }")
	g.P("  return msg.", ss.event.metadataField.GoName)
	g.P("}")
	g.P()
	g.P("func (msg *", eventMessage.GoIdent, ") PSMKeys() *", ss.keyMessage.GoIdent, " {")
	g.P("  return msg.", ss.event.keyField.GoName)
	g.P("}")
	g.P()
	g.P("func (msg *", eventMessage.GoIdent, ") SetPSMKeys(inner *", ss.keyMessage.GoIdent, ") {")
	g.P("  msg.", ss.event.keyField.GoName, " = inner")
	g.P("}")
	g.P()
	g.P("// PSMEventKey returns the ", ss.eventName, "PSMEventKey for the event, implementing psm.IEvent")
	g.P("func (msg *", eventMessage.GoIdent, ") PSMEventKey() ", ss.namePrefix, "PSMEventKey {")
	g.P("   tt := msg.UnwrapPSMEvent()")
	g.P("   if tt == nil {")
	g.P("     return ", ss.namePrefix, "PSMEventNil")
	g.P("   }")
	g.P("	return tt.PSMEventKey()")
	g.P("}")
	g.P()
	g.P("// UnwrapPSMEvent implements psm.IEvent, returning the inner event message")
	g.P("func (msg *", eventMessage.GoIdent, ") UnwrapPSMEvent() ", ss.eventName, " {")
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

func (ss PSMEntity) eventPublishMetadata(g *protogen.GeneratedFile) {
	g.P("func (event *", ss.event.message.GoIdent, ") EventPublishMetadata() *", psmEventPublishMetadataStruct, " {")
	keyIdent := psmProtoImportPath.Ident("EventTenant")
	g.P("  tenantKeys := make([]*", keyIdent, ", 0)")
	for _, field := range ss.keyMessage.Fields {
		// TODO: Map tenant keys
		opt := proto.GetExtension(field.Desc.Options(), ext_j5pb.E_Key).(*schema_j5pb.EntityKey)
		if opt == nil {
			continue
		}
		if opt.Tenant != nil {
			if field.Desc.HasOptionalKeyword() {
				g.P("if event.Keys.", field.GoName, " != nil {")
				g.P("  tenantKeys = append(tenantKeys, &", keyIdent, "{")
				g.P("    TenantType: \"", *opt.Tenant, "\",")
				g.P("    TenantId:  *event.Keys.", field.GoName, ",")
				g.P("  })")
				g.P("}")
			} else {
				g.P("if event.Keys.", field.GoName, " != \"\" {")
				g.P("  tenantKeys = append(tenantKeys, &", keyIdent, "{")
				g.P("    TenantType: \"", *opt.Tenant, "\",")
				g.P("    TenantId:  event.Keys.", field.GoName, ",")
				g.P("  })")
				g.P("}")
			}
		}
	}
	g.P("  return &", psmEventPublishMetadataStruct, "{")
	g.P("    EventId:   event.", ss.event.metadataField.GoName, ".EventId,")
	g.P("    Sequence:  event.", ss.event.metadataField.GoName, ".Sequence,")
	g.P("    Timestamp: event.", ss.event.metadataField.GoName, ".Timestamp,")
	g.P("    Cause:     event.", ss.event.metadataField.GoName, ".Cause,")
	g.P("    Auth: &", psmProtoImportPath.Ident("PublishAuth"), "{")
	g.P("      TenantKeys: tenantKeys,")
	g.P("    },")
	g.P("  }")
	g.P("}")
}

func (ss PSMEntity) tableSpecAndConfig(g *protogen.GeneratedFile) {
	g.P("func ", ss.machineName, "Builder() *", smStateMachineConfig, "[")
	ss.writeBaseTypes(g)
	g.P("] {")
	g.P("return &", smStateMachineConfig, "[")
	ss.writeBaseTypes(g)
	g.P("]{}")
	g.P("}")
	g.P()
}

func fieldByDesc(fields []*protogen.Field, desc protoreflect.Name) *protogen.Field {
	for _, f := range fields {
		if f.Desc.Name() == desc {
			return f
		}
	}
	return nil
}
