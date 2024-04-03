package pquery

import (
	"fmt"
	"strings"

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

func buildTsvColumnMap(messageFields protoreflect.FieldDescriptors) (map[string]string, error) {
	out := make(map[string]string)

	for i := 0; i < messageFields.Len(); i++ {
		field := messageFields.Get(i)

		var colName string
		switch field.Kind() {
		case protoreflect.StringKind:
			fieldOpts, ok := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)
			if !ok {
				continue
			}

			switch fieldOpts.GetString_().GetWellKnown().(type) {
			case *psml_pb.StringRules_OpenText:
				searchOpts := fieldOpts.GetString_().GetOpenText().GetSearching()
				if searchOpts == nil {
					continue
				}

				if !searchOpts.Searchable {
					continue
				}

				colName = strings.ToLower(fmt.Sprintf("%s_%s", string(field.Parent().Name()), string(field.Name())))

				i := 1
				for {
					t := fmt.Sprintf("%s_%d", colName, i)
					if _, ok := out[t]; !ok {
						colName = t
						break
					}

					i++
				}

				//TODO: generate field accessor
				accessor := string(field.Name())

				out[accessor] = colName

			default:
				continue
			}
		case protoreflect.MessageKind:
			// recurse into nested messages
			// increment any conflicting fields from the nested message
			// append nested output to the current output
		default:
			continue
		}
	}

	return out, nil
}

func (ll *Lister[REQ, RES]) buildDynamicSearches(tableAlias string, searches []*psml_pb.Search) ([]sq.Sqlizer, error) {
	out := []sq.Sqlizer{}

	for i := range searches {
		col, ok := ll.tsvColumnMap[searches[i].GetField()]
		if !ok {
			return nil, fmt.Errorf("tsv column unknown for field name '%s'", searches[i].GetField())
		}

		fmt.Println("col: ", col)

		out = append(out, sq.And{sq.Expr(fmt.Sprintf("%s.%s @@ to_tsquery(?)", tableAlias, col), searches[i].GetValue())})
	}

	return out, nil
}
