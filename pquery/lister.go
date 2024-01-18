package pquery

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/elgris/sqrl"
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
	nestedField
	desc bool
}

type Lister[
	REQ ListRequest,
	RES ListResponse,
] struct {
	listReflection

	tableName  string
	dataColumn string
	auth       AuthProvider
	authJoin   []*LeftJoin
}

type ListerOption func(*listerOptions)

type listerOptions struct {
	tieBreakerFields []string
}

// TieBreakerFields is a list of fields to use as a tie breaker when the
// list request message does not specify these fields.
func WithTieBreakerFields(fields ...string) ListerOption {
	return func(lo *listerOptions) {
		lo.tieBreakerFields = fields
	}
}

type listReflection struct {
	pageSize uint64

	arrayField        protoreflect.FieldDescriptor
	pageResponseField protoreflect.FieldDescriptor
	pageRequestField  protoreflect.FieldDescriptor
	queryRequestField protoreflect.FieldDescriptor

	defaultSortFields []sortSpec
	tieBreakerFields  []sortSpec
}

func resolveListerOptions(options []ListerOption) listerOptions {
	optionsStruct := listerOptions{}
	for _, option := range options {
		option(&optionsStruct)
	}
	return optionsStruct
}

func ValidateListMethod(req protoreflect.MessageDescriptor, res protoreflect.MessageDescriptor, options ...ListerOption) error {
	_, err := buildListReflection(req, res, resolveListerOptions(options))
	return err
}

func buildListReflection(req protoreflect.MessageDescriptor, res protoreflect.MessageDescriptor, options listerOptions) (*listReflection, error) {
	var err error
	ll := listReflection{
		pageSize: uint64(20),
	}
	fields := res.Fields()

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
				return nil, fmt.Errorf("multiple repeated fields (%s and %s)", ll.arrayField.Name(), field.Name())
			}

			ll.arrayField = field
			continue
		}
		return nil, fmt.Errorf("unknown field in response: '%s' of type %s", field.Name(), field.Kind())
	}

	if ll.arrayField == nil {
		return nil, fmt.Errorf("no repeated field in response, %s must have a repeated message", res.FullName())
	}

	if ll.pageResponseField == nil {
		return nil, fmt.Errorf("no page field in response, %s must have a psm.list.v1.PageResponse", res.FullName())
	}

	ll.defaultSortFields = buildDefaultSorts(ll.arrayField.Message().Fields())

	ll.tieBreakerFields, err = buildTieBreakerFields(req, ll.arrayField.Message(), options.tieBreakerFields)
	if err != nil {
		return nil, err
	}

	if len(ll.defaultSortFields) == 0 && len(ll.tieBreakerFields) == 0 {
		return nil, fmt.Errorf("no default sort field found, %s must have at least one field annotated as default sort, or specify a tie breaker in %s", ll.arrayField.Message().FullName(), req.FullName())
	}

	requestFields := req.Fields()
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
			ll.queryRequestField = field
			continue
		}
	}

	if ll.pageRequestField == nil {
		return nil, fmt.Errorf("no page field in request, %s must have a psm.list.v1.PageRequest", req.FullName())
	}

	if ll.queryRequestField == nil {
		return nil, fmt.Errorf("no query field in request, %s must have a psm.list.v1.QueryRequest", req.FullName())
	}

	arrayFieldOpt := ll.arrayField.Options().(*descriptorpb.FieldOptions)
	validateOpt := proto.GetExtension(arrayFieldOpt, validate.E_Field).(*validate.FieldConstraints)
	if repeated := validateOpt.GetRepeated(); repeated != nil {
		if repeated.MaxItems != nil {
			ll.pageSize = *repeated.MaxItems
		}
	}

	return &ll, nil
}

func NewLister[
	REQ ListRequest,
	RES ListResponse,
](spec ListSpec[REQ, RES], options ...ListerOption) (*Lister[REQ, RES], error) {
	ll := &Lister[REQ, RES]{
		tableName:  spec.TableName,
		dataColumn: spec.DataColumn,
		auth:       spec.Auth,
		authJoin:   spec.AuthJoin,
	}

	descriptors := newMethodDescriptor[REQ, RES]()

	optionsStruct := resolveListerOptions(options)

	listFields, err := buildListReflection(descriptors.request, descriptors.response, optionsStruct)
	if err != nil {
		return nil, err
	}
	ll.listReflection = *listFields

	return ll, nil
}

func buildTieBreakerFields(req protoreflect.MessageDescriptor, arrayField protoreflect.MessageDescriptor, fallback []string) ([]sortSpec, error) {
	listRequestAnnotation, ok := proto.GetExtension(req.Options().(*descriptorpb.MessageOptions), psml_pb.E_ListRequest).(*psml_pb.ListRequestMessage)
	if ok && listRequestAnnotation != nil && len(listRequestAnnotation.SortTiebreaker) > 0 {
		tieBreakerFields := make([]sortSpec, 0, len(listRequestAnnotation.SortTiebreaker))
		for _, tieBreaker := range listRequestAnnotation.SortTiebreaker {
			field, err := findField(arrayField, tieBreaker)
			if err != nil {
				return nil, err
			}
			tieBreakerFields = append(tieBreakerFields, sortSpec{
				nestedField: *field,
				desc:        false,
			})
		}
		return tieBreakerFields, nil
	}

	if len(fallback) == 0 {
		return []sortSpec{}, nil
	}

	tieBreakerFields := make([]sortSpec, 0, len(fallback))
	for _, tieBreaker := range fallback {
		field, err := findField(arrayField, tieBreaker)
		if err != nil {
			return nil, err
		}
		tieBreakerFields = append(tieBreakerFields, sortSpec{
			nestedField: *field,
			desc:        false,
		})
	}
	return tieBreakerFields, nil

}

func buildDefaultSorts(messageFields protoreflect.FieldDescriptors) []sortSpec {
	var defaultSortFields []sortSpec

	for i := 0; i < messageFields.Len(); i++ {
		field := messageFields.Get(i)
		sortOpts := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)

		if sortOpts != nil {
			isDefaultSort := false

			switch field.Kind() {
			case protoreflect.DoubleKind:
				if sortOpts.GetDouble().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.Fixed32Kind:
				if sortOpts.GetFixed32().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.Fixed64Kind:
				if sortOpts.GetFixed64().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.FloatKind:
				if sortOpts.GetFloat().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.Int32Kind:
				if sortOpts.GetInt32().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.Int64Kind:
				if sortOpts.GetInt64().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.Sfixed32Kind:
				if sortOpts.GetSfixed32().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.Sfixed64Kind:
				if sortOpts.GetSfixed64().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.Sint32Kind:
				if sortOpts.GetSint32().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.Sint64Kind:
				if sortOpts.GetSint64().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.Uint32Kind:
				if sortOpts.GetUint32().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.Uint64Kind:
				if sortOpts.GetUint64().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			case protoreflect.MessageKind:
				if field.Message().FullName() == "google.protobuf.Timestamp" &&
					sortOpts.GetTimestamp().GetSorting().DefaultSort {
					isDefaultSort = true
				}
			}

			if isDefaultSort {
				defaultSortFields = append(defaultSortFields, sortSpec{
					nestedField: nestedField{
						field:    field,
						jsonPath: []string{field.JSONName()},
					},
					desc: true,
				})
			}
		} else if field.Kind() == protoreflect.MessageKind {
			subSort := buildDefaultSorts(field.Message().Fields())
			for _, subSortField := range subSort {
				subSortField.jsonPath = append([]string{field.JSONName()}, subSortField.jsonPath...)
			}
			defaultSortFields = append(defaultSortFields, subSort...)
		}
	}

	return defaultSortFields
}

func (ll *Lister[REQ, RES]) List(ctx context.Context, db Transactor, reqMsg proto.Message, resMsg proto.Message) error {

	res := resMsg.ProtoReflect()
	req := reqMsg.ProtoReflect()

	selectQuery, err := ll.BuildQuery(ctx, req, res)
	if err != nil {
		return err
	}

	var jsonRows = make([][]byte, 0, ll.pageSize)
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

	var nextToken string
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

func (ll *Lister[REQ, RES]) BuildQuery(ctx context.Context, req protoreflect.Message, res protoreflect.Message) (*sqrl.SelectBuilder, error) {

	as := newAliasSet()
	tableAlias := as.Next()

	selectQuery := sq.
		Select(fmt.Sprintf("%s.%s", tableAlias, ll.dataColumn)).
		From(fmt.Sprintf("%s AS %s", ll.tableName, tableAlias))

	sortFields := ll.defaultSortFields
	// TODO: Dynamic Sorts

	sortFields = append(sortFields, ll.tieBreakerFields...)

	for _, sortField := range sortFields {
		direction := "ASC"
		if sortField.desc {
			direction = "DESC"
		}
		selectQuery.OrderBy(fmt.Sprintf("%s.%s->>'%s' %s", tableAlias, ll.dataColumn, sortField.jsonbPath(), direction))
	}

	if ll.auth != nil {
		authFilter, err := ll.auth.AuthFilter(ctx)
		if err != nil {
			return nil, err
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
			return nil, err
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
				return nil, fmt.Errorf("decode token: %w", err)
			}

			if err := proto.Unmarshal(rowBytes, rowMessage.Interface()); err != nil {
				return nil, fmt.Errorf("unmarshal into %s from %s: %w", rowMessage.Descriptor().FullName(), string(rowBytes), err)
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
						return nil, fmt.Errorf("marshal %s: %w", name, err)
					}

					switch name {
					case "google.protobuf.Timestamp":
						ts := timestamppb.Timestamp{}
						if err := proto.Unmarshal(msgBytes, &ts); err != nil {
							return nil, fmt.Errorf("unmarshal %s: %w", name, err)
						}
						dbVal = ts.AsTime().Format(time.RFC3339Nano) // JSON Encoding
					default:
						return nil, fmt.Errorf("sort field %s is a message of type %s", sortField.field.Name(), name)
					}

				case string:
					dbVal = subType
				default:
					return nil, fmt.Errorf("unknown sort field type %T", dbVal)
				}

				selectQuery = selectQuery.Where(
					fmt.Sprintf("%s.%s->>'%s' %s ?",
						tableAlias,
						ll.dataColumn,
						sortField.jsonbPath(),
						equality,
					), dbVal)
			}
		}
	}

	return selectQuery, nil
}

type nestedField struct {
	jsonPath []string
	field    protoreflect.FieldDescriptor
}

func (nf nestedField) jsonbPath() string {
	return strings.Join(nf.jsonPath, "->")
}

func findField(message protoreflect.MessageDescriptor, path string) (*nestedField, error) {
	var fieldName protoreflect.Name
	var furtherPath string
	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 2 {
		fieldName = protoreflect.Name(parts[0])
		furtherPath = parts[1]
	} else {
		fieldName = protoreflect.Name(path)
	}

	field := message.Fields().ByName(fieldName)
	if field == nil {
		return nil, fmt.Errorf("no field named '%s' in %s", fieldName, message.FullName())
	}

	if furtherPath == "" {
		return &nestedField{
			field:    field,
			jsonPath: []string{field.JSONName()},
		}, nil
	}

	if field.Kind() != protoreflect.MessageKind {
		return nil, fmt.Errorf("field %s is not a message", fieldName)
	}
	spec, err := findField(field.Message(), furtherPath)
	if err != nil {
		return nil, fmt.Errorf("field %s: %w", parts[0], err)
	}
	return &nestedField{
		field:    spec.field,
		jsonPath: append([]string{field.JSONName()}, spec.jsonPath...),
	}, nil

}
