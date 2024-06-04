package pgmigrate

import (
	"testing"

	"github.com/pentops/protostate/testproto/gen/testpb"
)

func TestBuildStateMachineOneKey(t *testing.T) {

	fooSpec := testpb.DefaultFooPSMTableSpec.StateTableSpec()
	barSpec := testpb.DefaultBarPSMTableSpec.StateTableSpec()

	data, err := BuildStateMachineMigrations(fooSpec, barSpec)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(data))
}
