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
			}, {
				Name:     proto.String("bars"),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".test.Bar"),
				Number:   proto.Int32(2),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
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
		}, {
			Name: proto.String("Bar"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:   proto.String("bar_id"),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(1),
			}, {
				Name:   proto.String("foo_id"),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(2),
			}, {
				Name:   proto.String("name"),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(3),
			}},
		}},
	}

	file, err := protodesc.NewFile(fileDescriptor, protoregistry.GlobalFiles)
	if err != nil {
		t.Fatalf("Compiling test proto: %s", err.Error())
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

		ArrayJoin: &ArrayJoinSpec{
			TableName:     "bar",
			FieldInParent: "bars",
			DataColumn:    "state",
			On: []JoinField{{
				JoinColumn: "foo_id",
				RootColumn: "foo_id",
			}},
		},
	}

	gg, err := NewGetter(spec)
	if err != nil {
		t.Fatal(err.Error())
	}
	gg.SetQueryLogger(testLog(t))

	ctx := context.Background()
	conn := pgtest.GetTestDB(t, pgtest.WithSchemaName("pquery"))

	db := sqrlx.NewPostgres(conn)

	execAll(t, conn,
		"CREATE TABLE foo (foo_id text PRIMARY KEY, state jsonb)",
		`INSERT INTO foo (foo_id, state) VALUES 
		( 'id0', '{"foo_id": "id0", "name": "foo0"}'),
		( 'id1', '{"foo_id": "id1", "name": "foo1"}')
		`,

		`CREATE TABLE bar (bar_id text PRIMARY KEY, foo_id text REFERENCES foo(foo_id), state jsonb)`,
		`INSERT INTO bar (bar_id, foo_id, state) VALUES
		( 'bar0', 'id0', '{"bar_id": "bar0", "foo_id": "id0", "name": "bar0"}'),
		( 'bar1', 'id0', '{"bar_id": "bar1", "foo_id": "id1", "name": "bar1"}')
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

	bars := resMsg.GetField(t, "bars").List()
	if bars.Len() != 2 {
		t.Fatalf("expected 2 bars, got %d", bars.Len())
	}

}

func newDynamicMessage(md protoreflect.MessageDescriptor) *tDynamicMessage {
	return &tDynamicMessage{dynamicpb.NewMessage(md)}
}

func testLog(t *testing.T) QueryLogger {
	return func(qq sqrlx.Sqlizer) {
		stmt, args, err := qq.ToSql()
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Logf("Query: %s, args: %v", stmt, args)

	}
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
		if msg == nil {
			t.Fatalf("field %s: msg is nil", p)
		}
	}
	fieldDef := msg.Descriptor().Fields().ByName(protoreflect.Name(last))
	if fieldDef == nil {
		t.Fatalf("field %s: no such field", name)
	}
	field := msg.Get(fieldDef)
	if !field.IsValid() {
		t.Errorf("field %s is not valid", name)
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
