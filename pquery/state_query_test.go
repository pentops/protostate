package pquery

/*
func TestStateQuery(t *testing.T) {
	ctx := contex

	config := StateQuerySpec{
		TableName:              "foo",
		DataColumn:             "state",
		PrimaryKeyColumn:       "id",
		PrimaryKeyRequestField: protoreflect.Name("foo_id"),

		Events: &GetJoinSpec{
			TableName:        "foo_event",
			DataColumn:       "data",
			FieldInParent:    "events",
			ForeignKeyColumn: "foo_id",
		},

		Auth: AuthProviderFunc(func(ctx context.Context) (map[string]interface{}, error) {
			return map[string]interface{}{
				"tenant_id": testFoo.TenantId,
			}, nil
		}),

		EventsInGet: true,

		Get: &MethodDescriptor{
			Request:  (&testpb.GetFooRequest{}).ProtoReflect().Descriptor(),
			Response: (&testpb.GetFooResponse{}).ProtoReflect().Descriptor(),
		},

		List: &MethodDescriptor{
			Request:  (&testpb.ListFoosRequest{}).ProtoReflect().Descriptor(),
			Response: (&testpb.ListFoosResponse{}).ProtoReflect().Descriptor(),
		},

		ListEvents: &MethodDescriptor{
			Request:  (&testpb.ListFooEventsRequest{}).ProtoReflect().Descriptor(),
			Response: (&testpb.ListFooEventsResponse{}).ProtoReflect().Descriptor(),
		},
	}

	queryer, err := NewStateQuery(config)
	if err != nil {
		t.Fatal(err.Error())
	}



}*/
