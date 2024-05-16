package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/protostate/testproto/gen/testpb"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type FooTester struct {
	*testpb.FooPSMDB
	db      *sqrlx.Wrapper
	ActorID string
	Queryer *testpb.FooPSMQuerySet
}

func NewFooTestMachine(t *testing.T, db *sqrlx.Wrapper) *FooTester {

	actorID := uuid.NewString()

	systemActor, err := psm.NewSystemActor(actorID)
	if err != nil {
		t.Fatal(err.Error())
	}
	sm, err := testpb.NewFooPSM(testpb.
		DefaultFooPSMConfig().
		StoreEventStateSnapshot().
		SystemActor(systemActor))
	if err != nil {
		t.Fatal(err.Error())
	}

	sm.From(testpb.FooStatus_UNSPECIFIED).
		Transition(testpb.FooPSMTransition(func(
			ctx context.Context,
			state *testpb.FooState,
			event *testpb.FooEventType_Created,
		) error {
			state.Status = testpb.FooStatus_ACTIVE
			state.Data = &testpb.FooStateData{}

			state.Data.Name = event.Name
			state.Data.Field = event.Field
			state.Data.Description = event.Description
			state.Data.Characteristics = &testpb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}
			state.Data.Profiles = event.Profiles
			return nil
		}))

	sm.From(testpb.FooStatus_ACTIVE).
		Transition(testpb.FooPSMTransition(func(
			ctx context.Context,
			state *testpb.FooState,
			event *testpb.FooEventType_Updated,
		) error {
			state.Data.Field = event.Field
			state.Data.Name = event.Name
			state.Data.Description = event.Description
			state.Data.Characteristics = &testpb.FooCharacteristics{
				Weight: event.GetWeight(),
				Height: event.GetHeight(),
				Length: event.GetLength(),
			}

			return nil
		}))

	sm.From().
		Hook(testpb.FooPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			baton testpb.FooPSMHookBaton,
			state *testpb.FooState,
			event *testpb.FooEventType_Updated,
		) error {
			if event.Delete {
				baton.ChainEvent(&testpb.FooEventType_Deleted{})
			}
			return nil
		}))

	sm.From(testpb.FooStatus_ACTIVE).
		Transition(testpb.FooPSMTransition(func(
			ctx context.Context,
			state *testpb.FooState,
			event *testpb.FooEventType_Deleted,
		) error {
			state.Status = testpb.FooStatus_DELETED
			return nil
		}))

	queryer, err := testpb.NewFooPSMQuerySet(testpb.DefaultFooPSMQuerySpec(sm.StateTableSpec()), psm.StateQueryOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}
	return &FooTester{
		ActorID:  actorID,
		FooPSMDB: (*testpb.FooPSMDB)(sm.WithDB(db)),
		Queryer:  queryer,
		db:       db,
	}

}

func (ft *FooTester) GetFoo(t testing.TB, id string) *testpb.FooState {
	req := &testpb.GetFooRequest{
		FooId: id,
	}
	res := &testpb.GetFooResponse{}

	err := ft.Queryer.Get(context.Background(), ft.db, req, res)
	if err != nil {
		t.Fatal(err.Error())
	}
	return res.State
}

func (ft *FooTester) AssertFooName(t testing.TB, id string, name string) {
	state := ft.GetFoo(t, id)
	if state.Data.Name != name {
		t.Fatalf("expected name %s, got %s", name, state.Data.Name)
	}
}
