package main

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

type PSMEntity struct {
	specifiedName string

	namePrefix  string
	machineName string

	stateMessage *protogen.Message
	eventMessage *protogen.Message

	// Name of the inner event interface
	eventName string

	eventFields eventFieldGenerators
	stateFields stateFieldGenerators
}

func (ss PSMEntity) write(g *protogen.GeneratedFile) error {
	g.P("// StateObjectOptions: ", ss.machineName)

	return ss.addStateSet(g)
}

func (ss PSMEntity) printTypes(g *protogen.GeneratedFile) {
	g.P("*", ss.stateMessage.GoIdent, ",")
	g.P(ss.stateFields.statusField.Enum.GoIdent.GoName, ",")
	g.P("*", ss.eventMessage.GoIdent, ",")
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

	FooPSMConverter := ss.machineName + "Converter"
	DefaultFooPSMTableSpec := "Default" + ss.machineName + "TableSpec"

	g.P("func Default", ss.machineName, "Config() *", smStateMachineConfig, "[")
	ss.printTypes(g)
	g.P("] {")
	g.P("return ", smNewStateMachineConfig, "[")
	ss.printTypes(g)
	g.P("](", FooPSMConverter, "{}, ", DefaultFooPSMTableSpec, ")")
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

	g.P("type ", ss.machineName, "TransitionBaton = ", smTransitionBaton, "[*", ss.eventMessage.GoIdent, ", ", ss.eventName, "]")

	g.P()
	g.P("func ", ss.machineName,
		"Func[SE ", ss.eventName, "]",
		"(cb func(",
		protogen.GoImportPath("context").Ident("Context"), ", ",
		ss.machineName, "TransitionBaton, *",
		ss.stateMessage.GoIdent, ", SE) error) ", smTransitionFunc, "[")
	ss.printTypes(g)
	g.P("SE,")
	g.P("] {")
	g.P("return ", smTransitionFunc, "[")
	ss.printTypes(g)
	g.P("SE,")
	g.P("](cb)")
	g.P("}")

	g.P()
	g.P("type ", ss.eventName, "Key = string")
	g.P()
	g.P("const (")
	g.P(ss.namePrefix, "PSMEventNil ", ss.eventName, "Key = \"<nil>\"")
	for _, field := range ss.eventFields.eventTypeField.Message.Fields {
		g.P(ss.namePrefix, "PSMEvent", field.GoName, " ", ss.eventName, "Key = \"", field.Desc.Name(), "\"")
	}
	g.P(")")
	g.P()
	g.P("type ", ss.eventName, " interface {")
	g.P(protogen.GoImportPath("google.golang.org/protobuf/proto").Ident("Message"))
	g.P("PSMEventKey() ", ss.eventName, "Key")
	g.P("}")
	g.P()
	// Converting types
	if err := ss.addTypeConverter(g); err != nil {
		return err
	}
	g.P()
	if err := ss.eventTypeMethods(g); err != nil {
		return err
	}
	g.P()
	if err := ss.eventAliasMethods(g); err != nil {
		return err
	}
	g.P()

	for _, field := range ss.eventFields.eventTypeField.Message.Fields {
		g.P("func (*", field.Message.GoIdent, ") PSMEventKey() ", ss.eventName, "Key  {")
		g.P("		return ", ss.namePrefix, "PSMEvent", field.GoName)
		g.P("}")
	}

	// IF SEQUENCE

	if ss.eventFields.metadataFields.sequence != nil || ss.stateFields.lastSequenceField != nil {
		if ss.eventFields.metadataFields.sequence == nil {
			return fmt.Errorf("state message has sequence, but event does not")
		} else if ss.stateFields.lastSequenceField == nil {
			return fmt.Errorf("event message has sequence, but state does not")
		}
		if err := ss.addSequenceMethods(g); err != nil {
			return err
		}
	}

	return nil

}

func (ss PSMEntity) addSequenceMethods(g *protogen.GeneratedFile) error {

	g.P("func (ee *", ss.eventMessage.GoIdent, ") PSMSequence() uint64 {")
	g.P("  return ee.", ss.eventFields.metadataField.GoName, ".", ss.eventFields.metadataFields.sequence.GoName)
	g.P("}")
	g.P()
	g.P("func (ee *", ss.eventMessage.GoIdent, ") SetPSMSequence(seq uint64) {")
	g.P("  ee.", ss.eventFields.metadataField.GoName, ".", ss.eventFields.metadataFields.sequence.GoName, " = seq")
	g.P("}")
	g.P()
	g.P("func (st *", ss.stateMessage.GoIdent, ") LastPSMSequence() uint64 {")
	g.P("  return st.", ss.stateFields.lastSequenceField.GoName)
	g.P("}")
	g.P()
	g.P("func (st *", ss.stateMessage.GoIdent, ") SetLastPSMSequence(seq uint64) {")
	g.P("  st.", ss.stateFields.lastSequenceField.GoName, " = seq")
	g.P("}")

	return nil
}

func (ss PSMEntity) eventTypeMethods(g *protogen.GeneratedFile) error {

	g.P("func (etw *", ss.eventFields.eventTypeField.Message.GoIdent, ") UnwrapPSMEvent() ", ss.eventName, " {")
	g.P("   if etw == nil {")
	g.P("     return nil")
	g.P("   }")
	g.P("	switch v := etw.Type.(type) {")
	for _, field := range ss.eventFields.eventTypeField.Message.Fields {
		g.P("	case *", field.GoIdent, ":")
		g.P("		return v.", field.GoName)
	}
	g.P("	default:")
	g.P("		return nil")
	g.P("	}")
	g.P("}")

	g.P("func (etw *", ss.eventFields.eventTypeField.Message.GoIdent, ") PSMEventKey() ", ss.namePrefix, "PSMEventKey {")
	g.P("   tt := etw.UnwrapPSMEvent()")
	g.P("   if tt == nil {")
	g.P("     return ", ss.namePrefix, "PSMEventNil")
	g.P("   }")
	g.P("	return tt.PSMEventKey()")
	g.P("}")

	g.P("func (etw *", ss.eventFields.eventTypeField.Message.GoIdent, ") SetPSMEvent(inner ", ss.eventName, ") {")
	g.P("  switch v := inner.(type) {")
	for _, field := range ss.eventFields.eventTypeField.Message.Fields {
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

	g.P("func (ee *", ss.eventMessage.GoIdent, ") PSMEventKey() ", ss.namePrefix, "PSMEventKey {")
	g.P("	return ee.", ss.eventFields.eventTypeField.GoName, ".PSMEventKey()")
	g.P("}")
	g.P()
	g.P("func (ee *", ss.eventMessage.GoIdent, ") UnwrapPSMEvent() ", ss.eventName, " {")
	g.P("   return ee.", ss.eventFields.eventTypeField.GoName, ".UnwrapPSMEvent()")
	g.P("}")
	g.P()
	g.P("func (ee *", ss.eventMessage.GoIdent, ") SetPSMEvent(inner ", ss.eventName, ") {")
	g.P("  if ee.", ss.eventFields.eventTypeField.GoName, " == nil {")
	g.P("    ee.", ss.eventFields.eventTypeField.GoName, " = &", ss.eventFields.eventTypeField.Message.GoIdent, "{}")
	g.P("  }")
	g.P("  ee.", ss.eventFields.eventTypeField.GoName, ".SetPSMEvent(inner)")
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
	g.P("  State: ", smTableSpec, "[*", ss.stateMessage.GoIdent, "] {")
	g.P("    TableName: \"", ss.specifiedName, "\",")
	g.P("    DataColumn: \"state\",")
	g.P("    StoreExtraColumns: func(state *", ss.stateMessage.GoIdent, ") (map[string]interface{}, error) {")
	g.P("      return map[string]interface{}{")

	// Assume the NON KEY fields are stored as columns in the state table
	for _, field := range ss.eventFields.eventStateKeyFields {
		if field.isKey {
			continue
		}
		// stripping the prefix foo_ from the name in the event. In the DB, we
		// expect the primary key to be called just id, so foo_id -> id
		keyName := strings.TrimPrefix(string(field.stateField.Desc.Name()), ss.specifiedName+"_")
		g.P("      \"", keyName, "\": state.", field.eventField.GoName, ",")
	}
	g.P("      }, nil")
	g.P("    },")
	g.P("    PKFieldPaths: []string{")
	for _, field := range ss.eventFields.eventStateKeyFields {
		if !field.isKey {
			continue
		}
		g.P("\"", field.stateField.Desc.Name(), "\",")
	}
	g.P("    },")
	g.P("  },")

	g.P("  Event: ", smTableSpec, "[*", ss.eventMessage.GoIdent, "] {")
	g.P("  TableName: \"", ss.specifiedName, "_event\",")
	g.P("    DataColumn: \"data\",")
	g.P("    StoreExtraColumns: func(event *", ss.eventMessage.GoIdent, ") (map[string]interface{}, error) {")
	g.P("      metadata := event.", ss.eventFields.metadataField.GoName)
	g.P("      return map[string]interface{}{")
	g.P("        \"id\": metadata.", ss.eventFields.metadataFields.id.GoName, ",")
	g.P("        \"timestamp\": metadata.", ss.eventFields.metadataFields.timestamp.GoName, ",")
	if ss.eventFields.metadataFields.actor != nil {
		g.P("    \"actor\": metadata.", ss.eventFields.metadataFields.actor.GoName, ",")
	}
	// Assumes that all fields in the event marked as state key should be
	// directly written to the table. If not, they should not be in the
	// event, i.e. if they are derivable from the state, rather than
	// identifying the state, there is no need to copy them to the event.
	for _, field := range ss.eventFields.eventStateKeyFields {
		keyName := string(field.stateField.Desc.Name())
		g.P("      \"", keyName, "\": event.", field.eventField.GoName, ",")
	}
	g.P("      }, nil")
	g.P("    },")
	g.P("    PKFieldPaths: []string{")
	g.P("      \"", ss.eventFields.metadataField.Desc.Name(), ".", ss.eventFields.metadataFields.id.Desc.Name(), "\",")
	g.P("    },")
	g.P("    PK: func(event *", ss.eventMessage.GoIdent, ") (map[string]interface{}, error) {")
	g.P("      return map[string]interface{}{")
	g.P("        \"id\": event.", ss.eventFields.metadataField.GoName, ".", ss.eventFields.metadataFields.id.GoName, ",")
	g.P("      }, nil")
	g.P("    },")
	g.P("  },")

	g.P("  PrimaryKey: func(event *", ss.eventMessage.GoIdent, ") (map[string]interface{}, error) {")
	g.P("    return map[string]interface{}{")
	for _, field := range ss.eventFields.eventStateKeyFields {
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

	g.P("}")

	return nil
}

func (ss PSMEntity) addTypeConverter(g *protogen.GeneratedFile) error {
	timestamppb := protogen.GoImportPath("google.golang.org/protobuf/types/known/timestamppb")

	g.P("type ", ss.machineName, "Converter struct {}")
	g.P()
	g.P("func (c ", ss.machineName, "Converter) EmptyState(e *", ss.eventMessage.GoIdent, ") *", ss.stateMessage.GoIdent, " {")
	g.P("return &", ss.stateMessage.GoIdent, "{")
	for _, field := range ss.eventFields.eventStateKeyFields {
		g.P(field.stateField.GoName, ": e.", field.eventField.GoName, ",")
	}
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (c ", ss.machineName, "Converter) DeriveChainEvent(e *", ss.eventMessage.GoIdent, ", systemActor ", smSystemActor, ", eventKey string) *", ss.eventMessage.GoIdent, " {")
	g.P("  metadata := &", ss.eventFields.metadataField.Message.GoIdent, "{")
	g.P("  ", ss.eventFields.metadataFields.id.GoName, ": systemActor.NewEventID(e.", ss.eventFields.metadataField.GoName, ".", ss.eventFields.metadataFields.id.GoName, ", eventKey),")
	g.P("  ", ss.eventFields.metadataFields.timestamp.GoName, ": ", timestamppb.Ident("Now()"), ",")
	g.P("}")

	if ss.eventFields.metadataFields.actor != nil {
		g.P("actorProto := systemActor.ActorProto()")
		g.P("refl := metadata.ProtoReflect()")
		g.P("refl.Set(refl.Descriptor().Fields().ByName(\"", ss.eventFields.metadataFields.actor.Desc.Name(), "\"), actorProto)")
	}

	g.P("return &", ss.eventMessage.GoIdent, "{")
	g.P(ss.eventFields.metadataField.GoName, ": metadata,")
	for _, field := range ss.eventFields.eventStateKeyFields {
		g.P(field.eventField.GoName, ": e.", field.eventField.GoName, ",")
	}
	g.P("}")
	g.P("}")
	g.P()

	g.P("func (c ", ss.machineName, "Converter) CheckStateKeys(s *", ss.stateMessage.GoIdent, ", e *", ss.eventMessage.GoIdent, ") error {")
	for _, field := range ss.eventFields.eventStateKeyFields {
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
