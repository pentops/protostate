package integration

import (
	"context"
	"fmt"

	"github.com/pentops/o5-auth/gen/o5/auth/v1/auth_pb"
	"github.com/pentops/protostate/psm"
)

type tokenCtxKey struct{}

type token struct {
	claim *auth_pb.Claim
}

func (t *token) WithToken(ctx context.Context) context.Context {
	return context.WithValue(ctx, tokenCtxKey{}, t)
}

func TokenFromCtx(ctx context.Context) (*token, error) {
	token, exists := ctx.Value(tokenCtxKey{}).(*token)
	if !exists {
		return nil, fmt.Errorf("no token found")
	}

	return token, nil
}

func newTokenQueryStateOption() psm.StateQueryOptions {
	return psm.StateQueryOptions{
		Auth: psm.ClaimTenantProvider(func(ctx context.Context) (*auth_pb.Action, error) {
			token, err := TokenFromCtx(ctx)
			if err != nil {
				return nil, err
			}
			return &auth_pb.Action{
				Actor: &auth_pb.Actor{
					Claim: token.claim,
				},
			}, nil
		}),
	}
}
