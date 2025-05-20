package pquery

import (
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/protostate/internal/pgstore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"maps"
)

func validateSearchesAnnotations(msg protoreflect.FieldDescriptors) error {
	fields := make([]protoreflect.FieldDescriptor, msg.Len())
	for i := range msg.Len() {
		fields[i] = msg.Get(i)
	}
	_, err := validateSearchesAnnotationsInner(fields)
	if err != nil {
		return fmt.Errorf("search validation: %w", err)
	}
	return nil
}

func validateSearchesAnnotationsInner(fields []protoreflect.FieldDescriptor) (map[string]protoreflect.Name, error) {
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

	ids := make(map[string]protoreflect.Name)
	oneofs := map[string][]protoreflect.FieldDescriptor{}

	for _, field := range fields {

		if oneof := field.ContainingOneof(); oneof != nil {
			name := string(oneof.Name())
			oneofs[name] = append(oneofs[name], field)
			continue
		}

		err := validateSearchAnnotationsField(ids, field)
		if err != nil {
			return nil, err
		}
	}

	for _, oneofFields := range oneofs {

		// oneof fields can have the same field identifier on different branches but must be unique:
		//  - within the branch, as before
		//  - with existing parent keys
		//  - with other oneofs

		combinedBranchIDs := make(map[string]protoreflect.Name)

		for _, field := range oneofFields {

			// collect a new set of IDs for this branch as if it is the root of
			// the message.
			branchIDs := make(map[string]protoreflect.Name)
			err := validateSearchAnnotationsField(branchIDs, field)
			if err != nil {
				return nil, err
			}

			// Compare the branch IDs to the root IDs, if a branch conflicts
			// with the root that is an error.
			for searchKey, usedIn := range branchIDs {
				if existing, ok := ids[searchKey]; ok {
					return nil, fmt.Errorf("field identifier '%s' is used at %s and %s within the same oneof branch", searchKey, existing, usedIn)
				}

				// the latest wins here, this makes a worse error message but
				// doesn't actually effect the outcome.
				combinedBranchIDs[searchKey] = usedIn
			}

		}

		// The IDs inside each branch are unique, and unique with the parent keys.

		// We still need to ensure that the IDs are unique with other oneofs.
		maps.Copy(ids, combinedBranchIDs)

	}

	return ids, nil
}

func validateSearchAnnotationsField(ids map[string]protoreflect.Name, field protoreflect.FieldDescriptor) error {

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

}

func validateQueryRequestSearches(message protoreflect.MessageDescriptor, searches []*list_j5pb.Search) error {
	for _, search := range searches {

		spec, err := pgstore.NewJSONPath(message, pgstore.ParseJSONPathSpec(search.GetField()))
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

func buildTsvColumnMap(message protoreflect.MessageDescriptor) map[string]string {
	out := make(map[string]string)

	for i := range message.Fields().Len() {
		field := message.Fields().Get(i)

		switch field.Kind() {
		case protoreflect.StringKind:
			fieldOpts, ok := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), list_j5pb.E_Field).(*list_j5pb.FieldConstraint)
			if !ok {
				continue
			}

			switch fieldOpts.GetString_().GetWellKnown().(type) {
			case *list_j5pb.StringRules_OpenText:
				searchOpts := fieldOpts.GetString_().GetOpenText().GetSearching()
				if searchOpts == nil || !searchOpts.Searchable {
					continue
				}

				out[field.TextName()] = searchOpts.GetFieldIdentifier()
			}

			continue
		case protoreflect.MessageKind:
			nestedMap := buildTsvColumnMap(field.Message())

			for nk, nv := range nestedMap {
				k := fmt.Sprintf("%s.%s", field.TextName(), nk)
				out[k] = nv
			}
		}
	}

	return out
}

func (ll *Lister[REQ, RES]) buildDynamicSearches(tableAlias string, searches []*list_j5pb.Search) ([]sq.Sqlizer, error) {
	out := []sq.Sqlizer{}

	for i := range searches {
		col, ok := ll.tsvColumnMap[camelToSnake(searches[i].GetField())]
		if !ok {
			return nil, fmt.Errorf("unknown field name '%s'", searches[i].GetField())
		}

		out = append(out, sq.And{sq.Expr(fmt.Sprintf("%s.%s @@ phraseto_tsquery(?)", tableAlias, col), searches[i].GetValue())})
	}

	return out, nil
}
