package pquery

import (
	"context"
	"fmt"

	"github.com/pentops/sqrlx.go/sqrlx"
)

type aliasSet int

func (as *aliasSet) Next(name string) string {
	*as++
	return fmt.Sprintf("_%s__a%d", name, *as)
}

func newAliasSet() *aliasSet {
	return new(aliasSet)
}

type Transactor interface {
	Transact(ctx context.Context, opts *sqrlx.TxOptions, callback sqrlx.Callback) error
}

type AuthProvider interface {
	AuthFilter(ctx context.Context) (map[string]string, error)
}

type AuthProviderFunc func(ctx context.Context) (map[string]string, error)

func (f AuthProviderFunc) AuthFilter(ctx context.Context) (map[string]string, error) {
	return f(ctx)
}

// LeftJoin is a specification for joining in the form
// <TableName> ON <TableName>.<JoinKeyColumn> = <Main>.<MainKeyColumn>
// Main is defined in the outer struct holding this LeftJoin
type LeftJoin struct {
	TableName string
	On        JoinFields
}
