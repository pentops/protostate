package pquery

import (
	"strings"
	"testing"

	"github.com/pentops/flowtest/prototest"
	"github.com/pentops/golib/gl"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/j5/lib/j5schema"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type composed struct {
	FooListRequest  string
	FooListResponse string
	Foo             string
}

func (c composed) toString() string {
	out := ""
	if c.FooListRequest != "" {
		out += "message FooListRequest {\n" + c.FooListRequest + "\n}\n"
	} else {
		out += `message FooListRequest {
				j5.list.v1.PageRequest page = 1;
				j5.list.v1.QueryRequest query = 2;

				option (j5.list.v1.list_request) = {
					sort_tiebreaker: ["id"]
				};
			}`
	}

	if c.FooListResponse != "" {
		out += "message FooListResponse {\n" + c.FooListResponse + "\n}\n"
	} else {
		out += `message FooListResponse {
				repeated Foo foos = 1;
				j5.list.v1.PageResponse page = 2;
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

type J5Proto struct {
	proto.Message
}

type tableMod func(t testing.TB, spec *TableSpec, req, res protoreflect.MessageDescriptor)

func testListReflection(t testing.TB, input string, spec tableMod) (*ListReflectionSet, error) {
	t.Helper()
	pdf := prototest.DescriptorsFromSource(t, map[string]string{
		"test.proto": `
				syntax = "proto3";

				package test;

				// Import everything which may be used
				import "j5/ext/v1/annotations.proto";
				import "j5/list/v1/page.proto";
				import "j5/list/v1/query.proto";
				import "j5/list/v1/annotations.proto";
				import "j5/types/date/v1/date.proto";
				import "buf/validate/validate.proto";
				import "google/protobuf/timestamp.proto";

				service FooService {
					rpc FooList(FooListRequest) returns (FooListResponse) {}
				}

				` + input,
	})

	requestDesc := pdf.MessageByName(t, "test.FooListRequest")
	responseDesc := pdf.MessageByName(t, "test.FooListResponse")
	table := &TableSpec{
		DataColumn: "data",
	}
	if spec != nil {
		spec(t, table, requestDesc, responseDesc)
	}

	schemaSet := j5schema.NewSchemaCache()
	requestObj, err := schemaSet.Schema(requestDesc)
	if err != nil {
		return nil, err
	}
	responseObj, err := schemaSet.Schema(responseDesc)
	if err != nil {
		return nil, err
	}
	ms := &j5schema.MethodSchema{
		Request:  requestObj.(*j5schema.ObjectSchema),
		Response: responseObj.(*j5schema.ObjectSchema),
	}

	return buildListReflection(ms, *table)
}

func TestBuildListReflection(t *testing.T) {

	runHappy := func(name string, input string, spec tableMod, callback func(*testing.T, *ListReflectionSet)) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Helper()
			set, err := testListReflection(t, input, spec)

			if err != nil {
				t.Fatal(err)
			}
			callback(t, set)
		})
	}
	runSad := func(name string, input string, spec tableMod, wantError string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Helper()
			_, err := testListReflection(t, input, spec)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), wantError) {
				t.Errorf("expected error to contain '%s', got '%s'", wantError, err.Error())
			}
		})
	}

	// Successes

	runHappy("full success", `
		message FooListRequest {
			j5.list.v1.PageRequest page = 1;
			j5.list.v1.QueryRequest query = 2;

			option (j5.list.v1.list_request) = {
				sort_tiebreaker: ["id"]
			};
		}

		message FooListResponse {
			repeated Foo foos = 1;
			j5.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
		}
		`,
		nil,
		func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 1 {
				t.Error("expected one sort tiebreaker")
			} else {
				field := lr.tieBreakerFields[0]
				assert.Equal(t, "->>'id'", field.Path.JSONBArrowPath())
			}
			assert.Equal(t, uint64(20), lr.defaultPageSize)
		})

	runHappy("dynamic filter", `
		message FooListRequest {
			j5.list.v1.PageRequest page = 1;
			j5.list.v1.QueryRequest query = 2;
			option (j5.list.v1.list_request) = {
				sort_tiebreaker: ["val"]
			};
		}

		message FooListResponse {
			repeated Foo foos = 1;
			j5.list.v1.PageResponse page = 2;
		}

		message Foo {
			int64 val = 1 [(j5.list.v1.field).int64.filtering.filterable = true];
		}
		`, nil, func(t *testing.T, lr *ListReflectionSet) {

		statements, err := lr.buildDynamicFilter("ALIAS", []*list_j5pb.Filter{{
			Type: &list_j5pb.Filter_Field{
				Field: &list_j5pb.Field{
					Name: "val",
					Type: &list_j5pb.FieldType{
						Type: &list_j5pb.FieldType_Value{
							Value: "e10",
						},
					},
				},
			},
		}})

		if err != nil {
			t.Fatal(err)
		}

		if len(statements) != 1 {
			t.Fatal("expected one statement, got", len(statements))
		}
		statement := statements[0]

		txt, params, err := statement.ToSql()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("SQL: %s, Params: %v", txt, params)
		want := "(jsonb_path_query_array(ALIAS.data, '$.val') @> ?::jsonb)"
		if txt != want {
			t.Errorf("SQL mismatch:\n got: %s\nwant: %s", txt, want)
		}

	})

	runHappy("override default page size by validation", `
		message FooListRequest {
			j5.list.v1.PageRequest page = 1;
			j5.list.v1.QueryRequest query = 2;

			option (j5.list.v1.list_request) = {
				sort_tiebreaker: ["id"]
			};
		}

		message FooListResponse {
			repeated Foo foos = 1 [
				(buf.validate.field).repeated.max_items = 10
			];
			j5.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
		}
		`,
		nil,
		func(t *testing.T, lr *ListReflectionSet) {
			assert.EqualValues(t, int(10), int(lr.defaultPageSize))
		})

	runHappy("tie breaker fallback", composed{
		FooListRequest: `
			j5.list.v1.PageRequest page = 1;
			j5.list.v1.QueryRequest query = 2;
		`,
	}.toString(),
		func(t testing.TB, table *TableSpec, req, res protoreflect.MessageDescriptor) {
			table.FallbackSortColumns = []ProtoField{{
				valueColumn: gl.Ptr("id"),
				pathInRoot:  JSONPathSpec{"id"},
			}}
		},
		func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 1 {
				t.Error("expected one sort tiebreaker")
			} else {
				field := lr.tieBreakerFields[0]
				assert.Equal(t, "->>'id'", field.Path.JSONBArrowPath())
			}
		})

	runHappy("sort by bar", `
		message FooListRequest {
			j5.list.v1.PageRequest page = 1;
			j5.list.v1.QueryRequest query = 2;

			option (j5.list.v1.list_request) = {
				sort_tiebreaker: ["bar.id"]
			};
		}

		message FooListResponse {
			repeated Foo foos = 1;
			j5.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
			Bar bar = 2;
		}

		message Bar {
			string id = 1;
		}
		`,
		nil,
		func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 1 {
				t.Error("expected one sort tiebreaker")
			} else {
				field := lr.tieBreakerFields[0]
				assert.Equal(t, "->'bar'->>'id'", field.Path.JSONBArrowPath())
				assert.Equal(t, "$.bar.id", field.Path.JSONPathQuery())
			}
		})

	runHappy("sort by bar by walking", `
		message FooListRequest {
			j5.list.v1.PageRequest page = 1;
			j5.list.v1.QueryRequest query = 2;

			option (j5.list.v1.list_request) = {
			};
		}

		message FooListResponse {
			repeated Foo foos = 1;
			j5.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
			Bar bar = 2;
		}

		message Bar {
			string id = 1;
			google.protobuf.Timestamp timestamp = 2 [
				(j5.list.v1.field).timestamp = {
					sorting: {
						default_sort: true
					}
				}
			];
		}
		`,
		nil,
		func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 0 {
				t.Error("expected no sort tiebreaker")
			}

			if len(lr.defaultSortFields) != 1 {
				t.Error("expected one sort tiebreaker, got", len(lr.tieBreakerFields))
			} else {
				field := lr.defaultSortFields[0]
				assert.Equal(t, "->'bar'->>'timestamp'", field.Path.JSONBArrowPath())
				assert.Equal(t, "$.bar.timestamp", field.Path.JSONPathQuery())
			}
		})

	runHappy("filter by bar date", `
		message FooListRequest {
			j5.list.v1.PageRequest page = 1;
			j5.list.v1.QueryRequest query = 2;

			option (j5.list.v1.list_request) = {
				sort_tiebreaker: ["bar.id"]
			};
		}

		message FooListResponse {
			repeated Foo foos = 1;
			j5.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
			Bar bar = 2;
		}

		message Bar {
			string id = 1;
			j5.types.date.v1.Date date = 2 [
				(j5.list.v1.field).date.filtering = {
					filterable: true,
					default_filters: ["2025-01-01"]
				}
			];
		}
		`,
		nil,
		func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.defaultFilterFields) != 1 {
				t.Error("expected one filter field, got", len(lr.defaultFilterFields))
			} else {
				field := lr.defaultFilterFields[0]
				assert.Equal(t, "->'bar'->>'date'", field.Path.JSONBArrowPath())
				assert.Equal(t, "$.bar.date", field.Path.JSONPathQuery())
			}
		})

	// Response Errors

	runSad("non message field in response", composed{
		FooListResponse: `
			repeated Foo foos = 1;
			j5.list.v1.PageResponse page = 2;
			Foo dangling = 3;
		`,
	}.toString(),
		nil,
		"unknown field")

	runSad("non message in response", composed{
		FooListResponse: `
			repeated Foo foos = 1;
			j5.list.v1.PageResponse page = 2;
			string dangling = 3;
		`,
	}.toString(),
		nil,
		"unknown field in response",
	)

	runSad("extra array field in response", composed{
		FooListResponse: `
			repeated Foo foos = 1;
			j5.list.v1.PageResponse page = 2;
			repeated Foo dangling = 3;
		`,
	}.toString(),
		nil,
		"multiple repeated")

	runSad("no array field in response", composed{
		FooListResponse: `
			j5.list.v1.PageResponse page = 2;
			`,
	}.toString(),
		nil,
		"does not contain a repeated message field",
	)

	runSad("no page field in response", composed{
		FooListResponse: `
			repeated Foo foos = 1;
			`,
	}.toString(),
		nil,
		"no page field in response",
	)

	// Request Errors

	runSad("no fallback sort field", composed{
		FooListRequest: `
			j5.list.v1.PageRequest page = 1;
			j5.list.v1.QueryRequest query = 2;
		`,
	}.toString(),
		nil,
		"no default sort field",
	)

	runSad("tie breaker not in response", composed{
		FooListRequest: `
			j5.list.v1.PageRequest page = 1;
			j5.list.v1.QueryRequest query = 2;
			option (j5.list.v1.list_request) = {
				sort_tiebreaker: ["missing"]
			};
			`,
	}.toString(),
		nil,
		`unknown proto field "missing"`,
	)

	runSad("no page field", composed{
		FooListRequest: `
			j5.list.v1.QueryRequest query = 2;

			option (j5.list.v1.list_request) = {
				sort_tiebreaker: ["id"]
			};
			`,
	}.toString(),
		nil,
		"no page field in request",
	)

	runSad("no query field", composed{
		FooListRequest: `
			j5.list.v1.PageRequest page = 1;

			option (j5.list.v1.list_request) = {
				sort_tiebreaker: ["id"]
			};
			`,
	}.toString(),
		nil,
		"no query field in request",
	)

	runSad("repeated field sort", `
		message FooListRequest {
			j5.list.v1.PageRequest page = 1;
			j5.list.v1.QueryRequest query = 2;
		}

		message FooListResponse {
			repeated Foo foos = 1;
			j5.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
			int64 seq = 2 [(j5.list.v1.field).int64.sorting = {sortable: true, default_sort: true}];
			repeated int64 weight = 3 [(j5.list.v1.field).int64.sorting.sortable = true];
		}
		`,
		nil,
		"list constraints not supported for arrays",
	)

	/*
		Test Change: Repeated isn't walked for sorting, but isn't invalid in case Profile is used in another non-array context.
		runSad("repeated sub field sort", `
			message FooListRequest {
				j5.list.v1.PageRequest page = 1;
				j5.list.v1.QueryRequest query = 2;
			}

			message FooListResponse {
				repeated Foo foos = 1;
				j5.list.v1.PageResponse page = 2;
			}

			message Foo {
				string id = 1;
				int64 seq = 2 [(j5.list.v1.field).int64.sorting = {sortable: true, default_sort: true}];
				repeated Profile profiles = 3;
			}

			message Profile {
				string name = 1;
				int64 weight = 2 [(j5.list.v1.field).int64.sorting.sortable = true];
			}
			`,
			nil,
			"sorting not allowed on subfield of repeated parent",
		)
	*/
}
