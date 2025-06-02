package state

import (
	"fmt"

	"github.com/pentops/j5/gen/j5/ext/v1/ext_j5pb"
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
}

func (ss PSMEntity) Write(g *protogen.GeneratedFile) {
	g.P("// PSM ", ss.machineName)
	g.P()

	ss.typeAlias(g, "", smStateMachine)
	ss.typeAlias(g, "DB", smDBStateMachine)
	ss.typeAlias(g, "EventSpec", smEventSpec)
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

	g.P("func (msg *", ss.keyMessage.GoIdent, ") PSMKeyValues() (map[string]string, error) {")
	g.P("  keyset := map[string]string{")
	for _, columnSpec := range keyColumns {
		if !columnSpec.Required {
			continue
		}
		field := fieldByDesc(ss.keyMessage.Fields, columnSpec.ProtoName)
		g.P("      \"", columnSpec.ColumnName, "\": msg.", field.GoName, ",")
	}
	g.P("  }")
	for _, columnSpec := range keyColumns {
		if columnSpec.Required {
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
		opt := proto.GetExtension(field.Desc.Options(), ext_j5pb.E_Key).(*ext_j5pb.PSMKeyFieldOptions)
		if opt == nil {
			continue
		}
		if opt.TenantType != nil {
			if field.Desc.HasOptionalKeyword() {
				g.P("if event.Keys.", field.GoName, " != nil {")
				g.P("  tenantKeys = append(tenantKeys, &", keyIdent, "{")
				g.P("    TenantType: \"", *opt.TenantType, "\",")
				g.P("    TenantId:  *event.Keys.", field.GoName, ",")
				g.P("  })")
				g.P("}")
			} else {
				g.P("if event.Keys.", field.GoName, " != \"\" {")
				g.P("  tenantKeys = append(tenantKeys, &", keyIdent, "{")
				g.P("    TenantType: \"", *opt.TenantType, "\",")
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

const (
	docMutation  = "runs at the start of a transition to merge the event information into the state data object. The state object is mutable in this context."
	docLogicHook = "runs after the mutation is complete. This hook can trigger side effects, including chained events, which are additional events processed by the state machine. Use this for Business Logic which determines the 'next step' in processing."
	docDataHook  = "runs after the mutations, and can be used to update data in tables which are not controlled as the state machine, e.g. for pre-calculating fields for performance reasons. Use of this hook prevents (future) transaction optimizations, as the transaction state when the function is called must needs to match the processing state, but only for this single transition, unlike the GeneralEventDataHook."
	docLinkHook  = "runs after the mutation and logic hook, and can be used to link the state machine to other state machines in the same database transaction"

	docGeneralLogic = "runs once per transition at the state-machine level regardless of which transition / event is being processed. It runs exactly once per transition, with the state object in the final state after the transition but prior to processing any further events. Chained events are added to the *end* of the event queue for the transaction, and side effects are published (as always) when the transaction is committed. The function MUST be pure, i.e. It MUST NOT produce any side-effects outside of the HookBaton, and MUST NOT modify the state."

	docGeneralStateData = "runs at the state-machine level regardless of which transition / event is being processed. It runs at-least once before committing a database transaction after multiple transitions are complete. This hook has access only to the final state after the transitions and is used to update other tables based on the resulting state. It MUST be idempotent, it may be called after injecting externally-held state data."
	docGeneralEventData = "runs after each transition at the state-machine level regardless of which transition / event is being processed. It runs exactly once per transition, before any other events are processed. The presence of this hook type prevents (future) transaction optimizations, so should be used sparingly."
	docEventPublish     = " EventPublishHook runs for each transition, at least once before committing a database transaction after multiple transitions are complete. It should publish a derived version of the event using the publisher."
	docEventUpsert      = "runs for each transition, at least once before committing a database transaction after multiple transitions are complete. It should publish a derived version of the event using the publisher."
)

func doc(name string, content string) string {
	return fmt.Sprintf("// %s %s", name, content)
}

func (ss PSMEntity) transitionFuncTypes(g *protogen.GeneratedFile) {

	// FooMutation
	mutationName := fmt.Sprintf("%sMutation", ss.machineName)
	g.P(doc(mutationName, docMutation))
	g.P("func ", mutationName, "[SE ", ss.eventName, "]",
		"(cb func(*", ss.state.dataField.Message.GoIdent, ", SE) error) ", smTransitionMutation, "[")
	ss.writeBaseTypesWithSE(g)
	g.P("] {")
	g.P("return ", smTransitionMutation, "[")
	ss.writeBaseTypesWithSE(g)
	g.P("](cb)")
	g.P("}")
	g.P()

	hookBatonType := ss.typeAlias(g, "HookBaton", smStateHookBaton)
	callbackBatonType := ss.typeAlias(g, "FullBaton", smFullBaton)

	type param int
	const (
		paramContext param = iota
		paramTx
		paramBaton
		paramState
		paramStateData
		paramEvent
		paramSpecificEvent
		paramLinkFunc
		paramPublisher
	)

	type callbackType struct {
		name   string
		docStr string

		hasLink          bool
		hasSpecificEvent bool

		params []param
	}

	addFunc := func(spec callbackType, returnType any) {
		funcName := fmt.Sprintf("%s%s", ss.machineName, spec.name)
		g.P(doc(funcName, spec.docStr))
		if spec.hasSpecificEvent || spec.hasLink {
			g.P("func ", funcName, "[")
			if spec.hasSpecificEvent {
				g.P("\tSE ", ss.eventName, ",")
			}
			if spec.hasLink {
				g.P("\tDK ", smIKeyset, ",")
				g.P("\tDIE ", smIInnerEvent, ",")
			}
			g.P("](")
		} else {
			g.P("func ", funcName, "(")
		}

		if spec.hasLink {
			g.P("\tlinkDestination ", smLinkDestination, "[DK, DIE], ")
		}

		g.P("\tcb func(")

		addParam := func(param ...any) {
			g.P(append(param, ", ")...)
		}

		for _, p := range spec.params {
			switch p {
			case paramContext:
				addParam(contextContext)
			case paramTx:
				addParam(sqrlxTransaction)
			case paramBaton:
				addParam(hookBatonType)
			case paramState:
				addParam("*", ss.state.message.GoIdent)
			case paramStateData:
				addParam("*", ss.state.dataField.Message.GoIdent)
			case paramEvent:
				addParam("*", ss.event.message.GoIdent)
			case paramSpecificEvent:
				addParam("SE")
			case paramLinkFunc:
				addParam("func(DK, DIE)")
			case paramPublisher:
				addParam(smPublisher)

			default:
				panic(fmt.Sprintf("unknown param type %d in callback %s", p, spec.name))
			}
		}

		g.P("\t) error) ", returnType, "[")
		if spec.hasSpecificEvent {
			ss.writeBaseTypesWithSE(g)
		} else {
			ss.writeBaseTypes(g)
		}
		g.P("] {")
	}

	type transitionType struct {
		callbackType
		runOnFollow bool
		body        func()
	}

	addTransitionHook := func(spec transitionType) {
		addFunc(spec.callbackType, smTransitionHook)
		g.P("return ", smTransitionHook, "[")
		ss.writeBaseTypesWithSE(g)
		g.P("]{")
		// types of the callback are fixed by smTransitionHook
		g.P("Callback: func(",
			"ctx ", contextContext, ", ",
			"tx ", sqrlxTransaction, ", ",
			"baton ", callbackBatonType, ", ",
			"state *", ss.state.message.GoIdent, ", ",
			"event *", ss.event.message.GoIdent, ", ",
			") error {")
		// func body
		spec.body()

		g.P("},")
		if spec.runOnFollow {
			g.P("RunOnFollow: true,")
		} else {
			g.P("RunOnFollow: false,")
		}
		g.P("}")
		g.P("}")
	}

	convertType := func() {
		g.P("	asType, ok := any(event.UnwrapPSMEvent()).(SE)")
		g.P("	if !ok {")
		g.P("		name := event.ProtoReflect().Descriptor().FullName()")
		g.P(`		return fmt.Errorf("unexpected event type in transition: %s [IE] does not match [SE] (%T)", name, new(SE))`)
		g.P("	}")
	}

	// FooLogicHook
	addTransitionHook(transitionType{
		callbackType: callbackType{
			name:             "LogicHook",
			docStr:           docLogicHook,
			hasSpecificEvent: true,
			params: []param{
				paramContext,
				paramBaton,
				paramState,
				paramSpecificEvent,
			},
		},
		runOnFollow: false,
		body: func() {
			convertType()
			g.P("return cb(ctx, baton, state, asType)")
		},
	})

	// FooDataHook
	addTransitionHook(transitionType{
		callbackType: callbackType{
			name:             "DataHook",
			docStr:           docDataHook,
			hasSpecificEvent: true,
			params: []param{
				paramContext,
				paramTx,
				paramStateData,
				paramSpecificEvent,
			},
		},
		runOnFollow: true,
		body: func() {
			convertType()
			g.P("return cb(ctx, tx, state.PSMData(), asType)")
		},
	})

	// FooPSMLinkHook
	addTransitionHook(transitionType{
		callbackType: callbackType{
			name:             "LinkHook",
			docStr:           docLinkHook,
			hasLink:          true,
			hasSpecificEvent: true,
			params: []param{
				paramContext,
				paramState,
				paramSpecificEvent,
				paramLinkFunc,
			},
		},
		runOnFollow: false,
		body: func() {
			g.P("return ", smRunLinkHook, "(ctx, linkDestination, cb, tx, state, event)")
		},
	})

	type generalType struct {
		callbackType
		runOnFollow bool
		body        func()
	}

	addGeneralHook := func(spec generalType) {
		addFunc(spec.callbackType, smGeneralEventHook)
		g.P("return ", smGeneralEventHook, "[")
		ss.writeBaseTypes(g)
		g.P("]{")
		// types of the callback are fixed by smTransitionHook
		g.P("Callback: func(",
			"ctx ", contextContext, ", ",
			"tx ", sqrlxTransaction, ", ",
			"baton ", callbackBatonType, ", ",
			"state *", ss.state.message.GoIdent, ", ",
			"event *", ss.event.message.GoIdent, ", ",
			") error {")
		spec.body()

		g.P("},")
		if spec.runOnFollow {
			g.P("RunOnFollow: true,")
		} else {
			g.P("RunOnFollow: false,")
		}
		g.P("}")
		g.P("}")
	}

	type generalStateType struct {
		callbackType
		runOnFollow bool
		body        func()
	}

	addGeneralStateHook := func(spec generalStateType) {
		addFunc(spec.callbackType, smGeneralStateHook)
		g.P("return ", smGeneralStateHook, "[")
		ss.writeBaseTypes(g)
		g.P("]{")
		// types of the callback are fixed by smTransitionHook
		g.P("Callback: func(",
			"ctx ", contextContext, ", ",
			"tx ", sqrlxTransaction, ", ",
			"baton ", callbackBatonType, ", ",
			"state *", ss.state.message.GoIdent, ", ",
			") error {")
		spec.body()

		g.P("},")
		if spec.runOnFollow {
			g.P("RunOnFollow: true,")
		} else {
			g.P("RunOnFollow: false,")
		}
		g.P("}")
		g.P("}")

	}

	// FooPSMGenericLogicHook
	addGeneralHook(generalType{
		callbackType: callbackType{
			name:   "GeneralLogicHook",
			docStr: docGeneralLogic,
			params: []param{
				paramContext,
				paramBaton,
				paramState,
				paramEvent,
			},
		},
		runOnFollow: false,
		body: func() {
			g.P("return cb(ctx, baton, state, event)")
		},
	})

	// FooPSMGeneralStateDataHook
	addGeneralStateHook(generalStateType{
		callbackType: callbackType{
			name:   "GeneralStateDataHook",
			docStr: docGeneralStateData,
			params: []param{
				paramContext,
				paramTx,
				paramState,
			},
		},
		runOnFollow: true,
		body: func() {
			g.P("return cb(ctx, tx, state)")
		},
	})

	// FooPSMGeneralStateDataHook
	addGeneralHook(generalType{
		callbackType: callbackType{
			name:   "GeneralEventDataHook",
			docStr: docGeneralEventData,
			params: []param{
				paramContext,
				paramTx,
				paramState,
				paramEvent,
			},
		},
		runOnFollow: true,
		body: func() {
			g.P("return cb(ctx, tx, state, event)")
		},
	})

	// FooPSMEventPublishHook
	addGeneralHook(generalType{
		callbackType: callbackType{
			name:   "EventPublishHook",
			docStr: docEventPublish,
			params: []param{
				paramContext,
				paramPublisher,
				paramState,
				paramEvent,
			},
		},
		runOnFollow: false,
		body: func() {
			g.P("return cb(ctx, baton, state, event)")
		},
	})

	// FooPSMUpsertPublishHook
	addGeneralHook(generalType{
		callbackType: callbackType{
			name:   "UpsertPublishHook",
			docStr: docEventUpsert,
			params: []param{
				paramContext,
				paramPublisher,
				paramState,
			},
		},
		runOnFollow: false,
		body: func() {
			g.P("return cb(ctx, baton, state)")
		},
	})
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
