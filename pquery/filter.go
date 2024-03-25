package pquery

import (
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
	field      *nestedField
	oneof      *nestedOneof
	filterVals []interface{}
}

func (fs filterSpec) jsonbPath() string {
	out := strings.Builder{}
	out.WriteString("$")

	if fs.field != nil {
		for idx := range fs.field.fieldPath {
			out.WriteString(fmt.Sprintf(".%s", fs.field.fieldPath[idx].JSONName()))

			if fs.field.fieldPath[idx].Cardinality() == protoreflect.Repeated {
				out.WriteString("[*]")
			}
		}
	} else if fs.oneof != nil {
		for idx := range fs.oneof.fieldPath {
			out.WriteString(fmt.Sprintf(".%s", fs.oneof.fieldPath[idx].JSONName()))
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
					field: &nestedField{
						jsonPath:  []string{field.JSONName()},
						fieldPath: []protoreflect.FieldDescriptor{field},
						field:     field,
					},
					filterVals: vals,
				})
			}
		} else if field.Kind() == protoreflect.MessageKind {
			subFilter, err := buildDefaultFilters(field.Message().Fields())
			if err != nil {
				return nil, err
			}

			for _, subFilterField := range subFilter {
				subFilterField.field.jsonPath = append([]string{field.JSONName()}, subFilterField.field.jsonPath...)
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
			f, err := findField(ll.arrayField.Message(), filters[i].GetField().GetName())
			if err != nil {
				of, err := findOneOfField(ll.arrayField.Message(), filters[i].GetField().GetName())
				if err != nil {
					return nil, fmt.Errorf("dynamic filter: %w", err)
				}

				o, err := ll.buildDynamicFilterOneof(tableAlias, of, filters[i])
				if err != nil {
					return nil, fmt.Errorf("dynamic filter: oneof: %w", err)
				}

				out = append(out, o)
			}

			if f != nil {
				o, err := ll.buildDynamicFilterField(tableAlias, f, filters[i])
				if err != nil {
					return nil, fmt.Errorf("dynamic filter: build field: %w", err)
				}

				out = append(out, o)
			}
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

func (ll *Lister[REQ, RES]) buildDynamicFilterField(tableAlias string, f *nestedField, filter *psml_pb.Filter) (sq.Sqlizer, error) {
	var out sq.And

	field := &filterSpec{
		field: f,
	}

	if filter.GetField() == nil {
		return nil, fmt.Errorf("dynamic filter: field is nil")
	}

	switch filter.GetField().GetType().(type) {
	case *psml_pb.Field_Value:
		val, err := validateFilterableField(field.field, filter.GetField().GetValue())
		if err != nil {
			return nil, fmt.Errorf("dynamic filter: value validation: %w", err)
		}

		out = sq.And{sq.Expr(fmt.Sprintf("jsonb_path_query_array(%s.%s, '%s') @> ?", tableAlias, ll.dataColumn, field.jsonbPath()), pg.JSONB(val))}
	case *psml_pb.Field_Range:
		var min, max interface{}
		min = ""
		max = ""
		var err error

		rawMin := filter.GetField().GetRange().GetMin()
		rawMax := filter.GetField().GetRange().GetMax()

		if rawMin != "" {
			min, err = validateFilterableField(field.field, rawMin)
			if err != nil {
				return nil, fmt.Errorf("dynamic filter: range min validation: %w", err)
			}
		}

		if rawMax != "" {
			max, err = validateFilterableField(field.field, rawMax)
			if err != nil {
				return nil, fmt.Errorf("dynamic filter: range max validation: %w", err)
			}
		}

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

func validateFilterableField(field protoreflect.FieldDescriptor, reqValue string) (interface{}, error) {
	filterable := false
	var val interface{}

	fieldOpts := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)

	if fieldOpts != nil {
		switch fieldOps := fieldOpts.Type.(type) {
		case *psml_pb.FieldConstraint_Double:
			if fieldOps.Double.Filtering != nil && fieldOps.Double.Filtering.Filterable {
				v, err := strconv.ParseFloat(reqValue, 64)
				if err != nil {
					return nil, fmt.Errorf("parsing double: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Fixed32:
			if fieldOps.Fixed32.Filtering != nil && fieldOps.Fixed32.Filtering.Filterable {
				v, err := strconv.ParseUint(reqValue, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("parsing fixed32: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Fixed64:
			if fieldOps.Fixed64.Filtering != nil && fieldOps.Fixed64.Filtering.Filterable {
				v, err := strconv.ParseUint(reqValue, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("parsing fixed64: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Float:
			if fieldOps.Float.Filtering != nil && fieldOps.Float.Filtering.Filterable {
				v, err := strconv.ParseFloat(reqValue, 32)
				if err != nil {
					return nil, fmt.Errorf("parsing float: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Int32:
			if fieldOps.Int32.Filtering != nil && fieldOps.Int32.Filtering.Filterable {
				v, err := strconv.ParseInt(reqValue, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("parsing int32: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Int64:
			if fieldOps.Int64.Filtering != nil && fieldOps.Int64.Filtering.Filterable {
				v, err := strconv.ParseInt(reqValue, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("parsing int64: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Sfixed32:
			if fieldOps.Sfixed32.Filtering != nil && fieldOps.Sfixed32.Filtering.Filterable {
				v, err := strconv.ParseInt(reqValue, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("parsing sfixed32: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Sfixed64:
			if fieldOps.Sfixed64.Filtering != nil && fieldOps.Sfixed64.Filtering.Filterable {
				v, err := strconv.ParseInt(reqValue, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("parsing sfixed64: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Sint32:
			if fieldOps.Sint32.Filtering != nil && fieldOps.Sint32.Filtering.Filterable {
				v, err := strconv.ParseInt(reqValue, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("parsing sint32: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Sint64:
			if fieldOps.Sint64.Filtering != nil && fieldOps.Sint64.Filtering.Filterable {
				v, err := strconv.ParseInt(reqValue, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("parsing sint64: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Uint32:
			if fieldOps.Uint32.Filtering != nil && fieldOps.Uint32.Filtering.Filterable {
				v, err := strconv.ParseUint(reqValue, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("parsing uint32: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Uint64:
			if fieldOps.Uint64.Filtering != nil && fieldOps.Uint64.Filtering.Filterable {
				v, err := strconv.ParseUint(reqValue, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("parsing uint64: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_Bool:
			if fieldOps.Bool.Filtering != nil && fieldOps.Bool.Filtering.Filterable {
				v, err := strconv.ParseBool(reqValue)
				if err != nil {
					return nil, fmt.Errorf("parsing bool: %w", err)
				}

				val = v
				filterable = true
			}
		case *psml_pb.FieldConstraint_String_:
			switch fieldOps := fieldOps.String_.WellKnown.(type) {
			case *psml_pb.StringRules_Date:
				if fieldOps.Date.Filtering != nil && fieldOps.Date.Filtering.Filterable {
					_, err := time.Parse("2006-01-02", reqValue)
					if err != nil {
						_, err = time.Parse("2006-01", reqValue)
						if err != nil {
							_, err = time.Parse("2006", reqValue)
							if err != nil {
								return nil, fmt.Errorf("parsing date: %w", err)
							}
						}
					}

					val = reqValue
					filterable = true
				}
			case *psml_pb.StringRules_ForeignKey:
				switch fieldOps := fieldOps.ForeignKey.Type.(type) {
				case *psml_pb.ForeignKeyRules_UniqueString:
					if fieldOps.UniqueString.Filtering != nil && fieldOps.UniqueString.Filtering.Filterable {
						val = reqValue
						filterable = true
					}
				case *psml_pb.ForeignKeyRules_Uuid:
					if fieldOps.Uuid.Filtering != nil && fieldOps.Uuid.Filtering.Filterable {
						_, err := uuid.Parse(reqValue)
						if err != nil {
							return nil, fmt.Errorf("parsing uuid: %w", err)
						}

						val = reqValue
						filterable = true
					}
				}
			}
		case *psml_pb.FieldConstraint_Enum:
			if fieldOps.Enum.Filtering != nil && fieldOps.Enum.Filtering.Filterable {
				name := strings.ToTitle(reqValue)
				prefix := strings.TrimSuffix(string(field.Enum().Values().Get(0).Name()), "_UNSPECIFIED")

				if !strings.HasPrefix(reqValue, prefix) {
					name = prefix + "_" + name
				}
				eval := field.Enum().Values().ByName(protoreflect.Name(name))

				if eval == nil {
					return nil, fmt.Errorf("enum value %s is not a valid enum value for field", reqValue)
				}

				val = eval.Name()
				filterable = true
			}
		case *psml_pb.FieldConstraint_Timestamp:
			if fieldOps.Timestamp.Filtering != nil && fieldOps.Timestamp.Filtering.Filterable {
				t, err := time.Parse(time.RFC3339, reqValue)
				if err != nil {
					return nil, fmt.Errorf("parsing timestamp: %w", err)
				}

				val = t
				filterable = true
			}
		default:
			return nil, fmt.Errorf("field %s of unsupported filtering type", field.JSONName())
		}
	}

	if !filterable {
		return nil, fmt.Errorf("field %s is not filterable", field.JSONName())
	}

	return val, nil
}
