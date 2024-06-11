package pgstore

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type Table interface {
	TableName() string
}

type NestedField struct {

	// The column containing the element JSONB
	RootColumn string

	// The path from the column root to the node
	Path Path
}

type ProtoFieldSpec struct {
	// The column holding the data, either directly or JSONB from here using the
	// path
	ColumnName string

	// The path from the column root to the node being specified (not the path
	// of the node from a 'root node' of the table)
	Path ProtoPathSpec

	// The path from the nominal table root message to the node being specified
	PathFromRoot ProtoPathSpec
}

func (nf *NestedField) ProtoChild(name protoreflect.Name) (*NestedField, error) {
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
	if len(nf.Path.path) == 0 {
		return fmt.Sprintf("%s.%s", inTable, nf.RootColumn)
	}
	return fmt.Sprintf("%s.%s%s", inTable, nf.RootColumn, nf.Path.JSONBArrowPath())
}

type pathNode struct {
	name  protoreflect.Name
	field protoreflect.FieldDescriptor
	oneof protoreflect.OneofDescriptor
}

type Path struct {
	root protoreflect.MessageDescriptor
	path []pathNode

	leafField protoreflect.FieldDescriptor
	leafOneof protoreflect.OneofDescriptor
}

func (pp Path) Root() protoreflect.MessageDescriptor {
	return pp.root
}

func (pp Path) LeafField() protoreflect.FieldDescriptor {
	return pp.leafField
}

func (pp Path) LeafOneof() protoreflect.OneofDescriptor {
	return pp.leafOneof
}

func (pp Path) Leaf() protoreflect.Descriptor {
	if pp.leafField != nil {
		return pp.leafField
	}
	return pp.leafOneof
}

func (pp Path) Child(name protoreflect.Name) (*Path, error) {
	if pp.leafField == nil {
		return nil, fmt.Errorf("child requires a message field")
	}
	if pp.leafField.Kind() != protoreflect.MessageKind {
		return nil, fmt.Errorf("child requires a message field")
	}
	field := pp.leafField.Message().Fields().ByName(name)
	if field == nil {
		return nil, fmt.Errorf("field %s not found in message %s", name, pp.leafField.Message().FullName())
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

func (pp *Path) JSONBArrowPath() string {
	elements := make([]string, 0, len(pp.path))
	end, path := pp.path[len(pp.path)-1], pp.path[:len(pp.path)-1]
	for _, part := range path {
		if part.oneof != nil {
			continue // Ignore the node, it isn't in the JSONB tree
		}
		if part.field.IsList() {
			panic("list fields not supported by JSONBArrowPath()")
		}
		elements = append(elements, fmt.Sprintf("->'%s'", part.field.JSONName()))
	}

	return fmt.Sprintf("%s->>'%s'", strings.Join(elements, ""), end.field.JSONName())
}

func (pp *Path) JSONPathQuery() string {
	elements := make([]string, 1, len(pp.path)+1)
	elements[0] = "$" // Used sometimes, not always?
	for _, part := range pp.path {
		if part.oneof != nil {
			continue // Ignore the node, it isn't in the JSONB tree
		}
		if part.field == nil {
			panic(fmt.Sprintf("invalid path: %v", pp.DebugName()))
		}
		elements = append(elements, fmt.Sprintf(".%s", part.field.JSONName()))
		if part.field.IsList() {
			elements = append(elements, "[*]")
		}
	}

	return strings.Join(elements, "")
}

// WalkPathNodes visits every field in the message tree other than the root
// message itself, calling the callback for each.
func WalkPathNodes(rootMessage protoreflect.MessageDescriptor, callback func(Path) error) error {
	root := &Path{
		root: rootMessage,
	}
	return root.walk(rootMessage, callback)
}

func (pp Path) walk(msg protoreflect.MessageDescriptor, callback func(Path) error) error {
	fields := msg.Fields()
	// walks only fields, not oneofs.
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldPath := append(pp.path, pathNode{
			name:  field.Name(),
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

		if field.Kind() != protoreflect.MessageKind {
			continue
		}
		if err := fieldPathSpec.walk(field.Message(), callback); err != nil {
			return fmt.Errorf("walking %s: %w", field.Name(), err)
		}
	}

	return nil

}

// An element in a path from a root message to a leaf node
// Messages use field name strings
// Repeated uses index numbes as strings (1 = "1")
// Maps use the map keys, which are always strings in J5 land.
// OneOf is not included as it doesn't appear in the proto tree
type ProtoPathSpec []string

func ParseProtoPathSpec(path string) ProtoPathSpec {
	return ProtoPathSpec(strings.Split(path, "."))
}

func (pp ProtoPathSpec) String() string {
	return strings.Join(pp, ".")
}

func NewProtoPath(message protoreflect.MessageDescriptor, fieldPath ProtoPathSpec) (*Path, error) {

	if len(fieldPath) == 0 {
		return nil, fmt.Errorf("fieldPath must have at least one element")
	}

	pathSpec := &Path{
		root: message,
		path: make([]pathNode, 0, len(fieldPath)),
	}

	walkMessage := message
	var pathElem string
	walkPath := fieldPath
	var inOneof *int

	for {

		pathElem, walkPath = walkPath[0], walkPath[1:]
		node := pathNode{
			name: protoreflect.Name(pathElem),
		}
		field := walkMessage.Fields().ByName(protoreflect.Name(pathElem))
		if field != nil {
			node.field = field
			// Check that it isn't a oneof.
			if inOneof != nil {
				if field.ContainingOneof().Index() != *inOneof {
					return nil, fmt.Errorf("field %s not in oneof %d", pathElem, *inOneof)
				}
				inOneof = nil
			}
		} else {
			if inOneof != nil {
				return nil, fmt.Errorf("field %s not found in oneof %d", pathElem, *inOneof)
			}
			// Oneof needn't be specified as it doens't appear in the node tree,
			// but if a oneof is named, this adds it as a node in the tree,
			// hoping that the next element will be an actual field.
			oneof := walkMessage.Oneofs().ByName(protoreflect.Name(pathElem))
			if oneof != nil {
				node.oneof = oneof
			} else {
				return nil, fmt.Errorf("no field named '%s' in message %s", pathElem, walkMessage.FullName())
			}
			idx := oneof.Index()
			inOneof = &idx

			pathSpec.path = append(pathSpec.path, node)
			if len(walkPath) == 0 {
				pathSpec.leafOneof = node.oneof
				break
			}
			continue
		}

		if field.IsMap() {
			return nil, fmt.Errorf("unimplemented: map fields in path spec")
		}

		pathSpec.path = append(pathSpec.path, node)

		if len(walkPath) == 0 {
			pathSpec.leafField = node.field
			break
		}

		if field.Kind() != protoreflect.MessageKind {
			return nil, fmt.Errorf("field %s is not a message, but path elements remain (%v)", pathElem, walkPath)
		}

		walkMessage = field.Message()

	}

	return pathSpec, nil
}

// Like ProtoPathSpec but uses JSON field names
type JSONPathSpec []string

func ParseJSONPathSpec(path string) JSONPathSpec {
	return JSONPathSpec(strings.Split(path, "."))
}

func (jp JSONPathSpec) String() string {
	return strings.Join(jp, ".")
}

func NewJSONPath(message protoreflect.MessageDescriptor, fieldPath JSONPathSpec) (*Path, error) {

	if len(fieldPath) == 0 {
		return nil, fmt.Errorf("fieldPath must have at least one element")
	}

	pathSpec := &Path{
		root: message,
		path: make([]pathNode, 0, len(fieldPath)),
	}

	walkMessage := message
	var pathElem string
	walkPath := fieldPath

	for {

		pathElem, walkPath = walkPath[0], walkPath[1:]
		node := pathNode{
			name: protoreflect.Name(pathElem),
		}
		field := walkMessage.Fields().ByJSONName(pathElem)
		if field == nil {

			// Very Special Edge Case: Oneof wrapper types allow the client to
			// filter based on the type of the oneof. So the oneof can be at the
			// end of the path, and the field can be the oneof wrapper type.

			if len(walkPath) == 0 && pathElem == "type" {
				oneof := walkMessage.Oneofs().ByName(protoreflect.Name("type"))
				if oneof != nil {
					node.oneof = oneof
					pathSpec.path = append(pathSpec.path, node)
					pathSpec.leafOneof = oneof
					break
				}
			}

			return nil, fmt.Errorf("JSON field '%s' not found in message %s", pathElem, walkMessage.FullName())
		}

		node.field = field

		if field.IsMap() {
			return nil, fmt.Errorf("unimplemented: map fields in path spec")
		}

		pathSpec.path = append(pathSpec.path, node)
		if len(walkPath) == 0 {
			pathSpec.leafField = node.field
			break
		}

		if field.Kind() != protoreflect.MessageKind {
			return nil, fmt.Errorf("field %s is not a message, but path elements remain", pathElem)
		}

		walkMessage = field.Message()

	}

	return pathSpec, nil
}

func (pp *Path) GetValue(msg protoreflect.Message) (protoreflect.Value, error) {
	if len(pp.path) == 0 {
		return protoreflect.Value{}, fmt.Errorf("empty path")
	}
	var val protoreflect.Value

	var walkNode pathNode
	walkMessage := msg

	remainingPath := pp.path
	for {
		walkNode, remainingPath = remainingPath[0], remainingPath[1:]
		if walkNode.oneof != nil {
			// ignore the oneof
			if len(remainingPath) == 0 {
				return protoreflect.Value{}, fmt.Errorf("oneof at leaf")
			}
			continue
		}

		if walkNode.field == nil {
			return protoreflect.Value{}, fmt.Errorf("no field or oneof")
		}

		// Has vs Get, Has returns false if the field is set to the default value for
		// scalar types. We still want the fields if they are set to the default value,
		// and can use validation of a field's existence before this point to ensure
		// that the field is available.
		val = walkMessage.Get(walkNode.field)
		if len(remainingPath) == 0 {
			return val, nil
		}

		if walkNode.field.Kind() != protoreflect.MessageKind {
			return protoreflect.Value{}, fmt.Errorf("field %s is not a message", walkNode.field.Name())
		}
		walkMessage = val.Message()
	}
}
