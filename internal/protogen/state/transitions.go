package state

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

const (
	docMutation     = "runs at the start of a transition to merge the event information into the state data object. The state object is mutable in this context."
	docLogicHook    = "runs after the mutation is complete. This hook can trigger side effects, including chained events, which are additional events processed by the state machine. Use this for Business Logic which determines the 'next step' in processing."
	docDataHook     = "runs after the mutations, and can be used to update data in tables which are not controlled as the state machine, e.g. for pre-calculating fields for performance reasons. Use of this hook prevents (future) transaction optimizations, as the transaction state when the function is called must needs to match the processing state, but only for this single transition, unlike the GeneralEventDataHook."
	docLinkHook     = "runs after the mutation and logic hook, and can be used to link the state machine to other state machines in the same database transaction"
	docLinkDBHook   = "like LinkHook, but has access to the current transaction for reads only (not enforced), use in place of controller logic to look up existing state."
	docGeneralLogic = "runs once per transition at the state-machine level regardless of which transition / event is being processed. It runs exactly once per transition, with the state object in the final state after the transition but prior to processing any further events. Chained events are added to the *end* of the event queue for the transaction, and side effects are published (as always) when the transaction is committed. The function MUST be pure, i.e. It MUST NOT produce any side-effects outside of the HookBaton, and MUST NOT modify the state."

	docGeneralStateData = "runs at the state-machine level regardless of which transition / event is being processed. It runs at-least once before committing a database transaction after multiple transitions are complete. This hook has access only to the final state after the transitions and is used to update other tables based on the resulting state. It MUST be idempotent, it may be called after injecting externally-held state data."
	docGeneralEventData = "runs after each transition at the state-machine level regardless of which transition / event is being processed. It runs exactly once per transition, before any other events are processed. The presence of this hook type prevents (future) transaction optimizations, so should be used sparingly."
	docEventPublish     = " EventPublishHook runs for each transition, at least once before committing a database transaction after multiple transitions are complete. It should publish a derived version of the event using the publisher."
	docEventUpsert      = "runs for each transition, at least once before committing a database transaction after multiple transitions are complete. It should publish a derived version of the event using the publisher."
)

func doc(name string, content string) string {
	return fmt.Sprintf("// %s %s", name, content)
}

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

func (ss PSMEntity) addTransitionFunc(g *protogen.GeneratedFile, spec callbackType, returnType any) {
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
			addParam(ss.alias.hookBaton)
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
	ss.writeBaseTypes(g)
	g.P("] {")
}

type transitionType struct {
	callbackType
	runOnFollow bool
	prep        func()
	body        func()
}

func (ss PSMEntity) addTransitionHook(g *protogen.GeneratedFile, spec transitionType) {
	ss.addTransitionFunc(g, spec.callbackType, smTransitionHook)
	g.P("eventType := (*new(SE)).PSMEventKey()")
	if spec.prep != nil {
		spec.prep()
	}
	g.P("return ", smTransitionHook, "[")
	ss.writeBaseTypes(g)
	g.P("]{")
	// types of the callback are fixed by smTransitionHook
	g.P("Callback: func(",
		"ctx ", contextContext, ", ",
		"tx ", sqrlxTransaction, ", ",
		"baton ", ss.alias.callbackBaton, ", ",
		"state *", ss.state.message.GoIdent, ", ",
		"event *", ss.event.message.GoIdent, ", ",
		") error {")
	// func body
	spec.body()

	g.P("},")
	g.P("EventType: eventType,")
	if spec.runOnFollow {
		g.P("RunOnFollow: true,")
	} else {
		g.P("RunOnFollow: false,")
	}
	g.P("}")
	g.P("}")
}

type generalType struct {
	callbackType
	runOnFollow bool
	hasEvent    bool
	body        func()
}

func (ss PSMEntity) addGeneralHook(g *protogen.GeneratedFile, spec generalType) {
	returnType := smGeneralStateHook
	if spec.hasEvent {
		returnType = smGeneralEventHook
	}
	ss.addTransitionFunc(g, spec.callbackType, returnType)
	g.P("return ", returnType, "[")
	ss.writeBaseTypes(g)
	g.P("]{")
	// types of the callback are fixed by smTransitionHook
	g.P("Callback: func(")
	g.P("ctx ", contextContext, ", ")
	g.P("tx ", sqrlxTransaction, ", ")
	g.P("baton ", ss.alias.callbackBaton, ", ")
	g.P("state *", ss.state.message.GoIdent, ", ")
	if spec.hasEvent {
		g.P("event *", ss.event.message.GoIdent, ", ")
	}
	g.P(") error {")
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

	convertType := func() {
		g.P("	asType, ok := any(event.UnwrapPSMEvent()).(SE)")
		g.P("	if !ok {")
		g.P("		name := event.ProtoReflect().Descriptor().FullName()")
		g.P(`		return fmt.Errorf("unexpected event type in transition: %s [IE] does not match [SE] (%T)", name, new(SE))`)
		g.P("	}")
	}

	// FooLogicHook
	ss.addTransitionHook(g, transitionType{
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
	ss.addTransitionHook(g, transitionType{
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

	ss.addTransitionHook(g, transitionType{
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
		prep: func() {
			g.P("wrapped := func(ctx context.Context, tx sqrlx.Transaction, state *", ss.state.message.GoIdent, ", event SE, add func(DK, DIE)) error {")
			g.P("	return cb(ctx, state, event, add)")
			g.P("}")

		},
		body: func() {
			g.P("return ", smRunLinkHook, "(ctx, linkDestination, wrapped, tx, state, event)")
		},
	})

	ss.addTransitionHook(g, transitionType{
		callbackType: callbackType{
			name:             "LinkDBHook",
			docStr:           docLinkDBHook,
			hasLink:          true,
			hasSpecificEvent: true,
			params: []param{
				paramContext,
				paramTx,
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

	ss.addGeneralHook(g, generalType{
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
		hasEvent:    true,
		body: func() {
			g.P("return cb(ctx, baton, state, event)")
		},
	})

	ss.addGeneralHook(g, generalType{
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

	ss.addGeneralHook(g, generalType{
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
		hasEvent:    true,
		runOnFollow: true,
		body: func() {
			g.P("return cb(ctx, tx, state, event)")
		},
	})

	ss.addGeneralHook(g, generalType{
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
		hasEvent:    true,
		body: func() {
			g.P("return cb(ctx, baton, state, event)")
		},
	})

	ss.addGeneralHook(g, generalType{
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
