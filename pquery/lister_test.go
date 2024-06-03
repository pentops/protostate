package pquery

import (
	"strings"
	"testing"

	"github.com/pentops/flowtest/prototest"
	"github.com/pentops/protostate/pgstore"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
)

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

	type tableMod func(t testing.TB, spec *TableSpec, req, res protoreflect.MessageDescriptor)

	build := func(t testing.TB, input string, spec tableMod) (*ListReflectionSet, error) {
		pdf := prototest.DescriptorsFromSource(t, map[string]string{
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

		requestDesc := pdf.MessageByName(t, "test.ListFoosRequest")
		responseDesc := pdf.MessageByName(t, "test.ListFoosResponse")

		table := &TableSpec{}
		if spec != nil {
			spec(t, table, requestDesc, responseDesc)
		}

		return buildListReflection(requestDesc, responseDesc, *table)
	}

	runHappy := func(name string, input string, spec tableMod, callback func(*testing.T, *ListReflectionSet)) {
		t.Run(name, func(t *testing.T) {
			t.Helper()
			set, err := build(t, input, spec)

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
			_, err := build(t, input, spec)
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

	runHappy("override default page size by validation", `
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
		nil,
		func(t *testing.T, lr *ListReflectionSet) {
			assert.EqualValues(t, int(10), int(lr.defaultPageSize))
		})

	runHappy("tie breaker fallback", composed{
		ListFoosRequest: `
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;
		`,
	}.toString(),
		func(t testing.TB, table *TableSpec, req, res protoreflect.MessageDescriptor) {
			table.FallbackSortColumns = []pgstore.ProtoFieldSpec{{
				ColumnName: table.DataColumn,
				Path:       pgstore.ProtoPathSpec{"id"},
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

	// Response Errors

	runSad("non message field in response", composed{
		ListFoosResponse: `
			repeated Foo foos = 1;
			psm.list.v1.PageResponse page = 2;
			Foo dangling = 3;
		`,
	}.toString(),
		nil,
		"unknown field")

	runSad("non message in response", composed{
		ListFoosResponse: `
			repeated Foo foos = 1;
			psm.list.v1.PageResponse page = 2;
			string dangling = 3;
		`,
	}.toString(),
		nil,
		"should be a message",
	)

	runSad("extra array field in response", composed{
		ListFoosResponse: `
			repeated Foo foos = 1;
			psm.list.v1.PageResponse page = 2;
			repeated Foo dangling = 3;
		`,
	}.toString(),
		nil,
		"multiple repeated fields")

	runSad("no array field in response", composed{
		ListFoosResponse: `
			psm.list.v1.PageResponse page = 2;
			`,
	}.toString(),
		nil,
		"no repeated field in response",
	)

	runSad("no page field in response", composed{
		ListFoosResponse: `
			repeated Foo foos = 1;
			`,
	}.toString(),
		nil,
		"no page field in response",
	)

	// Request Errors

	runSad("no fallback sort field", composed{
		ListFoosRequest: `
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;
		`,
	}.toString(),
		nil,
		"no default sort field",
	)

	runSad("tie breaker not in response", composed{
		ListFoosRequest: `
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;
			option (psm.list.v1.list_request) = {
				sort_tiebreaker: ["missing"]
			};
			`,
	}.toString(),
		nil,
		"no field named 'missing'",
	)

	runSad("no page field", composed{
		ListFoosRequest: `
			psm.list.v1.QueryRequest query = 2;

			option (psm.list.v1.list_request) = {
				sort_tiebreaker: ["id"]
			};
			`,
	}.toString(),
		nil,
		"no page field in request",
	)

	runSad("no query field", composed{
		ListFoosRequest: `
			psm.list.v1.PageRequest page = 1;

			option (psm.list.v1.list_request) = {
				sort_tiebreaker: ["id"]
			};
			`,
	}.toString(),
		nil,
		"no query field in request",
	)

	runSad("repeated field sort", `
		message ListFoosRequest {
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;
		}

		message ListFoosResponse {
			repeated Foo foos = 1;
			psm.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
			int64 seq = 2 [(psm.list.v1.field).int64.sorting = {sortable: true, default_sort: true}];
			repeated int64 weight = 3 [(psm.list.v1.field).int64.sorting.sortable = true];
		}
		`,
		nil,
		"sorting not allowed on repeated field",
	)

	runSad("repeated sub field sort", `
		message ListFoosRequest {
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;
		}

		message ListFoosResponse {
			repeated Foo foos = 1;
			psm.list.v1.PageResponse page = 2;
		}

		message Foo {
			string id = 1;
			int64 seq = 2 [(psm.list.v1.field).int64.sorting = {sortable: true, default_sort: true}];
			repeated Profile profiles = 3;
		}

		message Profile {
			string name = 1;
			int64 weight = 2 [(psm.list.v1.field).int64.sorting.sortable = true];
		}
		`,
		nil,
		"sorting not allowed on subfield of repeated parent",
	)
}
