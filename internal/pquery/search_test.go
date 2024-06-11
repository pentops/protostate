package pquery

import (
	"fmt"
	"testing"

	"github.com/pentops/flowtest/prototest"
	"github.com/stretchr/testify/assert"
)

func TestBuildTsvColumnMap(t *testing.T) {
	descFiles := prototest.DescriptorsFromSource(t, map[string]string{
		"test.proto": `
		syntax = "proto3";

		import "psm/list/v1/annotations.proto";

		package test;

		message Foo {
			string unoptioned_field = 1;
			string optioned_field = 2 [(psm.list.v1.field).string.open_text.searching = {
				searchable: true,
				field_identifier: "optioned_field"
			}];
			Bar bar = 3;
		}

		message Bar {
			string unoptioned_field = 1;
			string optioned_field = 2 [(psm.list.v1.field).string.open_text.searching = {
				searchable: true,
				field_identifier: "bar_optioned_field"
			}];
		}
	`})

	fooDesc := descFiles.MessageByName(t, "test.Foo")

	columnMap := buildTsvColumnMap(fooDesc)
	assert.Len(t, columnMap, 2)

	for f, c := range columnMap {
		t.Log("field: ", f, "\tcolumn: ", c)
	}
}

func TestValidateSearchAnnotations(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		descFiles := prototest.DescriptorsFromSource(t, map[string]string{
			"test.proto": `
		syntax = "proto3";

		import "psm/list/v1/annotations.proto";

		package test;

		message Foo {
			string unoptioned_field = 1;
			string optioned_field = 2 [(psm.list.v1.field).string.open_text.searching = {
				searchable: true,
				field_identifier: "optioned_field"
			}];
			Bar bar = 3;
		}

		message Bar {
			string unoptioned_field = 1;
			string optioned_field = 2 [(psm.list.v1.field).string.open_text.searching = {
				searchable: true,
				field_identifier: "bar_optioned_field"
			}];
		}

	`})

		fooDesc := descFiles.MessageByName(t, "test.Foo")

		err := validateSearchesAnnotations(fooDesc.Fields())
		assert.NoError(t, err)
	})

	t.Run("multi path duplicates", func(t *testing.T) {
		descFiles := prototest.DescriptorsFromSource(t, map[string]string{
			"test.proto": `
		syntax = "proto3";

		import "psm/list/v1/annotations.proto";

		package test;

		message Msg {
			string unoptioned_field = 1;
			string optioned_field = 2 [(psm.list.v1.field).string.open_text.searching = {
				searchable: true,
				field_identifier: "optioned_field"
			}];
		}

		message Foo {
			oneof type {
				Type1 type1 = 1;
				Type2 type2 = 2;
			}
		}

		message Bar {
			Type1 type1 = 1;
			Type2 type2 = 2;
		}

		message Baz {
			oneof set1 {
				Type1 s1type1 = 1;
				Type2 s1type2 = 2;
			}

			oneof set2 {
				Type3 type3 = 3;
				Type4 type4 = 4;
			}

		}

		message Type1 {
			Msg msg = 1;
		}
		message Type2 {
			Msg msg = 1;
		}
		message Type3 {
			Msg msg = 1;
		}
		message Type4 {
			Msg msg = 1;
		}
	`})

		// Both Type1 and Type2 import the same Msg messaage, which means they
		// have the same field identifier.

		// Bar is not mutually exclusive, so is not OK

		// This will be a common pattern for event messages.

		// The two instances of Msg within Foo are mutually exclusive, so is OK
		fooDesc := descFiles.MessageByName(t, "test.Foo")
		err := validateSearchesAnnotations(fooDesc.Fields())
		assert.NoError(t, err)

		// The two instances of Msg within Bar are NOT mutually exclusive, this
		// is not OK.
		barDesc := descFiles.MessageByName(t, "test.Bar")
		err = validateSearchesAnnotations(barDesc.Fields())
		assert.Error(t, err)

		// Each oneof in Baz is OK by itself, but Type1 and Type3 can be set
		// together, and so the search key can be duplicated.
		bazDesc := descFiles.MessageByName(t, "test.Baz")
		err = validateSearchesAnnotations(bazDesc.Fields())
		fmt.Println(err)
		assert.Error(t, err)
	})

	t.Run("duplicate field identifier", func(t *testing.T) {
		descFiles := prototest.DescriptorsFromSource(t, map[string]string{
			"test.proto": `
		syntax = "proto3";

		import "psm/list/v1/annotations.proto";

		package test;

		message Foo {
			string unoptioned_field = 1;
			string optioned_field = 2 [(psm.list.v1.field).string.open_text.searching = {
				searchable: true,
				field_identifier: "optioned_field"
			}];
			Bar bar = 3;
		}

		message Bar {
			string unoptioned_field = 1;
			string optioned_field = 2 [(psm.list.v1.field).string.open_text.searching = {
				searchable: true,
				field_identifier: "optioned_field"
			}];
		}
	`})

		fooDesc := descFiles.MessageByName(t, "test.Foo")

		err := validateSearchesAnnotations(fooDesc.Fields())
		assert.Error(t, err)
	})

	t.Run("missing field identifier", func(t *testing.T) {
		descFiles := prototest.DescriptorsFromSource(t, map[string]string{
			"test.proto": `
		syntax = "proto3";

		import "psm/list/v1/annotations.proto";

		package test;

		message Foo {
			string unoptioned_field = 1;
			string optioned_field = 2 [(psm.list.v1.field).string.open_text.searching = {
				searchable: true,
				field_identifier: ""
			}];
		}
	`})

		fooDesc := descFiles.MessageByName(t, "test.Foo")

		err := validateSearchesAnnotations(fooDesc.Fields())
		assert.Error(t, err)
	})
}
