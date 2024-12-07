package pquery

import (
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ColumnDest interface {
	NewRow() ScanDest
}

type ScanDest interface {
	ScanTo() interface{}
	Unmarshal(protoreflect.Message) error
}

type SelectBuilder interface {
	Column(into ColumnDest, stmt string, args ...interface{})
	LeftJoin(join string, rest ...interface{})
	TableAlias(tableName string) string
}

type selectBuilder struct {
	*sq.SelectBuilder
	aliasSet  *aliasSet
	rootAlias string
	columns   []ColumnDest
}

func newSelectBuilder(rootTable string) *selectBuilder {
	as := newAliasSet()
	rootAlias := as.Next(rootTable)
	sb := sq.Select().
		From(fmt.Sprintf("%s AS %s", rootTable, rootAlias))

	return &selectBuilder{
		SelectBuilder: sb,
		aliasSet:      as,
		rootAlias:     rootAlias,
	}
}

func (sb *selectBuilder) NewRow() ([]ScanDest, []interface{}) {
	fields := make([]ScanDest, 0, len(sb.columns))
	rowCols := make([]interface{}, 0, len(sb.columns))
	for _, inQuery := range sb.columns {
		colRow := inQuery.NewRow()
		fields = append(fields, colRow)
		rowCols = append(rowCols, colRow.ScanTo())
	}
	return fields, rowCols
}

func (sb *selectBuilder) Column(into ColumnDest, stmt string, args ...interface{}) {
	sb.SelectBuilder.Column(stmt, args...)
	sb.columns = append(sb.columns, into)
}

func (sb *selectBuilder) LeftJoin(join string, rest ...interface{}) {
	sb.SelectBuilder.LeftJoin(join, rest...)
}

func (sb *selectBuilder) TableAlias(tableName string) string {
	return sb.aliasSet.Next(tableName)
}

type ColumnSpec interface {
	ApplyQuery(parentAlias string, sb SelectBuilder)
}

type jsonColumn struct {
	sqlColumn string
	field     protoreflect.FieldDescriptor
}

func newJsonColumn(sqlColumn string, protoField protoreflect.FieldDescriptor) jsonColumn {
	return jsonColumn{
		sqlColumn: sqlColumn,
		field:     protoField,
	}
}

func (jc jsonColumn) ApplyQuery(tableAlias string, sb SelectBuilder) {
	sb.Column(jc, fmt.Sprintf("%s.%s", tableAlias, jc.sqlColumn))
}

func (jc jsonColumn) NewRow() ScanDest {
	return &jsonFieldRow{
		field: jc.field,
	}
}

// jsonFieldRow is a jsonb SQL field mapped to a proto field.
type jsonFieldRow struct {
	field protoreflect.FieldDescriptor
	data  []byte
}

func (jc *jsonFieldRow) ScanTo() interface{} {
	return &jc.data
}

func (jc *jsonFieldRow) Unmarshal(resReflect protoreflect.Message) error {

	if jc.data == nil {
		return status.Error(codes.NotFound, "not found")
	}

	msg := resReflect
	if jc.field != nil {
		msg = resReflect.Mutable(jc.field).Message()
		resReflect.Set(jc.field, protoreflect.ValueOf(msg))
	}

	if err := protojson.Unmarshal(jc.data, msg.Interface()); err != nil {
		return err
	}

	return nil
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
