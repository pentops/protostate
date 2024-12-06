package pquery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/protovalidate-go"
	sq "github.com/elgris/sqrl"
	"github.com/pentops/protostate/internal/dbconvert"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type GetRequest interface {
	proto.Message
}

type GetResponse interface {
	proto.Message
}

type GetSpec[
	REQ GetRequest,
	RES GetResponse,
] struct {
	TableName  string
	DataColumn string
	Auth       AuthProvider
	AuthJoin   []*KeyJoin

	PrimaryKey func(REQ) (map[string]interface{}, error)

	StateResponseField protoreflect.Name

	ArrayJoin *ArrayJoinSpec

	// ResponseDescriptor must describe the RES message, defaults to new(RES).ProtoReflect().Descriptor()
	ResponseDescriptor protoreflect.MessageDescriptor
}

// JoinConstraint defines a
// LEFT JOIN <JoinTable>
// ON <JoinTable>.<JoinColumn> = <RootTable>.<RootColumn>
type JoinField struct {
	JoinColumn string // The name of the column in the table being introduced
	RootColumn string // The name of the column in the root table
}

type JoinFields []JoinField

func (jc JoinFields) Reverse() JoinFields {
	out := make(JoinFields, 0, len(jc))
	for _, c := range jc {
		out = append(out, JoinField{
			JoinColumn: c.RootColumn,
			RootColumn: c.JoinColumn,
		})
	}
	return out
}

func (jc JoinFields) SQL(rootAlias string, joinAlias string) string {
	conditions := make([]string, 0, len(jc))
	for _, c := range jc {
		conditions = append(conditions, fmt.Sprintf("%s.%s = %s.%s",
			joinAlias,
			c.JoinColumn,
			rootAlias,
			c.RootColumn,
		))
	}
	return strings.Join(conditions, " AND ")
}

type ArrayJoinSpec struct {
	TableName     string
	DataColumn    string
	On            JoinFields
	FieldInParent protoreflect.Name
}

func (gc ArrayJoinSpec) validate() error {
	if gc.TableName == "" {
		return fmt.Errorf("missing TableName")
	}
	if gc.DataColumn == "" {
		return fmt.Errorf("missing DataColumn")
	}
	if gc.On == nil {
		return fmt.Errorf("missing On")
	}

	return nil
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

	stateMsg := resReflect.NewField(jc.field)
	if err := protojson.Unmarshal(jc.data, stateMsg.Message().Interface()); err != nil {
		return err
	}
	resReflect.Set(jc.field, stateMsg)
	return nil
}

type Getter[
	REQ GetRequest,
	RES proto.Message,
] struct {
	tableName  string
	primaryKey func(REQ) (map[string]interface{}, error)
	auth       AuthProvider
	authJoin   []*KeyJoin

	queryLogger QueryLogger

	validator *protovalidate.Validator

	columns []ColumnSpec
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

func NewGetter[
	REQ GetRequest,
	RES GetResponse,
](spec GetSpec[REQ, RES]) (*Getter[REQ, RES], error) {
	resDesc := spec.ResponseDescriptor
	if resDesc == nil {
		descriptors := newMethodDescriptor[REQ, RES]()
		resDesc = descriptors.response
	}

	sc := &Getter[REQ, RES]{
		tableName:  spec.TableName,
		primaryKey: spec.PrimaryKey,
		auth:       spec.Auth,
		authJoin:   spec.AuthJoin,
	}

	// TODO: Use an annotation not a passed in name
	defaultState := false
	if spec.StateResponseField == "" {
		defaultState = true
		spec.StateResponseField = protoreflect.Name("state")
	}

	stateField := resDesc.Fields().ByName(spec.StateResponseField)
	if stateField == nil {
		if defaultState {
			return nil, fmt.Errorf("no 'state' field in proto message, StateResponseField is left blank")
		}
		return nil, fmt.Errorf("no '%s' field in proto message", spec.StateResponseField)
	}
	sc.columns = append(sc.columns, newJsonColumn(spec.DataColumn, stateField))

	if spec.DataColumn == "" {
		return nil, fmt.Errorf("GetSpec missing DataColumn")
	}

	if spec.PrimaryKey == nil {
		return nil, fmt.Errorf("GetSpec missing PrimaryKey function")
	}

	if spec.TableName == "" {
		return nil, fmt.Errorf("GetSpec missing TableName")
	}

	if spec.ArrayJoin != nil {
		if err := spec.ArrayJoin.validate(); err != nil {
			return nil, fmt.Errorf("invalid join spec: %w", err)
		}

		joinField := resDesc.Fields().ByName(protoreflect.Name(spec.ArrayJoin.FieldInParent))
		if joinField == nil {
			return nil, fmt.Errorf("field %s not found in response message", spec.ArrayJoin.FieldInParent)
		}

		if !joinField.IsList() {
			return nil, fmt.Errorf("field %s, in join spec, is not a list", spec.ArrayJoin.FieldInParent)
		}

		sc.columns = append(sc.columns, &jsonArrayColumn{
			ArrayJoinSpec: *spec.ArrayJoin,
			fieldInParent: joinField,
		})
	}

	var err error
	sc.validator, err = protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize validator: %w", err)
	}

	return sc, nil
}

func (gc *Getter[REQ, RES]) SetQueryLogger(logger QueryLogger) {
	gc.queryLogger = logger
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

func (gc *Getter[REQ, RES]) Get(ctx context.Context, db Transactor, reqMsg REQ, resMsg RES) error {

	sb := newSelectBuilder(gc.tableName)

	resReflect := resMsg.ProtoReflect()

	if err := gc.validator.Validate(reqMsg); err != nil {
		return err
	}

	primaryKeyFields, err := gc.primaryKey(reqMsg)
	if err != nil {
		return err
	}

	if len(primaryKeyFields) == 0 {
		return fmt.Errorf("PrimaryKey() returned no fields")
	}

	rootFilter, err := dbconvert.FieldsToEqMap(sb.rootAlias, primaryKeyFields)
	if err != nil {
		return err
	}
	sb.Where(rootFilter)

	for pkField := range rootFilter {
		sb.GroupBy(pkField)
	}

	if gc.auth != nil {
		authAlias := sb.rootAlias
		for _, join := range gc.authJoin {
			priorAlias := authAlias
			authAlias = sb.TableAlias(join.TableName)
			sb.LeftJoin(fmt.Sprintf(
				"%s AS %s ON %s",
				join.TableName,
				authAlias,
				join.On.SQL(priorAlias, authAlias),
			))
		}

		authFilter, err := gc.auth.AuthFilter(ctx)
		if err != nil {
			return err
		}

		if len(authFilter) > 0 {
			claimFilter := map[string]interface{}{}
			for k, v := range authFilter {
				claimFilter[fmt.Sprintf("%s.%s", authAlias, k)] = v
			}
			sb.Where(claimFilter)
		}
	}

	for _, join := range gc.columns {
		join.ApplyQuery(sb.rootAlias, sb)
	}

	joins := make([]ScanDest, 0, len(gc.columns))
	rowCols := make([]interface{}, 0, len(sb.columns))
	for _, inQuery := range sb.columns {
		colRow := inQuery.NewRow()
		joins = append(joins, colRow)
		rowCols = append(rowCols, colRow.ScanTo())
	}

	if gc.queryLogger != nil {
		gc.queryLogger(sb.SelectBuilder)
	}

	if err := db.Transact(ctx, &sqrlx.TxOptions{
		ReadOnly:  true,
		Retryable: true,
		Isolation: sql.LevelReadCommitted,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		row := tx.SelectRow(ctx, sb.SelectBuilder)

		err := row.Scan(rowCols...)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				var pkDescription string
				if len(primaryKeyFields) == 1 {
					for _, v := range primaryKeyFields {
						pkDescription = fmt.Sprintf("%v", v)
					}
				} else {
					all := make([]string, 0, len(primaryKeyFields))
					for k, v := range primaryKeyFields {
						all = append(all, fmt.Sprintf("%s=%v", k, v))
					}
					pkDescription = strings.Join(all, ", ")
				}

				return status.Errorf(codes.NotFound, "entity %s not found", pkDescription)
			}
			return err
		}

		return nil
	}); err != nil {
		query, _, _ := sb.ToSql()
		return fmt.Errorf("%s: %w", query, err)
	}

	for _, join := range joins {
		if err := join.Unmarshal(resReflect); err != nil {
			return err
		}
	}

	return nil

}
