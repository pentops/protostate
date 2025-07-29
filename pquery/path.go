package pquery

import (
	"fmt"
	"strings"

	"github.com/pentops/j5/lib/j5reflect"
	"github.com/pentops/j5/lib/j5schema"
)

type Table interface {
	TableName() string
}

type NestedField struct {

	// The column containing the element JSONB
	RootColumn string

	// The path from the column root to the node
	Path Path

	// ValueColumn contains the value of the field directly in the table in
	// addition to nested within the JSONB data.
	ValueColumn *string
}

func (nf *NestedField) ProtoChild(name string) (*NestedField, error) {
	pathChild, err := nf.Path.Child(name)
	if err != nil {
		return nil, err
	}
	return &NestedField{
		RootColumn: nf.RootColumn,
		Path:       *pathChild,
	}, nil
}

func (nf *NestedField) Selector(inTable string) string {
	if nf.ValueColumn != nil {
		return fmt.Sprintf("%s.%s", inTable, *nf.ValueColumn)
	}
	if len(nf.Path.path) == 0 {
		return fmt.Sprintf("%s.%s", inTable, nf.RootColumn)
	}
	return fmt.Sprintf("%s.%s%s", inTable, nf.RootColumn, nf.Path.JSONBArrowPath())
}

type pathNode struct {
	name  string
	field *j5schema.ObjectProperty
}

type Path struct {
	root *j5schema.ObjectSchema
	path []pathNode

	leafField *j5schema.ObjectProperty
	leafOneof *j5schema.OneofSchema // If the leaf is a oneof, this is the oneof schema
}

func (pp Path) Root() *j5schema.ObjectSchema {
	return pp.root
}

func (pp Path) LeafField() *j5schema.ObjectProperty {
	return pp.leafField
}

func (pp Path) LeafOneof() *j5schema.OneofSchema {
	return pp.leafOneof
}

func (pp Path) Child(name string) (*Path, error) {
	if pp.leafField == nil {
		return nil, fmt.Errorf("child requires a message field")
	}

	var propSet j5schema.PropertySet

	switch st := pp.leafField.Schema.(type) {
	case *j5schema.ObjectField:
		propSet = st.ObjectSchema().Properties
	case *j5schema.OneofField:
		propSet = st.OneofSchema().Properties
	default:
		return nil, fmt.Errorf("child requires a message field")
	}

	field := propSet.ByJSONName(name)
	if field == nil {
		return nil, fmt.Errorf("field %s not found in message %s", name, pp.leafField.FullName())
	}
	return &Path{
		root: pp.root,
		path: append(pp.path, pathNode{
			name:  name,
			field: field,
		}),
	}, nil
}

// IDPath uniquely identifies the path within a specific root type context
func (pp Path) IDPath() string {
	return pp.pathNodeNames()
}

func (pp Path) DebugName() string {
	return fmt.Sprintf("%s:%s", pp.root.FullName(), pp.pathNodeNames())
}

func (pp Path) pathNodeNames() string {
	names := make([]string, 0, len(pp.path))
	for _, node := range pp.path {
		names = append(names, string(node.name))
	}
	return strings.Join(names, ".")
}

func (pp *Path) pathParts() []string {
	parts := make([]string, 0, len(pp.path))
	for _, node := range pp.path {
		parts = append(parts, node.field.JSONName)
	}
	return parts
}

func (pp *Path) JSONBArrowPath() string {
	elements := make([]string, 0, len(pp.path))
	end, path := pp.path[len(pp.path)-1], pp.path[:len(pp.path)-1]
	for _, part := range path {
		if obj, ok := part.field.Schema.(*j5schema.ObjectField); ok {
			if obj.Flatten {
				continue // Flattened fields are not in the JSONB tree
			}
		}
		if part.field == nil {
			panic(fmt.Sprintf("invalid path: %v", pp.DebugName()))
		}

		switch part.field.Schema.(type) {
		case *j5schema.MapField:
			panic("map fields not supported by JSONBArrowPath()")
		case *j5schema.ArrayField:
			panic("array fields not supported by JSONBArrowPath()")
		}
		elements = append(elements, fmt.Sprintf("->'%s'", part.field.JSONName))
	}

	return fmt.Sprintf("%s->>'%s'", strings.Join(elements, ""), end.field.JSONName)
}

func (pp *Path) JSONPathQuery() string {
	elements := make([]string, 1, len(pp.path)+1)
	elements[0] = "$" // Used sometimes, not always?
	for _, part := range pp.path {
		if obj, ok := part.field.Schema.(*j5schema.ObjectField); ok {
			if obj.Flatten {
				continue // Flattened fields are not in the JSONB tree
			}
		}
		if part.field == nil {
			panic(fmt.Sprintf("invalid path: %v", pp.DebugName()))
		}
		elements = append(elements, fmt.Sprintf(".%s", part.field.JSONName))
		switch part.field.Schema.(type) {
		case *j5schema.MapField:
			panic("map fields not supported by JSONBArrowPath()")
		case *j5schema.ArrayField:
			elements = append(elements, "[*]")
		}
	}

	if pp.leafOneof != nil {
		// Can't use quotes, must escape the !
		elements = append(elements, `.\!type`)
	}

	return strings.Join(elements, "")
}

// WalkPathNodes visits every field in the message tree other than the root
// message itself, calling the callback for each.
func WalkPathNodes(rootMessage *j5schema.ObjectSchema, callback func(Path) error) error {
	root := &Path{
		root: rootMessage,
	}
	return root.walk(rootMessage.Properties, callback)
}

func (pp Path) walk(props j5schema.PropertySet, callback func(Path) error) error {
	// walks only fields, not oneofs.
	for _, field := range props {
		fieldPath := append(pp.path, pathNode{
			name:  field.JSONName,
			field: field,
		})
		fieldPathSpec := Path{
			root:      pp.root,
			path:      fieldPath,
			leafField: field,
		}

		if err := callback(fieldPathSpec); err != nil {
			return err
		}

		switch ft := field.Schema.(type) {
		case *j5schema.ObjectField:
			if err := fieldPathSpec.walk(ft.ObjectSchema().Properties, callback); err != nil {
				return fmt.Errorf("walking %s: %w", field.JSONName, err)
			}

		case *j5schema.OneofField:
			if err := fieldPathSpec.walk(ft.OneofSchema().Properties, callback); err != nil {
				return fmt.Errorf("walking %s: %w", field.JSONName, err)
			}
		}
	}

	return nil

}

// Like ProtoPathSpec but uses JSON field names
type JSONPathSpec []string

func JSONPath(path ...string) JSONPathSpec {
	return JSONPathSpec(path)
}

func ParseJSONPathSpec(path string) JSONPathSpec {
	return JSONPathSpec(strings.Split(path, "."))
}

func (jp JSONPathSpec) String() string {
	return strings.Join(jp, ".")
}

func NewJSONPath(rootMessage *j5schema.ObjectSchema, fieldPath JSONPathSpec) (*Path, error) {
	return newPath(rootMessage, []string(fieldPath), clientPath)
}

func NewOuterPath(rootMessage *j5schema.ObjectSchema, fieldPath JSONPathSpec) (*Path, error) {
	return newPath(rootMessage, []string(fieldPath), outerPath)
}

type pathType int

const (
	clientPath pathType = iota // the JSON path from the client side, flattened fields are anonymous
	outerPath                  // flattened fields included with their name
)

type hasPropertySet interface {
	AllProperties() j5schema.PropertySet
	ClientProperties() j5schema.PropertySet
	FullName() string
}

func objectProperty(obj hasPropertySet, pathElem string, pt pathType) (*j5schema.ObjectProperty, error) {
	switch pt {
	case clientPath:
		prop := obj.ClientProperties().ByJSONName(pathElem)
		if prop == nil {
			return nil, fmt.Errorf("client property %q not found in %s", pathElem, obj.FullName())
		}
		return prop, nil
	case outerPath:
		prop := obj.AllProperties().ByJSONName(pathElem)
		if prop == nil {
			return nil, fmt.Errorf("property %q not found in %s", pathElem, obj.FullName())
		}
		return prop, nil

	default:
		return nil, fmt.Errorf("unknown path type %d", pt)
	}

}

func newPath(rootMessage *j5schema.ObjectSchema, fieldPath []string, pathType pathType) (*Path, error) {

	if len(fieldPath) == 0 {
		return nil, fmt.Errorf("fieldPath must have at least one element")
	}

	pathSpec := &Path{
		root: rootMessage,
		path: make([]pathNode, 0, len(fieldPath)),
	}

	var walkMessage j5schema.RootSchema
	walkMessage = rootMessage
	var pathElem string
	walkPath := fieldPath

	for {
		pathElem, walkPath = walkPath[0], walkPath[1:]
		var node pathNode
		switch walkMessage := walkMessage.(type) {
		case *j5schema.ObjectSchema:
			field, err := objectProperty(walkMessage, pathElem, pathType)
			if err != nil {
				return nil, err
			}
			node = pathNode{
				name:  pathElem,
				field: field,
			}

		case *j5schema.OneofSchema:

			if len(walkPath) == 0 && pathElem == "!type" {
				// Very Special Edge Case: Oneof wrapper types allow the client to
				// filter based on the type of the oneof. So the oneof can be at the
				// end of the path, and the field can be the oneof wrapper type.
				pathSpec.leafOneof = walkMessage
				return pathSpec, nil
			}

			field, err := objectProperty(walkMessage, pathElem, pathType)
			if err != nil {
				return nil, err
			}

			node = pathNode{
				name:  pathElem,
				field: field,
			}
		default:
			return nil, fmt.Errorf("field %q in %s.%s is not an object or oneof, no such field %q", pathElem, rootMessage.FullName(), strings.Join(fieldPath, "."), strings.Join(walkPath, "."))
		}

		pathSpec.path = append(pathSpec.path, node)
		pathSpec.leafField = node.field

		if len(walkPath) == 0 {
			return pathSpec, nil
		}

		switch nextType := node.field.Schema.(type) {
		case *j5schema.ObjectField:
			walkMessage = nextType.ObjectSchema()
		case *j5schema.OneofField:
			walkMessage = nextType.OneofSchema()
		case j5schema.RootSchema:
			walkMessage = nextType
		case *j5schema.ArrayField:
			items := nextType.ItemSchema
			switch items := items.(type) {
			case *j5schema.ObjectField:
				walkMessage = items.ObjectSchema()
			case *j5schema.OneofField:
				walkMessage = items.OneofSchema()
			default:
				return nil, fmt.Errorf("array field %q in %s.%s is an array, but not of an object or oneof, no such field %q", pathElem, rootMessage.FullName(), strings.Join(fieldPath, "."), strings.Join(walkPath, "."))
			}
		case *j5schema.MapField:
			return nil, fmt.Errorf("map fields not supported in path %q", strings.Join(fieldPath, "."))
		default:
			return nil, fmt.Errorf("field %q in %s.%s is a scalar, no such field %q", pathElem, rootMessage.FullName(), strings.Join(fieldPath, "."), strings.Join(walkPath, "."))
		}
	}

}

func (pp *Path) GetValue(msg j5reflect.Object) (any, bool, error) {
	if len(pp.path) == 0 {
		return nil, false, fmt.Errorf("empty path")
	}

	lastField, _, err := msg.GetField(pp.pathParts()...)
	if err != nil {
		return nil, false, fmt.Errorf("getting field %s: %w", pp.pathNodeNames(), err)
	}

	if pp.leafOneof != nil {
		// this is a type filter.
		oneof, ok := lastField.AsOneof()
		if !ok {
			return nil, false, fmt.Errorf("field %s is not a oneof", pp.pathNodeNames())
		}

		field, ok, err := oneof.GetOne()
		if err != nil {
			return nil, false, fmt.Errorf("getting oneof field %s: %w", pp.pathNodeNames(), err)
		}
		if !ok {
			return nil, false, nil // No value set in the oneof
		}

		val := field.NameInParent()
		return val, true, nil
	}

	scalar, ok := lastField.AsScalar()
	if !ok {
		return nil, false, fmt.Errorf("field %s is not a scalar", pp.pathNodeNames())
	}

	goVal, err := scalar.ToGoValue()
	if err != nil {
		return nil, false, fmt.Errorf("converting field %s to Go value: %w", pp.pathNodeNames(), err)
	}
	if goVal == nil {
		return nil, false, nil // No value set in the scalar
	}
	return goVal, true, nil
}
