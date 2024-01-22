package pquery

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type ResultSet struct {
	*protoregistry.Files
}

func (rs ResultSet) MessageByName(t testing.TB, name protoreflect.FullName) protoreflect.MessageDescriptor {
	t.Helper()
	md, err := rs.FindDescriptorByName(name)
	if err != nil {
		t.Fatal(err)
	}
	return md.(protoreflect.MessageDescriptor)
}
func DescriptorsFromSource(t testing.TB, sourceFiles map[string]string) ResultSet {
	t.Helper()

	parser := protoparse.Parser{
		ImportPaths:           []string{""},
		IncludeSourceCodeInfo: false,
		// Load everything which the runtime has already leaded, includes all
		// the google, buf, psm etc packages, so long as something which uses
		// them has already beein imported
		LookupImport: desc.LoadFileDescriptor,
		Accessor: func(filename string) (io.ReadCloser, error) {
			if content, ok := sourceFiles[filename]; ok {
				return io.NopCloser(strings.NewReader(content)), nil
			}
			return nil, fmt.Errorf("file not found: %s", filename)
		},
	}

	customDesc, err := parser.ParseFiles("test.proto")
	if err != nil {
		t.Fatal(err)
	}

	realDesc := desc.ToFileDescriptorSet(customDesc...)
	descFiles, err := protodesc.NewFiles(realDesc)
	if err != nil {
		t.Fatal(err)
	}

	return ResultSet{
		Files: descFiles,
	}
}

func TestFieldPath(t *testing.T) {
	descFiles := DescriptorsFromSource(t, map[string]string{
		"test.proto": `
		syntax = "proto3";

		import "psm/list/v1/page.proto";
		import "psm/list/v1/query.proto";

		package test;

		message Foo {
			string id = 1;
			Bar bar = 2;
		}

		message Bar {
			string id = 1;
		}
	`})

	fooDesc := descFiles.MessageByName(t, "test.Foo")

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
			namedFieldPath := make([]string, len(field.fieldPath))
			for i, field := range field.fieldPath {
				namedFieldPath[i] = string(field.Name())
			}
			assert.Equal(t, tc.path, namedFieldPath)

		})
	}

}

/*message ListFoosRequest {
	psm.list.v1.PageRequest page = 1;
	psm.list.v1.QueryRequest query = 2;

	option (psm.list.v1.list_request) = {
		sort_tiebreaker: ["id"]
	};
}

message Foo {
	string id = 1;
	Bar bar = 2;
}

message Bar {
	string id = 1;
}

message ListFoosResponse {
	repeated Foo foos = 1;
	psm.list.v1.PageResponse page = 2;
}*/

type composed struct {
	ListFoosRequest  string
	ListFoosResponse string
	Foo              string
}

func (c composed) toString() string {
	out := ""
	if c.ListFoosRequest != "" {
		out += "message ListFoosRequest {\n" + c.ListFoosRequest + "\n}\n"
	} else {
		out += `message ListFoosRequest {
				psm.list.v1.PageRequest page = 1;
				psm.list.v1.QueryRequest query = 2;

				option (psm.list.v1.list_request) = {
					sort_tiebreaker: ["id"]
				};
			}`
	}

	if c.ListFoosResponse != "" {
		out += "message ListFoosResponse {\n" + c.ListFoosResponse + "\n}\n"
	} else {
		out += `message ListFoosResponse {
				repeated Foo foos = 1;
				psm.list.v1.PageResponse page = 2;
			}`
	}

	if c.Foo != "" {
		out += "message Foo {\n" + c.Foo + "\n}\n"
	} else {
		out += `message Foo {
				string id = 1;
			}`
	}
	return out
}

func TestBuildListReflection(t *testing.T) {

	build := func(t testing.TB, input string, options listerOptions) (*ListReflectionSet, error) {
		pdf := DescriptorsFromSource(t, map[string]string{
			"test.proto": `
				syntax = "proto3";

				package test;

				// Import everything which may be used
				import "psm/list/v1/page.proto";
				import "psm/list/v1/query.proto";
				import "psm/list/v1/annotations.proto";
				import "buf/validate/validate.proto";
				import "google/protobuf/timestamp.proto";
				` + input,
		})

		responseDesc := pdf.MessageByName(t, "test.ListFoosResponse")
		requestDesc := pdf.MessageByName(t, "test.ListFoosRequest")

		return buildListReflection(requestDesc, responseDesc, options)
	}

	runHappy := func(name string, input string, options listerOptions, callback func(*testing.T, *ListReflectionSet)) {
		t.Run(name, func(t *testing.T) {
			set, err := build(t, input, options)
			if err != nil {
				t.Fatal(err)
			}
			callback(t, set)
		})
	}
	runSad := func(name string, input string, options listerOptions, wantError string) {
		t.Run(name, func(t *testing.T) {
			_, err := build(t, input, options)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), wantError) {
				t.Errorf("expected error to contain '%s', got '%s'", wantError, err.Error())
			}
		})
	}

	runHappy("simple", `
		message ListFoosRequest {
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;

			option (psm.list.v1.list_request) = {
				sort_tiebreaker: ["id"]
			};
		}

		message ListFoosResponse {
			repeated Foo foos = 1;
			psm.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
		}
		`,
		listerOptions{},
		func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 1 {
				t.Error("expected one sort tiebreaker")
			} else {
				field := lr.tieBreakerFields[0]
				assert.Equal(t, "id", field.jsonbPath())
			}
			assert.Equal(t, uint64(20), lr.pageSize)
		})

	runHappy("override page size", `
		message ListFoosRequest {
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;

			option (psm.list.v1.list_request) = {
				sort_tiebreaker: ["id"]
			};
		}

		message ListFoosResponse {
			repeated Foo foos = 1 [
				(buf.validate.field).repeated.max_items = 10
			];
			psm.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
		}
		`,
		listerOptions{},
		func(t *testing.T, lr *ListReflectionSet) {
			assert.EqualValues(t, int(10), int(lr.pageSize))
		})

	runSad("non message field in response", composed{
		ListFoosResponse: `
			repeated Foo foos = 1;
			psm.list.v1.PageResponse page = 2;
			Foo dangling = 3;
		`,
	}.toString(),
		listerOptions{},
		"unknown field")

	runSad("non message in response", composed{
		ListFoosResponse: `
			repeated Foo foos = 1;
			psm.list.v1.PageResponse page = 2;
			string dangling = 3;
		`,
	}.toString(),
		listerOptions{},
		"should be a message",
	)

	runSad("extra array field in response", composed{
		ListFoosResponse: `
			repeated Foo foos = 1;
			psm.list.v1.PageResponse page = 2;
			repeated Foo dangling = 3;
		`,
	}.toString(),
		listerOptions{},
		"multiple repeated fields")

	runSad("no array field in response", composed{
		ListFoosResponse: `
			psm.list.v1.PageResponse page = 2;
			`,
	}.toString(),
		listerOptions{},
		"no repeated field in response",
	)

	runSad("no page field in response", composed{
		ListFoosResponse: `
			repeated Foo foos = 1;
			`,
	}.toString(),
		listerOptions{},
		"no page field in response",
	)

	runSad("no fallback sort field", composed{
		ListFoosRequest: `
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;
		`,
	}.toString(),
		listerOptions{},
		"no default sort field",
	)

	runHappy("external tie breaker", composed{
		ListFoosRequest: `
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;
		`,
	}.toString(),
		listerOptions{
			tieBreakerFields: []string{"id"},
		},
		func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 1 {
				t.Error("expected one sort tiebreaker")
			} else {
				field := lr.tieBreakerFields[0]
				assert.Equal(t, "id", field.jsonbPath())
			}
		})

	runSad("tie breaker not in response", composed{

		ListFoosRequest: `
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;
			option (psm.list.v1.list_request) = {
				sort_tiebreaker: ["missing"]
			};
			`,
	}.toString(),
		listerOptions{},
		"no field named 'missing'",
	)

	runHappy("sort by bar", `
		message ListFoosRequest {
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;

			option (psm.list.v1.list_request) = {
				sort_tiebreaker: ["bar.id"]
			};
		}

		message ListFoosResponse {
			repeated Foo foos = 1;
			psm.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
			Bar bar = 2;
		}

		message Bar {
			string id = 1;
		}
		`,
		listerOptions{},
		func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 1 {
				t.Error("expected one sort tiebreaker")
			} else {
				field := lr.tieBreakerFields[0]
				assert.Equal(t, "bar->id", field.jsonbPath())

				fieldPath := make([]string, len(field.fieldPath))
				for i, field := range field.fieldPath {
					fieldPath[i] = string(field.Name())
				}
				assert.Equal(t, []string{"bar", "id"}, fieldPath)
			}
		})

	runHappy("sort by bar by walking", `
		message ListFoosRequest {
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;

			option (psm.list.v1.list_request) = {
			};
		}

		message ListFoosResponse {
			repeated Foo foos = 1;
			psm.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
			Bar bar = 2;
		}

		message Bar {
			string id = 1;
			google.protobuf.Timestamp timestamp = 2 [
				(psm.list.v1.field).timestamp = {
					sorting: {
						default_sort: true
					}
				}
			];
		}
		`,
		listerOptions{},
		func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 0 {
				t.Error("expected no sort tiebreaker")
			}

			if len(lr.defaultSortFields) != 1 {
				t.Error("expected one sort tiebreaker, got", len(lr.tieBreakerFields))
			} else {
				field := lr.defaultSortFields[0]
				assert.Equal(t, "bar->timestamp", field.jsonbPath())

				fieldPath := make([]string, len(field.fieldPath))
				for i, field := range field.fieldPath {
					fieldPath[i] = string(field.Name())
				}
				assert.Equal(t, []string{"bar", "timestamp"}, fieldPath)
			}
		})

}
