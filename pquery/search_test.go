package pquery

import (
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

		err := validateSearchesAnnotations(nil, fooDesc.Fields())
		assert.NoError(t, err)
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

		err := validateSearchesAnnotations(nil, fooDesc.Fields())
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

		err := validateSearchesAnnotations(nil, fooDesc.Fields())
		assert.Error(t, err)
	})
}
