package integration

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/j5/gen/j5/auth/v1/auth_j5pb"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	"github.com/pentops/protostate/internal/testproto/gen/test/v1/test_spb"
	"github.com/pentops/protostate/pquery"
	"google.golang.org/protobuf/proto"
)

func TestDefaultFiltering(t *testing.T) {
	uu := NewSchemaUniverse(t)
	ss := NewStepper(t)
	defer ss.RunSteps(t)

	var queryer *pquery.Lister

	ss.Setup(func(ctx context.Context, t flowtest.Asserter) error {
		queryer = uu.FooLister(t)
		uu.SetupFoo(t, 10, func(ii int, foo *TestObject) {
			if ii < 2 {
				foo.SetScalar(pquery.JSONPath("status"), "DELETED")
			}
		})
		return nil
	})

	ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(10),
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != 8 {
			t.Fatalf("expected %d states, got %d", 8, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for _, state := range res.Foo {
			if state.Status != test_pb.FooStatus_ACTIVE {
				t.Fatalf("expected status %s, got %s", test_pb.FooStatus_ACTIVE, state.Status)
			}
		}

		if res.Page != nil {
			t.Fatalf("page response should be empty")
		}
	})
}

func TestFilteringWithAuthScope(t *testing.T) {
	uu := NewSchemaUniverse(t)
	ss := NewStepper(t)
	defer ss.RunSteps(t)

	var queryer *pquery.Lister

	tenantID1 := uuid.NewString()
	tenantID2 := uuid.NewString()

	tenants := []string{tenantID1, tenantID2}

	ss.Setup(func(ctx context.Context, t flowtest.Asserter) error {
		queryer = uu.FooLister(t)
		uu.SetupFoo(t, 20, func(idx int, foo *TestObject) {
			// recreating the old function logic
			tenantId := idx % len(tenants)
			tenantID := tenants[tenantId]
			ii := idx / len(tenants)
			ti := tenantId

			foo.SetScalar(pquery.JSONPath("tenantId"), tenantID)

			weight := ((10 + int64(ii)) * (int64(ti) + 1))
			height := ((50 - int64(ii)) * (int64(ti) + 1))
			length := (int64(ii%2) * (int64(ti) + 1))

			foo.SetScalar(pquery.JSONPath("data", "characteristics", "weight"), weight)
			foo.SetScalar(pquery.JSONPath("data", "characteristics", "height"), height)
			foo.SetScalar(pquery.JSONPath("data", "characteristics", "length"), length)
			createdAt := time.Now()
			foo.SetScalar(pquery.JSONPath("metadata", "createdAt"), createdAt.Format(time.RFC3339Nano))

		})
		return nil
	})

	tkn := &token{
		claim: &auth_j5pb.Claim{
			TenantType: "tenant",
			TenantId:   tenantID1,
		},
	}

	ss.Step("List Page", func(ctx context.Context, t flowtest.Asserter) {
		ctx = tkn.WithToken(ctx)

		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								Name: "data.characteristics.weight",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Range{
										Range: &list_j5pb.Range{
											Min: "12",
											Max: "15",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		res := &test_spb.FooListResponse{}
		err := queryer.List(ctx, uu.DB, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != 4 {
			t.Fatalf("expected %d states, got %d", 4, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s (%s)", ii, state.Data.Field, state.Metadata.CreatedAt.AsTime().Format(time.RFC3339Nano))
		}

		for ii, state := range res.Foo {
			if *state.Keys.TenantId != tenantID1 {
				t.Errorf("expected tenant ID %s, got %s", tenantID1, state.Keys.TenantId)
			}
			if state.Data.Characteristics.Weight != int64(15-ii) {
				t.Errorf("expected weight %d, got %d", 15-ii, state.Data.Characteristics.Weight)
			}

		}

		if res.Page != nil {
			t.Fatalf("page response should be empty")
		}

	})
}

func TestDynamicFiltering(t *testing.T) {
	uu := NewSchemaUniverse(t)

	queryer := uu.FooLister(t)
	tenantID := uuid.NewString()
	uu.SetupFoo(t, 60, func(ii int, foo *TestObject) {
		weight := (10 + int64(ii))
		height := (50 - int64(ii))
		length := (int64(ii % 2))

		foo.SetScalar(pquery.JSONPath("tenantId"), tenantID)
		foo.SetScalar(pquery.JSONPath("data", "characteristics", "weight"), weight)
		foo.SetScalar(pquery.JSONPath("data", "characteristics", "height"), height)
		foo.SetScalar(pquery.JSONPath("data", "characteristics", "length"), length)
		createdAt := time.Now()
		foo.SetScalar(pquery.JSONPath("metadata", "createdAt"), createdAt.Format(time.RFC3339Nano))

		label := fmt.Sprintf("foo %d at %s (weighted %d, height %d, length %d)", ii, createdAt.Format(time.RFC3339Nano), weight, height, length)
		foo.SetScalar(pquery.JSONPath("data", "field"), label)

		val, err := foo.GetOrCreateValue("data", "profiles")
		if err != nil {
			t.Fatalf("failed to get or create profiles: %v", err)
		}

		objArray, ok := val.AsArrayOfObject()
		if !ok {
			t.Fatalf("expected profiles to be an array of objects, got %T", val)
		}

		profile1, _ := objArray.NewObjectElement()
		profile2, _ := objArray.NewObjectElement()

		if err := SetScalar(profile1, pquery.JSONPath("name"), fmt.Sprintf("profile %d", ii)); err != nil {
			t.Fatalf("failed to set profile name: %v", err)
		}

		if err := SetScalar(profile1, pquery.JSONPath("place"), int64(ii)+50); err != nil {
			t.Fatalf("failed to set profile place: %v", err)
		}

		if err := SetScalar(profile2, pquery.JSONPath("name"), fmt.Sprintf("profile %d", ii)); err != nil {
			t.Fatalf("failed to set profile name: %v", err)
		}

		if err := SetScalar(profile2, pquery.JSONPath("place"), int64(ii)+15); err != nil {
			t.Fatalf("failed to set profile place: %v", err)
		}

		switch ii {
		case 10:
			foo.SetScalar(pquery.JSONPath("data", "shape", "circle", "radius"), int64(10))
		case 11:
			foo.SetScalar(pquery.JSONPath("data", "shape", "square", "side"), int64(10))
		}
	})

	t.Run("Single Range Filter", func(t *testing.T) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								Name: "data.characteristics.weight",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Range{
										Range: &list_j5pb.Range{
											Min: "12",
											Max: "15",
										},
									},
								},
							},
						},
					},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != 4 {
			t.Fatalf("expected %d states, got %d", 4, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for ii, state := range res.Foo {
			if state.Data.Characteristics.Weight != int64(15-ii) {
				t.Fatalf("expected weight %d, got %d", 15-ii, state.Data.Characteristics.Weight)
			}
		}

		if res.Page != nil {
			t.Fatalf("page response should be empty")
		}
	})

	t.Run("Min Range Filter", func(t *testing.T) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								Name: "data.characteristics.weight",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Range{
										Range: &list_j5pb.Range{
											Min: "12",
										},
									},
								},
							},
						},
					},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != 5 {
			for _, foo := range res.Foo {
				t.Logf("Foo: %s, Weight: %d", foo.Data.Field, foo.Data.Characteristics.Weight)
			}
			t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for _, state := range res.Foo {
			if state.Data.Characteristics.Weight < int64(12) {
				t.Fatalf("expected weights greater than or equal to %d, got %d", 12, state.Data.Characteristics.Weight)
			}
		}
	})

	t.Run("Max Range Filter", func(t *testing.T) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								Name: "data.characteristics.weight",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Range{
										Range: &list_j5pb.Range{
											Max: "15",
										},
									},
								},
							},
						},
					},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(res.Foo) != 5 {
			t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for _, state := range res.Foo {
			if state.Data.Characteristics.Weight > int64(15) {
				t.Fatalf("expected weight less than or equal to %d, got %d", 15, state.Data.Characteristics.Weight)
			}
		}
	})

	t.Run("Multi Range Filter", func(t *testing.T) {
		nextToken := ""
		query := &list_j5pb.QueryRequest{
			Filters: []*list_j5pb.Filter{
				{
					Type: &list_j5pb.Filter_Or{
						Or: &list_j5pb.Or{
							Filters: []*list_j5pb.Filter{
								{
									Type: &list_j5pb.Filter_Field{
										Field: &list_j5pb.Field{
											Name: "data.characteristics.weight",
											Type: &list_j5pb.FieldType{
												Type: &list_j5pb.FieldType_Range{
													Range: &list_j5pb.Range{
														Min: "12",
														Max: "20",
													},
												},
											},
										},
									},
								},
								{
									Type: &list_j5pb.Filter_Field{
										Field: &list_j5pb.Field{
											Name: "data.characteristics.height",
											Type: &list_j5pb.FieldType{
												Type: &list_j5pb.FieldType_Range{
													Range: &list_j5pb.Range{
														Min: "16",
														Max: "18",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		{
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(10),
				},
				Query: query,
			}
			res := &test_spb.FooListResponse{}

			err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			if len(res.Foo) != 10 {
				t.Fatalf("expected %d states, got %d", 10, len(res.Foo))
			}

			for ii, state := range res.Foo[:3] {
				if state.Data.Characteristics.Weight != int64(44-ii) {
					t.Fatalf("expected weight %d, got %d", 44-ii, state.Data.Characteristics.Weight)
				}
			}

			for ii, state := range res.Foo[3:] {
				if state.Data.Characteristics.Weight != int64(20-ii) {
					t.Fatalf("expected weight %d, got %d", 20-ii, state.Data.Characteristics.Weight)
				}
			}

			pageResp := res.Page

			if pageResp.GetNextToken() == "" {
				t.Fatalf("NextToken should not be empty")
			}
			if pageResp.NextToken == nil {
				t.Fatalf("Should not be the final page")
			}

			nextToken = pageResp.GetNextToken()

		}

		{
			dec, err := base64.StdEncoding.DecodeString(nextToken)
			if err != nil {
				t.Fatalf("failed to decode next token: %v", err)
			}
			t.Log(string(dec))
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(10),
					Token:    &nextToken,
				},
				Query: query,
			}
			res := &test_spb.FooListResponse{}

			err = queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Field)
			}

			if len(res.Foo) != 2 {
				t.Fatalf("expected %d states, got %d", 2, len(res.Foo))
			}

			for ii, state := range res.Foo {
				if state.Data.Characteristics.Weight != int64(13-ii) {
					t.Fatalf("expected weight %d, got %d", 13-ii, state.Data.Characteristics.Weight)
				}
			}

			if res.Page != nil {
				t.Fatalf("page response should be empty")
			}
		}
	})

	t.Run("Flattened filterable fields", func(t *testing.T) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								Name: "tenantId",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Value{
										Value: tenantID,
									},
								},
							},
						},
					},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Foo) == 0 {
			t.Fatalf("expected to receive results filtered by tenantId, but got none")
		}

		for _, foo := range res.Foo {
			if *foo.Keys.TenantId != tenantID {
				t.Fatalf("expected tenantId %s, got %s", tenantID, *foo.Keys.TenantId)
			}
		}
	})

	t.Run("Non filterable fields", func(t *testing.T) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								Name: "foo_id",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Value{
										Value: "d34d826f-afe3-410d-8326-4e9af3f09467",
									},
								},
							},
						},
					},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("Enum values - Short", func(t *testing.T) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								Name: "status",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Value{
										Value: "ACTIVE",
									},
								},
							},
						},
					},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Foo) != 5 {
			t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for _, state := range res.Foo {
			if state.Status != test_pb.FooStatus_ACTIVE {
				t.Fatalf("expected status %s, got %s", test_pb.FooStatus_ACTIVE, state.Status)
			}
		}
	})

	t.Run("Enum values - Full", func(t *testing.T) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								Name: "status",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Value{
										Value: "FOO_STATUS_ACTIVE",
									},
								},
							},
						},
					},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Foo) != 5 {
			t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
		}

		for ii, state := range res.Foo {
			t.Logf("%d: %s", ii, state.Data.Field)
		}

		for _, state := range res.Foo {
			if state.Status != test_pb.FooStatus_ACTIVE {
				t.Fatalf("expected status %s, got %s", test_pb.FooStatus_ACTIVE, state.Status)
			}
		}

	})

	t.Run("List Page bad enum name", func(t *testing.T) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								Name: "status",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Value{
										Value: "FOO_STATUS_UNUSED",
									},
								},
							},
						},
					},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("Single Complex Range Filter", func(t *testing.T) {
		nextToken := ""
		{
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "data.profiles.place",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Range{
											Range: &list_j5pb.Range{
												Min: "15",
												Max: "21",
											},
										},
									},
								},
							},
						},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Profiles)
			}

			if len(res.Foo) != 5 {
				t.Fatalf("expected %d states, got %d", 5, len(res.Foo))
			}

			for _, state := range res.Foo {
				matched := false
				for _, profile := range state.Data.Profiles {
					if profile.Place >= 17 && profile.Place <= 21 {
						matched = true
						break
					}
				}

				if !matched {
					t.Fatalf("expected at least one profile to match the filter")
				}
			}

			pageResp := res.Page

			if pageResp.GetNextToken() == "" {
				t.Fatalf("NextToken should not be empty")
			}
			if pageResp.NextToken == nil {
				t.Fatalf("Should not be the final page")
			}

			nextToken = pageResp.GetNextToken()
		}

		{
			req := &test_spb.FooListRequest{
				Page: &list_j5pb.PageRequest{
					PageSize: proto.Int64(5),
					Token:    &nextToken,
				},
				Query: &list_j5pb.QueryRequest{
					Filters: []*list_j5pb.Filter{
						{
							Type: &list_j5pb.Filter_Field{
								Field: &list_j5pb.Field{
									Name: "data.profiles.place",
									Type: &list_j5pb.FieldType{
										Type: &list_j5pb.FieldType_Range{
											Range: &list_j5pb.Range{
												Min: "15",
												Max: "21",
											},
										},
									},
								},
							},
						},
					},
				},
			}
			res := &test_spb.FooListResponse{}

			err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
			if err != nil {
				t.Fatal(err.Error())
			}

			for ii, state := range res.Foo {
				t.Logf("%d: %s", ii, state.Data.Profiles)
			}

			if len(res.Foo) != 2 {
				t.Fatalf("expected %d states, got %d", 2, len(res.Foo))
			}

			for _, state := range res.Foo {
				matched := false
				for _, profile := range state.Data.Profiles {
					if profile.Place >= 15 && profile.Place <= 16 {
						matched = true
						break
					}
				}

				if !matched {
					t.Fatalf("expected at least one profile to match the filter")
				}
			}
		}
	})

	t.Run("Oneof filter - happy", func(t *testing.T) {
		req := &test_spb.FooListRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								// match the JSON encoding
								// {
								//   "event": {
								//     "!type": "created"
								//     "created": {...}
								//   }
								//   ...
								// }
								Name: "data.shape.!type",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Value{
										Value: "circle",
									},
								},
							},
						},
					},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Foo) != 1 {
			t.Fatalf("expected %d states, got %d", 1, len(res.Foo))
		}

		for ii, event := range res.Foo {
			switch event.Data.Shape.Type.(type) {
			case *test_pb.FooData_Shape_Circle_:
			default:
				t.Fatalf("expected event to be of type %T, got %T", &test_pb.FooData_Shape_Circle_{}, event.Data.Shape.Type)
			}

			t.Logf("%d: %s", ii, event.Data.Shape)
		}
	})

	t.Run("Oneof filter - bad name", func(t *testing.T) {
		req := &test_spb.FooEventsRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(5),
			},
			Query: &list_j5pb.QueryRequest{
				Filters: []*list_j5pb.Filter{
					{
						Type: &list_j5pb.Filter_Field{
							Field: &list_j5pb.Field{
								Name: "data.shape.!type",
								Type: &list_j5pb.FieldType{
									Type: &list_j5pb.FieldType_Value{
										Value: "damaged",
									},
								},
							},
						},
					},
				},
			},
		}
		res := &test_spb.FooListResponse{}

		err := queryer.List(t.Context(), uu.DB, req.J5Object(), res.J5Object())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
