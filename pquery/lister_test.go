package pquery

import (
	"fmt"
	"strings"
	"testing"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/pentops/protostate/gen/list/v1/psml_pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func ptr[T any](s T) *T {
	return &s
}

type TestFoo struct {
	ListFoosRequest  *descriptorpb.DescriptorProto
	Foo              *descriptorpb.DescriptorProto
	Bar              *descriptorpb.DescriptorProto
	ListFoosResponse *descriptorpb.DescriptorProto
}

func NewTestFoo() *TestFoo {
	tf := &TestFoo{
		ListFoosRequest: &descriptorpb.DescriptorProto{
			Name: ptr("ListFoosRequest"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     ptr("page"),
				Number:   ptr(int32(1)),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: ptr(".psm.list.v1.PageRequest"),
				Options:  &descriptorpb.FieldOptions{},
			}, {
				Name:     ptr("query"),
				Number:   ptr(int32(2)),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: ptr(".psm.list.v1.QueryRequest"),
				Options:  &descriptorpb.FieldOptions{},
			}},
			Options: &descriptorpb.MessageOptions{},
		},
		Foo: &descriptorpb.DescriptorProto{
			Name: ptr("Foo"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:    ptr("id"),
				Number:  ptr(int32(1)),
				Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Options: &descriptorpb.FieldOptions{},
			}, {
				Name:     ptr("bar"),
				Number:   ptr(int32(2)),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: ptr(".test.Bar"),
				Options:  &descriptorpb.FieldOptions{},
			}},
			Options: &descriptorpb.MessageOptions{},
		},
		Bar: &descriptorpb.DescriptorProto{
			Name: ptr("Bar"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:    ptr("id"),
				Number:  ptr(int32(1)),
				Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Options: &descriptorpb.FieldOptions{},
			}},
			Options: &descriptorpb.MessageOptions{},
		},
		ListFoosResponse: &descriptorpb.DescriptorProto{
			Name: ptr("ListFoosResponse"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     ptr("foos"),
				Number:   ptr(int32(1)),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: ptr(".test.Foo"),
				Options:  &descriptorpb.FieldOptions{},
			}, {
				Name:     ptr("page"),
				Number:   ptr(int32(2)),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: ptr(".psm.list.v1.PageResponse"),
				Options:  &descriptorpb.FieldOptions{},
			}},
			Options: &descriptorpb.MessageOptions{},
		},
	}

	tf.SetListRequest(&psml_pb.ListRequestMessage{
		SortTiebreaker: []string{"id"},
	})

	return tf
}

func (tf *TestFoo) FileDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    ptr("test.proto"),
		Package: ptr("test"),
		Dependency: []string{
			"psm/list/v1/page.proto",
			"psm/list/v1/query.proto",
		},
		MessageType: []*descriptorpb.DescriptorProto{
			tf.ListFoosRequest,
			tf.Foo,
			tf.Bar,
			tf.ListFoosResponse,
		},
	}
}

func (tf *TestFoo) SetListRequest(listRequest *psml_pb.ListRequestMessage) {
	proto.SetExtension(tf.ListFoosRequest.Options, psml_pb.E_ListRequest, listRequest)
}

func TestFieldPath(t *testing.T) {

	tf := NewTestFoo()
	pdf, err := protodesc.NewFile(tf.FileDescriptor(), protoregistry.GlobalFiles)
	if err != nil {
		t.Fatal(fmt.Errorf("converting to descriptor: %w", err))
	}

	fooDesc := pdf.Messages().ByName("Foo")

	for _, tc := range []struct {
		name string
		path []string
	}{
		{
			name: "id",
			path: []string{"id"},
		},
		{
			name: "bar.id",
			path: []string{"bar", "id"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field, err := findField(fooDesc, tc.name)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.path, field.jsonPath)
		})
	}

}

func TestBuildListReflection(t *testing.T) {

	for _, tc := range []struct {
		name      string
		mods      func(*TestFoo)
		options   []ListerOption
		wantError string
		assert    func(*testing.T, *ListReflectionSet)
	}{{
		name: "full success",
		mods: func(tf *TestFoo) {},
		assert: func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 1 {
				t.Error("expected one sort tiebreaker")
			} else {
				field := lr.tieBreakerFields[0]
				assert.Equal(t, "id", field.jsonbPath())
			}
			assert.Equal(t, uint64(20), lr.pageSize)
		},
	}, {
		name: "override page size by validation",
		mods: func(tf *TestFoo) {
			proto.SetExtension(tf.ListFoosResponse.Field[0].Options, validate.E_Field, &validate.FieldConstraints{
				Type: &validate.FieldConstraints_Repeated{
					Repeated: &validate.RepeatedRules{
						MaxItems: ptr(uint64(10)),
					},
				},
			})
		},
		assert: func(t *testing.T, lr *ListReflectionSet) {
			assert.EqualValues(t, 10, lr.pageSize)
		},
	}, {
		name: "non message field in response",
		mods: func(tf *TestFoo) {
			tf.ListFoosResponse.Field = append(tf.ListFoosResponse.Field, &descriptorpb.FieldDescriptorProto{
				Name:     ptr("dangling"),
				Number:   ptr(int32(3)),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: ptr(".test.Foo"),
			})
		},
		wantError: "unknown field",
	}, {
		name: "non repeated field in response",
		mods: func(tf *TestFoo) {
			tf.ListFoosResponse.Field = append(tf.ListFoosResponse.Field, &descriptorpb.FieldDescriptorProto{
				Name:   ptr("dangling"),
				Number: ptr(int32(3)),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			})
		},
		wantError: "should be a message",
	}, {
		name: "extra array field in response",
		mods: func(tf *TestFoo) {

			tf.ListFoosResponse.Field = append(tf.ListFoosResponse.Field, &descriptorpb.FieldDescriptorProto{
				Name:     ptr("double"),
				Number:   ptr(int32(3)),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: ptr(".test.Foo"),
			})
		},
		wantError: "multiple repeated fields",
	}, {
		name: "no array field",
		mods: func(tf *TestFoo) {
			tf.ListFoosResponse.Field = tf.ListFoosResponse.Field[1:]
		},
		wantError: "no repeated field in response",
	}, {
		name: "no page field",
		mods: func(tf *TestFoo) {
			tf.ListFoosResponse.Field = tf.ListFoosResponse.Field[0:1]
		},
		wantError: "no page field in response",
	}, {
		name: "no sort or tie breaker",
		mods: func(tf *TestFoo) {
			tf.SetListRequest(&psml_pb.ListRequestMessage{})
		},
		wantError: "no default sort field",
	}, {
		name: "tie breaker fallback",
		mods: func(tf *TestFoo) {
			tf.SetListRequest(&psml_pb.ListRequestMessage{})
		},
		options: []ListerOption{
			WithTieBreakerFields("id"),
		},
		assert: func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 1 {
				t.Error("expected one sort tiebreaker")
			} else {
				field := lr.tieBreakerFields[0]
				assert.Equal(t, "id", field.jsonbPath())
			}
		},
	}, {
		name: "unknown tie breaker",
		mods: func(tf *TestFoo) {
			tf.SetListRequest(&psml_pb.ListRequestMessage{
				SortTiebreaker: []string{"unknown"},
			})
		},
		wantError: "no field named 'unknown'",
	}, {
		name: "sort by bar",
		mods: func(tf *TestFoo) {
			tf.SetListRequest(&psml_pb.ListRequestMessage{
				SortTiebreaker: []string{"bar.id"},
			})
		},
		assert: func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 1 {
				t.Error("expected one sort tiebreaker")
			} else {
				field := lr.tieBreakerFields[0]
				assert.Equal(t, "bar->id", field.jsonbPath())
			}
		},
	}} {

		t.Run(tc.name, func(t *testing.T) {

			tf := NewTestFoo()
			tc.mods(tf)
			pdf, err := protodesc.NewFile(tf.FileDescriptor(), protoregistry.GlobalFiles)
			if err != nil {
				t.Fatal(fmt.Errorf("converting to descriptor: %w", err))
			}
			responseDesc := pdf.Messages().ByName("ListFoosResponse")
			requestDesc := pdf.Messages().ByName("ListFoosRequest")

			options := resolveListerOptions(tc.options)
			listReflection, err := buildListReflection(requestDesc, responseDesc, options)

			if tc.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				msg := err.Error()
				if !strings.Contains(msg, tc.wantError) {
					t.Errorf("expected error to contain '%s', got '%s'", tc.wantError, msg)
				}
				return
			}

			tc.assert(t, listReflection)

		})
	}

}
