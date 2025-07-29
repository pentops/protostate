package psmigrate

import (
	"github.com/pentops/protostate/internal/pgstore/psmpg"
	"github.com/pentops/protostate/psm"
)

func BuildStateMachineMigrations(specs ...psm.QueryTableSpec) ([]byte, error) {
	return psmpg.BuildStateMachineMigrations(specs...)
}
