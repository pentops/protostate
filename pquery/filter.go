package pquery

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/elgris/sqrl/pg"
	"github.com/google/uuid"
	"github.com/pentops/protostate/gen/list/v1/psml_pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type filterSpec struct {
	field      protoreflect.FieldDescriptor
	fieldPath  []protoreflect.FieldDescriptor
	filterVals []interface{}
}

func (fs filterSpec) jsonbPath() string {
	out := strings.Builder{}
	out.WriteString("$")

	for idx := range fs.fieldPath {
		out.WriteString(fmt.Sprintf(".%s", fs.fieldPath[idx].JSONName()))

		if fs.fieldPath[idx].Cardinality() == protoreflect.Repeated {
			out.WriteString("[*]")
		}
	}

	return out.String()
}

func buildDefaultFilters(messageFields protoreflect.FieldDescriptors) ([]filterSpec, error) {
	var filters []filterSpec

	for i := 0; i < messageFields.Len(); i++ {
		field := messageFields.Get(i)
		fieldOpts := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)
		vals := []interface{}{}

		if fieldOpts != nil {
			switch fieldOps := fieldOpts.Type.(type) {
			case *psml_pb.FieldConstraint_Float:
				if fieldOps.Float.Filtering != nil && fieldOps.Float.Filtering.Filterable {
					for _, val := range fieldOps.Float.Filtering.DefaultFilters {
						v, err := strconv.ParseFloat(val, 32)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Double:
				if fieldOps.Double.Filtering != nil && fieldOps.Double.Filtering.Filterable {
					for _, val := range fieldOps.Double.Filtering.DefaultFilters {
						v, err := strconv.ParseFloat(val, 64)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Fixed32:
				if fieldOps.Fixed32.Filtering != nil && fieldOps.Fixed32.Filtering.Filterable {
					for _, val := range fieldOps.Fixed32.Filtering.DefaultFilters {
						v, err := strconv.ParseUint(val, 10, 32)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Fixed64:
				if fieldOps.Fixed64.Filtering != nil && fieldOps.Fixed64.Filtering.Filterable {
					for _, val := range fieldOps.Fixed64.Filtering.DefaultFilters {
						v, err := strconv.ParseUint(val, 10, 64)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Int32:
				if fieldOps.Int32.Filtering != nil && fieldOps.Int32.Filtering.Filterable {
					for _, val := range fieldOps.Int32.Filtering.DefaultFilters {
						v, err := strconv.ParseInt(val, 10, 32)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Int64:
				if fieldOps.Int64.Filtering != nil && fieldOps.Int64.Filtering.Filterable {
					for _, val := range fieldOps.Int64.Filtering.DefaultFilters {
						v, err := strconv.ParseInt(val, 10, 64)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Sfixed32:
				if fieldOps.Sfixed32.Filtering != nil && fieldOps.Sfixed32.Filtering.Filterable {
					for _, val := range fieldOps.Sfixed32.Filtering.DefaultFilters {
						v, err := strconv.ParseInt(val, 10, 32)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Sfixed64:
				if fieldOps.Sfixed64.Filtering != nil && fieldOps.Sfixed64.Filtering.Filterable {
					for _, val := range fieldOps.Sfixed64.Filtering.DefaultFilters {
						v, err := strconv.ParseInt(val, 10, 64)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Sint32:
				if fieldOps.Sint32.Filtering != nil && fieldOps.Sint32.Filtering.Filterable {
					for _, val := range fieldOps.Sint32.Filtering.DefaultFilters {
						v, err := strconv.ParseInt(val, 10, 32)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Sint64:
				if fieldOps.Sint64.Filtering != nil && fieldOps.Sint64.Filtering.Filterable {
					for _, val := range fieldOps.Sint64.Filtering.DefaultFilters {
						v, err := strconv.ParseInt(val, 10, 64)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Uint32:
				if fieldOps.Uint32.Filtering != nil && fieldOps.Uint32.Filtering.Filterable {
					for _, val := range fieldOps.Uint32.Filtering.DefaultFilters {
						v, err := strconv.ParseUint(val, 10, 32)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Uint64:
				if fieldOps.Uint64.Filtering != nil && fieldOps.Uint64.Filtering.Filterable {
					for _, val := range fieldOps.Uint64.Filtering.DefaultFilters {
						v, err := strconv.ParseUint(val, 10, 64)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_Bool:
				if fieldOps.Bool.Filtering != nil && fieldOps.Bool.Filtering.Filterable {
					for _, val := range fieldOps.Bool.Filtering.DefaultFilters {
						v, err := strconv.ParseBool(val)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, v)
					}
				}

			case *psml_pb.FieldConstraint_String_:
				switch fieldOps := fieldOps.String_.WellKnown.(type) {
				case *psml_pb.StringRules_Date:
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

				case *psml_pb.StringRules_ForeignKey:
					switch fieldOps := fieldOps.ForeignKey.Type.(type) {
					case *psml_pb.ForeignKeyRules_UniqueString:
						if fieldOps.UniqueString.Filtering != nil && fieldOps.UniqueString.Filtering.Filterable {
							for _, val := range fieldOps.UniqueString.Filtering.DefaultFilters {
								vals = append(vals, val)
							}
						}

					case *psml_pb.ForeignKeyRules_Uuid:
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

			case *psml_pb.FieldConstraint_Enum:
				if fieldOps.Enum.Filtering != nil && fieldOps.Enum.Filtering.Filterable {
					for _, val := range fieldOps.Enum.Filtering.DefaultFilters {
						vals = append(vals, val)
					}
				}

			case *psml_pb.FieldConstraint_Timestamp:
				if fieldOps.Timestamp.Filtering != nil && fieldOps.Timestamp.Filtering.Filterable {
					for _, val := range fieldOps.Timestamp.Filtering.DefaultFilters {
						t, err := time.Parse(time.RFC3339, val)
						if err != nil {
							return nil, fmt.Errorf("error parsing default filter (%s): %w", field.JSONName(), err)
						}

						vals = append(vals, timestamppb.New(t))
					}
				}
			}

			if len(vals) > 0 {
				filters = append(filters, filterSpec{
					fieldPath:  []protoreflect.FieldDescriptor{field},
					field:      field,
					filterVals: vals,
				})
			}
		} else if field.Kind() == protoreflect.MessageKind {
			subFilter, err := buildDefaultFilters(field.Message().Fields())
			if err != nil {
				return nil, err
			}

			for _, subFilterField := range subFilter {
				subFilterField.filterVals = append(vals, subFilterField.filterVals...)
			}
			filters = append(filters, subFilter...)
		}
	}

	return filters, nil
}

func (ll *Lister[REQ, RES]) buildDynamicFilter(tableAlias string, filters []*psml_pb.Filter) ([]sq.Sqlizer, error) {
	out := []sq.Sqlizer{}

	for i := range filters {
		switch filters[i].GetType().(type) {
		case *psml_pb.Filter_Field:
			spec, err := findFieldSpec(ll.arrayField.Message(), filters[i].GetField().GetName())
			if err != nil && !errors.Is(err, ErrOneof) {
				return nil, fmt.Errorf("dynamic filter: find field: %w", err)
			}

			var o sq.Sqlizer
			if errors.Is(err, ErrOneof) {
				ospec, err := findOneofSpec(ll.arrayField.Message(), filters[i].GetField().GetName())
				if err != nil {
					return nil, fmt.Errorf("dynamic filter: find oneof: %w", err)
				}

				o, err = ll.buildDynamicFilterOneof(tableAlias, ospec, filters[i])
				if err != nil {
					return nil, fmt.Errorf("dynamic filter: build oneof: %w", err)
				}
			} else {
				o, err = ll.buildDynamicFilterField(tableAlias, spec, filters[i])
				if err != nil {
					return nil, fmt.Errorf("dynamic filter: build field: %w", err)
				}
			}

			out = append(out, o)
		case *psml_pb.Filter_And:
			f, err := ll.buildDynamicFilter(tableAlias, filters[i].GetAnd().GetFilters())
			if err != nil {
				return nil, fmt.Errorf("dynamic filter: and: %w", err)
			}
			and := sq.And{}
			and = append(and, f...)

			out = append(out, and)
		case *psml_pb.Filter_Or:
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

func (ll *Lister[REQ, RES]) buildDynamicFilterField(tableAlias string, spec *fieldSpec, filter *psml_pb.Filter) (sq.Sqlizer, error) {
	var out sq.And

	field := &filterSpec{
		fieldPath: spec.fieldPath,
	}

	if filter.GetField() == nil {
		return nil, fmt.Errorf("dynamic filter: field is nil")
	}

	switch filter.GetField().GetType().(type) {
	case *psml_pb.Field_Value:
		val := filter.GetField().GetValue()
		if spec.field.Kind() == protoreflect.EnumKind {
			name := strings.ToTitle(val)
			prefix := strings.TrimSuffix(string(spec.field.Enum().Values().Get(0).Name()), "_UNSPECIFIED")

			if !strings.HasPrefix(val, prefix) {
				name = prefix + "_" + name
			}

			val = name
		}

		out = sq.And{sq.Expr(fmt.Sprintf("jsonb_path_query_array(%s.%s, '%s') @> ?", tableAlias, ll.dataColumn, field.jsonbPath()), pg.JSONB(val))}
	case *psml_pb.Field_Range:
		min := filter.GetField().GetRange().GetMin()
		max := filter.GetField().GetRange().GetMax()

		switch {
		case min != "" && max != "":
			exprStr := fmt.Sprintf("jsonb_path_query_array(%s.%s, '%s ?? (@ >= $min && @ <= $max)', jsonb_build_object('min', ?::text, 'max', ?::text)) <> '[]'::jsonb", tableAlias, ll.dataColumn, field.jsonbPath())
			out = sq.And{sq.Expr(exprStr, min, max)}
		case min != "":
			exprStr := fmt.Sprintf("jsonb_path_query_array(%s.%s, '%s ?? (@ >= $min)', jsonb_build_object('min', ?::text)) <> '[]'::jsonb", tableAlias, ll.dataColumn, field.jsonbPath())
			out = sq.And{sq.Expr(exprStr, min)}
		case max != "":
			exprStr := fmt.Sprintf("jsonb_path_query_array(%s.%s, '%s ?? (@ <= $max)', jsonb_build_object('max', ?::text)) <> '[]'::jsonb", tableAlias, ll.dataColumn, field.jsonbPath())
			out = sq.And{sq.Expr(exprStr, max)}
		}
	}

	return out, nil
}

func (ll *Lister[REQ, RES]) buildDynamicFilterOneof(tableAlias string, ospec *oneofSpec, filter *psml_pb.Filter) (sq.Sqlizer, error) {
	var out sq.And

	field := &filterSpec{
		fieldPath: ospec.fieldPath,
	}

	if filter.GetField() == nil {
		return nil, fmt.Errorf("dynamic filter: field is nil")
	}

	switch filter.GetField().GetType().(type) {
	case *psml_pb.Field_Value:
		val := filter.GetField().GetValue()

		// Val is used directly here instead of passed in as an expression
		// parameter. It has been sanitized by validation against the oneof
		// field names.
		exprStr := fmt.Sprintf("jsonb_array_length(jsonb_path_query_array(%s.%s, '%s ?? (exists(@.%s))')) > 0", tableAlias, ll.dataColumn, field.jsonbPath(), val)
		out = sq.And{sq.Expr(exprStr)}
	case *psml_pb.Field_Range:
		return nil, fmt.Errorf("oneofs cannot be filtered by range")
	}

	return out, nil
}

func validateQueryRequestFilters(message protoreflect.MessageDescriptor, filters []*psml_pb.Filter) error {
	for i := range filters {
		switch filters[i].GetType().(type) {
		case *psml_pb.Filter_Field:
			return validateQueryRequestFilterField(message, filters[i].GetField())
		case *psml_pb.Filter_And:
			return validateQueryRequestFilters(message, filters[i].GetAnd().GetFilters())
		case *psml_pb.Filter_Or:
			return validateQueryRequestFilters(message, filters[i].GetOr().GetFilters())
		}
	}

	return nil
}

func validateQueryRequestFilterField(message protoreflect.MessageDescriptor, filterField *psml_pb.Field) error {
	// validate fields exist from the request query
	err := validateFieldName(message, filterField.GetName())
	if err != nil {
		return fmt.Errorf("field name: %w", err)
	}

	spec, err := findFieldSpec(message, filterField.GetName())
	if err != nil && !errors.Is(err, ErrOneof) {
		return fmt.Errorf("find field: %w", err)
	}

	// validate the fields are annotated correctly for the request query
	// and the values are valid for the field

	filterable := false

	if errors.Is(err, ErrOneof) {
		ospec, err := findOneofSpec(message, filterField.GetName())
		if err != nil {
			return fmt.Errorf("find field: %w", err)
		}

		filterOpts, ok := proto.GetExtension(ospec.oneof.Options().(*descriptorpb.OneofOptions), psml_pb.E_Oneof).(*psml_pb.OneofConstraint)
		if !ok {
			return fmt.Errorf("requested filter field '%s' does not have any filterable constraints defined", filterField.Name)
		}

		if filterOpts != nil {
			filterable = filterOpts.GetFiltering().Filterable

			if filterable {
				switch filterField.Type.(type) {
				case *psml_pb.Field_Value:
					val := filterField.GetValue()

					found := false
					for i := 0; i < ospec.oneof.Fields().Len(); i++ {
						f := ospec.oneof.Fields().Get(i)
						if strings.EqualFold(string(f.Name()), val) {
							found = true
							break
						}
					}

					if !found {
						return fmt.Errorf("filter value '%s' is not found in oneof '%s'", val, filterField.Name)
					}
				case *psml_pb.Field_Range:
					return fmt.Errorf("oneofs cannot be filtered by range")
				}
			}
		}

		if !filterable {
			return fmt.Errorf("requested filter field '%s' is not filterable", filterField.Name)
		}

		return nil
	}

	filterOpts, ok := proto.GetExtension(spec.field.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)
	if !ok {
		return fmt.Errorf("requested filter field '%s' does not have any filterable constraints defined", filterField.Name)
	}

	if filterOpts != nil {
		switch spec.field.Kind() {
		case protoreflect.DoubleKind:
			filterable = filterOpts.GetDouble().GetFiltering().Filterable
		case protoreflect.Fixed32Kind:
			filterable = filterOpts.GetFixed32().GetFiltering().Filterable
		case protoreflect.Fixed64Kind:
			filterable = filterOpts.GetFixed64().GetFiltering().Filterable
		case protoreflect.FloatKind:
			filterable = filterOpts.GetFloat().GetFiltering().Filterable
		case protoreflect.Int32Kind:
			filterable = filterOpts.GetInt32().GetFiltering().Filterable
		case protoreflect.Int64Kind:
			filterable = filterOpts.GetInt64().GetFiltering().Filterable
		case protoreflect.Sfixed32Kind:
			filterable = filterOpts.GetSfixed32().GetFiltering().Filterable
		case protoreflect.Sfixed64Kind:
			filterable = filterOpts.GetSfixed64().GetFiltering().Filterable
		case protoreflect.Sint32Kind:
			filterable = filterOpts.GetSint32().GetFiltering().Filterable
		case protoreflect.Sint64Kind:
			filterable = filterOpts.GetSint64().GetFiltering().Filterable
		case protoreflect.Uint32Kind:
			filterable = filterOpts.GetUint32().GetFiltering().Filterable
		case protoreflect.Uint64Kind:
			filterable = filterOpts.GetUint64().GetFiltering().Filterable
		case protoreflect.BoolKind:
			filterable = filterOpts.GetBool().GetFiltering().Filterable
		case protoreflect.EnumKind:
			filterable = filterOpts.GetEnum().GetFiltering().Filterable
		case protoreflect.StringKind:
			switch filterOpts.GetString_().WellKnown.(type) {
			case *psml_pb.StringRules_Date:
				filterable = filterOpts.GetString_().GetDate().Filtering.Filterable
			case *psml_pb.StringRules_ForeignKey:
				switch filterOpts.GetString_().GetForeignKey().GetType().(type) {
				case *psml_pb.ForeignKeyRules_UniqueString:
					filterable = filterOpts.GetString_().GetForeignKey().GetUniqueString().Filtering.Filterable
				case *psml_pb.ForeignKeyRules_Uuid:
					filterable = filterOpts.GetString_().GetForeignKey().GetUuid().Filtering.Filterable
				}
			}
		case protoreflect.MessageKind:
			if spec.field.Message().FullName() == "google.protobuf.Timestamp" {
				filterable = filterOpts.GetTimestamp().GetFiltering().Filterable
			}
		}

		if filterable {
			switch filterField.Type.(type) {
			case *psml_pb.Field_Value:
				err := validateFilterFieldValue(filterOpts, spec, filterField.GetValue())
				if err != nil {
					return fmt.Errorf("filter value: %w", err)
				}
			case *psml_pb.Field_Range:
				err := validateFilterFieldValue(filterOpts, spec, filterField.GetRange().GetMin())
				if err != nil {
					return fmt.Errorf("filter min value: %w", err)
				}

				err = validateFilterFieldValue(filterOpts, spec, filterField.GetRange().GetMax())
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
}

func validateFilterFieldValue(filterOpts *psml_pb.FieldConstraint, spec *fieldSpec, value string) error {
	if value == "" {
		return nil
	}

	switch spec.field.Kind() {
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
			prefix := strings.TrimSuffix(string(spec.field.Enum().Values().Get(0).Name()), "_UNSPECIFIED")

			if !strings.HasPrefix(value, prefix) {
				name = prefix + "_" + name
			}
			eval := spec.field.Enum().Values().ByName(protoreflect.Name(name))

			if eval == nil {
				return fmt.Errorf("enum value %s is not a valid enum value for field", value)
			}
		}
	case protoreflect.StringKind:
		switch filterOpts.GetString_().WellKnown.(type) {
		case *psml_pb.StringRules_Date:
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
		case *psml_pb.StringRules_ForeignKey:
			switch filterOpts.GetString_().GetForeignKey().GetType().(type) {
			case *psml_pb.ForeignKeyRules_Uuid:
				if filterOpts.GetString_().GetForeignKey().GetUuid().Filtering.Filterable {
					_, err := uuid.Parse(value)
					if err != nil {
						return fmt.Errorf("parsing uuid: %w", err)
					}
				}
			}
		}
	case protoreflect.MessageKind:
		if spec.field.Message().FullName() == "google.protobuf.Timestamp" {
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
