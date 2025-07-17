package pquery

import (
	"fmt"

	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/j5/gen/j5/schema/v1/schema_j5pb"
	"github.com/pentops/j5/lib/j5schema"
	"github.com/pentops/protostate/internal/pgstore"
)

type sortSpec struct {
	*pgstore.NestedField
	desc bool
}

func (ss sortSpec) errorName() string {
	return ss.Path.JSONPathQuery()
}

func buildTieBreakerFields(dataColumn string, req *j5schema.ObjectSchema, arrayField *j5schema.ObjectSchema, fallback []ProtoField) ([]sortSpec, error) {

	if req.ListRequest != nil && len(req.ListRequest.SortTiebreaker) > 0 {
		tieBreakerFields := make([]sortSpec, 0, len(req.ListRequest.SortTiebreaker))
		for _, tieBreaker := range req.ListRequest.SortTiebreaker {
			spec, err := pgstore.NewOuterPath(arrayField, tieBreaker)
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

		path, err := pgstore.NewOuterPath(arrayField, tieBreaker.pathInRoot)
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

func getFieldSorting(field *j5schema.ObjectProperty) *list_j5pb.SortingConstraint {
	scalar, ok := field.Schema.(*j5schema.ScalarSchema)
	if !ok {
		return nil // only scalars are sortable
	}

	schema := scalar.ToJ5Field()

	switch st := schema.Type.(type) {
	case *schema_j5pb.Field_Float:
		if st.Float.ListRules == nil {
			return nil
		}
		return st.Float.ListRules.Sorting
	case *schema_j5pb.Field_Integer:
		if st.Integer.ListRules == nil {
			return nil
		}
		return st.Integer.ListRules.Sorting
	case *schema_j5pb.Field_Timestamp:
		if st.Timestamp.ListRules == nil {
			return nil
		}
		return st.Timestamp.ListRules.Sorting
	case *schema_j5pb.Field_Decimal:
		if st.Decimal.ListRules == nil {
			return nil
		}
		return st.Decimal.ListRules.Sorting
	default:
		return nil
	}
}

func buildDefaultSorts(columnName string, message *j5schema.ObjectSchema) ([]sortSpec, error) {
	var defaultSortFields []sortSpec

	err := pgstore.WalkPathNodes(message, func(path pgstore.Path) error {
		field := path.LeafField()
		if field == nil {
			return nil // oneof or something
		}

		sortConstraint := getFieldSorting(field)
		if sortConstraint == nil {
			return nil // not a sortable field
		}
		if sortConstraint.DefaultSort {
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
		spec, err := pgstore.NewJSONPath(ll.arrayObject, pathSpec)
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

func validateQueryRequestSorts(message *j5schema.ObjectSchema, sorts []*list_j5pb.Sort) error {
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

		sortAnnotation := getFieldSorting(field)
		if sortAnnotation == nil || !sortAnnotation.Sortable {
			return fmt.Errorf("requested sort field '%s' is not sortable", sort.Field)
		}
	}

	return nil
}
