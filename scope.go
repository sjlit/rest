package rest

import (
	"context"
	"net/http"

	"git.nobla.cn/golang/rest/schema"
)

type scopeKey struct{}

var runtimeScopeKey = &scopeKey{}

type RuntimeScope struct {
	User            string
	ModuleName      string
	TableName       string
	Scenario        string
	Schemas         []schema.Schema
	Request         *http.Request
	PrimaryKeyValue any
	Context         context.Context
}

func WithRuntimeScope(ctx context.Context, scope *RuntimeScope) context.Context {
	return context.WithValue(ctx, runtimeScopeKey, scope)
}

func RuntimeScopeFromContext(ctx context.Context) *RuntimeScope {
	if v := ctx.Value(runtimeScopeKey); v != nil {
		if s, ok := v.(*RuntimeScope); ok {
			return s
		}
	}
	return nil
}
