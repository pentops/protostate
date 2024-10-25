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
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/internal/pgstore"
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

type ListRequest interface {
	proto.Message
}

type ListResponse interface {
	proto.Message
}

type TableSpec struct {
	TableName string

	Auth     AuthProvider
	AuthJoin []*LeftJoin

	DataColumn string // TODO: Replace with array Columns []Column

	// List of postgres column names to sort by if no other unique sort is found.
	FallbackSortColumns []pgstore.ProtoFieldSpec
}

type Column struct {
	Name string

	// The point within the root element which is stored in the column. An empty
	// path means this stores the root element,
	MountPoint *pgstore.Path
}

type ListSpec[REQ ListRequest, RES ListResponse] struct {
	TableSpec
	RequestFilter func(REQ) (map[string]interface{}, error)
}

type QueryLogger func(sqrlx.Sqlizer)

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

	// TODO: This should be an array/map of columns to data types, allowing
	// multiple JSONB values, as well as cached field values direcrly on the
	// table
	dataColumn string
}

func BuildListReflection(req protoreflect.MessageDescriptor, res protoreflect.MessageDescriptor, table TableSpec) (*ListReflectionSet, error) {
	return buildListReflection(req, res, table)
}

func buildListReflection(req protoreflect.MessageDescriptor, res protoreflect.MessageDescriptor, table TableSpec) (*ListReflectionSet, error) {
	var err error
	ll := ListReflectionSet{
		defaultPageSize: uint64(20),
		dataColumn:      table.DataColumn,
	}
	fields := res.Fields()

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		msg := field.Message()
		if msg == nil {
			return nil, fmt.Errorf("field %s is a '%s', but should be a message", field.Name(), field.Kind())
		}

		if msg.FullName() == "j5.list.v1.PageResponse" {
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
		return nil, fmt.Errorf("no page field in response, %s must have a j5.list.v1.PageResponse", res.FullName())
	}

	err = validateListAnnotations(ll.arrayField.Message().Fields())
	if err != nil {
		return nil, fmt.Errorf("validate list annotations on %s: %w", ll.arrayField.Message().FullName(), err)
	}

	ll.defaultSortFields, err = buildDefaultSorts(ll.dataColumn, ll.arrayField.Message())
	if err != nil {
		return nil, fmt.Errorf("default sorts: %w", err)
	}

	ll.tieBreakerFields, err = buildTieBreakerFields(ll.dataColumn, req, ll.arrayField.Message(), table.FallbackSortColumns)
	if err != nil {
		return nil, fmt.Errorf("tie breaker fields: %w", err)
	}

	if len(ll.defaultSortFields) == 0 && len(ll.tieBreakerFields) == 0 {
		return nil, fmt.Errorf("no default sort field found, %s must have at least one field annotated as default sort, or specify a tie breaker in %s", ll.arrayField.Message().FullName(), req.FullName())
	}

	f, err := buildDefaultFilters(ll.dataColumn, ll.arrayField.Message())
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
			case "j5.list.v1.PageRequest":
				ll.pageRequestField = field
				continue
			case "j5.list.v1.QueryRequest":
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
		return nil, fmt.Errorf("no page field in request, %s must have a j5.list.v1.PageRequest", req.FullName())
	}

	if ll.queryRequestField == nil {
		return nil, fmt.Errorf("no query field in request, %s must have a j5.list.v1.QueryRequest", req.FullName())
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

	tableName string

	queryLogger QueryLogger

	auth     AuthProvider
	authJoin []*LeftJoin

	requestFilter func(REQ) (map[string]interface{}, error)

	validator *protovalidate.Validator
}

func NewLister[
	REQ ListRequest,
	RES ListResponse,
](spec ListSpec[REQ, RES]) (*Lister[REQ, RES], error) {
	ll := &Lister[REQ, RES]{
		tableName: spec.TableName,
		auth:      spec.Auth,
		authJoin:  spec.AuthJoin,
	}

	descriptors := newMethodDescriptor[REQ, RES]()

	listFields, err := buildListReflection(descriptors.request, descriptors.response, spec.TableSpec)
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

func (ll *Lister[REQ, RES]) SetQueryLogger(logger QueryLogger) {
	ll.queryLogger = logger
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

	if ll.queryLogger != nil {
		ll.queryLogger(selectQuery)
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
		pageResponse := &list_j5pb.PageResponse{
			NextToken: &nextToken,
		}

		res.Set(ll.pageResponseField, protoreflect.ValueOf(pageResponse.ProtoReflect()))
	}

	return nil
}

func (ll *Lister[REQ, RES]) BuildQuery(ctx context.Context, req protoreflect.Message, res protoreflect.Message) (*sqrl.SelectBuilder, error) {
	as := newAliasSet()
	tableAlias := as.Next(ll.tableName)

	selectQuery := sq.Select(fmt.Sprintf("%s.%s", tableAlias, ll.dataColumn)).
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

	reqQuery, ok := req.Get(ll.queryRequestField).Message().Interface().(*list_j5pb.QueryRequest)
	if ok && reqQuery != nil {
		if err := ll.validateQueryRequest(reqQuery); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "query validation: %s", err)
		}

		querySorts := reqQuery.GetSorts()
		if len(querySorts) > 0 {
			dynSorts, err := ll.buildDynamicSortSpec(querySorts)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "build sorts: %s", err)
			}

			sortFields = dynSorts
		}

		queryFilters := reqQuery.GetFilters()
		if len(queryFilters) > 0 {
			dynFilters, err := ll.buildDynamicFilter(tableAlias, queryFilters)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "build filters: %s", err)
			}

			filterFields = append(filterFields, dynFilters...)
		}

		querySearches := reqQuery.GetSearches()
		if len(querySearches) > 0 {
			searchFilters, err := ll.buildDynamicSearches(tableAlias, querySearches)
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
				or = append(or, sq.Expr(fmt.Sprintf("jsonb_path_query_array(%s.%s, '%s') @> ?", tableAlias, ll.dataColumn, spec.Path.JSONPathQuery()), pg.JSONB(val)))
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
		selectQuery.OrderBy(fmt.Sprintf("%s %s", sortField.Selector(tableAlias), direction))
	}

	if ll.auth != nil {
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

		authFilter, err := ll.auth.AuthFilter(ctx)
		if err != nil {
			return nil, err
		}

		if len(authFilter) > 0 {
			claimFilter := map[string]interface{}{}
			for k, v := range authFilter {
				claimFilter[fmt.Sprintf("%s.%s", authAlias, k)] = v
			}
			selectQuery.Where(claimFilter)
		}
	}

	pageSize, err := ll.getPageSize(req)
	if err != nil {
		return nil, err
	}

	selectQuery.Limit(pageSize + 1)

	reqPage, ok := req.Get(ll.pageRequestField).Message().Interface().(*list_j5pb.PageRequest)
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
			rowSelecter := sortField.Selector(tableAlias)
			valuePlaceholder := "?"

			fieldVal, err := sortField.Path.GetValue(rowMessage)
			if err != nil {
				return nil, fmt.Errorf("sort field %s: %w", sortField.errorName(), err)
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
					return nil, fmt.Errorf("sort field %s is a message of type %s", sortField.errorName(), name)
				}

			case string:
				dbVal = subType
				if sortField.desc {
					// String fields aren't valid for sorting, they can only be used
					// for the tie-breaker so the order itself is not important, only
					// that it is consistent
					return nil, fmt.Errorf("sort field %s is a string, strings cannot be sorted DESC", sortField.errorName())
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
				return nil, fmt.Errorf("sort field %s is of type %T", sortField.errorName(), dbVal)
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

	pageReq, ok := req.Get(ll.pageRequestField).Message().Interface().(*list_j5pb.PageRequest)
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

func (ll *Lister[REQ, RES]) validateQueryRequest(query *list_j5pb.QueryRequest) error {
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
