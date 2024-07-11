package integration

import (
	"context"
	"fmt"

	"github.com/pentops/o5-auth/gen/o5/auth/v1/auth_pb"
	"github.com/pentops/protostate/pquery"
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
		Auth: pquery.AuthProviderFunc(func(ctx context.Context) (map[string]string, error) {
			token, err := TokenFromCtx(ctx)
			if err != nil {
				return nil, err
			}

			out := map[string]string{}
			for k, v := range token.claim.Tenant {
				out[fmt.Sprintf("%s_id", k)] = v
			}

			return out, nil

		}),
	}
}
