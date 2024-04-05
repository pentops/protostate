package pquery

import (
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/protostate/gen/list/v1/psml_pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func validateQueryRequestSearches(message protoreflect.MessageDescriptor, searches []*psml_pb.Search) error {
	for _, search := range searches {
		// validate fields exist from the request query
		err := validateFieldName(message, search.GetField())
		if err != nil {
			return fmt.Errorf("field name: %w", err)
		}

		spec, err := findFieldSpec(message, search.GetField())
		if err != nil {
			return err
		}

		// validate the fields are annotated correctly for the request query
		searchOpts, ok := proto.GetExtension(spec.field.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)
		if !ok {
			return fmt.Errorf("requested search field '%s' does not have any searchable constraints defined", search.Field)
		}

		searchable := false
		if searchOpts != nil {
			switch spec.field.Kind() {
			case protoreflect.StringKind:
				switch searchOpts.GetString_().WellKnown.(type) {
				case *psml_pb.StringRules_OpenText:
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

func buildTsvColumnMap(message protoreflect.MessageDescriptor) (map[string]string, error) {
	out := make(map[string]string)

	for i := 0; i < message.Fields().Len(); i++ {
		field := message.Fields().Get(i)

		switch field.Kind() {
		case protoreflect.StringKind:
			fieldOpts, ok := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)
			if !ok {
				continue
			}

			switch fieldOpts.GetString_().GetWellKnown().(type) {
			case *psml_pb.StringRules_OpenText:
				searchOpts := fieldOpts.GetString_().GetOpenText().GetSearching()
				if searchOpts == nil || !searchOpts.Searchable {
					continue
				}

				if searchOpts.GetFieldIdentifier() == "" {
					return nil, fmt.Errorf("field '%s' is missing a field identifier", field.TextName())
				}

				out[field.TextName()] = searchOpts.GetFieldIdentifier()
			}

			continue
		case protoreflect.MessageKind:
			nestedMap, err := buildTsvColumnMap(field.Message())
			if err != nil {
				return nil, err
			}

			for nk, nv := range nestedMap {
				k := fmt.Sprintf("%s.%s", field.TextName(), nk)
				out[k] = nv
			}
		}
	}

	return out, nil
}

func (ll *Lister[REQ, RES]) buildDynamicSearches(tableAlias string, searches []*psml_pb.Search) ([]sq.Sqlizer, error) {
	out := []sq.Sqlizer{}

	for i := range searches {
		col, ok := ll.tsvColumnMap[searches[i].GetField()]
		if !ok {
			return nil, fmt.Errorf("unknown field name '%s'", searches[i].GetField())
		}

		out = append(out, sq.And{sq.Expr(fmt.Sprintf("%s.%s @@ phraseto_tsquery(?)", tableAlias, col), searches[i].GetValue())})
	}

	return out, nil
}
