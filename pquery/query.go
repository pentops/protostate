package pquery

import (
	"context"
	"fmt"

	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type aliasSet int

func (as *aliasSet) Next() string {
	*as++
	return fmt.Sprintf("_gc_alias_%d", *as)
}

func newAliasSet() *aliasSet {
	return new(aliasSet)
}

type Transactor interface {
	Transact(ctx context.Context, opts *sqrlx.TxOptions, callback sqrlx.Callback) error
}

type AuthProvider interface {
	AuthFilter(ctx context.Context) (map[string]interface{}, error)
}

type AuthProviderFunc func(ctx context.Context) (map[string]interface{}, error)

func (f AuthProviderFunc) AuthFilter(ctx context.Context) (map[string]interface{}, error) {
	return f(ctx)
}

// LeftJoin is a specification for joining in the form
// <TableName> ON <TableName>.<JoinKeyColumn> = <Main>.<MainKeyColumn>
// Main is defined in the outer struct holding this LeftJoin
type LeftJoin struct {
	TableName string
	On        JoinFields
}

// MethodDescriptor is the RequestResponse pair in the gRPC Method
type methodDescriptor[REQ proto.Message, RES proto.Message] struct {
	request  protoreflect.MessageDescriptor
	response protoreflect.MessageDescriptor
}

func newMethodDescriptor[REQ proto.Message, RES proto.Message]() *methodDescriptor[REQ, RES] {
	req := *new(REQ)
	res := *new(RES)
	return &methodDescriptor[REQ, RES]{
		request:  req.ProtoReflect().Descriptor(),
		response: res.ProtoReflect().Descriptor(),
	}
}
