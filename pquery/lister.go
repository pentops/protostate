package pquery

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/protovalidate-go"
	"github.com/elgris/sqrl"
	sq "github.com/elgris/sqrl"
	"github.com/elgris/sqrl/pg"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/dbconvert"
	"github.com/pentops/protostate/gen/list/v1/psml_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var dateRegex = regexp.MustCompile(`^\d{4}(-\d{2}(-\d{2})?)?$`)

var ErrOneof = fmt.Errorf("field is a oneof")
var ErrField = fmt.Errorf("field is a field")

type ListRequest interface {
	proto.Message
}

type ListResponse interface {
	proto.Message
}

type ListSpec[REQ ListRequest, RES ListResponse] struct {
	TableName  string
	DataColumn string

	Auth     AuthProvider
	AuthJoin []*LeftJoin

	RequestFilter func(REQ) (map[string]interface{}, error)
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

type ListReflectionSet struct {
	defaultPageSize uint64

	arrayField        protoreflect.FieldDescriptor
	pageResponseField protoreflect.FieldDescriptor
	pageRequestField  protoreflect.FieldDescriptor
	queryRequestField protoreflect.FieldDescriptor

	defaultSortFields []sortSpec
	tieBreakerFields  []sortSpec

	defaultFilterFields []filterSpec
	RequestFilterFields []protoreflect.FieldDescriptor

	tsvColumnMap map[string]string
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
	return fmt.Errorf("validate list method: %w", err)
}

func BuildListReflection(req protoreflect.MessageDescriptor, res protoreflect.MessageDescriptor, options ...ListerOption) (*ListReflectionSet, error) {
	return buildListReflection(req, res, resolveListerOptions(options))
}

func buildListReflection(req protoreflect.MessageDescriptor, res protoreflect.MessageDescriptor, options listerOptions) (*ListReflectionSet, error) {
	var err error
	ll := ListReflectionSet{
		defaultPageSize: uint64(20),
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

	err = validateListAnnotations(ll.arrayField.Message().Fields())
	if err != nil {
		return nil, fmt.Errorf("validate list annotations on %s: %w", ll.arrayField.Message().FullName(), err)
	}

	ll.defaultSortFields = buildDefaultSorts(ll.arrayField.Message().Fields())

	ll.tieBreakerFields, err = buildTieBreakerFields(req, ll.arrayField.Message(), options.tieBreakerFields)
	if err != nil {
		return nil, fmt.Errorf("tie breaker fields: %w", err)
	}

	if len(ll.defaultSortFields) == 0 && len(ll.tieBreakerFields) == 0 {
		return nil, fmt.Errorf("no default sort field found, %s must have at least one field annotated as default sort, or specify a tie breaker in %s", ll.arrayField.Message().FullName(), req.FullName())
	}

	f, err := buildDefaultFilters(ll.arrayField.Message().Fields())
	if err != nil {
		return nil, fmt.Errorf("default filters: %w", err)
	}

	ll.defaultFilterFields = f

	ll.tsvColumnMap = buildTsvColumnMap(ll.arrayField.Message())

	requestFields := req.Fields()
	for i := 0; i < requestFields.Len(); i++ {
		field := requestFields.Get(i)
		msg := field.Message()
		if msg != nil {
			switch msg.FullName() {
			case "psm.list.v1.PageRequest":
				ll.pageRequestField = field
				continue
			case "psm.list.v1.QueryRequest":
				ll.queryRequestField = field
				continue
			default:
				return nil, fmt.Errorf("unknown field in request: '%s' of type %s", field.Name(), field.Kind())
			}
		}

		// Assume this is a filter field
		switch field.Kind() {
		case protoreflect.StringKind:
			ll.RequestFilterFields = append(ll.RequestFilterFields, field)
		case protoreflect.BoolKind:
			ll.RequestFilterFields = append(ll.RequestFilterFields, field)
		default:
			return nil, fmt.Errorf("unsupported filter field in request: '%s' of type %s", field.Name(), field.Kind())
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
			ll.defaultPageSize = *repeated.MaxItems
		}
	}

	return &ll, nil
}

type Lister[REQ ListRequest, RES ListResponse] struct {
	ListReflectionSet

	tableName  string
	dataColumn string
	auth       AuthProvider
	authJoin   []*LeftJoin

	requestFilter func(REQ) (map[string]interface{}, error)

	validator *protovalidate.Validator
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
	ll.ListReflectionSet = *listFields
	ll.requestFilter = spec.RequestFilter

	ll.validator, err = protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize validator: %w", err)
	}

	return ll, nil
}

func (ll *Lister[REQ, RES]) List(ctx context.Context, db Transactor, reqMsg proto.Message, resMsg proto.Message) error {
	if err := ll.validator.Validate(reqMsg); err != nil {
		return fmt.Errorf("validating request %s: %w", reqMsg.ProtoReflect().Descriptor().FullName(), err)
	}

	res := resMsg.ProtoReflect()
	req := reqMsg.ProtoReflect()

	pageSize, err := ll.getPageSize(req)
	if err != nil {
		return fmt.Errorf("get page size: %w", err)
	}

	selectQuery, err := ll.BuildQuery(ctx, req, res)
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	txOpts := &sqrlx.TxOptions{
		ReadOnly:  true,
		Retryable: true,
		Isolation: sql.LevelReadCommitted,
	}

	var jsonRows = make([][]byte, 0, pageSize)
	err = db.Transact(ctx, txOpts, func(ctx context.Context, tx sqrlx.Transaction) error {
		rows, err := tx.Query(ctx, selectQuery)
		if err != nil {
			return fmt.Errorf("run select: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var json []byte
			if err := rows.Scan(&json); err != nil {
				return fmt.Errorf("row scan: %w", err)
			}

			jsonRows = append(jsonRows, json)
		}

		return rows.Err()
	})
	if err != nil {
		stmt, _, _ := selectQuery.ToSql()
		log.WithField(ctx, "query", stmt).Error("list query")
		return fmt.Errorf("list query: %w", err)
	}

	list := res.Mutable(ll.arrayField).List()
	res.Set(ll.arrayField, protoreflect.ValueOf(list))

	var nextToken string
	for idx, rowBytes := range jsonRows {
		rowMessage := list.NewElement().Message()

		err := protojson.Unmarshal(rowBytes, rowMessage.Interface())
		if err != nil {
			return fmt.Errorf("unmarshal into %s from %s: %w", rowMessage.Descriptor().FullName(), string(rowBytes), err)
		}

		if idx >= int(pageSize) {
			// TODO: This works but the token is huge.
			// The eventual solution will need to look at
			// the sorting and filtering of the query and either encode them
			// directly, or encode a subset of the message as required.
			lastBytes, err := proto.Marshal(rowMessage.Interface())
			if err != nil {
				return fmt.Errorf("marshalling final row: %w", err)
			}

			nextToken = base64.StdEncoding.EncodeToString(lastBytes)
			break
		}

		list.Append(protoreflect.ValueOf(rowMessage))
	}

	if nextToken != "" {
		pageResponse := &psml_pb.PageResponse{
			NextToken: &nextToken,
		}

		res.Set(ll.pageResponseField, protoreflect.ValueOf(pageResponse.ProtoReflect()))
	}

	return nil
}

func (ll *Lister[REQ, RES]) BuildQuery(ctx context.Context, req protoreflect.Message, res protoreflect.Message) (*sqrl.SelectBuilder, error) {
	as := newAliasSet()
	tableAlias := as.Next(ll.tableName)

	selectQuery := sq.
		Select(fmt.Sprintf("%s.%s", tableAlias, ll.dataColumn)).
		From(fmt.Sprintf("%s AS %s", ll.tableName, tableAlias))

	sortFields := ll.defaultSortFields
	sortFields = append(sortFields, ll.tieBreakerFields...)

	filterFields := []sq.Sqlizer{}
	if ll.requestFilter != nil {
		filter, err := ll.requestFilter(req.Interface().(REQ))
		if err != nil {
			return nil, err
		}

		and := sq.And{}
		for k := range filter {
			and = append(and, sq.Expr(fmt.Sprintf("%s.%s = ?", tableAlias, k), filter[k]))
		}

		if len(and) > 0 {
			selectQuery.Where(and)
		}
	}

	reqQuery, ok := req.Get(ll.queryRequestField).Message().Interface().(*psml_pb.QueryRequest)
	if ok && reqQuery != nil {
		if err := ll.validateQueryRequest(reqQuery); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "query validation: %s", err)
		}

		if len(reqQuery.GetSorts()) > 0 {
			dynSorts, err := ll.buildDynamicSortSpec(reqQuery.GetSorts())
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "build sorts: %s", err)
			}

			sortFields = dynSorts
		}

		if len(reqQuery.GetFilters()) > 0 {
			dynFilters, err := ll.buildDynamicFilter(tableAlias, reqQuery.GetFilters())
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "build filters: %s", err)
			}

			filterFields = append(filterFields, dynFilters...)
		}

		if len(reqQuery.GetSearches()) > 0 {
			searchFilters, err := ll.buildDynamicSearches(tableAlias, reqQuery.GetSearches())
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "build searches: %s", err)
			}

			filterFields = append(filterFields, searchFilters...)
		}
	}

	for i := range filterFields {
		selectQuery.Where(filterFields[i])
	}

	// apply default filters if no filters have been requested
	if ll.defaultFilterFields != nil && len(filterFields) == 0 {
		and := sq.And{}
		for _, spec := range ll.defaultFilterFields {
			or := sq.Or{}
			for _, val := range spec.filterVals {
				and = append(and, sq.Expr(fmt.Sprintf("jsonb_path_query_array(%s.%s, '%s') @> ?", tableAlias, ll.dataColumn, spec.jsonbPath()), pg.JSONB(val)))
			}

			and = append(and, or)
		}

		if len(and) > 0 {
			selectQuery.Where(and)
		}
	}

	for _, sortField := range sortFields {
		direction := "ASC"
		if sortField.desc {
			direction = "DESC"
		}
		selectQuery.OrderBy(fmt.Sprintf("%s.%s%s %s", tableAlias, ll.dataColumn, sortField.jsonbPath(), direction))
	}

	if ll.auth != nil {
		authFilter, err := ll.auth.AuthFilter(ctx)
		if err != nil {
			return nil, err
		}
		authAlias := tableAlias

		for _, join := range ll.authJoin {
			priorAlias := authAlias
			authAlias = as.Next(join.TableName)
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

		if len(authFilterMapped) > 0 {
			selectQuery = selectQuery.Where(authFilterMapped)
		}
	}

	pageSize, err := ll.getPageSize(req)
	if err != nil {
		return nil, err
	}

	selectQuery.Limit(pageSize + 1)

	reqPage, ok := req.Get(ll.pageRequestField).Message().Interface().(*psml_pb.PageRequest)
	if ok && reqPage != nil && reqPage.GetToken() != "" {
		rowMessage := dynamicpb.NewMessage(ll.arrayField.Message())

		rowBytes, err := base64.StdEncoding.DecodeString(reqPage.GetToken())
		if err != nil {
			return nil, fmt.Errorf("decode token: %w", err)
		}

		if err := proto.Unmarshal(rowBytes, rowMessage.Interface()); err != nil {
			return nil, fmt.Errorf("unmarshal into %s from %s: %w", rowMessage.Descriptor().FullName(), string(rowBytes), err)
		}

		lhsFields := make([]string, 0, len(sortFields))
		rhsValues := make([]interface{}, 0, len(sortFields))
		rhsPlaceholders := make([]string, 0, len(sortFields))

		for _, sortField := range sortFields {
			rowSelecter := fmt.Sprintf("%s.%s%s",
				tableAlias,
				ll.dataColumn,
				sortField.jsonbPath(),
			)
			valuePlaceholder := "?"

			fieldVal, err := findFieldValue(rowMessage, sortField.fieldPath)
			if err != nil {
				return nil, fmt.Errorf("sort field %s: %w", sortField.fieldName(), err)
			}

			dbVal := fieldVal.Interface()
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
					intVal := ts.AsTime().Round(time.Microsecond).UnixMicro()
					// Go rounds half-up.
					// Postgres is undocumented, but can only handle
					// microseconds.
					rowSelecter = fmt.Sprintf("(EXTRACT(epoch FROM (%s)::timestamp) * 1000000)::bigint", rowSelecter)
					if sortField.desc {
						intVal = intVal * -1
						rowSelecter = fmt.Sprintf("-1 * %s", rowSelecter)
					}
					dbVal = intVal
				default:
					return nil, fmt.Errorf("sort field %s is a message of type %s", sortField.field.name, name)
				}

			case string:
				dbVal = subType
				if sortField.desc {
					// String fields aren't valid for sorting, they can only be used
					// for the tie-breaker so the order itself is not important, only
					// that it is consistent
					return nil, fmt.Errorf("sort field %s is a string, strings cannot be sorted DESC", sortField.field.name)
				}

			case int64:
				if sortField.desc {
					dbVal = dbVal.(int64) * -1
					rowSelecter = fmt.Sprintf("-1 * (%s)::bigint", rowSelecter)
				}
			case int32:
				if sortField.desc {
					dbVal = dbVal.(int32) * -1
					rowSelecter = fmt.Sprintf("-1 * (%s)::integer", rowSelecter)
				}
			case float32:
				if sortField.desc {
					dbVal = dbVal.(float32) * -1
					rowSelecter = fmt.Sprintf("-1 * (%s)::real", rowSelecter)
				}
			case float64:
				if sortField.desc {
					dbVal = dbVal.(float64) * -1
					rowSelecter = fmt.Sprintf("-1 * (%s)::double precision", rowSelecter)
				}
			case bool:
				if sortField.desc {
					dbVal = !dbVal.(bool)
					rowSelecter = fmt.Sprintf("NOT (%s)::boolean", rowSelecter)
				}

			// TODO: Reversals for the other types that are sortable

			default:
				return nil, fmt.Errorf("sort field %s is of type %T", sortField.field.name, dbVal)
			}

			lhsFields = append(lhsFields, rowSelecter)
			rhsValues = append(rhsValues, dbVal)
			rhsPlaceholders = append(rhsPlaceholders, valuePlaceholder)
		}

		// From https://www.postgresql.org/docs/current/functions-comparisons.html#ROW-WISE-COMPARISON
		// >> for the <, <=, > and >= cases, the row elements are compared left-to-right, stopping as soon
		// >> as an unequal or null pair of elements is found. If either of this pair of elements is null,
		// >> the result of the row comparison is unknown (null); otherwise comparison of this pair of elements
		// >> determines the result. For example, ROW(1,2,NULL) < ROW(1,3,0) yields true, not null, because the
		// >> third pair of elements are not considered.
		//
		// This means that we can use the row comparison with the same fields as
		// the sort fields to exclude the exact record we want, rather than the
		// filter being applied to all fields equally which takes out valid
		// records.
		// `(1, 30) >= (1, 20)` is true, so is `1 >= 1 AND 30 >= 20`
		// `(2, 10) >= (1, 20)` is also true, but `2 >= 1 AND 10 >= 20` is false
		// Since the tuple comparison starts from the left and stops at the first term.
		//
		// The downside is that we have to negate the values to sort in reverse
		// order, as we don't get an operator per term. This gets strange for
		// some data types and will create some crazy indexes.
		//
		// TODO: Optimize the cases when the order is ASC and therefore we don't
		// need to flip, but also the cases where we can just reverse the whole
		// comparison and reverse all flips to simplify, noting again that it
		// does not actually matter in which order the string field is sorted...
		// or don't because indexes.

		selectQuery = selectQuery.Where(
			fmt.Sprintf("(%s) >= (%s)",
				strings.Join(lhsFields, ","),
				strings.Join(rhsPlaceholders, ","),
			), rhsValues...)
	}

	return selectQuery, nil
}

func (ll *Lister[REQ, RES]) getPageSize(req protoreflect.Message) (uint64, error) {
	pageSize := ll.defaultPageSize

	pageReq, ok := req.Get(ll.pageRequestField).Message().Interface().(*psml_pb.PageRequest)
	if ok && pageReq != nil && pageReq.PageSize != nil {
		pageSize = uint64(*pageReq.PageSize)

		if pageSize > ll.defaultPageSize {
			return 0, fmt.Errorf("page size exceeds the maximum allowed size of %d", ll.defaultPageSize)
		}
	}

	return pageSize, nil
}

func camelToSnake(jsonName string) string {
	var out strings.Builder
	for i, r := range jsonName {
		if unicode.IsUpper(r) {
			if i > 0 {
				out.WriteRune('_')
			}
			out.WriteRune(unicode.ToLower(r))
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func validateListAnnotations(fields protoreflect.FieldDescriptors) error {
	err := validateSortsAnnotations(fields)
	if err != nil {
		return fmt.Errorf("sort: %w", err)
	}

	err = validateFiltersAnnotations(fields)
	if err != nil {
		return fmt.Errorf("filter: %w", err)
	}

	err = validateSearchesAnnotations(fields)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	return nil
}

func (ll *Lister[REQ, RES]) validateQueryRequest(query *psml_pb.QueryRequest) error {
	err := validateQueryRequestSorts(ll.arrayField.Message(), query.GetSorts())
	if err != nil {
		return fmt.Errorf("sort validation: %w", err)
	}

	err = validateQueryRequestFilters(ll.arrayField.Message(), query.GetFilters())
	if err != nil {
		return fmt.Errorf("filter validation: %w", err)
	}

	err = validateQueryRequestSearches(ll.arrayField.Message(), query.GetSearches())
	if err != nil {
		return fmt.Errorf("search validation: %w", err)
	}

	return nil
}

func validateFieldName(message protoreflect.MessageDescriptor, path string) error {
	var name protoreflect.Name
	var remainder string

	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 2 {
		name = protoreflect.Name(camelToSnake(parts[0]))
		remainder = parts[1]
	} else {
		name = protoreflect.Name(camelToSnake(path))
	}

	field := message.Fields().ByName(name)
	if field == nil {
		oneof := message.Oneofs().ByName(name)
		if oneof == nil {
			return fmt.Errorf("no field named '%s' in %s", name, message.FullName())
		}
	}

	if remainder != "" {
		if field.Kind() != protoreflect.MessageKind {
			return fmt.Errorf("field %s is not a message", name)
		}

		return validateFieldName(field.Message(), remainder)
	}

	return nil
}

type fieldSpec struct {
	field     pathNode
	fieldPath []pathNode
}

type pathNode struct {
	name  protoreflect.Name
	field protoreflect.FieldDescriptor
	oneof protoreflect.OneofDescriptor
}

func findFieldSpec(message protoreflect.MessageDescriptor, path string) (*fieldSpec, error) {
	var name protoreflect.Name
	var remainder string

	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 2 {
		name = protoreflect.Name(camelToSnake(parts[0]))
		remainder = parts[1]
	} else {
		name = protoreflect.Name(camelToSnake(path))
	}

	node := pathNode{name: name}
	field := message.Fields().ByName(name)
	if field != nil {
		node.field = field
	} else {
		oneof := message.Oneofs().ByName(name)
		if oneof != nil {
			node.oneof = oneof
		} else {
			return nil, fmt.Errorf("no field named '%s' in %s", name, message.FullName())
		}
	}

	node.field = field

	if remainder == "" {
		return &fieldSpec{
			field:     node,
			fieldPath: []pathNode{node},
		}, nil
	}

	if field.Kind() != protoreflect.MessageKind {
		return nil, fmt.Errorf("field %s is not a message", name)
	}

	spec, err := findFieldSpec(field.Message(), remainder)
	if err != nil {
		return nil, fmt.Errorf("field %s: %w", name, err)
	}

	return &fieldSpec{
		field:     spec.field,
		fieldPath: append([]pathNode{node}, spec.fieldPath...),
	}, nil
}

func findFieldValue(msg protoreflect.Message, path []pathNode) (protoreflect.Value, error) {
	if len(path) == 0 {
		return protoreflect.Value{}, fmt.Errorf("empty path")
	}
	var val protoreflect.Value
	node := path[0]

	if node.field == nil {
		if node.oneof == nil {
			return protoreflect.Value{}, fmt.Errorf("no field or oneof")
		}
		// oneof doesn't appear in the proto walk.
		return findFieldValue(msg, path[1:])
	}

	field := node.field

	// Has vs Get, Has returns false if the field is set to the default value for
	// scalar types. We still want the fields if they are set to the default value,
	// and can use validation of a field's existence before this point to ensure
	// that the field is available.
	val = msg.Get(field)
	if len(path) == 1 {
		return val, nil
	}
	if field.Kind() != protoreflect.MessageKind {
		return protoreflect.Value{}, fmt.Errorf("field %s is not a message", field.Name())
	}

	return findFieldValue(val.Message(), path[1:])
}
