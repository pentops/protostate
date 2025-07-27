package pquery

import (
	"fmt"

	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/j5/gen/j5/schema/v1/schema_j5pb"
	"github.com/pentops/j5/lib/j5schema"
)

type sortSpec struct {
	*NestedField
	desc bool
}

func (ss sortSpec) errorName() string {
	return ss.Path.JSONPathQuery()
}

func buildTieBreakerFields(dataColumn string, req *j5schema.ObjectSchema, arrayField *j5schema.ObjectSchema, fallback []ProtoField) ([]sortSpec, error) {

	if req.ListRequest != nil && len(req.ListRequest.SortTiebreaker) > 0 {
		tieBreakerFields := make([]sortSpec, 0, len(req.ListRequest.SortTiebreaker))
		for _, tieBreaker := range req.ListRequest.SortTiebreaker {
			spec, err := NewJSONPath(arrayField, tieBreaker)
			if err != nil {
				return nil, fmt.Errorf("field %s in annotated sort tiebreaker for %s: %w", tieBreaker, req.FullName(), err)
			}

			tieBreakerFields = append(tieBreakerFields, sortSpec{
				NestedField: &NestedField{
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

		path, err := NewJSONPath(arrayField, tieBreaker.pathInRoot)
		if err != nil {
			return nil, fmt.Errorf("field %s in fallback sort tiebreaker for %s: %w", tieBreaker.pathInRoot, req.FullName(), err)
		}

		tieBreakerFields = append(tieBreakerFields, sortSpec{
			NestedField: &NestedField{
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

	err := WalkPathNodes(message, func(path Path) error {
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
				NestedField: &NestedField{
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

func (ll *Lister) buildDynamicSortSpec(sorts []*list_j5pb.Sort) ([]sortSpec, error) {
	results := []sortSpec{}
	direction := ""
	for _, sort := range sorts {
		pathSpec := ParseJSONPathSpec(sort.Field)
		spec, err := NewJSONPath(ll.arrayObject, pathSpec)
		if err != nil {
			return nil, fmt.Errorf("dynamic filter: find field: %w", err)
		}

		biggerSpec := &NestedField{
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

func validateQueryRequestSorts(message *j5schema.ObjectSchema, sorts []*list_j5pb.Sort) error {
	for _, sort := range sorts {
		pathSpec := ParseJSONPathSpec(sort.Field)
		spec, err := NewJSONPath(message, pathSpec)
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
