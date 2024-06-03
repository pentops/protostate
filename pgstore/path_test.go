package pgstore

import (
	"testing"

	"github.com/pentops/flowtest/prototest"
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
		path     ProtoPathSpec
		jsonPath JSONPathSpec
	}{
		{path: ProtoPathSpec{"id"}},
		{path: ProtoPathSpec{"profile", "weight"}},
		{path: ProtoPathSpec{"profile", "card", "size"}},
	}

	for _, tc := range tcs {
		if tc.jsonPath == nil {
			tc.jsonPath = JSONPathSpec(tc.path) // assume it's just lower case wingle word
		}
		t.Run(tc.path.String()+" Proto", func(t *testing.T) {
			spec, err := NewProtoPath(fooDesc, tc.path)
			if err != nil {
				t.Fatal(err)
			}

			if spec.leafField == nil {
				t.Fatal("expected field")
			}

			name := tc.path[len(tc.path)-1]
			assert.Equal(t, string(spec.leafField.Name()), name)
		})
		t.Run(tc.jsonPath.String()+" JSON", func(t *testing.T) {
			spec, err := NewJSONPath(fooDesc, tc.jsonPath)
			if err != nil {
				t.Fatal(err)
			}

			if spec.leafField == nil {
				t.Fatal("expected field")
			}

			name := tc.path[len(tc.path)-1]
			assert.Equal(t, string(spec.leafField.Name()), name)
		})
	}

	tcs = []struct {
		path     ProtoPathSpec
		jsonPath JSONPathSpec
	}{
		{path: ProtoPathSpec{"foo"}},
		{path: ProtoPathSpec{"profile", "weight", "size"}},
	}

	for _, tc := range tcs {
		if tc.jsonPath == nil {
			tc.jsonPath = JSONPathSpec(tc.path) // assume it's just lower case wingle word
		}
		t.Run(tc.path.String()+" Proto Sad", func(t *testing.T) {
			_, err := NewProtoPath(fooDesc, tc.path)
			if err == nil {
				t.Fatal("expected an error")
			}
		})
		t.Run(tc.jsonPath.String()+" JSON Sad", func(t *testing.T) {
			_, err := NewJSONPath(fooDesc, tc.jsonPath)
			if err == nil {
				t.Fatal("expected an error")
			}
		})
	}

	t.Run("Walk", func(t *testing.T) {

		expectedNodes := map[string]struct{}{
			"$.id":                {},
			"$.profile":           {},
			"$.profile.weight":    {},
			"$.profile.card":      {},
			"$.profile.card.size": {},
		}

		err := WalkPathNodes(fooDesc, func(node Path) error {
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
