package pquery

import (
	"fmt"

	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/protostate/internal/pgstore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type sortSpec struct {
	*pgstore.NestedField
	desc bool
}

func (ss sortSpec) errorName() string {
	return ss.NestedField.Path.JSONPathQuery()
}

func buildTieBreakerFields(dataColumn string, req protoreflect.MessageDescriptor, arrayField protoreflect.MessageDescriptor, fallback []ProtoField) ([]sortSpec, error) {
	listRequestAnnotation, ok := proto.GetExtension(req.Options().(*descriptorpb.MessageOptions), list_j5pb.E_ListRequest).(*list_j5pb.ListRequestMessage)
	if ok && listRequestAnnotation != nil && len(listRequestAnnotation.SortTiebreaker) > 0 {
		tieBreakerFields := make([]sortSpec, 0, len(listRequestAnnotation.SortTiebreaker))
		for _, tieBreaker := range listRequestAnnotation.SortTiebreaker {
			spec, err := pgstore.NewProtoPath(arrayField, pgstore.ParseProtoPathSpec(tieBreaker))
			if err != nil {
				return nil, fmt.Errorf("field %s in annotated sort tiebreaker for %s: %w", tieBreaker, req.FullName(), err)
			}

			tieBreakerFields = append(tieBreakerFields, sortSpec{
				NestedField: &pgstore.NestedField{
					RootColumn: dataColumn,
					Path:       *spec,
				},
				desc: false,
			})
		}

		return tieBreakerFields, nil
	}

	if len(fallback) == 0 {
		return []sortSpec{}, nil
	}

	tieBreakerFields := make([]sortSpec, 0, len(fallback))
	for _, tieBreaker := range fallback {

		path, err := pgstore.NewProtoPath(arrayField, tieBreaker.pathInRoot)
		if err != nil {
			return nil, fmt.Errorf("field %s in fallback sort tiebreaker for %s: %w", tieBreaker.pathInRoot, req.FullName(), err)
		}

		tieBreakerFields = append(tieBreakerFields, sortSpec{
			NestedField: &pgstore.NestedField{
				Path:        *path,
				RootColumn:  dataColumn,
				ValueColumn: tieBreaker.valueColumn,
			},
			desc: false,
		})
	}

	return tieBreakerFields, nil
}

func buildDefaultSorts(columnName string, message protoreflect.MessageDescriptor) ([]sortSpec, error) {
	var defaultSortFields []sortSpec

	err := pgstore.WalkPathNodes(message, func(path pgstore.Path) error {
		field := path.LeafField()
		if field == nil {
			return nil // oneof or something
		}

		fieldOpts := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), list_j5pb.E_Field).(*list_j5pb.FieldConstraint)

		if fieldOpts == nil {
			return nil
		}
		isDefaultSort := false

		switch fieldOps := fieldOpts.Type.(type) {
		case *list_j5pb.FieldConstraint_Double:
			if fieldOps.Double.Sorting != nil && fieldOps.Double.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Fixed32:
			if fieldOps.Fixed32.Sorting != nil && fieldOps.Fixed32.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Fixed64:
			if fieldOps.Fixed64.Sorting != nil && fieldOps.Fixed64.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Float:
			if fieldOps.Float.Sorting != nil && fieldOps.Float.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Int32:
			if fieldOps.Int32.Sorting != nil && fieldOps.Int32.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Int64:
			if fieldOps.Int64.Sorting != nil && fieldOps.Int64.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Sfixed32:
			if fieldOps.Sfixed32.Sorting != nil && fieldOps.Sfixed32.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Sfixed64:
			if fieldOps.Sfixed64.Sorting != nil && fieldOps.Sfixed64.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Sint32:
			if fieldOps.Sint32.Sorting != nil && fieldOps.Sint32.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Sint64:
			if fieldOps.Sint64.Sorting != nil && fieldOps.Sint64.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Uint32:
			if fieldOps.Uint32.Sorting != nil && fieldOps.Uint32.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Uint64:
			if fieldOps.Uint64.Sorting != nil && fieldOps.Uint64.Sorting.DefaultSort {
				isDefaultSort = true
			}
		case *list_j5pb.FieldConstraint_Timestamp:
			if fieldOps.Timestamp.Sorting != nil && fieldOps.Timestamp.Sorting.DefaultSort {
				isDefaultSort = true
			}
		}
		if isDefaultSort {
			defaultSortFields = append(defaultSortFields, sortSpec{
				NestedField: &pgstore.NestedField{
					RootColumn: columnName,
					Path:       path,
				},
				desc: true,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return defaultSortFields, nil
}

func (ll *Lister[REQ, RES]) buildDynamicSortSpec(sorts []*list_j5pb.Sort) ([]sortSpec, error) {
	results := []sortSpec{}
	direction := ""
	for _, sort := range sorts {
		pathSpec := pgstore.ParseJSONPathSpec(sort.Field)
		spec, err := pgstore.NewJSONPath(ll.arrayField.Message(), pathSpec)
		if err != nil {
			return nil, fmt.Errorf("dynamic filter: find field: %w", err)
		}

		biggerSpec := &pgstore.NestedField{
			Path:       *spec,
			RootColumn: ll.dataColumn,
		}

		results = append(results, sortSpec{
			NestedField: biggerSpec,
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
	for i := range fields.Len() {
		field := fields.Get(i)

		if field.Kind() == protoreflect.MessageKind {
			subFields := field.Message().Fields()

			for i := range subFields.Len() {
				subField := subFields.Get(i)

				if subField.Kind() == protoreflect.MessageKind {
					err := validateSortsAnnotations(subField.Message().Fields())
					if err != nil {
						return fmt.Errorf("message sort validation: %w", err)
					}
				} else {
					if field.Cardinality() == protoreflect.Repeated {
						// check options of subfield for sorting
						fieldOpts := proto.GetExtension(subField.Options().(*descriptorpb.FieldOptions), list_j5pb.E_Field).(*list_j5pb.FieldConstraint)
						if isSortingAnnotated(fieldOpts) {
							return fmt.Errorf("sorting not allowed on subfield of repeated parent: %s", field.Name())
						}
					}
				}
			}
		} else {
			if field.Cardinality() == protoreflect.Repeated {
				// check options of parent field for sorting
				fieldOpts := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), list_j5pb.E_Field).(*list_j5pb.FieldConstraint)
				if isSortingAnnotated(fieldOpts) {
					return fmt.Errorf("sorting not allowed on repeated field, must be a scalar: %s", field.Name())
				}
			}

		}
	}

	return nil
}

func isSortingAnnotated(opts *list_j5pb.FieldConstraint) bool {
	annotated := false

	if opts != nil {
		switch opts.Type.(type) {
		case *list_j5pb.FieldConstraint_Double:
			if opts.GetDouble().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Fixed32:
			if opts.GetFixed32().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Fixed64:
			if opts.GetFixed64().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Float:
			if opts.GetFloat().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Int32:
			if opts.GetInt32().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Int64:
			if opts.GetInt64().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Sfixed32:
			if opts.GetSfixed32().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Sfixed64:
			if opts.GetSfixed64().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Sint32:
			if opts.GetSint32().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Sint64:
			if opts.GetSint64().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Uint32:
			if opts.GetUint32().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Uint64:
			if opts.GetUint64().Sorting != nil {
				annotated = true
			}
		case *list_j5pb.FieldConstraint_Timestamp:
			if opts.GetTimestamp().Sorting != nil {
				annotated = true
			}
		}
	}

	return annotated
}

func validateQueryRequestSorts(message protoreflect.MessageDescriptor, sorts []*list_j5pb.Sort) error {
	for _, sort := range sorts {
		pathSpec := pgstore.ParseJSONPathSpec(sort.Field)
		spec, err := pgstore.NewJSONPath(message, pathSpec)
		if err != nil {
			return fmt.Errorf("find field %s: %w", sort.Field, err)
		}

		field := spec.LeafField()
		if field == nil {
			return fmt.Errorf("node %s is not a field", spec.DebugName())
		}

		// validate the fields are annotated correctly for the request query
		sortOpts, ok := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), list_j5pb.E_Field).(*list_j5pb.FieldConstraint)
		if !ok {
			return fmt.Errorf("requested sort field '%s' does not have any sortable constraints defined", sort.Field)
		}

		sortable := false
		if sortOpts != nil {
			switch field.Kind() {
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
				if field.Message().FullName() == "google.protobuf.Timestamp" {
					sortable = sortOpts.GetTimestamp().GetSorting().Sortable
				}
			}
		}

		if !sortable {
			return fmt.Errorf("requested sort field '%s' is not sortable", sort.Field)
		}
	}

	return nil
}
