package query

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

var (
	protoPackage        = protogen.GoImportPath("google.golang.org/protobuf/proto")
	stateMachinePackage = protogen.GoImportPath("github.com/pentops/protostate/psm")
	sqrlxPkg            = protogen.GoImportPath("github.com/pentops/sqrlx.go/sqrlx")
	contextPkg          = protogen.GoImportPath("context")
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
	GoName  string
	Service *protogen.Service

	GetMethod protogen.Method
	GetREQ    protogen.GoIdent
	GetRES    protogen.GoIdent

	ListMethod protogen.Method
	ListREQ    protogen.GoIdent
	ListRES    protogen.GoIdent

	ListEventsMethod *protogen.Method
	ListEventsREQ    *protogen.GoIdent
	ListEventsRES    *protogen.GoIdent

	ListRequestFilter       []ListFilterField
	ListEventsRequestFilter []ListFilterField
}

func (qs PSMQuerySet) Write(g *protogen.GeneratedFile) {
	g.P("// State Query Service for %s", qs.GoName)
	g.P("// QuerySet is the query set for the ", qs.GoName, " service.")
	g.P()
	qs.writeQuerySet(g)
	g.P()
	qs.writeQuerySpec(g)
	g.P()
	qs.writeDefaultService(g)
}

func (qs PSMQuerySet) writeQuerySet(g *protogen.GeneratedFile) {
	qs.genericTypeAlias(g, qs.GoName+"PSMQuerySet", "StateQuerySet")
	g.P("func New", qs.GoName, "PSMQuerySet(")
	g.P("smSpec ", stateMachinePackage.Ident("QuerySpec"), "[")
	qs.writeGenericTypeSet(g)
	g.P("],")
	g.P("options ", stateMachinePackage.Ident("StateQueryOptions"), ",")
	g.P(") (*", qs.GoName, "PSMQuerySet, error) {")
	g.P("return ", stateMachinePackage.Ident("BuildStateQuerySet"), "[")
	qs.writeGenericTypeSet(g)
	g.P("](smSpec, options)")
	g.P("}")
}

func (qs PSMQuerySet) writeQuerySpec(g *protogen.GeneratedFile) {
	qs.genericTypeAlias(g, qs.GoName+"PSMQuerySpec", "QuerySpec")
	g.P()
	g.P("func Default", qs.GoName, "PSMQuerySpec(tableSpec ", stateMachinePackage.Ident("QueryTableSpec"), ") ", qs.GoName, "PSMQuerySpec {")
	g.P("  return ", stateMachinePackage.Ident("QuerySpec"), "[")
	qs.writeGenericTypeSet(g)
	g.P("  ]{")
	g.P("    QueryTableSpec: tableSpec,")
	qs.listFilter(g, qs.ListREQ, "ListRequestFilter", qs.ListRequestFilter)
	qs.listFilter(g, *qs.ListEventsREQ, "ListEventsRequestFilter", qs.ListEventsRequestFilter)
	g.P("  }")
	g.P("}")
}

func (qs PSMQuerySet) writeDefaultService(g *protogen.GeneratedFile) {

	serviceName := qs.GoName + "QueryServiceImpl"
	g.P("type ", serviceName, " struct {")
	g.P("db ", sqrlxPkg.Ident("Transactor"))
	g.P("querySet *", qs.GoName, "PSMQuerySet")
	g.P("Unsafe", qs.Service.GoName, "Server")
	g.P("}")
	g.P()
	g.P("var _ ", qs.Service.GoName, "Server = &", serviceName, "{}")
	g.P()
	g.P("func New", serviceName, "(db ", sqrlxPkg.Ident("Transactor"), ", querySet *", qs.GoName, "PSMQuerySet) *", serviceName, " {")
	g.P("  return &", serviceName, "{")
	g.P("    db: db,")
	g.P("    querySet: querySet,")
	g.P("  }")
	g.P("}")
	g.P()
	g.P("func (s *", serviceName, ") ", qs.GetMethod.GoName, "(ctx ", contextPkg.Ident("Context"), ", req *", qs.GetREQ.GoName, ") (*", qs.GetRES.GoName, ", error) {")
	g.P("  resObject := &", qs.GetRES.GoName, "{}")
	g.P("  err := s.querySet.Get(ctx, s.db, req, resObject)")
	g.P("  if err != nil {")
	g.P("    return nil, err")
	g.P("  }")
	g.P("  return resObject, nil")
	g.P("}")
	g.P()
	g.P("func (s *", serviceName, ") ", qs.ListMethod.GoName, "(ctx ", contextPkg.Ident("Context"), ", req *", qs.ListREQ.GoName, ") (*", qs.ListRES.GoName, ", error) {")
	g.P("  resObject := &", qs.ListRES.GoName, "{}")
	g.P("  err := s.querySet.List(ctx, s.db, req, resObject)")
	g.P("  if err != nil {")
	g.P("    return nil, err")
	g.P("  }")
	g.P("  return resObject, nil")
	g.P("}")
	if qs.ListEventsMethod == nil {
		return
	}
	g.P()
	g.P("func (s *", serviceName, ") ", qs.ListEventsMethod.GoName, "(ctx ", contextPkg.Ident("Context"), ", req *", *qs.ListEventsREQ, ") (*", *qs.ListEventsRES, ", error) {")
	g.P("  resObject := &", *qs.ListEventsRES, "{}")
	g.P("  err := s.querySet.ListEvents(ctx, s.db, req, resObject)")
	g.P("  if err != nil {")
	g.P("    return nil, err")
	g.P("  }")
	g.P("  return resObject, nil")
	g.P("}")
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
