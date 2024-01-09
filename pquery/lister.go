package pquery

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	sq "github.com/elgris/sqrl"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/dbconvert"
	"github.com/pentops/protostate/gen/list/v1/psml_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	Auth     AuthProvider
	AuthJoin []*LeftJoin
}

type sortSpec struct {
	field protoreflect.FieldDescriptor
	desc  bool
}

type Lister[
	REQ ListRequest,
	RES ListResponse,
] struct {
	pageSize uint64

	arrayField         protoreflect.FieldDescriptor
	pageResponseField  protoreflect.FieldDescriptor
	pageRequestField   protoreflect.FieldDescriptor
	queryReequestField protoreflect.FieldDescriptor

	defaultSortFields []sortSpec

	tableName  string
	dataColumn string
	auth       AuthProvider
	authJoin   []*LeftJoin
}

func NewLister[
	REQ ListRequest,
	RES ListResponse,
](spec ListSpec[REQ, RES]) (*Lister[REQ, RES], error) {

	ll := &Lister[REQ, RES]{
		pageSize:   uint64(20),
		tableName:  spec.TableName,
		dataColumn: spec.DataColumn,
		auth:       spec.Auth,
		authJoin:   spec.AuthJoin,
	}

	descriptors := newMethodDescriptor[REQ, RES]()
	fields := descriptors.response.Fields()

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		msg := field.Message()
		if msg == nil {
			return nil, fmt.Errorf("field %s is a '%s', but should be a message", field.Name(), field.Kind())
		}

		if msg.FullName() == "psm.list.v1.PageResponse" {
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
		return nil, fmt.Errorf("no page field in response, must have a psm.list.v1.PageResponse")
	}

	messageFields := ll.arrayField.Message().Fields()
	for i := 0; i < messageFields.Len(); i++ {
		field := messageFields.Get(i)
		if strings.HasPrefix(string(field.Name()), "created") {
			ll.defaultSortFields = []sortSpec{{
				field: field,
				desc:  true,
			}}
			break
		}

		if len(ll.defaultSortFields) == 0 && strings.HasSuffix(string(field.Name()), "_id") {
			ll.defaultSortFields = append(ll.defaultSortFields, sortSpec{
				field: field,
				desc:  false,
			})
		}
	}

	requestFields := descriptors.request.Fields()
	for i := 0; i < requestFields.Len(); i++ {
		field := requestFields.Get(i)
		msg := field.Message()
		if msg == nil {
			continue
		}

		switch msg.FullName() {
		case "psm.list.v1.PageRequest":
			ll.pageRequestField = field
			continue
		case "psm.list.v1.QueryRequest":
			ll.queryReequestField = field
			continue
		}
	}

	if ll.pageRequestField == nil {
		return nil, fmt.Errorf("no page field in request, must have a psm.list.v1.PageRequest")
	}

	if ll.queryReequestField == nil {
		return nil, fmt.Errorf("no query field in request, must have a psm.list.v1.QueryRequest")
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
	req := reqMsg.ProtoReflect()

	var jsonRows = make([][]byte, 0, ll.pageSize)

	as := newAliasSet()
	tableAlias := as.Next()

	selectQuery := sq.
		Select(fmt.Sprintf("%s.%s", tableAlias, ll.dataColumn)).
		From(fmt.Sprintf("%s AS %s", ll.tableName, tableAlias))

		// TODO: Dynamic Sorts
	sortFields := ll.defaultSortFields

	for _, sortField := range sortFields {
		direction := "ASC"
		if sortField.desc {
			direction = "DESC"
		}
		selectQuery.OrderBy(fmt.Sprintf("%s.%s->>'%s' %s", tableAlias, ll.dataColumn, sortField.field.JSONName(), direction))
	}

	if ll.auth != nil {
		authFilter, err := ll.auth.AuthFilter(ctx)
		if err != nil {
			return err
		}
		authAlias := tableAlias

		for _, join := range ll.authJoin {
			priorAlias := authAlias
			authAlias = as.Next()
			selectQuery = selectQuery.LeftJoin(fmt.Sprintf(
				"%s AS %s ON %s",
				join.TableName,
				authAlias,
				join.On.SQL(priorAlias, authAlias),
			))
		}

		authFilterMapped, err := dbconvert.FieldsToEqMap(authAlias, authFilter)
		if err != nil {
			return err
		}

		selectQuery = selectQuery.Where(authFilterMapped)
	}

	selectQuery.Limit(ll.pageSize + 1)

	// TODO: Request Filters req := reqMsg.ProtoReflect()

	reqPage, ok := req.Get(ll.pageRequestField).Message().Interface().(*psml_pb.PageRequest)
	if ok && reqPage != nil {
		if reqPage.GetToken() != "" {
			rowMessage := dynamicpb.NewMessage(ll.arrayField.Message())

			rowBytes, err := base64.StdEncoding.DecodeString(reqPage.GetToken())
			if err != nil {
				return fmt.Errorf("decode token: %w", err)
			}

			if err := proto.Unmarshal(rowBytes, rowMessage.Interface()); err != nil {
				return fmt.Errorf("unmarshal into %s from %s: %w", rowMessage.Descriptor().FullName(), string(rowBytes), err)
			}

			for _, sortField := range sortFields {
				equality := ">="
				if sortField.desc {
					equality = "<="
				}

				dbVal := rowMessage.Get(sortField.field).Interface()
				switch subType := dbVal.(type) {
				case *dynamicpb.Message:
					name := subType.Descriptor().FullName()
					msgBytes, err := proto.Marshal(subType)
					if err != nil {
						return fmt.Errorf("marshal %s: %w", name, err)
					}

					switch name {
					case "google.protobuf.Timestamp":
						ts := timestamppb.Timestamp{}
						if err := proto.Unmarshal(msgBytes, &ts); err != nil {
							return fmt.Errorf("unmarshal %s: %w", name, err)
						}
						dbVal = ts.AsTime().Format(time.RFC3339Nano) // JSON Encoding
					default:
						return fmt.Errorf("sort field %s is a message of type %s", sortField.field.Name(), name)
					}

				default:
					return fmt.Errorf("unknown sort field type %T", dbVal)
				}

				selectQuery = selectQuery.Where(
					fmt.Sprintf("%s.%s->>'%s' %s ?",
						tableAlias,
						ll.dataColumn,
						sortField.field.JSONName(),
						equality,
					), dbVal)
			}
		}
	}

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
			nextToken = base64.StdEncoding.EncodeToString(lastBytes)
			break
		}
		list.Append(protoreflect.ValueOf(rowMessage))
	}

	pageResponse := &psml_pb.PageResponse{}
	if nextToken != "" {
		pageResponse.NextToken = &nextToken
	}

	res.Set(ll.pageResponseField, protoreflect.ValueOf(pageResponse.ProtoReflect()))

	return nil

}
