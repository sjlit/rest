# ResourceConfig Design — 消除 ResourceOption 泛型选项模式

## Problem

Current `Resource[T]` initialization uses a generic option pattern:

```go
userResource := rest.NewResource(userModel,
    rest.WithRouter[IntegUser](tr),
    rest.WithPrefix[IntegUser]("/api/v1"),
    rest.WithFormatter[IntegUser](formatter),
)
```

With dozens of models, this creates three pain points:

1. **Type parameter noise**: Every shared option (`WithRouter`, `WithPrefix`, `WithFormatter`) requires writing `[T]` even though the parameters have nothing to do with `T`.
2. **Configuration repetition**: `router`, `prefix`, and `formatter` are identical across all resources but must be repeated for every model.
3. **Manual `Register()`**: Each resource must be individually registered after creation.

## Solution

Replace the generic `ResourceOption[T]` functional-option pattern with a plain **non-generic `ResourceConfig` struct**.

### Changes

#### 1. Remove `ResourceOption[T]` and all `WithXxx[T]` helpers

Delete from `resource.go`:

- `type ResourceOption[T any] func(*Resource[T])`
- `func WithRouter[T any](router Router) ResourceOption[T]`
- `func WithResponder[T any](responder Responder) ResourceOption[T]`
- `func WithFormatter[T any](formatter *formats.Formatter) ResourceOption[T]`
- `func WithPrefix[T any](prefix string) ResourceOption[T]`
- `func WithTenantResolve[T any](tenantResolve ResolveTenantFunc) ResourceOption[T]`
- `func WithUserResolve[T any](userResolve ResolveUserFunc) ResourceOption[T]`

#### 2. Add `ResourceConfig`

```go
package rest

type ResourceConfig struct {
    Router        Router
    Responder     Responder
    Formatter     *formats.Formatter
    Prefix        string
    TenantResolve ResolveTenantFunc
    UserResolve   ResolveUserFunc
}
```

#### 3. Change `NewResource` signature

```go
// Old (removed):
// func NewResource[T any](model *Model[T], opts ...ResourceOption[T]) *Resource[T]

// New:
func NewResource[T any](model *Model[T], cfg ResourceConfig) *Resource[T] {
    return &Resource[T]{
        model:         model,
        router:        cfg.Router,
        responder:     cfg.Responder,
        formatter:     cfg.Formatter,
        prefix:        cfg.Prefix,
        tenantResolve: cfg.TenantResolve,
        userResolve:   cfg.UserResolve,
    }
}
```

#### 4. Preserve `Resource[T]` internals

`Resource[T]` struct and all methods (`Register`, `Create`, `Update`, `Delete`, `Detail`, `Search`, `Export`, `OpenApi`, `Respond`, `buildUri`, `buildQuery`, `findPrimaryKey`, `getRuntimeScope`) remain unchanged.

### New Usage

```go
cfg := rest.ResourceConfig{
    Router:    router,
    Prefix:    "/api/v1",
    Formatter: formatter,
}

userResource  := rest.NewResource(userModel, cfg)
orderResource := rest.NewResource(orderModel, cfg)

userResource.Register()
orderResource.Register()
```

### Files to modify

| File | Change |
|------|--------|
| `resource.go` | Remove `ResourceOption[T]` and 6 `WithXxx[T]` functions. Add `ResourceConfig`. Modify `NewResource` signature. |
| `integration_test.go` | Update 2 call sites of `NewResource` to use `ResourceConfig`. |
| `docs/superpowers/plans/2026-05-14-api-doc-auto-generation.md` | Update 2 example code blocks. |

### Backward compatibility

**Not preserved.** The generic option pattern is completely removed. Callers must migrate to `ResourceConfig`.

### Benefits

1. **Zero type parameters** for resource configuration — no more `[IntegUser]`, `[Order]`, `[Product]` noise.
2. **Single shared config** — create one `ResourceConfig` and reuse it across all models.
3. **Smaller API surface** — 7 exported symbols removed, 1 added (`ResourceConfig`).
4. **Simpler mentally** — plain struct is easier to understand than functional options with phantom type parameters.
