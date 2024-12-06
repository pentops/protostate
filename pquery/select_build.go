package pquery

import (
	"fmt"

	"github.com/lib/pq"
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
