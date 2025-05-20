package pquery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/protovalidate-go"
	sq "github.com/elgris/sqrl"
	"github.com/lib/pq"
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
	AuthJoin   []*LeftJoin

	PrimaryKey func(REQ) (map[string]any, error)

	StateResponseField protoreflect.Name

	Join *GetJoinSpec
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

type GetJoinSpec struct {
	TableName     string
	DataColumn    string
	On            JoinFields
	FieldInParent protoreflect.Name
}

func (gc GetJoinSpec) validate() error {
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

type Getter[
	REQ GetRequest,
	RES proto.Message,
] struct {
	stateField protoreflect.FieldDescriptor

	dataColumn string
	tableName  string
	primaryKey func(REQ) (map[string]any, error)
	auth       AuthProvider
	authJoin   []*LeftJoin

	queryLogger QueryLogger

	validator protovalidate.Validator

	join *getJoin
}

type getJoin struct {
	dataColumn    string
	tableName     string
	fieldInParent protoreflect.FieldDescriptor // wraps the ListFooEventResponse type
	on            JoinFields
}

func NewGetter[
	REQ GetRequest,
	RES GetResponse,
](spec GetSpec[REQ, RES]) (*Getter[REQ, RES], error) {
	descriptors := newMethodDescriptor[REQ, RES]()
	resDesc := descriptors.response

	sc := &Getter[REQ, RES]{
		dataColumn: spec.DataColumn,
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
	sc.stateField = resDesc.Fields().ByName(spec.StateResponseField)
	if sc.stateField == nil {
		if defaultState {
			return nil, fmt.Errorf("no 'state' field in proto message - did you mean to override StateResponseField?")
		}
		return nil, fmt.Errorf("no '%s' field in proto message", spec.StateResponseField)
	}

	if spec.PrimaryKey == nil {
		return nil, fmt.Errorf("missing PrimaryKey func")
	}

	if spec.Join != nil {

		if err := spec.Join.validate(); err != nil {
			return nil, fmt.Errorf("invalid join spec: %w", err)
		}

		joinField := resDesc.Fields().ByName(protoreflect.Name(spec.Join.FieldInParent))
		if joinField == nil {
			return nil, fmt.Errorf("field %s not found in response message", spec.Join.FieldInParent)
		}

		if !joinField.IsList() {
			return nil, fmt.Errorf("field %s, in join spec, is not a list", spec.Join.FieldInParent)
		}

		sc.join = &getJoin{
			tableName:     spec.Join.TableName,
			dataColumn:    spec.Join.DataColumn,
			fieldInParent: joinField,
			on:            spec.Join.On,
		}
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

func (gc *Getter[REQ, RES]) Get(ctx context.Context, db Transactor, reqMsg REQ, resMsg RES) error {

	as := newAliasSet()
	rootAlias := as.Next(gc.tableName)

	resReflect := resMsg.ProtoReflect()

	if err := gc.validator.Validate(reqMsg); err != nil {
		return err
	}

	primaryKeyFields, err := gc.primaryKey(reqMsg)
	if err != nil {
		return err
	}

	rootFilter, err := dbconvert.FieldsToEqMap(rootAlias, primaryKeyFields)
	if err != nil {
		return err
	}

	selectQuery := sq.
		Select().
		Column(fmt.Sprintf("%s.%s", rootAlias, gc.dataColumn)).
		From(fmt.Sprintf("%s AS %s", gc.tableName, rootAlias)).
		Where(rootFilter)

	for pkField := range rootFilter {
		selectQuery.GroupBy(pkField)
	}

	if gc.auth != nil {
		authAlias := rootAlias
		for _, join := range gc.authJoin {
			priorAlias := authAlias
			authAlias = as.Next(join.TableName)
			selectQuery = selectQuery.LeftJoin(fmt.Sprintf(
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
			claimFilter := map[string]any{}
			for k, v := range authFilter {
				claimFilter[fmt.Sprintf("%s.%s", authAlias, k)] = v
			}
			selectQuery.Where(claimFilter)
		}
	}

	if gc.join != nil {
		joinAlias := as.Next(gc.join.tableName)

		selectQuery.
			Column(fmt.Sprintf("ARRAY_AGG(%s.%s)", joinAlias, gc.join.dataColumn)).
			LeftJoin(fmt.Sprintf(
				"%s AS %s ON %s",
				gc.join.tableName,
				joinAlias,
				gc.join.on.SQL(rootAlias, joinAlias),
			))
	}

	var foundJSON []byte
	var joinedJSON pq.StringArray

	if gc.queryLogger != nil {
		gc.queryLogger(selectQuery)
	}

	if err := db.Transact(ctx, &sqrlx.TxOptions{
		ReadOnly:  true,
		Retryable: true,
		Isolation: sql.LevelReadCommitted,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		row := tx.SelectRow(ctx, selectQuery)

		var err error
		if gc.join != nil {
			err = row.Scan(&foundJSON, &joinedJSON)
		} else {
			err = row.Scan(&foundJSON)
		}
		if err != nil {
			return err
		}

		return nil
	}); err != nil {

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
		query, _, _ := selectQuery.ToSql()

		return fmt.Errorf("%s: %w", query, err)
	}

	if foundJSON == nil {
		return status.Error(codes.NotFound, "not found")
	}

	stateMsg := resReflect.NewField(gc.stateField)
	if err := protojson.Unmarshal(foundJSON, stateMsg.Message().Interface()); err != nil {
		return err
	}
	resReflect.Set(gc.stateField, stateMsg)

	if gc.join != nil {
		elementList := resReflect.Mutable(gc.join.fieldInParent).List()
		for _, eventBytes := range joinedJSON {
			if eventBytes == "" {
				continue
			}

			rowMessage := elementList.NewElement().Message()
			if err := protojson.Unmarshal([]byte(eventBytes), rowMessage.Interface()); err != nil {
				return fmt.Errorf("joined unmarshal: %w", err)
			}
			elementList.Append(protoreflect.ValueOf(rowMessage))
		}

	}

	return nil

}
