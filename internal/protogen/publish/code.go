package publish

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

const (
	EventPublishMetadata = "j5.state.v1.EventPublishMetadata"

	psmProtoImportPath = protogen.GoImportPath("github.com/pentops/j5/gen/j5/state/v1/psm_j5pb")
	psmImportPath      = protogen.GoImportPath("github.com/pentops/protostate/psm")
)

type PSMPublishSet struct {
	name string

	publishMessage *protogen.Message
	metadataField  *protogen.Field
	keyField       *protogen.Field
	eventField     *protogen.Field
	dataField      *protogen.Field
	statusField    *protogen.Field
}

func (ps PSMPublishSet) Write(g *protogen.GeneratedFile) {
	g.P("// Publish Toipc for ", ps.name)
	importPath := ps.keyField.Message.GoIdent.GoImportPath
	nameStub := strings.TrimSuffix(ps.keyField.Message.GoIdent.GoName, "Keys")

	publishHook := importPath.Ident(nameStub + "PSMEventPublishHook")

	g.P("func Publish", nameStub, "() ", psmImportPath.Ident("GeneralEventHook"), "[")
	g.P("*", ps.keyField.Message.GoIdent, ", // implements psm.IKeyset")
	g.P("*", importPath.Ident(nameStub+"State"), ", // implements psm.IState")
	g.P(ps.statusField.Enum.GoIdent, ", // implements psm.IStatusEnum")
	g.P("*", ps.dataField.Message.GoIdent, ", // implements psm.IStateData")
	g.P("*", importPath.Ident(nameStub+"Event"), ", // implements psm.IEvent")
	g.P(importPath.Ident(nameStub+"PSMEvent"), ", // implements psm.IInnerEvent")
	g.P("] {")
	g.P("  return ", publishHook, "(func(")
	g.P("    ctx ", protogen.GoImportPath("context").Ident("Context"), ",")
	g.P("    publisher ", psmImportPath.Ident("Publisher"), ",")
	g.P("    state *", importPath.Ident(nameStub+"State"), ",")
	g.P("    event *", importPath.Ident(nameStub+"Event"), ",")
	g.P("  ) error {")
	g.P("    publisher.Publish(&", ps.publishMessage.GoIdent, "{")
	g.P("      Metadata: event.EventPublishMetadata(),")
	g.P("      Keys:   event.Keys,")
	g.P("      Event:  event.Event,")
	g.P("      Data:   state.Data,")
	g.P("      Status: state.Status,")
	g.P("    })")
	g.P("    return nil")
	g.P("  })")
	g.P("}")
}
