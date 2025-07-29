package integration

import (
	"context"
	"fmt"

	"github.com/pentops/j5/gen/j5/auth/v1/auth_j5pb"
	"github.com/pentops/protostate/pquery"
)

type tokenCtxKey struct{}

type token struct {
	claim *auth_j5pb.Claim
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

func tokenAuth() pquery.AuthProvider {
	fieldMap := map[string]string{
		"tenant":      "tenant_id",
		"meta_tenant": "meta_tenant_id",
	}
	return pquery.AuthProviderFunc(func(ctx context.Context) (map[string]string, error) {
		token, err := TokenFromCtx(ctx)
		if err != nil {
			return nil, err
		}

		tt, ok := fieldMap[token.claim.TenantType]
		if !ok {
			return nil, fmt.Errorf("no field mapping for tenant type %s", token.claim.TenantType)
		}

		return map[string]string{
			tt: token.claim.TenantId,
		}, nil
	})
}
