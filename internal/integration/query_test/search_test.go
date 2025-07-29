package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/pquery"
	"google.golang.org/protobuf/proto"
)

func TestDynamicSearching(t *testing.T) {
	uu := NewSchemaUniverse(t)
	queryer := uu.FooLister(t)

	uu.SetupFoo(t, 30, func(ii int, foo *TestObject) {

		weight := (10 + int64(ii))

		foo.SetScalar(pquery.JSONPath("data", "characteristics", "weight"), weight)
		createdAt := time.Now()
		foo.SetScalar(pquery.JSONPath("metadata", "createdAt"), createdAt)
		foo.SetScalar(pquery.JSONPath("data", "field"), fmt.Sprintf("foo %d weighted %d", ii, weight))
	})

	t.Run("Simple Search Field", func(t *testing.T) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Searches: []*list_j5pb.Search{
					{
						Field: "data.field",
						Value: "weighted 30",
					},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != 1 {
			t.Fatalf("expected %d states, got %d", 1, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for ii, state := range res.Foo {
			if state.Data.Characteristics.Weight != int64(30-ii) {
				t.Fatalf("expected weight %d, got %d", 30-ii, state.Data.Characteristics.Weight)
			}
		}

		if res.Page != nil {
			t.Fatalf("page response should be empty")
		}
	})

}
