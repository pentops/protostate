package pquery

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestGetter(t *testing.T) {

	fileDescriptor := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("TestRequest"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:   proto.String("foo_id"),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(1),
			}},
		}, {
			Name: proto.String("TestResponse"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     proto.String("state"),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".test.Foo"),
				Number:   proto.Int32(1),
			}},
		}, {
			Name: proto.String("Foo"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:   proto.String("foo_id"),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(1),
			}, {
				Name:   proto.String("name"),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(2),
			}},
		}},
	}

	file, err := protodesc.NewFile(fileDescriptor, protoregistry.GlobalFiles)
	if err != nil {
		t.Fatal(err.Error())
	}

	reqDesc := file.Messages().ByName("TestRequest")
	if reqDesc == nil {
		t.Fatal("reqDesc is nil")
	}

	resDesc := file.Messages().ByName("TestResponse")
	if resDesc == nil {
		t.Fatal("resDesc is nil")
	}

	spec := GetSpec[*tDynamicMessage, *tDynamicMessage]{
		ResponseDescriptor: resDesc,
		DataColumn:         "state",
		TableName:          "foo",
		PrimaryKey: func(req *tDynamicMessage) (map[string]interface{}, error) {
			return map[string]interface{}{
				"foo_id": req.GetField(t, "foo_id").String(),
			}, nil
		},
	}

	gg, err := NewGetter(spec)
	if err != nil {
		t.Fatal(err.Error())
	}

	ctx := context.Background()
	conn := pgtest.GetTestDB(t, pgtest.WithSchemaName("pquery"))

	db := sqrlx.NewPostgres(conn)

	execAll(t, conn,
		"CREATE TABLE foo (foo_id text PRIMARY KEY, state jsonb)",
		`INSERT INTO foo (foo_id, state) VALUES 
		( 'id0', '{"foo_id": "id0", "name": "foo0"}'),
		( 'id1', '{"foo_id": "id1", "name": "foo1"}')
		`,
	)

	reqMsg := newDynamicMessage(reqDesc)
	reqMsg.SetField(t, "foo_id", protoreflect.ValueOf("id0"))

	resMsg := newDynamicMessage(resDesc)
	err = gg.Get(ctx, db, reqMsg, resMsg)
	if err != nil {
		t.Fatal(err.Error())
	}

	resMsg.AssertField(t, "state.foo_id", protoreflect.ValueOf("id0"))
	resMsg.AssertField(t, "state.name", protoreflect.ValueOf("foo0"))
}

func newDynamicMessage(md protoreflect.MessageDescriptor) *tDynamicMessage {
	return &tDynamicMessage{dynamicpb.NewMessage(md)}
}

type tDynamicMessage struct {
	*dynamicpb.Message
}

func (dm *tDynamicMessage) GetField(t *testing.T, name string) protoreflect.Value {
	path := strings.Split(name, ".")
	pathTo, last := path[:len(path)-1], path[len(path)-1]
	var msg protoreflect.Message = dm.Message
	for _, p := range pathTo {
		field := msg.Get(msg.Descriptor().Fields().ByName(protoreflect.Name(p)))
		if !field.IsValid() {
			t.Errorf("field %s is not valid", p)
		}
		msg = field.Message()
	}
	field := msg.Get(msg.Descriptor().Fields().ByName(protoreflect.Name(last)))
	if !field.IsValid() {
		t.Errorf("field %s is not valid", last)
	}
	return field
}

func (dm *tDynamicMessage) SetField(t *testing.T, name string, value protoreflect.Value) {
	path := strings.Split(name, ".")
	pathTo, last := path[:len(path)-1], path[len(path)-1]
	var msg protoreflect.Message = dm.Message
	for _, p := range pathTo {
		field := msg.Get(msg.Descriptor().Fields().ByName(protoreflect.Name(p)))
		if field.IsValid() {
			msg = field.Message()
			continue
		}
		field = protoreflect.ValueOf(msg.NewField(msg.Descriptor().Fields().ByName(protoreflect.Name(p))))
		msg.Set(msg.Descriptor().Fields().ByName(protoreflect.Name(p)), field)
		t.Errorf("field %s is not valid", p)
	}

	msg.Set(msg.Descriptor().Fields().ByName(protoreflect.Name(last)), value)
}

func (dm *tDynamicMessage) AssertField(t *testing.T, name string, value protoreflect.Value) {
	field := dm.GetField(t, name)
	if !field.Equal(value) {
		t.Errorf("expected %v, got %v", value, field)
	}
}

func execAll(t *testing.T, conn *sql.DB, queries ...string) {
	for _, query := range queries {
		_, err := conn.Exec(query)
		if err != nil {
			t.Fatal(err.Error())
		}
	}
}
