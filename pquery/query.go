package pquery

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"gopkg.daemonl.com/sqrlx"
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

// MethodDescriptor is the RequestResponse pair in the gRPC Method
type MethodDescriptor[REQ proto.Message, RES proto.Message] struct {
	Request  REQ //protoreflect.MessageDescriptor
	Response RES //protoreflect.MessageDescriptor
}
