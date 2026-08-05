package rest

import (
	"context"

	"git.nobla.cn/golang/rest/v3/schema"
)

type scopeKey struct{}

var runtimeScopeKey = &scopeKey{}

type RuntimeScope struct {
	User            string //用户ID
	ModuleName      string //模块名称
	TableName       string //表名称
	Scenario        string //场景
	TenantID        string //租户
	Schemas         []schema.Schema
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
