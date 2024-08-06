package migrate

import (
	"github.com/pentops/protostate/internal/pgstore/pgmigrate"
	"github.com/pentops/protostate/psm"
)

func BuildStateMachineMigrations(specs ...psm.QueryTableSpec) ([]byte, error) {
	return pgmigrate.BuildStateMachineMigrations(specs...)
}
