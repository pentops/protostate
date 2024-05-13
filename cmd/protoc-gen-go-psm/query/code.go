package query

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	protoPackage        = protogen.GoImportPath("google.golang.org/protobuf/proto")
	stateMachinePackage = protogen.GoImportPath("github.com/pentops/protostate/psm")

	eventMetadataProtoName = protoreflect.FullName("psm.state.v1.EventMetadata")
	stateMetadataProtoName = protoreflect.FullName("psm.state.v1.StateMetadata")
)

func quoteString(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

type ListFilterField struct {
	DBName   string
	Getter   string
	Optional bool
}

type PSMQuerySet struct {
	GoServiceName string
	GetREQ        protogen.GoIdent
	GetRES        protogen.GoIdent
	ListREQ       protogen.GoIdent
	ListRES       protogen.GoIdent
	ListEventsREQ *protogen.GoIdent
	ListEventsRES *protogen.GoIdent

	ListRequestFilter       []ListFilterField
	ListEventsRequestFilter []ListFilterField
}

func (qs PSMQuerySet) Write(g *protogen.GeneratedFile) {
	g.P("// State Query Service for %s", qs.GoServiceName)
	g.P("// QuerySet is the query set for the ", qs.GoServiceName, " service.")
	g.P()
	qs.genericTypeAlias(g, qs.GoServiceName+"PSMQuerySet", "StateQuerySet")
	g.P("func New", qs.GoServiceName, "PSMQuerySet(")
	g.P("smSpec ", stateMachinePackage.Ident("QuerySpec"), "[")
	qs.writeGenericTypeSet(g)
	g.P("],")
	g.P("options ", stateMachinePackage.Ident("StateQueryOptions"), ",")
	g.P(") (*", qs.GoServiceName, "PSMQuerySet, error) {")
	g.P("return ", stateMachinePackage.Ident("BuildStateQuerySet"), "[")
	qs.writeGenericTypeSet(g)
	g.P("](smSpec, options)")
	g.P("}")
	g.P()
	qs.genericTypeAlias(g, qs.GoServiceName+"PSMQuerySpec", "QuerySpec")
	g.P()
	g.P("func Default", qs.GoServiceName, "PSMQuerySpec(tableSpec ", stateMachinePackage.Ident("QueryTableSpec"), ") ", qs.GoServiceName, "PSMQuerySpec {")
	g.P("  return ", stateMachinePackage.Ident("QuerySpec"), "[")
	qs.writeGenericTypeSet(g)
	g.P("  ]{")
	g.P("    QueryTableSpec: tableSpec,")
	qs.listFilter(g, qs.ListREQ, "ListRequestFilter", qs.ListRequestFilter)
	qs.listFilter(g, *qs.ListEventsREQ, "ListEventsRequestFilter", qs.ListEventsRequestFilter)
	g.P("  }")
	g.P("}")
	g.P()
}

func (qs PSMQuerySet) writeGenericTypeSet(g *protogen.GeneratedFile) {
	g.P("*", qs.GetREQ, ",")
	g.P("*", qs.GetRES, ",")
	g.P("*", qs.ListREQ, ",")
	g.P("*", qs.ListRES, ",")
	if qs.ListEventsREQ != nil {
		g.P("*", *qs.ListEventsREQ, ",")
		g.P("*", *qs.ListEventsRES, ",")
	} else {
		g.P(protoPackage.Ident("Message"), ",")
		g.P(protoPackage.Ident("Message"), ",")
	}
}

func (qs PSMQuerySet) genericTypeAlias(g *protogen.GeneratedFile, typedName string, psmName string) {
	g.P("type ", typedName, " = ", stateMachinePackage.Ident(psmName), "[")
	qs.writeGenericTypeSet(g)
	g.P("]")
	g.P()
}

func (qs PSMQuerySet) listFilter(g *protogen.GeneratedFile, reqType protogen.GoIdent, name string, fields []ListFilterField) {
	// ListRequestFilter       func(ListREQ) (map[string]interface{}, error)
	g.P(name, ": func(req *", reqType, ") (map[string]interface{}, error) {")
	g.P("  filter := map[string]interface{}{}")
	for _, field := range fields {
		if field.Optional {
			g.P("  if req.", field.Getter, " != nil {")
			g.P("    filter[", quoteString(field.DBName), "] = *req.", field.Getter)
			g.P("  }")
		} else {
			g.P("  filter[", quoteString(field.DBName), "] = req.", field.Getter)
		}
	}
	g.P("  return filter, nil")
	g.P("},")
}
