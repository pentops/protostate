package pquery

import (
	"fmt"
	"strings"

	"github.com/pentops/protostate/gen/list/v1/psml_pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type sortSpec struct {
	nestedField
	desc bool
}

func (ss sortSpec) jsonbPath() string {
	out := strings.Builder{}
	last := len(ss.jsonPath) - 1
	for idx, part := range ss.jsonPath {
		// last part gets a double >
		if idx == last {
			out.WriteString("->>")
		} else {
			out.WriteString("->")
		}
		out.WriteString(fmt.Sprintf("'%s'", part))
	}

	return out.String()
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
		fieldOpts := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)

		if fieldOpts != nil {
			isDefaultSort := false

			switch fieldOps := fieldOpts.Type.(type) {
			case *psml_pb.FieldConstraint_Double:
				if fieldOps.Double.Sorting != nil && fieldOps.Double.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Fixed32:
				if fieldOps.Fixed32.Sorting != nil && fieldOps.Fixed32.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Fixed64:
				if fieldOps.Fixed64.Sorting != nil && fieldOps.Fixed64.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Float:
				if fieldOps.Float.Sorting != nil && fieldOps.Float.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Int32:
				if fieldOps.Int32.Sorting != nil && fieldOps.Int32.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Int64:
				if fieldOps.Int64.Sorting != nil && fieldOps.Int64.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Sfixed32:
				if fieldOps.Sfixed32.Sorting != nil && fieldOps.Sfixed32.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Sfixed64:
				if fieldOps.Sfixed64.Sorting != nil && fieldOps.Sfixed64.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Sint32:
				if fieldOps.Sint32.Sorting != nil && fieldOps.Sint32.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Sint64:
				if fieldOps.Sint64.Sorting != nil && fieldOps.Sint64.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Uint32:
				if fieldOps.Uint32.Sorting != nil && fieldOps.Uint32.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Uint64:
				if fieldOps.Uint64.Sorting != nil && fieldOps.Uint64.Sorting.DefaultSort {
					isDefaultSort = true
				}
			case *psml_pb.FieldConstraint_Timestamp:
				if fieldOps.Timestamp.Sorting != nil && fieldOps.Timestamp.Sorting.DefaultSort {
					isDefaultSort = true
				}
			}
			if isDefaultSort {
				defaultSortFields = append(defaultSortFields, sortSpec{
					nestedField: nestedField{
						field:     field,
						fieldPath: []protoreflect.FieldDescriptor{field},
						jsonPath:  []string{field.JSONName()},
					},
					desc: true,
				})
			}
		} else if field.Kind() == protoreflect.MessageKind {
			subSort := buildDefaultSorts(field.Message().Fields())
			for idx, subSortField := range subSort {
				subSortField.jsonPath = append([]string{field.JSONName()}, subSortField.jsonPath...)
				subSortField.fieldPath = append([]protoreflect.FieldDescriptor{field}, subSortField.fieldPath...)
				subSort[idx] = subSortField
			}
			defaultSortFields = append(defaultSortFields, subSort...)
		}
	}

	return defaultSortFields
}

func (ll *Lister[REQ, RES]) buildDynamicSortSpec(sorts []*psml_pb.Sort) ([]sortSpec, error) {
	results := []sortSpec{}
	direction := ""
	for _, sort := range sorts {
		// validate the fields requested exist
		nestedField, err := findField(ll.arrayField.Message(), sort.Field)
		if err != nil {
			return nil, fmt.Errorf("requested sort: %w", err)
		}

		if nestedField.field.Cardinality() == protoreflect.Repeated {
			return nil, fmt.Errorf("requested sort field '%s' is a repeated field, it must be a scalar", sort.Field)
		}

		// validate the fields requested are marked as sortable
		sortOpts, ok := proto.GetExtension(nestedField.field.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)
		if !ok {
			return nil, fmt.Errorf("requested sort field '%s' does not have any sortable constraints defined", sort.Field)
		}

		sortable := false
		if sortOpts != nil {
			switch nestedField.field.Kind() {
			case protoreflect.DoubleKind:
				sortable = sortOpts.GetDouble().GetSorting().Sortable
			case protoreflect.Fixed32Kind:
				sortable = sortOpts.GetFixed32().GetSorting().Sortable
			case protoreflect.Fixed64Kind:
				sortable = sortOpts.GetFixed64().GetSorting().Sortable
			case protoreflect.FloatKind:
				sortable = sortOpts.GetFloat().GetSorting().Sortable
			case protoreflect.Int32Kind:
				sortable = sortOpts.GetInt32().GetSorting().Sortable
			case protoreflect.Int64Kind:
				sortable = sortOpts.GetInt64().GetSorting().Sortable
			case protoreflect.Sfixed32Kind:
				sortable = sortOpts.GetSfixed32().GetSorting().Sortable
			case protoreflect.Sfixed64Kind:
				sortable = sortOpts.GetSfixed64().GetSorting().Sortable
			case protoreflect.Sint32Kind:
				sortable = sortOpts.GetSint32().GetSorting().Sortable
			case protoreflect.Sint64Kind:
				sortable = sortOpts.GetSint64().GetSorting().Sortable
			case protoreflect.Uint32Kind:
				sortable = sortOpts.GetUint32().GetSorting().Sortable
			case protoreflect.Uint64Kind:
				sortable = sortOpts.GetUint64().GetSorting().Sortable
			case protoreflect.MessageKind:
				if nestedField.field.Message().FullName() == "google.protobuf.Timestamp" {
					sortable = sortOpts.GetTimestamp().GetSorting().Sortable
				}
			}
		}

		if !sortable {
			return nil, fmt.Errorf("requested sort field '%s' is not sortable", sort.Field)
		}

		results = append(results, sortSpec{
			nestedField: *nestedField,
			desc:        sort.Descending,
		})

		// validate direction of all the fields is the same
		if direction == "" {
			direction = "ASC"
			if sort.Descending {
				direction = "DESC"
			}
		} else {
			if (direction == "DESC" && !sort.Descending) || (direction == "ASC" && sort.Descending) {
				return nil, fmt.Errorf("requested sorts have conflicting directions, they must all be the same")
			}
		}
	}

	return results, nil
}

func validateSortsAnnotations(fields protoreflect.FieldDescriptors) error {
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)

		if field.Kind() == protoreflect.MessageKind {
			subFields := field.Message().Fields()

			for i := 0; i < subFields.Len(); i++ {
				subField := subFields.Get(i)

				if subField.Kind() == protoreflect.MessageKind {
					err := validateSortsAnnotations(subField.Message().Fields())
					if err != nil {
						return err
					}
				} else {
					if field.Cardinality() == protoreflect.Repeated {
						fieldOpts := proto.GetExtension(subField.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)
						if fieldOpts != nil {
							invalid := false
							switch fieldOpts.Type.(type) {
							case *psml_pb.FieldConstraint_Double:
								if fieldOpts.GetDouble().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Fixed32:
								if fieldOpts.GetFixed32().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Fixed64:
								if fieldOpts.GetFixed64().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Float:
								if fieldOpts.GetFloat().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Int32:
								if fieldOpts.GetInt32().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Int64:
								if fieldOpts.GetInt64().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Sfixed32:
								if fieldOpts.GetSfixed32().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Sfixed64:
								if fieldOpts.GetSfixed64().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Sint32:
								if fieldOpts.GetSint32().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Sint64:
								if fieldOpts.GetSint64().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Uint32:
								if fieldOpts.GetUint32().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Uint64:
								if fieldOpts.GetUint64().Sorting != nil {
									invalid = true
								}
							case *psml_pb.FieldConstraint_Timestamp:
								if fieldOpts.GetTimestamp().Sorting != nil {
									invalid = true
								}
							}

							if invalid {
								return fmt.Errorf("sorting not allowed on subfield of repeated parent: %s", field.Name())
							}
						}
					}
				}
			}
		}
	}

	return nil
}
