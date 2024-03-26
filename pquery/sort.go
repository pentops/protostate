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
		nestedField, err := findField(ll.arrayField.Message(), sort.Field)
		if err != nil {
			return nil, fmt.Errorf("requested sort: %w", err)
		}

		results = append(results, sortSpec{
			nestedField: *nestedField,
			desc:        sort.Descending,
		})

		// TODO: Remove this constraint, we can sort by different directions once we have the reversal logic in place
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
						// check options of subfield for sorting
						fieldOpts := proto.GetExtension(subField.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)
						if isSortingAnnotated(fieldOpts) {
							return fmt.Errorf("sorting not allowed on subfield of repeated parent: %s", field.Name())
						}
					}
				}
			}
		} else {
			if field.Cardinality() == protoreflect.Repeated {
				// check options of parent field for sorting
				fieldOpts := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)
				if isSortingAnnotated(fieldOpts) {
					return fmt.Errorf("sorting not allowed on repeated field, must be a scalar: %s", field.Name())
				}
			}

		}
	}

	return nil
}

func isSortingAnnotated(opts *psml_pb.FieldConstraint) bool {
	annotated := false

	if opts != nil {
		switch opts.Type.(type) {
		case *psml_pb.FieldConstraint_Double:
			if opts.GetDouble().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Fixed32:
			if opts.GetFixed32().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Fixed64:
			if opts.GetFixed64().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Float:
			if opts.GetFloat().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Int32:
			if opts.GetInt32().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Int64:
			if opts.GetInt64().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Sfixed32:
			if opts.GetSfixed32().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Sfixed64:
			if opts.GetSfixed64().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Sint32:
			if opts.GetSint32().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Sint64:
			if opts.GetSint64().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Uint32:
			if opts.GetUint32().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Uint64:
			if opts.GetUint64().Sorting != nil {
				annotated = true
			}
		case *psml_pb.FieldConstraint_Timestamp:
			if opts.GetTimestamp().Sorting != nil {
				annotated = true
			}
		}
	}

	return annotated
}
