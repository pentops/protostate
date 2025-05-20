package pquery

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/elgris/sqrl/pg"
	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/protostate/internal/pgstore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type filterSpec struct {
	*pgstore.NestedField
	filterVals []any
}

func filtersForField(field protoreflect.FieldDescriptor) ([]any, error) {

	fieldOpts := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), list_j5pb.E_Field).(*list_j5pb.FieldConstraint)
	vals := []any{}

	if fieldOpts == nil || fieldOpts.Type == nil {
		return nil, nil
	}

	switch fieldOps := fieldOpts.Type.(type) {
	case *list_j5pb.FieldConstraint_Float:
		if fieldOps.Float.Filtering != nil && fieldOps.Float.Filtering.Filterable {
			for _, val := range fieldOps.Float.Filtering.DefaultFilters {
				v, err := strconv.ParseFloat(val, 32)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Double:
		if fieldOps.Double.Filtering != nil && fieldOps.Double.Filtering.Filterable {
			for _, val := range fieldOps.Double.Filtering.DefaultFilters {
				v, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Fixed32:
		if fieldOps.Fixed32.Filtering != nil && fieldOps.Fixed32.Filtering.Filterable {
			for _, val := range fieldOps.Fixed32.Filtering.DefaultFilters {
				v, err := strconv.ParseUint(val, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Fixed64:
		if fieldOps.Fixed64.Filtering != nil && fieldOps.Fixed64.Filtering.Filterable {
			for _, val := range fieldOps.Fixed64.Filtering.DefaultFilters {
				v, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Int32:
		if fieldOps.Int32.Filtering != nil && fieldOps.Int32.Filtering.Filterable {
			for _, val := range fieldOps.Int32.Filtering.DefaultFilters {
				v, err := strconv.ParseInt(val, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Int64:
		if fieldOps.Int64.Filtering != nil && fieldOps.Int64.Filtering.Filterable {
			for _, val := range fieldOps.Int64.Filtering.DefaultFilters {
				v, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Sfixed32:
		if fieldOps.Sfixed32.Filtering != nil && fieldOps.Sfixed32.Filtering.Filterable {
			for _, val := range fieldOps.Sfixed32.Filtering.DefaultFilters {
				v, err := strconv.ParseInt(val, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Sfixed64:
		if fieldOps.Sfixed64.Filtering != nil && fieldOps.Sfixed64.Filtering.Filterable {
			for _, val := range fieldOps.Sfixed64.Filtering.DefaultFilters {
				v, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Sint32:
		if fieldOps.Sint32.Filtering != nil && fieldOps.Sint32.Filtering.Filterable {
			for _, val := range fieldOps.Sint32.Filtering.DefaultFilters {
				v, err := strconv.ParseInt(val, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Sint64:
		if fieldOps.Sint64.Filtering != nil && fieldOps.Sint64.Filtering.Filterable {
			for _, val := range fieldOps.Sint64.Filtering.DefaultFilters {
				v, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Uint32:
		if fieldOps.Uint32.Filtering != nil && fieldOps.Uint32.Filtering.Filterable {
			for _, val := range fieldOps.Uint32.Filtering.DefaultFilters {
				v, err := strconv.ParseUint(val, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Uint64:
		if fieldOps.Uint64.Filtering != nil && fieldOps.Uint64.Filtering.Filterable {
			for _, val := range fieldOps.Uint64.Filtering.DefaultFilters {
				v, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_Bool:
		if fieldOps.Bool.Filtering != nil && fieldOps.Bool.Filtering.Filterable {
			for _, val := range fieldOps.Bool.Filtering.DefaultFilters {
				v, err := strconv.ParseBool(val)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, v)
			}
		}

	case *list_j5pb.FieldConstraint_String_:
		switch fieldOps := fieldOps.String_.WellKnown.(type) {
		case *list_j5pb.StringRules_Date:
			if fieldOps.Date.Filtering != nil && fieldOps.Date.Filtering.Filterable {
				for _, val := range fieldOps.Date.Filtering.DefaultFilters {
					if !dateRegex.MatchString(val) {
						return nil, fmt.Errorf("invalid date format for default filter (%s): %s", field.JSONName(), val)
					}

					// TODO: change to using ranges for date to handle whole
					// year, whole month, whole day
					vals = append(vals, val)
				}
			}

		case *list_j5pb.StringRules_ForeignKey:
			switch fieldOps := fieldOps.ForeignKey.Type.(type) {
			case *list_j5pb.ForeignKeyRules_UniqueString:
				if fieldOps.UniqueString.Filtering != nil && fieldOps.UniqueString.Filtering.Filterable {
					for _, val := range fieldOps.UniqueString.Filtering.DefaultFilters {
						vals = append(vals, val)
					}
				}

			case *list_j5pb.ForeignKeyRules_Id62:
				if fieldOps.Id62.Filtering != nil && fieldOps.Id62.Filtering.Filterable {
					for _, val := range fieldOps.Id62.Filtering.DefaultFilters {
						vals = append(vals, val)
					}
				}

			case *list_j5pb.ForeignKeyRules_Uuid:
				if fieldOps.Uuid.Filtering != nil && fieldOps.Uuid.Filtering.Filterable {
					for _, val := range fieldOps.Uuid.Filtering.DefaultFilters {
						_, err := uuid.Parse(val)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, val)
					}
				}

			}
		}

	case *list_j5pb.FieldConstraint_Enum:
		if fieldOps.Enum.Filtering != nil && fieldOps.Enum.Filtering.Filterable {
			for _, val := range fieldOps.Enum.Filtering.DefaultFilters {
				vals = append(vals, val)
			}
		}

	case *list_j5pb.FieldConstraint_Timestamp:
		if fieldOps.Timestamp.Filtering != nil && fieldOps.Timestamp.Filtering.Filterable {
			for _, val := range fieldOps.Timestamp.Filtering.DefaultFilters {
				t, err := time.Parse(time.RFC3339, val)
				if err != nil {
					return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
				}

				vals = append(vals, timestamppb.New(t))
			}
		}

	case *list_j5pb.FieldConstraint_Oneof:
		if fieldOps.Oneof.Filtering != nil && fieldOps.Oneof.Filtering.Filterable {
			for _, val := range fieldOps.Oneof.Filtering.DefaultFilters {
				vals = append(vals, val)
			}
		}

	default:
		return nil, fmt.Errorf("unknown field type for filter rules %T", fieldOps)
	}
	return vals, nil
}

func buildDefaultFilters(columnName string, message protoreflect.MessageDescriptor) ([]filterSpec, error) {
	var filters []filterSpec

	err := pgstore.WalkPathNodes(message, func(path pgstore.Path) error {
		field := path.LeafField()
		if field == nil {
			oneof := path.LeafOneofWrapper()
			if oneof == nil {
				return nil
			}
			field = oneof
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
			spec, err := pgstore.NewJSONPath(ll.arrayField.Message(), pathSpec)
			if err != nil {
				return nil, fmt.Errorf("dynamic filter: find field: %w", err)
			}

			biggerSpec := &pgstore.NestedField{
				Path:       *spec,
				RootColumn: ll.dataColumn,
			}

			var o sq.Sqlizer
			switch leaf := spec.Leaf().(type) {
			case protoreflect.OneofDescriptor:
				o, err = ll.buildDynamicFilterOneof(tableAlias, biggerSpec, filters[i])
				if err != nil {
					return nil, fmt.Errorf("dynamic filter: build oneof: %w", err)
				}
			case protoreflect.FieldDescriptor:
				o, err = ll.buildDynamicFilterField(tableAlias, biggerSpec, filters[i])
				if err != nil {
					return nil, fmt.Errorf("dynamic filter: build field: %w", err)
				}
			default:
				return nil, fmt.Errorf("unknown leaf type %T", leaf)
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

	leafField := spec.Path.LeafField()

	switch ft := filter.GetField().GetType().Type.(type) {
	case *list_j5pb.FieldType_Value:
		val := ft.Value
		if leafField.Kind() == protoreflect.EnumKind {
			name := strings.ToTitle(val)
			prefix := strings.TrimSuffix(string(leafField.Enum().Values().Get(0).Name()), "_UNSPECIFIED")

			if !strings.HasPrefix(val, prefix) {
				name = prefix + "_" + name
			}

			val = name
		}

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

func validateQueryRequestFilters(message protoreflect.MessageDescriptor, filters []*list_j5pb.Filter) error {
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

func validateQueryRequestFilterField(message protoreflect.MessageDescriptor, filterField *list_j5pb.Field) error {

	jsonPath := pgstore.ParseJSONPathSpec(filterField.GetName())
	spec, err := pgstore.NewJSONPath(message, jsonPath)
	if err != nil {
		return fmt.Errorf("find field: %w", err)
	}

	// validate the fields are annotated correctly for the request query
	// and the values are valid for the field

	switch leaf := spec.Leaf().(type) {
	case protoreflect.OneofDescriptor:
		filterable := false

		filterOpts := proto.GetExtension(leaf.Options().(*descriptorpb.OneofOptions), list_j5pb.E_Oneof).(*list_j5pb.OneofRules)

		if filterOpts == nil {

			fmt.Printf("No Filter Opts %s\n", filterField.GetName())
			wrapperField := spec.LeafOneofWrapper()
			if wrapperField != nil {
				fmt.Printf("WWWW %s\n", wrapperField.Name())
				fieldConstraint := proto.GetExtension(wrapperField.Options().(*descriptorpb.FieldOptions), list_j5pb.E_Field).(*list_j5pb.FieldConstraint)
				if fieldConstraint != nil {
					oneof := fieldConstraint.GetOneof()
					if oneof != nil && oneof.Filtering != nil {
						filterOpts = oneof
					}
				}
			}
		}

		if filterOpts != nil {
			filterable = filterOpts.GetFiltering().Filterable

			if filterable {
				switch filterField.Type.Type.(type) {
				case *list_j5pb.FieldType_Value:
					val := filterField.Type.GetValue()

					found := false
					for i := range leaf.Fields().Len() {
						f := leaf.Fields().Get(i)
						if strings.EqualFold(string(f.Name()), val) {
							found = true
							break
						}
					}

					if !found {
						return fmt.Errorf("filter value '%s' is not found in oneof '%s'", val, filterField.Name)
					}
				case *list_j5pb.FieldType_Range:
					return fmt.Errorf("oneofs cannot be filtered by range")
				}
			}
		}

		if !filterable {
			return fmt.Errorf("requested filter field '%s' is not filterable", filterField.Name)
		}

		return nil
	case protoreflect.FieldDescriptor:

		filterOpts, ok := proto.GetExtension(leaf.Options().(*descriptorpb.FieldOptions), list_j5pb.E_Field).(*list_j5pb.FieldConstraint)
		if !ok {
			return fmt.Errorf("requested filter field '%s' does not have any filterable constraints defined", filterField.Name)
		}

		filterable := false

		if filterOpts != nil {
			switch leaf.Kind() {
			case protoreflect.DoubleKind:
				if filterOpts.GetDouble().Filtering != nil {
					filterable = filterOpts.GetDouble().GetFiltering().Filterable
				}

			case protoreflect.Fixed32Kind:
				if filterOpts.GetFixed32().Filtering != nil {
					filterable = filterOpts.GetFixed32().GetFiltering().Filterable
				}

			case protoreflect.Fixed64Kind:
				if filterOpts.GetFixed64().Filtering != nil {
					filterable = filterOpts.GetFixed64().GetFiltering().Filterable
				}

			case protoreflect.FloatKind:
				if filterOpts.GetFloat().Filtering != nil {
					filterable = filterOpts.GetFloat().GetFiltering().Filterable
				}
			case protoreflect.Int32Kind:
				if filterOpts.GetInt32().Filtering != nil {
					filterable = filterOpts.GetInt32().GetFiltering().Filterable
				}

			case protoreflect.Int64Kind:
				if filterOpts.GetInt64().Filtering != nil {
					filterable = filterOpts.GetInt64().GetFiltering().Filterable
				}

			case protoreflect.Sfixed32Kind:
				if filterOpts.GetSfixed32().Filtering != nil {
					filterable = filterOpts.GetSfixed32().GetFiltering().Filterable
				}

			case protoreflect.Sfixed64Kind:
				if filterOpts.GetSfixed64().Filtering != nil {
					filterable = filterOpts.GetSfixed64().GetFiltering().Filterable
				}

			case protoreflect.Sint32Kind:
				if filterOpts.GetSint32().Filtering != nil {
					filterable = filterOpts.GetSint32().GetFiltering().Filterable
				}

			case protoreflect.Sint64Kind:
				if filterOpts.GetSint64().Filtering != nil {
					filterable = filterOpts.GetSint64().GetFiltering().Filterable
				}

			case protoreflect.Uint32Kind:
				if filterOpts.GetUint32().Filtering != nil {
					filterable = filterOpts.GetUint32().GetFiltering().Filterable
				}

			case protoreflect.Uint64Kind:
				if filterOpts.GetUint64().Filtering != nil {
					filterable = filterOpts.GetUint64().GetFiltering().Filterable
				}

			case protoreflect.BoolKind:
				if filterOpts.GetBool().Filtering != nil {
					filterable = filterOpts.GetBool().GetFiltering().Filterable
				}

			case protoreflect.EnumKind:
				if filterOpts.GetEnum().Filtering != nil {
					filterable = filterOpts.GetEnum().GetFiltering().Filterable
				}

			case protoreflect.StringKind:
				switch filterOpts.GetString_().WellKnown.(type) {
				case *list_j5pb.StringRules_Date:
					if filterOpts.GetString_().GetDate().Filtering != nil {
						filterable = filterOpts.GetString_().GetDate().Filtering.Filterable
					}

				case *list_j5pb.StringRules_ForeignKey:
					switch filterOpts.GetString_().GetForeignKey().GetType().(type) {
					case *list_j5pb.ForeignKeyRules_UniqueString:
						if filterOpts.GetString_().GetForeignKey().GetUniqueString().Filtering != nil {
							filterable = filterOpts.GetString_().GetForeignKey().GetUniqueString().Filtering.Filterable
						}

					case *list_j5pb.ForeignKeyRules_Id62:
						if filterOpts.GetString_().GetForeignKey().GetId62().Filtering != nil {
							filterable = filterOpts.GetString_().GetForeignKey().GetId62().Filtering.Filterable
						}

					case *list_j5pb.ForeignKeyRules_Uuid:
						if filterOpts.GetString_().GetForeignKey().GetUuid().Filtering != nil {
							filterable = filterOpts.GetString_().GetForeignKey().GetUuid().Filtering.Filterable
						}

					}
				}
			case protoreflect.MessageKind:
				if leaf.Message().FullName() == "google.protobuf.Timestamp" && filterOpts.GetTimestamp().Filtering != nil {
					filterable = filterOpts.GetTimestamp().GetFiltering().Filterable
				}
			}

			if filterable {
				switch filterField.Type.Type.(type) {
				case *list_j5pb.FieldType_Value:
					err := validateFilterFieldValue(filterOpts, leaf, filterField.Type.GetValue())
					if err != nil {
						return fmt.Errorf("filter value: %w", err)
					}
				case *list_j5pb.FieldType_Range:
					err := validateFilterFieldValue(filterOpts, leaf, filterField.Type.GetRange().GetMin())
					if err != nil {
						return fmt.Errorf("filter min value: %w", err)
					}

					err = validateFilterFieldValue(filterOpts, leaf, filterField.Type.GetRange().GetMax())
					if err != nil {
						return fmt.Errorf("filter max value: %w", err)
					}
				}
			}
		}

		if !filterable {
			return fmt.Errorf("requested filter field '%s' is not filterable", filterField.Name)
		}

		return nil
	default:
		return fmt.Errorf("unknown leaf type %v", leaf)
	}
}

func validateFilterFieldValue(filterOpts *list_j5pb.FieldConstraint, field protoreflect.FieldDescriptor, value string) error {
	if value == "" {
		return nil
	}

	switch field.Kind() {
	case protoreflect.DoubleKind:
		if filterOpts.GetDouble().GetFiltering().Filterable {
			_, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("parsing double: %w", err)
			}
		}
	case protoreflect.Fixed32Kind:
		if filterOpts.GetFixed32().GetFiltering().Filterable {
			_, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return fmt.Errorf("parsing fixed32: %w", err)
			}
		}
	case protoreflect.Fixed64Kind:
		if filterOpts.GetFixed64().GetFiltering().Filterable {
			_, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return fmt.Errorf("parsing fixed64: %w", err)
			}
		}
	case protoreflect.FloatKind:
		if filterOpts.GetFloat().GetFiltering().Filterable {
			_, err := strconv.ParseFloat(value, 32)
			if err != nil {
				return fmt.Errorf("parsing float: %w", err)
			}
		}
	case protoreflect.Int32Kind:
		if filterOpts.GetInt32().GetFiltering().Filterable {
			_, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return fmt.Errorf("parsing int32: %w", err)
			}
		}
	case protoreflect.Int64Kind:
		if filterOpts.GetInt64().GetFiltering().Filterable {
			_, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("parsing int64: %w", err)
			}
		}
	case protoreflect.Sfixed32Kind:
		if filterOpts.GetSfixed32().GetFiltering().Filterable {
			_, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return fmt.Errorf("parsing sfixed32: %w", err)
			}
		}
	case protoreflect.Sfixed64Kind:
		if filterOpts.GetSfixed64().GetFiltering().Filterable {
			_, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("parsing sfixed64: %w", err)
			}
		}
	case protoreflect.Sint32Kind:
		if filterOpts.GetSint32().GetFiltering().Filterable {
			_, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return fmt.Errorf("parsing sint32: %w", err)
			}
		}
	case protoreflect.Sint64Kind:
		if filterOpts.GetSint64().GetFiltering().Filterable {
			_, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("parsing sint64: %w", err)
			}
		}
	case protoreflect.Uint32Kind:
		if filterOpts.GetUint32().GetFiltering().Filterable {
			_, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return fmt.Errorf("parsing uint32: %w", err)
			}
		}
	case protoreflect.Uint64Kind:
		if filterOpts.GetUint64().GetFiltering().Filterable {
			_, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return fmt.Errorf("parsing uint64: %w", err)
			}
		}
	case protoreflect.BoolKind:
		if filterOpts.GetBool().GetFiltering().Filterable {
			_, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("parsing bool: %w", err)
			}
		}
	case protoreflect.EnumKind:
		if filterOpts.GetEnum().GetFiltering().Filterable {
			name := strings.ToTitle(value)
			prefix := strings.TrimSuffix(string(field.Enum().Values().Get(0).Name()), "_UNSPECIFIED")

			if !strings.HasPrefix(value, prefix) {
				name = prefix + "_" + name
			}
			eval := field.Enum().Values().ByName(protoreflect.Name(name))

			if eval == nil {
				return fmt.Errorf("enum value %s is not a valid enum value for field", value)
			}
		}
	case protoreflect.StringKind:
		switch filterOpts.GetString_().WellKnown.(type) {
		case *list_j5pb.StringRules_Date:
			if filterOpts.GetString_().GetDate().Filtering.Filterable {
				_, err := time.Parse("2006-01-02", value)
				if err != nil {
					_, err = time.Parse("2006-01", value)
					if err != nil {
						_, err = time.Parse("2006", value)
						if err != nil {
							return fmt.Errorf("parsing date: %w", err)
						}
					}
				}
			}
		case *list_j5pb.StringRules_ForeignKey:
			switch filterOpts.GetString_().GetForeignKey().GetType().(type) {
			case *list_j5pb.ForeignKeyRules_Uuid:
				if filterOpts.GetString_().GetForeignKey().GetUuid().Filtering.Filterable {
					_, err := uuid.Parse(value)
					if err != nil {
						return fmt.Errorf("parsing uuid: %w", err)
					}
				}
			}
		}
	case protoreflect.MessageKind:
		if field.Message().FullName() == "google.protobuf.Timestamp" {
			if filterOpts.GetTimestamp().GetFiltering().Filterable {
				_, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return fmt.Errorf("parsing timestamp: %w", err)
				}
			}
		}
	}

	return nil
}
