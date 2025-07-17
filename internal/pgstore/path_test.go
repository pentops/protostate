package pgstore

import (
	"testing"

	"github.com/pentops/flowtest/prototest"
	"github.com/pentops/j5/lib/j5schema"
	"github.com/stretchr/testify/assert"
)

func TestFindFieldSpec(t *testing.T) {
	descFiles := prototest.DescriptorsFromSource(t, map[string]string{
		"test.proto": `
		syntax = "proto3";


		package test;

		message Foo {
			string id = 1;
			Profile profile = 2;
		}

		message Profile {
			int64 weight = 1;
			ProfileType type = 2;
		}

		message ProfileType {
			oneof type {
				Card card = 2;
			}
		}


		message Card {
			int64 size = 1;
		}
	`})

	fooDesc := descFiles.MessageByName(t, "test.Foo")
	fooSchema, err := j5schema.Global.Schema(fooDesc)
	if err != nil {
		t.Fatal(err)
	}
	fooObject, ok := fooSchema.(*j5schema.ObjectSchema)
	if !ok {
		t.Fatal("expected Foo to be an ObjectSchema")
	}

	t.Run("Find Field", func(t *testing.T) {

		tcs := []struct {
			path    JSONPathSpec
			isOneof bool
		}{
			{path: JSONPathSpec{"id"}},
			{path: JSONPathSpec{"profile", "weight"}},
			{path: JSONPathSpec{"profile", "type"}},
			{path: JSONPathSpec{"profile", "type", "type"}, isOneof: true},
			{path: JSONPathSpec{"profile", "type", "card", "size"}},
		}

		for _, tc := range tcs {
			t.Run(tc.path.String()+" JSON", func(t *testing.T) {
				spec, err := NewJSONPath(fooObject, tc.path)
				if err != nil {
					t.Fatal(err)
				}

				if tc.isOneof {
					if spec.leafOneof == nil {
						t.Fatal("missing oneof field")
					}
				} else {
					if spec.leafField == nil {
						t.Fatal("missing leaf field")
					}
					name := tc.path[len(tc.path)-1]
					assert.Equal(t, string(spec.leafField.JSONName), name)
				}
			})
		}
	})

	t.Run("Sad Path", func(t *testing.T) {
		tcs := []struct {
			path JSONPathSpec
		}{
			{path: JSONPathSpec{"foo"}},
			{path: JSONPathSpec{"profile", "weight", "size"}},
		}

		for _, tc := range tcs {
			t.Run(tc.path.String()+" JSON Sad", func(t *testing.T) {
				_, err := NewJSONPath(fooObject, tc.path)
				if err == nil {
					t.Fatal("expected an error")
				}
			})
		}
	})

	t.Run("Walk", func(t *testing.T) {

		expectedNodes := map[string]struct{}{
			"$.id":                     {},
			"$.profile":                {},
			"$.profile.weight":         {},
			"$.profile.type":           {},
			"$.profile.type.card":      {},
			"$.profile.type.card.size": {},
		}

		err := WalkPathNodes(fooObject, func(node Path) error {
			pathQuery := node.JSONPathQuery()
			t.Log(pathQuery)
			_, ok := expectedNodes[pathQuery]
			if !ok {
				t.Fatalf("unexpected node %s", pathQuery)
			}
			delete(expectedNodes, pathQuery)

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		for node := range expectedNodes {
			t.Errorf("expected node %s", node)
		}
	})
}
