package pquery

import (
	"fmt"
	"regexp"
	"strings"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/j5/gen/j5/schema/v1/schema_j5pb"
	"github.com/pentops/j5/lib/j5schema"
)

/*
	func validateSearchesAnnotations(props []*j5schema.ObjectProperty) error {
		_, err := validateSearchesAnnotationsInner(props)
		if err != nil {
			return fmt.Errorf("search validation: %w", err)
		}
		return nil
	}

	func validateSearchesAnnotationsInner(fields []*j5schema.ObjectProperty) (map[string]protoreflect.Name, error) {
		// search annotations have a 'field_identifier' which specifies the database column name to use for the text-search-vector.
		// This function validates that the field_identifier is unique for the given field-set
		// In cases where the field is a message, it will recurse into the message to validate the field identifiers

		// When the same message is used in multiple places, any field of that
		// message or a child message will, by definition, have the same field
		// identifier, and is therefore invalid.

		// Search annotation cannot be used in repeated or map fields
		// Recursion is already invalid.

		// Oneof fields, however, can have the same field identifier on different
		// branches.

		ids := make(map[string]string)

		for _, field := range fields {

			err := validateSearchAnnotationsField(ids, field)
			if err != nil {
				return nil, err
			}
		}

		return ids, nil
	}

func validateSearchAnnotationsField(ids map[string]string, field *j5schema.ObjectProperty) error {

		switch bigType := field.Schema.(type) {
		case *j5schema.ObjectField:
			fields := bigType.ObjectSchema().Fields()
			searchIdentifiers, err := validateSearchesAnnotationsInner(fields)
			if err != nil {
				return fmt.Errorf("object search validation: %w", err)
			}

			for searchKey, usedIn := range searchIdentifiers {
				if existing, ok := ids[searchKey]; ok {
					return fmt.Errorf("field identifier '%s' is already used at %s", searchKey, existing)
				}
				ids[searchKey] = protoreflect.Name(fmt.Sprintf("%s.%s", field.JSONName, usedIn))
			}

		case *j5schema.OneofField:
			fields := bigType.OneofSchema().Fields()
			searchIdentifiers, err := validateSearchesAnnotationsInner(fields)
			if err != nil {
				return fmt.Errorf("oneof search validation: %w", err)
			}
			for searchKey, usedIn := range searchIdentifiers {
				if existing, ok := ids[searchKey]; ok {
					return fmt.Errorf("field identifier '%s' is already used at %s", searchKey, existing)
				}
				ids[searchKey] = protoreflect.Name(fmt.Sprintf("%s.%s", field.JSONName, usedIn))
			}

		case *j5schema.MapField:
			items := bigType.MapSchema()

		case *j5schema.ArrayField:

		case *j5schema.ScalarField:

			schema := bigType

		}
	}

	switch field.Kind() {
	case protoreflect.StringKind:
		fieldOpts, ok := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), list_j5pb.E_Field).(*list_j5pb.FieldConstraint)
		if !ok {
			return nil
		}

		switch fieldOpts.GetString_().GetWellKnown().(type) {
		case *list_j5pb.StringRules_OpenText:
			searchOpts := fieldOpts.GetString_().GetOpenText().GetSearching()
			if searchOpts == nil || !searchOpts.Searchable {
				return nil
			}

			if searchOpts.GetFieldIdentifier() == "" {
				return fmt.Errorf("field '%s' is missing a field identifier", field.FullName())
			}

			if existing, ok := ids[searchOpts.GetFieldIdentifier()]; ok {
				return fmt.Errorf("field identifier '%s' is already used at %s", searchOpts.GetFieldIdentifier(), existing)
			}

			ids[searchOpts.GetFieldIdentifier()] = field.Name()
		}

		return nil
	case protoreflect.MessageKind:
		msg := field.Message().Fields()
		fields := make([]protoreflect.FieldDescriptor, msg.Len())
		for i := range msg.Len() {
			fields[i] = msg.Get(i)
		}

		searchIdentifiers, err := validateSearchesAnnotationsInner(fields)
		if err != nil {
			return fmt.Errorf("message search validation: %w", err)
		}
		for searchKey, usedIn := range searchIdentifiers {
			if existing, ok := ids[searchKey]; ok {
				return fmt.Errorf("field identifier '%s' is already used at %s", searchKey, existing)
			}

			ids[searchKey] = protoreflect.Name(fmt.Sprintf("%s.%s", field.Name(), usedIn))
		}
	}

	return nil

	func validateQueryRequestSearches(message *j5schema.ObjectSchema, searches []*list_j5pb.Search) error {
		for _, search := range searches {

			spec, err := NewJSONPath(message, ParseJSONPathSpec(search.GetField()))
			if err != nil {
				return fmt.Errorf("field spec: %w", err)
			}

			field := spec.LeafField()
			if field == nil {
				return fmt.Errorf("leaf '%s' is not a field", spec.JSONPathQuery())
			}

			// validate the fields are annotated correctly for the request query
			searchOpts, ok := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), list_j5pb.E_Field).(*list_j5pb.FieldConstraint)
			if !ok {
				return fmt.Errorf("requested search field '%s' does not have any searchable constraints defined", search.Field)
			}

			searchable := false
			if searchOpts != nil {
				switch field.Kind() {
				case protoreflect.StringKind:
					switch searchOpts.GetString_().WellKnown.(type) {
					case *list_j5pb.StringRules_OpenText:
						searchable = searchOpts.GetString_().GetOpenText().GetSearching().Searchable
					}
				}
			}

			if !searchable {
				return fmt.Errorf("requested search field '%s' is not searchable", search.Field)
			}
		}

		return nil
	}
*/
var rePgUnsafe = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func TSVColumns(message *j5schema.ObjectSchema) ([]*TSVColumn, error) {
	return tsvColumns(message)
}

func buildTsvColumnMap(message *j5schema.ObjectSchema) (map[string]string, error) {
	out := make(map[string]string)
	paths, err := tsvColumns(message)
	if err != nil {
		return nil, fmt.Errorf("build tsv column map: %w", err)
	}

	for _, path := range paths {
		out[path.IDPath()] = path.ColumnName
	}

	return out, nil
}

type TSVColumn struct {
	Path
	ColumnName string
}

func tsvColumns(message *j5schema.ObjectSchema) ([]*TSVColumn, error) {

	usedColNames := make(map[string]struct{})
	out := []*TSVColumn{}

	err := WalkPathNodes(message, func(path Path) error {
		field := path.LeafField()
		switch bigSchema := field.Schema.(type) {

		case *j5schema.ScalarSchema:
			innerSchema := bigSchema.ToJ5Field()
			switch ist := innerSchema.Type.(type) {
			case *schema_j5pb.Field_String_:
				if ist.String_.ListRules == nil || ist.String_.ListRules.Searching == nil || !ist.String_.ListRules.Searching.Searchable {
					return nil // not searchable
				}

				idPath := path.IDPath()
				columnName := rePgUnsafe.ReplaceAllString(idPath, "_")
				columnName = strings.ToLower(columnName)
				if _, exists := usedColNames[columnName]; exists {
					// the character set isn't sufficient to guarantee
					// uniqueness when dropping the casing, e.g., `foo_d`
					// becomes `fooD`, which becomes `food`, and `food` itself
					// could also be valid.
					// It's unlikely to come up, but better throw here than
					// behave unexpectedly later.
					return fmt.Errorf("duplicate tsv column name %q for path %q", columnName, idPath)
				}
				usedColNames[columnName] = struct{}{}

				out = append(out, &TSVColumn{
					Path:       path,
					ColumnName: fmt.Sprintf("tsv_%s", columnName),
				})
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (ll *Lister) buildDynamicSearches(tableAlias string, searches []*list_j5pb.Search) ([]sq.Sqlizer, error) {
	out := []sq.Sqlizer{}

	for i := range searches {
		col, ok := ll.tsvColumnMap[searches[i].GetField()]
		if !ok {
			return nil, fmt.Errorf("unknown search field %q", searches[i].GetField())
		}

		out = append(out, sq.And{sq.Expr(fmt.Sprintf("%s.%s @@ phraseto_tsquery(?)", tableAlias, col), searches[i].GetValue())})
	}

	return out, nil
}
