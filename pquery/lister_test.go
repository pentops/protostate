package pquery

import (
	"strings"
	"testing"

	"github.com/pentops/flowtest/prototest"
	"github.com/stretchr/testify/assert"
)

func TestValidateFieldName(t *testing.T) {
	descFiles := prototest.DescriptorsFromSource(t, map[string]string{
		"test.proto": `
		syntax = "proto3";

		import "psm/list/v1/page.proto";
		import "psm/list/v1/query.proto";

		package test;

		message Foo {
			string id = 1;
			Profile profile = 2;
		}

		message Profile {
			int64 weight = 1;

			oneof type {
				Card card = 2;
			}
		}

		message Card {
			int64 size = 1;
		}
	`})

	fooDesc := descFiles.MessageByName(t, "test.Foo")

	tcs := []struct {
		name string
	}{
		{name: "id"},
		{name: "profile.weight"},
		{name: "profile.type"},
		{name: "profile.card.size"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFieldName(fooDesc, tc.name)
			if err != nil {
				t.Fatal(err)
			}
		})
	}

	tcs = []struct {
		name string
	}{
		{name: "foo"},
		{name: "profile.weight.size"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFieldName(fooDesc, tc.name)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestFindFieldSpec(t *testing.T) {
	descFiles := prototest.DescriptorsFromSource(t, map[string]string{
		"test.proto": `
		syntax = "proto3";

		import "psm/list/v1/page.proto";
		import "psm/list/v1/query.proto";

		package test;

		message Foo {
			string id = 1;
			Profile profile = 2;
		}

		message Profile {
			int64 weight = 1;

			oneof type {
				Card card = 2;
			}
		}

		message Card {
			int64 size = 1;
		}
	`})

	fooDesc := descFiles.MessageByName(t, "test.Foo")

	tcs := []struct {
		name string
	}{
		{name: "id"},
		{name: "profile.weight"},
		{name: "profile.card.size"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := findFieldSpec(fooDesc, tc.name)
			if err != nil {
				t.Fatal(err)
			}

			if spec.field.field == nil {
				t.Fatal("expected field")
			}

			parts := strings.Split(tc.name, ".")
			name := parts[len(parts)-1]
			assert.Equal(t, string(spec.field.field.Name()), name)
		})
	}

	tcs = []struct {
		name string
	}{
		{name: "foo"},
		//{name: "profile.type"},
		{name: "profile.weight.size"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := findFieldSpec(fooDesc, tc.name)
			if err == nil {
				t.Fatal("expected error")
			}

			if spec != nil {
				t.Fatal("expected no spec")
			}
		})
	}
}

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

		responseDesc := pdf.MessageByName(t, "test.ListFoosResponse")
		requestDesc := pdf.MessageByName(t, "test.ListFoosRequest")

		return buildListReflection(requestDesc, responseDesc, options)
	}

	runHappy := func(name string, input string, options listerOptions, callback func(*testing.T, *ListReflectionSet)) {
		t.Run(name, func(t *testing.T) {
			t.Helper()
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
		listerOptions{},
		func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 1 {
				t.Error("expected one sort tiebreaker")
			} else {
				field := lr.tieBreakerFields[0]
				assert.Equal(t, "->>'id'", field.jsonbPath())
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
		listerOptions{},
		func(t *testing.T, lr *ListReflectionSet) {
			assert.EqualValues(t, int(10), int(lr.defaultPageSize))
		})

	runHappy("tie breaker fallback", composed{
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
				assert.Equal(t, "->>'id'", field.jsonbPath())
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
		listerOptions{},
		func(t *testing.T, lr *ListReflectionSet) {
			if len(lr.tieBreakerFields) != 1 {
				t.Error("expected one sort tiebreaker")
			} else {
				field := lr.tieBreakerFields[0]
				assert.Equal(t, "->'bar'->>'id'", field.jsonbPath())

				fieldPath := make([]string, len(field.fieldPath))
				for i, field := range field.fieldPath {
					fieldPath[i] = string(field.field.Name())
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
				assert.Equal(t, "->'bar'->>'timestamp'", field.jsonbPath())

				fieldPath := make([]string, len(field.fieldPath))
				for i, field := range field.fieldPath {
					fieldPath[i] = string(field.field.Name())
				}
				assert.Equal(t, []string{"bar", "timestamp"}, fieldPath)
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

	// Request Errors

	runSad("no fallback sort field", composed{
		ListFoosRequest: `
			psm.list.v1.PageRequest page = 1;
			psm.list.v1.QueryRequest query = 2;
		`,
	}.toString(),
		listerOptions{},
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
		listerOptions{},
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
		listerOptions{},
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
		listerOptions{},
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
		listerOptions{},
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
		listerOptions{},
		"sorting not allowed on subfield of repeated parent",
	)
}
