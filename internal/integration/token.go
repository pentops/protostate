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
	fieldMap := map[string]string{
		"tenant":      "tenant_id",
		"meta_tenant": "meta_tenant_id",
	}
	return psm.StateQueryOptions{
		Auth: pquery.AuthProviderFunc(func(ctx context.Context) (map[string]string, error) {
			token, err := TokenFromCtx(ctx)
			if err != nil {
				return nil, err
			}
			keys := token.claim.Tenant

			filter := map[string]string{}
			for key, value := range keys {
				if field, exists := fieldMap[key]; !exists {
					return nil, fmt.Errorf("no field mapping for key %s", key)
				} else {
					filter[field] = value
				}
			}

			return filter, nil
		}),
	}
}
