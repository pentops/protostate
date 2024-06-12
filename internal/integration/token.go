package integration

import (
	"context"
	"fmt"

	"github.com/pentops/protostate/pquery"
	"github.com/pentops/protostate/psm"
)

type tokenCtxKey struct{}

type token struct {
	tenantID string
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
		Auth: pquery.AuthProviderFunc(func(ctx context.Context) (map[string]interface{}, error) {
			authMap := map[string]interface{}{}

			token, err := TokenFromCtx(ctx)
			if err != nil {
				return nil, err
			}

			if token.tenantID != "" {
				authMap["tenant_id"] = token.tenantID
			}

			return authMap, nil
		}),
	}
}
