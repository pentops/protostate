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
	"github.com/pentops/protostate/dbconvert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.daemonl.com/sqrlx"
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
	AuthJoin   *LeftJoin

	PrimaryKey func(REQ) (map[string]interface{}, error)

	StateResponseField protoreflect.Name

	Join *GetJoinSpec

	Method *MethodDescriptor[REQ, RES]
}

type GetJoinSpec struct {
	TableName     string
	DataColumn    string
	On            map[string]string // <TableName>.<key> = <parent>.<val>
	FieldInParent string
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
	primaryKey func(REQ) (map[string]interface{}, error)
	auth       AuthProvider
	authJoin   *LeftJoin

	validator *protovalidate.Validator

	join *getJoin
}

type getJoin struct {
	Table      string
	DataColunn string
	on         map[string]string // <TableName>.<key> = <parent>.<val>

	fieldInParent protoreflect.FieldDescriptor // wraps the ListFooEventResponse type
}

func NewGetter[
	REQ GetRequest,
	RES GetResponse,
](spec GetSpec[REQ, RES]) (*Getter[REQ, RES], error) {

	if spec.Method == nil {
		return nil, fmt.Errorf("missing Method")
	}

	resDesc := spec.Method.Response.ProtoReflect().Descriptor()

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
			Table:         spec.Join.TableName,
			DataColunn:    spec.Join.DataColumn,
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

func (gc *Getter[REQ, RES]) Get(ctx context.Context, db Transactor, reqMsg REQ, resMsg RES) error {

	as := newAliasSet()
	rootAlias := as.Next()

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
		authFilter, err := gc.auth.AuthFilter(ctx)
		if err != nil {
			return err
		}

		authAlias := rootAlias
		if gc.authJoin != nil {
			authAlias = as.Next()
			// LEFT JOIN
			//   <t> AS authAlias
			//   ON authAlias.<authJoin.foreignKeyColumn> = rootAlias.<authJoin.primaryKeyColumn>
			selectQuery = selectQuery.LeftJoin(fmt.Sprintf(
				"%s AS %s ON %s.%s = %s.%s",
				gc.authJoin.TableName,
				authAlias,
				authAlias,
				gc.authJoin.JoinKeyColumn,
				rootAlias,
				gc.authJoin.MainKeyColumn,
			))
		}

		for k, v := range authFilter {
			selectQuery = selectQuery.Where(sq.Eq{fmt.Sprintf("%s.%s", authAlias, k): v})
		}
	}

	if gc.join != nil {
		joinAlias := as.Next()

		joinConditions := make([]string, 0, len(primaryKeyFields))
		for joinColumn, rootColumn := range gc.join.on {
			joinConditions = append(joinConditions, fmt.Sprintf("%s.%s = %s.%s", joinAlias, joinColumn, rootAlias, rootColumn))
		}

		selectQuery.
			Column(fmt.Sprintf("ARRAY_AGG(%s.%s)", joinAlias, gc.join.DataColunn)).
			LeftJoin(fmt.Sprintf(
				"%s AS %s ON %s",
				gc.join.Table,
				joinAlias,
				strings.Join(joinConditions, " AND "),
			))
	}

	var foundJSON []byte
	var joinedJSON pq.ByteaArray

	query, args, err := selectQuery.ToSql()
	if err != nil {
		return err
	}
	fmt.Print(query+"\n", args)

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
		return err
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
			if eventBytes == nil {
				continue
			}

			rowMessage := elementList.NewElement().Message()
			if err := protojson.Unmarshal(eventBytes, rowMessage.Interface()); err != nil {
				return fmt.Errorf("joined unmarshal: %w", err)
			}
			elementList.Append(protoreflect.ValueOf(rowMessage))
		}

	}

	return nil

}