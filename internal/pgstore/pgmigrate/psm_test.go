package pgmigrate

import (
	"testing"

	"github.com/pentops/protostate/internal/testproto/gen/testpb"
	"github.com/pentops/protostate/psm"
)

func TestBuildStateMachineOneKey(t *testing.T) {

	fooSpec, err := psm.BuildQueryTableSpec(
		(&testpb.FooState{}).ProtoReflect().Descriptor(),
		(&testpb.FooEvent{}).ProtoReflect().Descriptor(),
	)
	if err != nil {
		t.Fatal(err)
	}

	barSpec, err := psm.BuildQueryTableSpec(
		(&testpb.BarState{}).ProtoReflect().Descriptor(),
		(&testpb.BarEvent{}).ProtoReflect().Descriptor(),
	)
	if err != nil {
		t.Fatal(err)
	}

	data, err := BuildStateMachineMigrations(fooSpec, barSpec)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(data))
}
