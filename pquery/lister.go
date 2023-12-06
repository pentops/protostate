package pquery

import (
	"context"
	"database/sql"
	"fmt"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	sq "github.com/elgris/sqrl"
	query_pb "github.com/pentops/listify-go/query/v1"
	"github.com/pentops/log.go/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gopkg.daemonl.com/sqrlx"
)

type ListRequest interface {
	proto.Message
}

type ListResponse interface {
	proto.Message
}

type ListSpec[
	REQ ListRequest,
	RES ListResponse,
] struct {
	TableName  string
	DataColumn string

	Method *MethodDescriptor[REQ, RES]

	Auth     AuthProvider
	AuthJoin []*LeftJoin
}

type Lister[
	REQ ListRequest,
	RES ListResponse,
] struct {
	pageSize   uint64
	selecQuery func(ctx context.Context) (*sq.SelectBuilder, error)

	arrayField        protoreflect.FieldDescriptor
	pageResponseField protoreflect.FieldDescriptor
}

// LeftJoin is a specification for joining in the form
// <TableName> ON <TableName>.<JoinKeyColumn> = <Main>.<MainKeyColumn>
// Main is defined in the outer struct holding this LeftJoin
type LeftJoin struct {
	TableName     string
	JoinKeyColumn string
	MainKeyColumn string
}

func NewLister[
	REQ ListRequest,
	RES ListResponse,
](spec ListSpec[REQ, RES]) (*Lister[REQ, RES], error) {

	ll := &Lister[REQ, RES]{
		pageSize: uint64(20),
		selecQuery: func(ctx context.Context) (*sq.SelectBuilder, error) {
			as := newAliasSet()
			tableAlias := as.Next()

			selectQuery := sq.
				Select(fmt.Sprintf("%s.%s", tableAlias, spec.DataColumn)).
				From(fmt.Sprintf("%s AS %s", spec.TableName, tableAlias))

			if spec.Auth != nil {

				authFilter, err := spec.Auth.AuthFilter(ctx)
				if err != nil {
					return nil, err
				}
				authAlias := tableAlias

				for _, join := range spec.AuthJoin {
					priorAlias := authAlias
					authAlias = as.Next()
					// LEFT JOIN
					//   <t> AS authAlias
					//   ON authAlias.<authJoin.foreignKeyColumn> = tableAlias.<authJoin.primaryKeyColumn>
					selectQuery = selectQuery.LeftJoin(fmt.Sprintf(
						"%s AS %s ON %s.%s = %s.%s",
						join.TableName,
						authAlias,
						authAlias,
						join.JoinKeyColumn,
						priorAlias,
						join.MainKeyColumn,
					))
				}

				for k, v := range authFilter {
					selectQuery = selectQuery.Where(sq.Eq{fmt.Sprintf("%s.%s", authAlias, k): v})
				}
			}
			return selectQuery, nil
		},
	}

	fields := spec.Method.Response.ProtoReflect().Descriptor().Fields()

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		msg := field.Message()
		if msg == nil {
			return nil, fmt.Errorf("field %s is a '%s', but should be a listify.query.v1.PageResponse", field.Name(), field.Kind())
		}

		if msg.FullName() == "listify.query.v1.PageResponse" {
			ll.pageResponseField = field
			continue
		}

		if field.Cardinality() == protoreflect.Repeated {
			if ll.arrayField != nil {
				return nil, fmt.Errorf("multiple array fields")
			}

			ll.arrayField = field
			continue
		}
		return nil, fmt.Errorf("unknown field in response: '%s' of type %s", field.Name(), field.Kind())
	}

	if ll.arrayField == nil {
		return nil, fmt.Errorf("no array field")
	}

	if ll.pageResponseField == nil {
		return nil, fmt.Errorf("no page field in response, must have a listify.query.v1.PageResponse")
	}

	arrayFieldOpt := ll.arrayField.Options().(*descriptorpb.FieldOptions)
	validateOpt := proto.GetExtension(arrayFieldOpt, validate.E_Field).(*validate.FieldConstraints)
	if repeated := validateOpt.GetRepeated(); repeated != nil {
		if repeated.MaxItems != nil {
			ll.pageSize = *repeated.MaxItems
		}
	}

	return ll, nil
}

func (ll *Lister[REQ, RES]) List(ctx context.Context, db Transactor, reqMsg proto.Message, resMsg proto.Message) error {

	res := resMsg.ProtoReflect()

	var jsonRows = make([][]byte, 0, ll.pageSize)

	selectQuery, err := ll.selecQuery(ctx)
	if err != nil {
		return err
	}
	selectQuery.Limit(ll.pageSize + 1)

	// TODO: Request Filters req := reqMsg.ProtoReflect()
	// TODO: Request Sorts
	// TODO: Pagination in Request

	var nextToken string
	if err := db.Transact(ctx, &sqrlx.TxOptions{
		ReadOnly:  true,
		Retryable: true,
		Isolation: sql.LevelReadCommitted,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		rows, err := tx.Query(ctx, selectQuery)
		if err != nil {
			return fmt.Errorf("run select: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var json []byte
			if err := rows.Scan(&json); err != nil {
				return err
			}
			jsonRows = append(jsonRows, json)
		}
		return rows.Err()
	}); err != nil {
		stmt, _, _ := selectQuery.ToSql()
		log.WithField(ctx, "query", stmt).Error("list query")
		return fmt.Errorf("list query: %w", err)
	}

	list := res.Mutable(ll.arrayField).List()
	res.Set(ll.arrayField, protoreflect.ValueOf(list))

	for idx, rowBytes := range jsonRows {
		rowMessage := list.NewElement().Message()
		if err := protojson.Unmarshal(rowBytes, rowMessage.Interface()); err != nil {
			return fmt.Errorf("unmarshal into %s from %s: %w", rowMessage.Descriptor().FullName(), string(rowBytes), err)
		}
		if idx >= int(ll.pageSize) {
			// This is just pretend. The eventual solution will need to look at
			// the actual sorting and filtering of the query to determine the
			// next token.
			lastBytes, err := proto.Marshal(rowMessage.Interface())
			if err != nil {
				return fmt.Errorf("marshalling final row: %w", err)
			}
			nextToken = string(lastBytes)
			break
		}
		list.Append(protoreflect.ValueOf(rowMessage))
	}

	pageResponse := &query_pb.PageResponse{
		NextToken: nextToken,
		FinalPage: nextToken != "",
	}

	res.Set(ll.pageResponseField, protoreflect.ValueOf(pageResponse.ProtoReflect()))

	return nil

}
