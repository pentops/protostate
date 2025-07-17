package pquery

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/elgris/sqrl/pg"
	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/j5/gen/j5/schema/v1/schema_j5pb"
	"github.com/pentops/j5/lib/id62"
	"github.com/pentops/j5/lib/j5schema"
	"github.com/pentops/protostate/internal/pgstore"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type filterSpec struct {
	*pgstore.NestedField
	filterVals []any
}

func filtersForField(field *j5schema.ObjectProperty) ([]any, error) {

	switch bigType := field.Schema.(type) {
	case *j5schema.EnumField:
		schema := bigType.Schema()

		if bigType.ListRules == nil || bigType.ListRules.Filtering == nil || !bigType.ListRules.Filtering.Filterable {
			return nil, nil
		}

		vals := []any{}
		for _, val := range bigType.ListRules.Filtering.DefaultFilters {
			option := schema.OptionByName(val)
			if option == nil {
				return nil, fmt.Errorf("default filter value '%s' not found in enum '%s'", val, field.JSONName)
			}

			vals = append(vals, option.Name())
		}

		return vals, nil

	case *j5schema.OneofField:
		if bigType.ListRules == nil || bigType.ListRules.Filtering == nil || !bigType.ListRules.Filtering.Filterable {
			return nil, nil
		}

		vals := []any{}
		for _, val := range bigType.ListRules.Filtering.DefaultFilters {
			// The value is the name of the set field
			vals = append(vals, val)
		}

		return vals, nil

	case *j5schema.ScalarSchema:
		j5Field := field.Schema.ToJ5Field()

		switch st := j5Field.Type.(type) {
		case *schema_j5pb.Field_Float:

			if st.Float.ListRules == nil || st.Float.ListRules.Filtering == nil || !st.Float.ListRules.Filtering.Filterable {
				return nil, nil
			}

			var size int
			switch st.Float.Format {
			case schema_j5pb.FloatField_FORMAT_FLOAT32:
				size = 32
			case schema_j5pb.FloatField_FORMAT_FLOAT64:
				size = 64
			default:
				return nil, fmt.Errorf("unknown float format for default filter (%s): %s", field.JSONName, st.Float.Format)
			}

			vals := []any{}
			for _, val := range st.Float.ListRules.Filtering.DefaultFilters {

				v, err := strconv.ParseFloat(val, size)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName, err)
				}

				vals = append(vals, v)
			}

			return vals, nil

		case *schema_j5pb.Field_Integer:
			if st.Integer.ListRules == nil || st.Integer.ListRules.Filtering == nil || !st.Integer.ListRules.Filtering.Filterable {
				return nil, nil
			}

			vals := []any{}
			for _, val := range st.Integer.ListRules.Filtering.DefaultFilters {
				var v any
				var err error

				switch st.Integer.Format {
				case schema_j5pb.IntegerField_FORMAT_INT32:
					v, err = strconv.ParseInt(val, 10, 32)
					if err != nil {
						return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName, err)
					}

				case schema_j5pb.IntegerField_FORMAT_INT64:
					v, err = strconv.ParseInt(val, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName, err)
					}

				case schema_j5pb.IntegerField_FORMAT_UINT32:
					v, err = strconv.ParseUint(val, 10, 32)
					if err != nil {
						return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName, err)
					}

				case schema_j5pb.IntegerField_FORMAT_UINT64:
					v, err = strconv.ParseUint(val, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName, err)
					}

				default:
					return nil, fmt.Errorf("unknown integer format for default filter (%s): %s", field.JSONName, st.Integer.Format)
				}

				vals = append(vals, v)

			}
			return vals, nil

		case *schema_j5pb.Field_Bool:
			if st.Bool.ListRules == nil || st.Bool.ListRules.Filtering == nil || !st.Bool.ListRules.Filtering.Filterable {
				return nil, nil
			}
			vals := []any{}
			for _, val := range st.Bool.ListRules.Filtering.DefaultFilters {
				v, err := strconv.ParseBool(val)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName, err)
				}

				vals = append(vals, v)
			}
			return vals, nil

		case *schema_j5pb.Field_Key:

			if st.Key.ListRules == nil || st.Key.ListRules.Filtering == nil || !st.Key.ListRules.Filtering.Filterable {
				return nil, nil
			}

			vals := []any{}
			for _, val := range st.Key.ListRules.Filtering.DefaultFilters {
				vals = append(vals, val)
			}
			return vals, nil

		case *schema_j5pb.Field_Timestamp:
			if st.Timestamp.ListRules == nil || st.Timestamp.ListRules.Filtering == nil || !st.Timestamp.ListRules.Filtering.Filterable {
				return nil, nil
			}

			vals := []any{}
			for _, val := range st.Timestamp.ListRules.Filtering.DefaultFilters {
				t, err := time.Parse(time.RFC3339, val)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName, err)
				}
				vals = append(vals, timestamppb.New(t))
			}
			return vals, nil

		case *schema_j5pb.Field_Date:
			if st.Date.ListRules == nil || st.Date.ListRules.Filtering == nil || !st.Date.ListRules.Filtering.Filterable {
				return nil, nil
			}
			vals := []any{}
			for _, val := range st.Date.ListRules.Filtering.DefaultFilters {
				t, err := time.Parse(time.DateOnly, val)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName, err)
				}
				vals = append(vals, t)
			}
			return vals, nil
		case *schema_j5pb.Field_String_:
			// no filters definable
			return nil, nil

		case *schema_j5pb.Field_Decimal:
			if st.Decimal.ListRules == nil || st.Decimal.ListRules.Filtering == nil || !st.Decimal.ListRules.Filtering.Filterable {
				return nil, nil
			}

			vals := []any{}
			for _, val := range st.Decimal.ListRules.Filtering.DefaultFilters {
				v, err := decimal.NewFromString(val)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName, err)
				}
				vals = append(vals, v)
			}
			return vals, nil

		default:
			return nil, fmt.Errorf("unknown scalar type for default filter (%s): %T", field.JSONName, st)
		}
	default:
		return nil, fmt.Errorf("unknown field type for filter rules %T", bigType)
	}
}

func buildDefaultFilters(columnName string, message *j5schema.ObjectSchema) ([]filterSpec, error) {
	var filters []filterSpec

	err := pgstore.WalkPathNodes(message, func(path pgstore.Path) error {
		field := path.LeafField()
		if field == nil {
			return nil
		}

		vals, err := filtersForField(field)
		if err != nil {
			return fmt.Errorf("filters for field: %w", err)
		}

		if len(vals) > 0 {
			filters = append(filters, filterSpec{
				NestedField: &pgstore.NestedField{
					Path:       path,
					RootColumn: columnName,
				},
				filterVals: vals,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk path nodes: %w", err)
	}

	return filters, nil
}

func (ll *Lister[REQ, RES]) buildDynamicFilter(tableAlias string, filters []*list_j5pb.Filter) ([]sq.Sqlizer, error) {
	out := []sq.Sqlizer{}

	for i := range filters {
		switch filters[i].GetType().(type) {
		case *list_j5pb.Filter_Field:
			pathSpec := pgstore.ParseJSONPathSpec(filters[i].GetField().GetName())
			spec, err := pgstore.NewJSONPath(ll.arrayObject, pathSpec)
			if err != nil {
				return nil, fmt.Errorf("dynamic filter: find field: %w", err)
			}

			biggerSpec := &pgstore.NestedField{
				Path:       *spec,
				RootColumn: ll.dataColumn,
			}

			o, err := ll.buildDynamicFilterField(tableAlias, biggerSpec, filters[i])
			if err != nil {
				return nil, fmt.Errorf("dynamic filter: build field: %w", err)
			}

			out = append(out, o)
		case *list_j5pb.Filter_And:
			f, err := ll.buildDynamicFilter(tableAlias, filters[i].GetAnd().GetFilters())
			if err != nil {
				return nil, fmt.Errorf("dynamic filter: and: %w", err)
			}
			and := sq.And{}
			and = append(and, f...)

			out = append(out, and)
		case *list_j5pb.Filter_Or:
			f, err := ll.buildDynamicFilter(tableAlias, filters[i].GetOr().GetFilters())
			if err != nil {
				return nil, fmt.Errorf("dynamic filter: or: %w", err)
			}
			or := sq.Or{}
			or = append(or, f...)

			out = append(out, or)
		}
	}

	return out, nil
}

func (ll *Lister[REQ, RES]) buildDynamicFilterField(tableAlias string, spec *pgstore.NestedField, filter *list_j5pb.Filter) (sq.Sqlizer, error) {
	var out sq.And

	if filter.GetField() == nil {
		return nil, fmt.Errorf("dynamic filter: field is nil")
	}

	switch ft := filter.GetField().GetType().Type.(type) {
	case *list_j5pb.FieldType_Value:
		val := ft.Value

		out = sq.And{sq.Expr(
			fmt.Sprintf("jsonb_path_query_array(%s.%s, '%s') @> ?",
				tableAlias,
				spec.RootColumn,
				spec.Path.JSONPathQuery(),
			), pg.JSONB(val))}

	case *list_j5pb.FieldType_Range:
		min := ft.Range.GetMin()
		max := ft.Range.GetMax()

		switch {
		case min != "" && max != "":
			exprStr := fmt.Sprintf("jsonb_path_query_array(%s.%s, '%s ?? (@ >= $min && @ <= $max)', jsonb_build_object('min', ?::text, 'max', ?::text)) <> '[]'::jsonb", tableAlias, spec.RootColumn, spec.Path.JSONPathQuery())
			out = sq.And{sq.Expr(exprStr, min, max)}
		case min != "":
			exprStr := fmt.Sprintf("jsonb_path_query_array(%s.%s, '%s ?? (@ >= $min)', jsonb_build_object('min', ?::text)) <> '[]'::jsonb", tableAlias, spec.RootColumn, spec.Path.JSONPathQuery())
			out = sq.And{sq.Expr(exprStr, min)}
		case max != "":
			exprStr := fmt.Sprintf("jsonb_path_query_array(%s.%s, '%s ?? (@ <= $max)', jsonb_build_object('max', ?::text)) <> '[]'::jsonb", tableAlias, spec.RootColumn, spec.Path.JSONPathQuery())
			out = sq.And{sq.Expr(exprStr, max)}
		}
	}

	return out, nil
}

func (ll *Lister[REQ, RES]) buildDynamicFilterOneof(tableAlias string, ospec *pgstore.NestedField, filter *list_j5pb.Filter) (sq.Sqlizer, error) {
	var out sq.And

	if filter.GetField() == nil {
		return nil, fmt.Errorf("dynamic filter: field is nil")
	}

	switch ft := filter.GetField().GetType().Type.(type) {
	case *list_j5pb.FieldType_Value:
		val := ft.Value

		// Val is used directly here instead of passed in as an expression
		// parameter. It has been sanitized by validation against the oneof
		// field names.
		exprStr := fmt.Sprintf("jsonb_array_length(jsonb_path_query_array(%s.%s, '%s ?? (exists(@.%s))')) > 0", tableAlias, ospec.RootColumn, ospec.Path.JSONPathQuery(), val)
		out = sq.And{sq.Expr(exprStr)}
	case *list_j5pb.FieldType_Range:
		return nil, fmt.Errorf("oneofs cannot be filtered by range")
	}

	return out, nil
}

func validateQueryRequestFilters(message *j5schema.ObjectSchema, filters []*list_j5pb.Filter) error {
	for i := range filters {
		switch filters[i].GetType().(type) {
		case *list_j5pb.Filter_Field:
			return validateQueryRequestFilterField(message, filters[i].GetField())
		case *list_j5pb.Filter_And:
			return validateQueryRequestFilters(message, filters[i].GetAnd().GetFilters())
		case *list_j5pb.Filter_Or:
			return validateQueryRequestFilters(message, filters[i].GetOr().GetFilters())
		}
	}

	return nil
}

func validateFiltersAnnotations(_ protoreflect.FieldDescriptors) error {

	// TODO: This existed pre refactor, check if it should do something.
	return nil
}

func validateQueryRequestFilterField(message *j5schema.ObjectSchema, filterField *list_j5pb.Field) error {

	jsonPath := pgstore.ParseJSONPathSpec(filterField.GetName())
	spec, err := pgstore.NewJSONPath(message, jsonPath)
	if err != nil {
		return fmt.Errorf("find field: %w", err)
	}

	fieldSpec := spec.LeafField()
	if !schemaIsFilterable(fieldSpec) {
		return fmt.Errorf("requested filter field '%s' is not filterable", filterField.Name)
	}

	switch filterField.Type.Type.(type) {
	case *list_j5pb.FieldType_Value:
		err := validateFilterFieldValue(fieldSpec, filterField.Type.GetValue())
		if err != nil {
			return fmt.Errorf("filter value: %w", err)
		}
	case *list_j5pb.FieldType_Range:
		err := validateFilterFieldValue(fieldSpec, filterField.Type.GetRange().GetMin())
		if err != nil {
			return fmt.Errorf("filter min value: %w", err)
		}

		err = validateFilterFieldValue(fieldSpec, filterField.Type.GetRange().GetMax())
		if err != nil {
			return fmt.Errorf("filter max value: %w", err)
		}
	}

	return nil
}

func schemaIsFilterable(prop *j5schema.ObjectProperty) bool {

	switch bigSchema := prop.Schema.(type) {
	case *j5schema.EnumField:
		return bigSchema.ListRules != nil && bigSchema.ListRules.Filtering != nil && bigSchema.ListRules.Filtering.Filterable
	case *j5schema.OneofField:
		return bigSchema.ListRules != nil && bigSchema.ListRules.Filtering != nil && bigSchema.ListRules.Filtering.Filterable
	case *j5schema.ScalarSchema:
		j5Field := bigSchema.ToJ5Field()
		switch st := j5Field.Type.(type) {
		case *schema_j5pb.Field_Float:
			return st.Float.ListRules != nil && st.Float.ListRules.Filtering != nil && st.Float.ListRules.Filtering.Filterable
		case *schema_j5pb.Field_Integer:
			return st.Integer.ListRules != nil && st.Integer.ListRules.Filtering != nil && st.Integer.ListRules.Filtering.Filterable
		case *schema_j5pb.Field_Bool:
			return st.Bool.ListRules != nil && st.Bool.ListRules.Filtering != nil && st.Bool.ListRules.Filtering.Filterable
		case *schema_j5pb.Field_Key:
			return st.Key.ListRules != nil && st.Key.ListRules.Filtering != nil && st.Key.ListRules.Filtering.Filterable
		case *schema_j5pb.Field_Timestamp:
			return st.Timestamp.ListRules != nil && st.Timestamp.ListRules.Filtering != nil && st.Timestamp.ListRules.Filtering.Filterable
		case *schema_j5pb.Field_Date:
			return st.Date.ListRules != nil && st.Date.ListRules.Filtering != nil && st.Date.ListRules.Filtering.Filterable
		case *schema_j5pb.Field_Decimal:
			return st.Decimal.ListRules != nil && st.Decimal.ListRules.Filtering != nil && st.Decimal.ListRules.Filtering.Filterable
		default:
			// If we don't know the type, we assume it's not filterable
			return false
		}
	default:
		// If we don't know the type, we assume it's not filterable
		return false
	}
}

func validateFilterFieldValue(prop *j5schema.ObjectProperty, value string) error {
	if value == "" {
		return nil
	}

	switch bigType := prop.Schema.(type) {
	case *j5schema.EnumField:
		schema := bigType.Schema()
		if bigType.ListRules == nil || bigType.ListRules.Filtering == nil || !bigType.ListRules.Filtering.Filterable {
			return fmt.Errorf("enum field '%s' is not filterable", prop.JSONName)
		}
		if schema.OptionByName(value) == nil {
			return fmt.Errorf("enum value '%s' is not a valid option for field '%s'", value, prop.JSONName)
		}
	case *j5schema.OneofField:
		if bigType.ListRules == nil || bigType.ListRules.Filtering == nil || !bigType.ListRules.Filtering.Filterable {
			return fmt.Errorf("oneof field '%s' is not filterable", prop.JSONName)
		}
		oneof := bigType.OneofSchema()
		if oneof.Properties.ByJSONName(value) == nil {
			return fmt.Errorf("oneof value '%s' is not a valid option for field '%s'", value, prop.JSONName)
		}

	case *j5schema.ScalarSchema:
		j5Field := bigType.ToJ5Field()
		switch st := j5Field.Type.(type) {
		case *schema_j5pb.Field_Float:
			if st.Float.ListRules == nil || st.Float.ListRules.Filtering == nil || !st.Float.ListRules.Filtering.Filterable {
				return fmt.Errorf("float field '%s' is not filterable", prop.JSONName)
			}
			if st.Float.Format == schema_j5pb.FloatField_FORMAT_FLOAT32 {
				_, err := strconv.ParseFloat(value, 32)
				if err != nil {
					return fmt.Errorf("parsing float32 value '%s' for field '%s': %w", value, prop.JSONName, err)
				}
			} else if st.Float.Format == schema_j5pb.FloatField_FORMAT_FLOAT64 {
				_, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return fmt.Errorf("parsing float64 value '%s' for field '%s': %w", value, prop.JSONName, err)
				}
			} else {
				return fmt.Errorf("unknown float format '%s' for field '%s'", st.Float.Format, prop.JSONName)
			}
		case *schema_j5pb.Field_Integer:
			if st.Integer.ListRules == nil || st.Integer.ListRules.Filtering == nil || !st.Integer.ListRules.Filtering.Filterable {
				return fmt.Errorf("integer field '%s' is not filterable", prop.JSONName)
			}
			switch st.Integer.Format {
			case schema_j5pb.IntegerField_FORMAT_INT32:
				_, err := strconv.ParseInt(value, 10, 32)
				if err != nil {
					return fmt.Errorf("parsing int32 value '%s' for field '%s': %w", value, prop.JSONName, err)
				}
			case schema_j5pb.IntegerField_FORMAT_INT64:
				_, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("parsing int64 value '%s' for field '%s': %w", value, prop.JSONName, err)
				}
			case schema_j5pb.IntegerField_FORMAT_UINT32:
				_, err := strconv.ParseUint(value, 10, 32)
				if err != nil {
					return fmt.Errorf("parsing uint32 value '%s' for field '%s': %w", value, prop.JSONName, err)
				}
			case schema_j5pb.IntegerField_FORMAT_UINT64:
				_, err := strconv.ParseUint(value, 10, 64)
				if err != nil {
					return fmt.Errorf("parsing uint64 value '%s' for field '%s': %w", value, prop.JSONName, err)
				}
			default:
				return fmt.Errorf("unknown integer format '%s' for field '%s'", st.Integer.Format, prop.JSONName)
			}
		case *schema_j5pb.Field_Bool:
			if st.Bool.ListRules == nil || st.Bool.ListRules.Filtering == nil || !st.Bool.ListRules.Filtering.Filterable {
				return fmt.Errorf("bool field '%s' is not filterable", prop.JSONName)
			}
			_, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("parsing bool value '%s' for field '%s': %w", value, prop.JSONName, err)
			}
		case *schema_j5pb.Field_Key:
			if st.Key.ListRules == nil || st.Key.ListRules.Filtering == nil || !st.Key.ListRules.Filtering.Filterable {
				return fmt.Errorf("key field '%s' is not filterable", prop.JSONName)
			}
			switch ft := st.Key.Format.Type.(type) {
			case *schema_j5pb.KeyFormat_Uuid:
				if _, err := uuid.Parse(value); err != nil {
					return fmt.Errorf("parsing uuid value '%s' for field '%s': %w", value, prop.JSONName, err)
				}

			case *schema_j5pb.KeyFormat_Informal_:
				// pass
			case *schema_j5pb.KeyFormat_Id62:
				if _, err := id62.Parse(value); err != nil {
					return fmt.Errorf("parsing id62 value '%s' for field '%s': %w", value, prop.JSONName, err)
				}

			case *schema_j5pb.KeyFormat_Custom_:
				re, err := regexp.Compile(ft.Custom.Pattern)
				if err != nil {
					return fmt.Errorf("parsing custom key regex '%s' for field '%s': %w", ft.Custom.Pattern, prop.JSONName, err)
				}
				if !re.MatchString(value) {
					return fmt.Errorf("value '%s' for field '%s' does not match custom key regex '%s'", value, prop.JSONName, ft.Custom.Pattern)
				}

			default:
				return fmt.Errorf("unknown key type for field '%s'", prop.JSONName)
			}
		case *schema_j5pb.Field_Timestamp:
			if st.Timestamp.ListRules == nil || st.Timestamp.ListRules.Filtering == nil || !st.Timestamp.ListRules.Filtering.Filterable {
				return fmt.Errorf("timestamp field '%s' is not filterable", prop.JSONName)
			}
			_, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return fmt.Errorf("parsing timestamp value '%s' for field '%s': %w", value, prop.JSONName, err)
			}
		case *schema_j5pb.Field_Date:
			if st.Date.ListRules == nil || st.Date.ListRules.Filtering == nil || !st.Date.ListRules.Filtering.Filterable {
				return fmt.Errorf("date field '%s' is not filterable", prop.JSONName)
			}
			_, err := time.Parse("2006-01-02", value)
			if err != nil {
				_, err = time.Parse("2006-01", value)
				if err != nil {
					_, err = time.Parse("2006", value)
					if err != nil {
						return fmt.Errorf("parsing date value '%s' for field '%s': %w", value, prop.JSONName, err)
					}
				}
			}
		case *schema_j5pb.Field_Decimal:
			if st.Decimal.ListRules == nil || st.Decimal.ListRules.Filtering == nil || !st.Decimal.ListRules.Filtering.Filterable {
				return fmt.Errorf("decimal field '%s' is not filterable", prop.JSONName)
			}
			_, err := decimal.NewFromString(value)
			if err != nil {
				return fmt.Errorf("parsing decimal value '%s' for field '%s': %w", value, prop.JSONName, err)
			}
		case *schema_j5pb.Field_String_:
			return fmt.Errorf("string field '%s' is not filterable", prop.JSONName)
		default:
			return fmt.Errorf("unknown scalar type for field '%s': %T", prop.JSONName, st)
		}
	default:
		// If we don't know the type, we assume it's not filterable
		return fmt.Errorf("unknown field type for filter validation: %T", bigType)
	}
	return nil
}
