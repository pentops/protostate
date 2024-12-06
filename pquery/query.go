package pquery

import (
	"context"
	"fmt"

	"github.com/lib/pq"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type aliasSet int

func (as *aliasSet) Next(name string) string {
	*as++
	return fmt.Sprintf("_%s__a%d", name, *as)
}

func newAliasSet() *aliasSet {
	return new(aliasSet)
}

type Transactor interface {
	Transact(ctx context.Context, opts *sqrlx.TxOptions, callback sqrlx.Callback) error
}

type AuthProvider interface {
	AuthFilter(ctx context.Context) (map[string]string, error)
}

type AuthProviderFunc func(ctx context.Context) (map[string]string, error)

func (f AuthProviderFunc) AuthFilter(ctx context.Context) (map[string]string, error) {
	return f(ctx)
}

// KeyJoin is a join on a primary (or unique) key in the RHS table.
// LEFT JOIN <TableName> ON <TableName>.<JoinKeyColumn> = <Main>.<MainKeyColumn>
// Main is defined in the outer struct holding this KeyJoin
type KeyJoin struct {
	TableName string
	On        JoinFields
}

// MethodDescriptor is the RequestResponse pair in the gRPC Method
type methodDescriptor[REQ proto.Message, RES proto.Message] struct {
	request  protoreflect.MessageDescriptor
	response protoreflect.MessageDescriptor
}

func newMethodDescriptor[REQ proto.Message, RES proto.Message]() *methodDescriptor[REQ, RES] {
	req := *new(REQ)
	res := *new(RES)
	return &methodDescriptor[REQ, RES]{
		request:  req.ProtoReflect().Descriptor(),
		response: res.ProtoReflect().Descriptor(),
	}
}

type SelectBuilder interface {
	Column(into ColumnDest, stmt string, args ...interface{})
	LeftJoin(join string, rest ...interface{})
	TableAlias(tableName string) string
}

type ColumnSpec interface {
	ApplyQuery(parentAlias string, sb SelectBuilder)
}

type jsonArrayColumn struct {
	ArrayJoinSpec
	fieldInParent protoreflect.FieldDescriptor // wraps the ListFooEventResponse type
}

func (join *jsonArrayColumn) NewRow() ScanDest {
	scanDest := pq.StringArray{}
	return &jsonArrayFieldRow{
		fieldInParent: join.fieldInParent,
		column:        &scanDest,
	}
}

func (join *jsonArrayColumn) ApplyQuery(parentTable string, sb SelectBuilder) {
	joinAlias := sb.TableAlias(join.TableName)
	sb.Column(join, fmt.Sprintf("ARRAY_AGG(%s.%s)", joinAlias, join.DataColumn))
	sb.LeftJoin(fmt.Sprintf(
		"%s AS %s ON %s",
		join.TableName,
		joinAlias,
		join.On.SQL(parentTable, joinAlias),
	))
}

type jsonArrayFieldRow struct {
	fieldInParent protoreflect.FieldDescriptor
	column        *pq.StringArray
}

func (join *jsonArrayFieldRow) ScanTo() interface{} {
	return join.column
}

func (join *jsonArrayFieldRow) Unmarshal(resReflect protoreflect.Message) error {

	elementList := resReflect.Mutable(join.fieldInParent).List()
	for _, eventBytes := range *join.column {
		if eventBytes == "" {
			continue
		}

		rowMessage := elementList.NewElement().Message()
		if err := protojson.Unmarshal([]byte(eventBytes), rowMessage.Interface()); err != nil {
			return fmt.Errorf("joined unmarshal: %w", err)
		}
		elementList.Append(protoreflect.ValueOf(rowMessage))
	}

	return nil
}
