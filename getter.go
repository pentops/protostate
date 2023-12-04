package protostate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	sq "github.com/elgris/sqrl"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.daemonl.com/sqrlx"
)

type GetSpec struct {
	TableName        string
	DataColumn       string
	PrimaryKeyColumn string
	Auth             AuthProvider
	AuthJoin         *LeftJoin

	PrimaryKeyRequestField protoreflect.Name
	StateResponseField     protoreflect.Name

	Join *GetJoinSpec

	Method *MethodDescriptor
}

type GetJoinSpec struct {
	TableName        string
	DataColumn       string
	ForeignKeyColumn string

	FieldInParent protoreflect.Name
}

func (gc GetJoinSpec) validate() error {
	if gc.TableName == "" {
		return fmt.Errorf("missing TableName")
	}
	if gc.DataColumn == "" {
		return fmt.Errorf("missing DataColumn")
	}
	if gc.ForeignKeyColumn == "" {
		return fmt.Errorf("missing ForeignKeyColumn")
	}
	if gc.FieldInParent == "" {
		return fmt.Errorf("missing FieldInParent")
	}

	return nil
}

type Getter struct {
	stateField     protoreflect.FieldDescriptor
	requestPKField protoreflect.FieldDescriptor

	dataColumn       string
	tableName        string
	primaryKeyColumn string
	auth             AuthProvider
	authJoin         *LeftJoin

	validator *protovalidate.Validator

	join *getJoin
}

type getJoin struct {
	Table            string
	DataColunn       string
	ForeignKeyColumn string

	fieldInParent protoreflect.FieldDescriptor // wraps the ListFooEventResponse type
}

func NewGetter(spec GetSpec) (*Getter, error) {

	if spec.Method == nil {
		return nil, fmt.Errorf("missing Method")
	}
	if spec.Method.Request == nil {
		return nil, fmt.Errorf("missing Method.Request")
	}
	if spec.Method.Response == nil {
		return nil, fmt.Errorf("missing Method.Response")
	}

	reqDesc := spec.Method.Request
	resDesc := spec.Method.Response

	sc := &Getter{
		dataColumn:       spec.DataColumn,
		tableName:        spec.TableName,
		primaryKeyColumn: spec.PrimaryKeyColumn,
		auth:             spec.Auth,
		authJoin:         spec.AuthJoin,
	}

	// TODO: Use an annotation not a passed in name
	sc.requestPKField = reqDesc.Fields().ByName(spec.PrimaryKeyRequestField)
	if sc.requestPKField == nil {
		return nil, fmt.Errorf("request message has no field %s: %s", spec.PrimaryKeyRequestField, reqDesc.FullName())
	}

	// TODO: Use an annotation not a passed in name
	if spec.StateResponseField == "" {
		spec.StateResponseField = protoreflect.Name("state")
	}
	sc.stateField = resDesc.Fields().ByName(spec.StateResponseField)
	if sc.stateField == nil {
		return nil, fmt.Errorf("no 'state' field in proto message")
	}

	if spec.Join != nil {
		joinField := resDesc.Fields().ByName(protoreflect.Name(spec.Join.FieldInParent))
		if joinField == nil {
			return nil, fmt.Errorf("field %s not found in response message", spec.Join.FieldInParent)
		}

		if !joinField.IsList() {
			return nil, fmt.Errorf("field %s, in join spec, is not a list", spec.Join.FieldInParent)
		}

		sc.join = &getJoin{
			Table:            spec.Join.TableName,
			DataColunn:       spec.Join.DataColumn,
			fieldInParent:    joinField,
			ForeignKeyColumn: spec.Join.ForeignKeyColumn,
		}
	}

	var err error
	sc.validator, err = protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize validator: %w", err)
	}

	return sc, nil
}

func (gc *Getter) Get(ctx context.Context, db Transactor, reqMsg proto.Message, resMsg proto.Message) error {

	as := newAliasSet()
	rootAlias := as.Next()

	resReflect := resMsg.ProtoReflect()
	reqReflect := reqMsg.ProtoReflect()

	if err := gc.validator.Validate(reqMsg); err != nil {
		return err
	}

	idVal := reqReflect.Get(gc.requestPKField).Interface()

	selectQuery := sq.
		Select().
		Column(fmt.Sprintf("%s.%s", rootAlias, gc.dataColumn)).
		From(fmt.Sprintf("%s AS %s", gc.tableName, rootAlias)).
		Where(sq.Eq{
			fmt.Sprintf("%s.%s", rootAlias, gc.primaryKeyColumn): idVal,
		}).GroupBy(fmt.Sprintf("%s.%s", rootAlias, gc.primaryKeyColumn))

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

		selectQuery.
			Column(fmt.Sprintf("ARRAY_AGG(%s.%s)", joinAlias, gc.join.DataColunn)).
			LeftJoin(fmt.Sprintf(
				"%s AS %s ON %s.%s = %s.%s",
				gc.join.Table,
				joinAlias,
				joinAlias,
				gc.join.ForeignKeyColumn,
				rootAlias,
				gc.primaryKeyColumn,
			))
	}

	var foundJSON []byte
	var joinedJSON pq.ByteaArray

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
				return status.Errorf(codes.NotFound, "entity %s not found", idVal)
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
