package pquery

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/elgris/sqrl/pg"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/j5/gen/j5/schema/v1/schema_j5pb"
	"github.com/pentops/j5/j5types/date_j5t"
	"github.com/pentops/j5/lib/j5codec"
	"github.com/pentops/j5/lib/j5reflect"
	"github.com/pentops/j5/lib/j5schema"
	"github.com/pentops/j5/lib/j5validate"
	"github.com/pentops/log.go/log"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ListRequest interface {
	j5reflect.Object
}

type ListResponse interface {
	j5reflect.Object
}

type TableSpec struct {
	TableName string

	Auth     AuthProvider
	AuthJoin []*LeftJoin

	DataColumn string // TODO: Replace with array Columns []Column

	// List of fields to sort by if no other unique sort is found.
	FallbackSortColumns []ProtoField
}

func (ts *TableSpec) Validate() error {
	if ts.TableName == "" {
		return fmt.Errorf("table name must be set")
	}
	if ts.DataColumn == "" {
		return fmt.Errorf("data column must be set")
	}
	return nil
}

// ProtoField represents a field within a the root data.
type ProtoField struct {
	// path from the root object to this field
	pathInRoot JSONPathSpec

	// optional column name when the field is also stored as a scalar directly
	valueColumn *string
}

func NewJSONField(protoPath string, columnName *string) ProtoField {
	pp := ParseJSONPathSpec(protoPath)
	return ProtoField{
		valueColumn: columnName,
		pathInRoot:  pp,
	}
}

type Column struct {
	Name string

	// The point within the root element which is stored in the column. An empty
	// path means this stores the root element,
	MountPoint *Path
}

type ListSpec struct {
	Method *j5schema.MethodSchema

	TableSpec
	RequestFilter func(j5reflect.Object) (map[string]any, error)
}

func (ls *ListSpec) Validate() error {
	if ls.Method == nil {
		return fmt.Errorf("list spec must have a method")
	}
	if err := ls.TableSpec.Validate(); err != nil {
		return fmt.Errorf("validate table spec: %w", err)
	}
	return nil
}

type QueryLogger func(sqrlx.Sqlizer)

type ListReflectionSet struct {
	method          *j5schema.MethodSchema
	defaultPageSize uint64

	arrayField  *j5schema.ObjectProperty
	arrayObject *j5schema.ObjectSchema

	pageResponseField *j5schema.ObjectProperty
	pageRequestField  *j5schema.ObjectProperty
	queryRequestField *j5schema.ObjectProperty

	defaultSortFields []sortSpec
	tieBreakerFields  []sortSpec

	defaultFilterFields []filterSpec
	RequestFilterFields []*j5schema.ObjectProperty

	tsvColumnMap map[string]string // map[JSON Path]PGColumName

	// TODO: This should be an array/map of columns to data types, allowing
	// multiple JSONB values, as well as cached field values direcrly on the
	// table
	dataColumn string
}

func BuildListReflection(method *j5schema.MethodSchema, table TableSpec) (*ListReflectionSet, error) {
	return buildListReflection(method, table)
}

func buildListReflection(method *j5schema.MethodSchema, table TableSpec) (*ListReflectionSet, error) {

	var err error
	ll := ListReflectionSet{
		method:          method,
		defaultPageSize: uint64(20),
		dataColumn:      table.DataColumn,
	}

	for _, field := range method.Response.Properties {
		switch ft := field.Schema.(type) {
		case *j5schema.ObjectField:

			if ft.ObjectSchema().FullName() == "j5.list.v1.PageResponse" {
				ll.pageResponseField = field
				continue
			}

		case *j5schema.ArrayField:

			objectField, ok := ft.ItemSchema.(*j5schema.ObjectField)
			if !ok {
				return nil, fmt.Errorf("array field %s must be an object field, got %s", field.FullName(), ft.ItemSchema.FullName())
			}

			if ll.arrayField != nil {
				return nil, fmt.Errorf("multiple repeated fields (%s and %s)", ll.arrayField.FullName(), field.FullName())
			}

			ll.arrayField = field
			ll.arrayObject = objectField.ObjectSchema()
			continue

		}

		return nil, fmt.Errorf("unknown field in list response: '%s' of type %s", field.FullName(), field.Schema.TypeName())
	}

	if ll.arrayField == nil {
		return nil, fmt.Errorf("no repeated field in response, %s must have a repeated message", method.Response.FullName())
	}

	if ll.pageResponseField == nil {
		return nil, fmt.Errorf("no page field in response, %s must have a j5.list.v1.PageResponse", method.Response.FullName())
	}

	err = validateListAnnotations(ll.arrayObject)
	if err != nil {
		return nil, fmt.Errorf("validate list annotations on %s: %w", ll.arrayField.FullName(), err)
	}

	ll.defaultSortFields, err = buildDefaultSorts(ll.dataColumn, ll.arrayObject)
	if err != nil {
		return nil, fmt.Errorf("default sorts: %w", err)
	}

	ll.tieBreakerFields, err = buildTieBreakerFields(ll.dataColumn, method.Request, ll.arrayObject, table.FallbackSortColumns)
	if err != nil {
		return nil, fmt.Errorf("tie breaker fields: %w", err)
	}

	if len(ll.defaultSortFields) == 0 && len(ll.tieBreakerFields) == 0 {
		return nil, fmt.Errorf("no default sort field found, %s must have at least one field annotated as default sort, or specify a tie breaker in %s", ll.arrayField.FullName(), method.Request.FullName())
	}

	f, err := buildDefaultFilters(ll.dataColumn, ll.arrayObject)
	if err != nil {
		return nil, fmt.Errorf("default filters: %w", err)
	}

	ll.defaultFilterFields = f

	ll.tsvColumnMap, err = buildTsvColumnMap(ll.arrayObject)
	if err != nil {
		return nil, fmt.Errorf("build tsv column map: %w", err)
	}

	for _, field := range method.Request.Properties {
		switch ft := field.Schema.(type) {
		case *j5schema.ObjectField:
			switch ft.ObjectSchema().FullName() {
			case "j5.list.v1.PageRequest":
				ll.pageRequestField = field
			case "j5.list.v1.QueryRequest":
				ll.queryRequestField = field
			default:
				return nil, fmt.Errorf("unknown field in request: '%s' of type %s", field.FullName(), field.Schema.TypeName())
			}

		case *j5schema.ScalarSchema:
			ll.RequestFilterFields = append(ll.RequestFilterFields, field)

		default:
			return nil, fmt.Errorf("unsupported filter field in request: '%s' of type %s", field.FullName(), field.Schema.TypeName())
		}
	}

	if ll.pageRequestField == nil {
		return nil, fmt.Errorf("no page field in request, %s must have a j5.list.v1.PageRequest", method.Request.FullName())
	}

	if ll.queryRequestField == nil {
		return nil, fmt.Errorf("no query field in request, %s must have a j5.list.v1.QueryRequest", method.Request.FullName())
	}

	arraySchema := ll.arrayField.Schema.(*j5schema.ArrayField)
	if arraySchema.Rules != nil {
		if arraySchema.Rules.MaxItems != nil {
			ll.defaultPageSize = *arraySchema.Rules.MaxItems
		}
	}

	return &ll, nil
}

type Lister struct {
	ListReflectionSet

	tableName string

	queryLogger QueryLogger

	auth     AuthProvider
	authJoin []*LeftJoin

	requestFilter func(j5reflect.Object) (map[string]any, error)

	validator *j5validate.Validator
}

func NewLister(spec ListSpec) (*Lister, error) {
	ll := &Lister{
		tableName: spec.TableName,
		auth:      spec.Auth,
		authJoin:  spec.AuthJoin,
	}

	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("validate list spec: %w", err)
	}

	listFields, err := buildListReflection(spec.Method, spec.TableSpec)
	if err != nil {
		return nil, err
	}
	ll.ListReflectionSet = *listFields

	ll.requestFilter = spec.RequestFilter

	ll.validator = j5validate.Global

	return ll, nil
}

func (ll *Lister) SetQueryLogger(logger QueryLogger) {
	ll.queryLogger = logger
}

func (ll *Lister) List(ctx context.Context, db Transactor, req, res j5reflect.Object) error {
	if err := ll.validator.Validate(req); err != nil {
		return fmt.Errorf("validating request %s: %w", req.SchemaName(), err)
	}

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

	listField, err := res.GetOrCreateValue(ll.arrayField.JSONName)
	if err != nil {
		return err
	}
	list, ok := listField.AsArrayOfObject()
	if !ok {
		return fmt.Errorf("field %s in response is not an array", ll.arrayField.FullName())
	}

	var nextToken string
	for idx, rowBytes := range jsonRows {
		rowMessage, _ := list.NewObjectElement()

		err := j5codec.Global.JSONToReflect(rowBytes, rowMessage)
		if err != nil {
			return fmt.Errorf("unmarshal into %s from %s: %w", rowMessage.SchemaName(), string(rowBytes), err)
		}

		if idx >= int(pageSize) {
			// TODO: This works but the token is huge.
			// The eventual solution will need to look at
			// the sorting and filtering of the query and either encode them
			// directly, or encode a subset of the message as required.

			nextToken, err = encodePageToken(rowMessage, selectQuery.sortFields)
			if err != nil {
				return fmt.Errorf("encode page token: %w", err)
			}
			break
		}
	}

	// TODO: This drops the pagination token from the list.
	// Consider adding support to J5Reflect to create the object element without
	// attaching it to the array
	if list.Length() > int(pageSize) {
		list.Truncate(int(pageSize))
	}

	if nextToken != "" {
		pageRes, err := res.GetOrCreateValue(ll.pageResponseField.JSONName)
		if err != nil {
			return err
		}

		cont, ok := pageRes.AsContainer()
		if !ok {
			return fmt.Errorf("field %s in response is not a container", ll.pageResponseField.FullName())
		}

		nextTokenVal, err := cont.GetOrCreateValue("nextToken")
		if err != nil {
			return fmt.Errorf("get nextToken field in response: %w", err)
		}
		nextTokenScalar, ok := nextTokenVal.AsScalar()
		if !ok {
			return fmt.Errorf("field %s in response is not a scalar", nextTokenVal.FullTypeName())
		}
		if err := nextTokenScalar.SetGoValue(nextToken); err != nil {
			return fmt.Errorf("set nextToken field in response: %w", err)
		}
	}

	return nil
}

func encodePageToken(rowMessage j5reflect.Object, sortFields []sortSpec) (string, error) {
	vals := map[string]any{}
	for _, sortField := range sortFields {
		fieldVal, _, err := sortField.Path.GetValue(rowMessage)
		if err != nil {
			return "", fmt.Errorf("sort field %s: %w", sortField.errorName(), err)
		}
		fmt.Printf("Set Sort Field %s to %v\n", sortField.errorName(), fieldVal)
		vals[sortField.Path.IDPath()] = fieldVal

	}
	valJSON, err := json.Marshal(vals)
	if err != nil {
		return "", fmt.Errorf("marshal page token values: %w", err)
	}
	return base64.StdEncoding.EncodeToString(valJSON), nil
}

func decodePageToken(token string, sortFields []sortSpec) (map[string]any, error) {
	vals := map[string]any{}
	jsonBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("decode page token: %w", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(jsonBytes)))
	dec.UseNumber()
	err = dec.Decode(&vals)
	if err != nil {
		return nil, fmt.Errorf("unmarshal page token: %w", err)
	}
	if len(vals) != len(sortFields) {
		return nil, fmt.Errorf("page token has %d values, expected %d", len(vals), len(sortFields))
	}

	return vals, nil
}

func fieldAs[T any](obj j5reflect.Object, path ...string) (val T, ok bool, err error) {

	endField, ok, err := obj.GetField(path...)
	if err != nil || !ok {
		return val, ok, err
	}

	objField, ok := endField.AsObject()
	if !ok {
		return val, false, fmt.Errorf("field %s in %s not an object", path, obj.SchemaName())
	}

	if !objField.IsSet() {
		return val, false, nil
	}

	val, ok = objField.ProtoReflect().Interface().(T)
	if !ok {
		return val, false, fmt.Errorf("field %s in %s not of type %T", path, obj.SchemaName(), (*T)(nil))
	}

	return val, true, nil

}

type Query struct {
	*sq.SelectBuilder
	sortFields []sortSpec
}

func (ll *Lister) BuildQuery(ctx context.Context, req j5reflect.Object, res j5reflect.Object) (*Query, error) {

	err := assertObjectsMatch(ll.method, req, res)
	if err != nil {
		return nil, err
	}

	as := newAliasSet()
	tableAlias := as.Next(ll.tableName)

	selectQuery := sq.Select(fmt.Sprintf("%s.%s", tableAlias, ll.dataColumn)).
		From(fmt.Sprintf("%s AS %s", ll.tableName, tableAlias))

	sortFields := ll.defaultSortFields
	sortFields = append(sortFields, ll.tieBreakerFields...)

	filterFields := []sq.Sqlizer{}
	if ll.requestFilter != nil {
		filter, err := ll.requestFilter(req)
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

	reqQuery, ok, err := fieldAs[*list_j5pb.QueryRequest](req, ll.queryRequestField.JSONName)
	if err != nil {
		return nil, err
	}
	if ok {
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
			claimFilter := map[string]any{}
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

	reqPage, ok, err := fieldAs[*list_j5pb.PageRequest](req, ll.pageRequestField.JSONName)
	if err != nil {
		return nil, fmt.Errorf("get page request field: %w", err)
	}
	if ok && reqPage.GetToken() != "" {

		filter, err := ll.addPageFilter(reqPage.GetToken(), sortFields, tableAlias)
		if err != nil {
			return nil, err
		}
		selectQuery.Where(filter)

	}

	return &Query{
		SelectBuilder: selectQuery,
		sortFields:    sortFields,
	}, nil
}

func (ll *Lister) addPageFilter(token string, sortFields []sortSpec, tableAlias string) (sq.Sqlizer, error) {

	lhsFields := make([]string, 0, len(sortFields))
	rhsValues := make([]any, 0, len(sortFields))
	rhsPlaceholders := make([]string, 0, len(sortFields))

	pageFields, err := decodePageToken(token, sortFields)
	if err != nil {
		return nil, fmt.Errorf("decode page token: %w", err)
	}

	for _, sortField := range sortFields {
		rowSelecter := sortField.Selector(tableAlias)
		valuePlaceholder := "?"

		dbVal, ok := pageFields[sortField.Path.IDPath()]
		if !ok {
			return nil, fmt.Errorf("page token missing value for sort field %s", sortField.errorName())
		}

		switch schema := sortField.Path.leafField.Schema.(type) {
		//case *j5schema.EnumField:

		case *j5schema.ScalarSchema:

			ft := schema.ToJ5Field()

			switch ft := ft.Type.(type) {
			case *schema_j5pb.Field_String_, *schema_j5pb.Field_Key:
				dbVal, ok = dbVal.(string)
				if !ok {
					return nil, fmt.Errorf("sort field %s is a string, but value is not a string", sortField.errorName())
				}
				if sortField.desc {
					// String fields aren't valid for sorting, they can only be used
					// for the tie-breaker so the order itself is not important, only
					// that it is consistent
					return nil, fmt.Errorf("sort field %s is a string, strings cannot be sorted DESC", sortField.errorName())
				}

			case *schema_j5pb.Field_Integer:
				number, ok := dbVal.(json.Number)
				if !ok {
					stringVal, ok := dbVal.(string)
					if !ok {
						return nil, fmt.Errorf("sort field %s is an integer, but value is %+v", sortField.errorName(), dbVal)
					}
					number = json.Number(stringVal)
				}
				int64val, err := number.Int64()
				if err != nil {
					return nil, fmt.Errorf("sort field %s is an integer, but value is not a valid integer: %w", sortField.errorName(), err)
				}

				switch ft.Integer.Format {
				case schema_j5pb.IntegerField_FORMAT_INT32, schema_j5pb.IntegerField_FORMAT_UINT32:

					if sortField.desc {
						int64val = int64val * -1
						rowSelecter = fmt.Sprintf("-1 * (%s)::integer", rowSelecter)
					}

				case schema_j5pb.IntegerField_FORMAT_INT64, schema_j5pb.IntegerField_FORMAT_UINT64:

					if sortField.desc {
						int64val = int64val * -1
						rowSelecter = fmt.Sprintf("-1 * (%s)::bigint", rowSelecter)
					}

				default:
					panic(fmt.Sprintf("unknown integer format %s for field %s", ft.Integer.Format, sortField.errorName()))
				}
				dbVal = int64val

			case *schema_j5pb.Field_Float:
				number, ok := dbVal.(json.Number)
				if !ok {
					stringVal, ok := dbVal.(string)
					if !ok {
						return nil, fmt.Errorf("sort field %s is a float, but value is not a number", sortField.errorName())
					}
					number = json.Number(stringVal)
				}
				float64val, err := number.Float64()
				if err != nil {
					return nil, fmt.Errorf("sort field %s is a float, but value is not a valid float: %w", sortField.errorName(), err)
				}
				switch ft.Float.Format {
				case schema_j5pb.FloatField_FORMAT_FLOAT32:
					if sortField.desc {
						float64val = float64val * -1
						rowSelecter = fmt.Sprintf("-1 * (%s)::real", rowSelecter)
					}

				case schema_j5pb.FloatField_FORMAT_FLOAT64:
					if sortField.desc {
						float64val = float64val * -1
						rowSelecter = fmt.Sprintf("-1 * (%s)::double precision", rowSelecter)
					}
				}
				dbVal = float64val

			case *schema_j5pb.Field_Bool:
				if sortField.desc {
					dbVal = !dbVal.(bool)
					rowSelecter = fmt.Sprintf("NOT (%s)::boolean", rowSelecter)
				}

			case *schema_j5pb.Field_Timestamp:

				stringVal, ok := dbVal.(string)
				if !ok {
					return nil, fmt.Errorf("sort field %s is a timestamp, but value is not a string", sortField.errorName())
				}
				ts, err := time.Parse(time.RFC3339, stringVal)
				if err != nil {
					return nil, fmt.Errorf("sort field %s is a timestamp, but value is not a valid timestamp: %w", sortField.errorName(), err)
				}

				intVal := ts.Round(time.Microsecond).UnixMicro()
				// Go rounds half-up.
				// Postgres is undocumented, but can only handle
				// microseconds.
				rowSelecter = fmt.Sprintf("(EXTRACT(epoch FROM (%s)::timestamp) * 1000000)::bigint", rowSelecter)
				if sortField.desc {
					intVal = intVal * -1
					rowSelecter = fmt.Sprintf("-1 * %s", rowSelecter)
				}
				dbVal = intVal

			case *schema_j5pb.Field_Date:
				stringVal, ok := dbVal.(string)
				if !ok {
					return nil, fmt.Errorf("sort field %s is a date, but value is not a string", sortField.errorName())
				}

				date, err := date_j5t.DateFromString(stringVal)
				if err != nil {
					return nil, fmt.Errorf("sort field %s is a date, but value is not a valid date: %w", sortField.errorName(), err)
				}
				intVal := date.AsTime(time.UTC).Round(time.Microsecond).Unix()

				rowSelecter = fmt.Sprintf("(EXTRACT(epoch FROM (%s)::date))::bigint", rowSelecter)
				if sortField.desc {
					intVal = intVal * -1
					rowSelecter = fmt.Sprintf("-1 * %s", rowSelecter)
				}
				dbVal = intVal

			}

		default:
			return nil, fmt.Errorf("sort field %s is not a scalar or enum", sortField.errorName())
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

	return sq.Expr(
		fmt.Sprintf("(%s) >= (%s)",
			strings.Join(lhsFields, ","),
			strings.Join(rhsPlaceholders, ","),
		), rhsValues...), nil

}
func (ll *Lister) getPageSize(req j5reflect.Object) (uint64, error) {
	pageField, ok, err := req.GetField(ll.pageRequestField.JSONName, "pageSize")
	if err != nil {
		return 0, fmt.Errorf("get page request field %s: %w", ll.pageRequestField.JSONName, err)
	}
	if !ok {
		return ll.defaultPageSize, nil
	}

	val, ok := pageField.AsScalar()
	if !ok {
		return 0, fmt.Errorf("field %s in request is not a scalar", ll.pageRequestField.JSONName)
	}
	intVal, err := val.ToGoValue()
	if err != nil {
		return 0, fmt.Errorf("convert page request field %s to Go value: %w", ll.pageRequestField.JSONName, err)
	}

	int64Val, ok := intVal.(int64)
	if !ok {
		return 0, fmt.Errorf("field %s in request is not an int64, got %T", ll.pageRequestField.JSONName, intVal)
	}

	returnVal := uint64(int64Val)
	if returnVal > ll.defaultPageSize {
		return 0, fmt.Errorf("page size %d exceeds maximum of %d", val, ll.defaultPageSize)
	}
	return returnVal, nil
}

func validateListAnnotations(object *j5schema.ObjectSchema) error {
	/*
		err := validateSortsAnnotations(object.Properties)
		if err != nil {
			return fmt.Errorf("sort: %w", err)
		}*/

	/*
		err = validateSearchesAnnotations(object)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}*/

	return nil
}

func (ll *Lister) validateQueryRequest(query *list_j5pb.QueryRequest) error {
	err := validateQueryRequestSorts(ll.arrayObject, query.GetSorts())
	if err != nil {
		return fmt.Errorf("sort validation: %w", err)
	}

	err = validateQueryRequestFilters(ll.arrayObject, query.GetFilters())
	if err != nil {
		return fmt.Errorf("filter validation: %w", err)
	}

	/*
		err = validateQueryRequestSearches(ll.arrayObject, query.GetSearches())
		if err != nil {
			return fmt.Errorf("search validation: %w", err)
		}*/

	return nil
}
