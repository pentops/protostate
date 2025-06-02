package psm

import (
	"context"
	"fmt"
	"testing"

	"github.com/pentops/sqrlx.go/sqrlx"
	"github.com/stretchr/testify/assert"
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

func (v testS) PSMData() *testSD {
	return &testSD{
		IStateData: v,
	}
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

	hs := &transitionSet[*testK, *testS, testST, *testSD, *testE, testIE]{}

	gotTransitions := []string{}

	hs.From(testST(1)).OnEvent("event1").
		LogicHook(TransitionHook[*testK, *testS, testST, *testSD, *testE, testIE, *testIE1]{
			Callback: func(
				ctx context.Context,
				_ sqrlx.Transaction,
				baton CallbackBaton[*testK, *testS, testST, *testSD, *testE, testIE],
				state *testS,
				event *testE,
			) error {
				gotTransitions = append(gotTransitions, "A2")
				return nil
			},
			RunOnFollow: true,
		}).
		Mutate(TransitionMutation[*testK, *testS, testST, *testSD, *testE, testIE, *testIE1](func(
			state *testSD,
			event *testIE1,
		) error {
			gotTransitions = append(gotTransitions, "A1")
			return nil
		}))

	hs.From().OnEvent("event1").LogicHook(
		TransitionHook[*testK, *testS, testST, *testSD, *testE, testIE, *testIE1]{
			Callback: func(
				ctx context.Context,
				_ sqrlx.Transaction,
				baton CallbackBaton[*testK, *testS, testST, *testSD, *testE, testIE],
				state *testS,
				event *testE,
			) error {
				gotTransitions = append(gotTransitions, "B")
				return nil
			},
			RunOnFollow: true,
		})

	hs.From().LogicHook(
		TransitionHook[*testK, *testS, testST, *testSD, *testE, testIE, *testIE1]{
			Callback: func(
				ctx context.Context,
				_ sqrlx.Transaction,
				baton CallbackBaton[*testK, *testS, testST, *testSD, *testE, testIE],
				state *testS,
				event *testE,
			) error {
				gotTransitions = append(gotTransitions, "C")
				return nil
			},
			RunOnFollow: true,
		})

	ctx := context.Background()

	state := &testS{
		status: 1,
	}

	event := &testE{
		innerEvent: &testIE1{
			data: "e1",
		},
	}

	hookSet, err := hs.buildTransition(state.status, "event1")
	if err != nil {
		t.Fatal(err)
	}

	err = hookSet.runMutations(ctx, state, event)
	if err != nil {
		t.Fatal(err)
	}

	err = hookSet.runHooks(ctx, nil, nil, state, event)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("transitions: %v", gotTransitions)
	assert.Equal(t, []string{"A1", "A2", "B", "C"}, gotTransitions)

}
