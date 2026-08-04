# ResourceConfig Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the generic `ResourceOption[T]` functional-option pattern with a plain non-generic `ResourceConfig` struct.

**Architecture:** Remove `ResourceOption[T]` and all `WithXxx[T]` helpers from `resource.go`. Introduce `ResourceConfig` struct with the same fields. Change `NewResource` to accept `ResourceConfig` instead of variadic options. Update all call sites in tests and docs.

**Tech Stack:** Go 1.25, GORM v2

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `resource.go` | Modify | Remove `ResourceOption[T]` and 6 `WithXxx[T]` functions. Add `ResourceConfig`. Change `NewResource` signature. |
| `integration_test.go` | Modify | Update 2 call sites (`TestIntegrationOpenAPIEndpoint`, `TestIntegrationOpenAPIDisabledByDefault`) to use `ResourceConfig`. |
| `docs/superpowers/plans/2026-05-14-api-doc-auto-generation.md` | Modify | Update 2 example code blocks that show `NewResource` usage. |

---

### Task 1: Replace ResourceOption with ResourceConfig in resource.go

**Files:**
- Modify: `resource.go:21-69` — Remove type and option helpers
- Modify: `resource.go:535-543` — Change `NewResource` signature and body

- [ ] **Step 1: Remove `ResourceOption[T]` and all `WithXxx[T]` helpers**

Replace lines 21-69 in `resource.go`:

```go
// Old lines 21-69 to delete entirely:
// type (
//     ResourceOption[T any] func(*TypedResource[T])
//     ...
// )
// func WithRouter[T any](...) ...
// ... etc

// New code (insert after the closing `)` of the existing `type` block at line 33):
type ResourceConfig struct {
    Router        Router
    Responder     Responder
    Formatter     *formats.Formatter
    Prefix        string
    TenantResolve ResolveTenantFunc
    UserResolve   ResolveUserFunc
}
```

The exact edit: delete lines 21-69 (the `ResourceOption` type and all 6 `WithXxx` functions), then add `ResourceConfig` as a standalone type.

Resulting structure at the top of `resource.go`:

```go
type (
    Resource[T any] struct {
        model         *TypedModel[T]
        prefix        string
        router        Router
        responder     Responder
        formatter     *formats.Formatter
        tenantResolve ResolveTenantFunc
        userResolve   ResolveUserFunc
    }
)

type ResourceConfig struct {
    Router        Router
    Responder     Responder
    Formatter     *formats.Formatter
    Prefix        string
    TenantResolve ResolveTenantFunc
    UserResolve   ResolveUserFunc
}
```

- [ ] **Step 2: Change `NewResource` signature and body**

Replace lines 535-543 in `resource.go`:

```go
// Old:
// func NewTypedResource[T any](model *TypedModel[T], opts ...ResourceOption[T]) *TypedResource[T] {
//     r := &TypedResource[T]{
//         model: model,
//     }
//     for _, cb := range opts {
//         cb(r)
//     }
//     return r
// }

// New:
func NewTypedResource[T any](model *TypedModel[T], cfg ResourceConfig) *TypedResource[T] {
    return &TypedResource[T]{
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

- [ ] **Step 3: Verify compilation of resource.go**

Run: `go build ./...`

Expected: PASS (or at least `resource.go` compiles; other files may fail until Task 2)

- [ ] **Step 4: Commit**

```bash
git add resource.go
git commit -m "feat(resource): replace ResourceOption[T] with ResourceConfig

Remove the generic functional-option pattern for Resource initialization.
All ResourceOption[T] helpers and the ResourceOption type are deleted.
NewResource now accepts a plain ResourceConfig struct.

BREAKING CHANGE: callers must migrate from variadic options to ResourceConfig."
```

---

### Task 2: Update integration tests to use ResourceConfig

**Files:**
- Modify: `integration_test.go:172-176` — `TestIntegrationOpenAPIEndpoint` Resource creation
- Modify: `integration_test.go:226-230` — `TestIntegrationOpenAPIDisabledByDefault` Resource creation

- [ ] **Step 1: Update `TestIntegrationOpenAPIEndpoint`**

Replace lines 172-176 in `integration_test.go`:

```go
// Old:
//     tr := &testRouter{}
//     userResource := NewResource(userModel,
//         WithRouter[IntegUser](tr),
//         WithPrefix[IntegUser]("/api/v1"),
//     )
//     userResource.Register()

// New:
    tr := &testRouter{}
    userResource := NewResource(userModel, ResourceConfig{
        Router: tr,
        Prefix: "/api/v1",
    })
    userResource.Register()
```

- [ ] **Step 2: Update `TestIntegrationOpenAPIDisabledByDefault`**

Replace lines 226-230 in `integration_test.go`:

```go
// Old:
//     tr := &testRouter{}
//     userResource := NewResource(userModel,
//         WithRouter[IntegUser](tr),
//         WithPrefix[IntegUser]("/api/v1"),
//     )
//     userResource.Register()

// New:
    tr := &testRouter{}
    userResource := NewResource(userModel, ResourceConfig{
        Router: tr,
        Prefix: "/api/v1",
    })
    userResource.Register()
```

- [ ] **Step 3: Run integration tests**

Run: `go test ./... -run TestIntegrationOpenAPI -v`

Expected: Both `TestIntegrationOpenAPIEndpoint` and `TestIntegrationOpenAPIDisabledByDefault` PASS.

- [ ] **Step 4: Run full test suite**

Run: `go test ./...`

Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add integration_test.go
git commit -m "test(integration): migrate Resource creation to ResourceConfig"
```

---

### Task 3: Update plan document example code

**Files:**
- Modify: `docs/superpowers/plans/2026-05-14-api-doc-auto-generation.md:981-986`
- Modify: `docs/superpowers/plans/2026-05-14-api-doc-auto-generation.md:1074-1079`

- [ ] **Step 1: Update first example block**

Replace lines 981-986:

```go
// Old:
//     mux := http.NewServeMux()
//     userResource := NewResource(userModel,
//         WithRouter[IntegUser](&testRouter{mux: mux}),
//         WithPrefix[IntegUser]("/api/v1"),
//     )
//     userResource.Register()

// New:
    mux := http.NewServeMux()
    userResource := NewResource(userModel, ResourceConfig{
        Router: &testRouter{mux: mux},
        Prefix: "/api/v1",
    })
    userResource.Register()
```

- [ ] **Step 2: Update second example block**

Replace lines 1074-1079:

```go
// Old:
//     mux := http.NewServeMux()
//     userResource := NewResource(userModel,
//         WithRouter[IntegUser](&testRouter{mux: mux}),
//         WithPrefix[IntegUser]("/api/v1"),
//     )
//     userResource.Register()

// New:
    mux := http.NewServeMux()
    userResource := NewResource(userModel, ResourceConfig{
        Router: &testRouter{mux: mux},
        Prefix: "/api/v1",
    })
    userResource.Register()
```

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/plans/2026-05-14-api-doc-auto-generation.md
git commit -m "docs(plan): update example code to use ResourceConfig"
```

---

### Task 4: Final verification

- [ ] **Step 1: Full build**

Run: `go build ./...`

Expected: Zero errors.

- [ ] **Step 2: Full test suite**

Run: `go test ./...`

Expected: All tests PASS.

- [ ] **Step 3: Confirm no remaining references to old API**

Run: `grep -r "ResourceOption\|WithRouter\|WithResponder\|WithFormatter\|WithPrefix\|WithTenantResolve\|WithUserResolve" --include="*.go" .`

Expected: No matches in `.go` files (except possibly in vendor/ if any).

- [ ] **Step 4: Commit (if any lingering fixes)**

If any fixes were needed, commit them. Otherwise this task is complete.

---

## Self-Review

**Spec coverage:**
- Delete `ResourceOption[T]` — Task 1, Step 1
- Delete all `WithXxx[T]` helpers — Task 1, Step 1
- Add `ResourceConfig` — Task 1, Step 1
- Change `NewResource` signature — Task 1, Step 2
- Update `integration_test.go` — Task 2
- Update plan doc examples — Task 3

**Placeholder scan:** None found. Every step contains exact code and commands.

**Type consistency:**
- `ResourceConfig` fields match `TypedResource[T]` fields exactly
- `NewTypedResource[T]` returns `*TypedResource[T]` unchanged
- Field names consistent across struct definition, assignment, and usage
