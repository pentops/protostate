package psm

import (
	"context"
	"fmt"
	"testing"
)

type testK struct {
	IKeyset
}

type testS struct {
	IState[*testK, testST, *testSD]

	status testST
}

func (v testS) Status() testST {
	return v.status
}

type testST int32

func (v testST) ShortString() string {
	return fmt.Sprintf("S%d", v)
}

func (v testST) String() string {
	return fmt.Sprintf("TEST_STATUS_S%d", v)
}

type testSD struct {
	IStateData
}

type testE struct {
	IEvent[*testK, *testS, testST, *testSD, testIE]

	innerEvent testIE
}

func (v testE) UnwrapPSMEvent() testIE {
	return v.innerEvent
}

type testIE interface {
	IInnerEvent
}

type testIE1 struct {
	IPSMMessage
	data string
}

func (*testIE1) PSMEventKey() string {
	return "event1"
}

func TestTransitionBuilder(t *testing.T) {

	hs := &hookSet[*testK, *testS, testST, *testSD, *testE, testIE]{}

	gotTransitions := []string{}

	hs.From(testST(1)).OnEvent("event1").LogicHook(
		TransitionLogicHook[*testK, *testS, testST, *testSD, *testE, testIE, *testIE1](func(
			ctx context.Context,
			baton HookBaton[*testK, *testS, testST, *testSD, *testE, testIE],
			state *testS,
			event *testIE1,
		) error {
			gotTransitions = append(gotTransitions, "A:"+event.data)

			return nil
		}))

	hs.From().OnEvent("event1").LogicHook(
		TransitionLogicHook[*testK, *testS, testST, *testSD, *testE, testIE, *testIE1](func(
			ctx context.Context,
			baton HookBaton[*testK, *testS, testST, *testSD, *testE, testIE],
			state *testS,
			event *testIE1,
		) error {
			gotTransitions = append(gotTransitions, "B:"+event.data)

			return nil
		}))

	hs.From().LogicHook(
		TransitionLogicHook[*testK, *testS, testST, *testSD, *testE, testIE, *testIE1](func(
			ctx context.Context,
			baton HookBaton[*testK, *testS, testST, *testSD, *testE, testIE],
			state *testS,
			event *testIE1,
		) error {
			gotTransitions = append(gotTransitions, "C:"+event.data)

			return nil
		}))

	ctx := context.Background()

	state := &testS{
		status: 1,
	}

	event := &testE{
		innerEvent: &testIE1{
			data: "test",
		},
	}

	hookSet, err := hs.findTransitions(ctx, state.status, "event1")
	if err != nil {
		t.Fatal(err)
	}

	err = hookSet.runTransitionHooks(ctx, nil, nil, state, event)
	if err != nil {
		t.Fatal(err)
	}

	if len(gotTransitions) != 3 {
		t.Fatalf("unexpected transitions: %v", gotTransitions)
	}
	for idx, want := range []string{"A:test", "B:test", "C:test"} {
		if gotTransitions[idx] != want {
			t.Fatalf("unexpected transitions[%d]: %v", idx, gotTransitions[idx])
		}
	}

	t.Logf("transitions: %v", gotTransitions)

}
