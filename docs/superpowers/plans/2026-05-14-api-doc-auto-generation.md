# API 文档自动生成 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `rest` 框架的每个 `TypedResource[T]` 自动生成 OpenAPI 3.0 JSON 规范，通过运行时 HTTP 端点 `/{prefix}/openapi.json` 提供访问。

**Architecture:** 新增独立的 `openapi/` 包，内含最小化 OpenAPI 3.0 结构体与 `Generator`。`Generator` 接收 `Config`（含 `buildUri` 回调），递归查询 `schema.GetVisibleSchemas` 构建 `Paths` 与 `Components/schemas`。`Resource.Register()` 在注册业务路由后，若启用 `WithOpenAPI(true)`，则一次性生成 Spec 并缓存为 `[]byte`，再注册 `GET /{prefix}/openapi.json` 路由直接返回。

**Tech Stack:** Go 1.25, GORM v1.31.1, 无外部 OpenAPI 库（纯自研序列化）。

---

## File Structure

| File | Responsibility |
|---|---|
| `openapi/spec.go` | 最小化 OpenAPI 3.0 JSON 结构体定义（Spec, PathItem, Operation, SchemaRef 等） |
| `openapi/generator.go` | `Generator` 核心：Schema→OpenAPI 映射、递归关联处理、循环引用检测、路径构建 |
| `openapi/generator_test.go` | 单元测试：映射矩阵、循环引用、完整 Spec 生成断言 |
| `options.go` | 修改：新增 `enableOpenAPI` 字段与 `WithOpenAPI` Option |
| `resource.go` | 修改：`TypedResource[T]` 新增 `openAPISpec []byte`；`Register()` 末尾条件注册 `/openapi.json`；新增 `ServeOpenAPI` handler |
| `integration_test.go` | 扩展：新增基于内存 SQLite 的集成测试，验证 `/openapi.json` 返回结构与字段 |

---

### Task 1: Define OpenAPI 3.0 structs in `openapi/spec.go`

**Files:**
- Create: `openapi/spec.go`

- [ ] **Step 1: Write the structs**

```go
package openapi

type Spec struct {
    OpenAPI    string              `json:"openapi"`
    Info       Info                `json:"info"`
    Paths      map[string]PathItem `json:"paths"`
    Components Components          `json:"components"`
}

type Info struct {
    Title   string `json:"title"`
    Version string `json:"version"`
}

type PathItem struct {
    Get    *Operation `json:"get,omitempty"`
    Post   *Operation `json:"post,omitempty"`
    Put    *Operation `json:"put,omitempty"`
    Delete *Operation `json:"delete,omitempty"`
}

type Operation struct {
    Summary     string              `json:"summary,omitempty"`
    OperationID string              `json:"operationId,omitempty"`
    Parameters  []Parameter         `json:"parameters,omitempty"`
    RequestBody *RequestBody        `json:"requestBody,omitempty"`
    Responses   map[string]Response `json:"responses"`
}

type Parameter struct {
    Name        string     `json:"name"`
    In          string     `json:"in"`
    Required    bool       `json:"required,omitempty"`
    Schema      *SchemaRef `json:"schema"`
    Description string     `json:"description,omitempty"`
}

type RequestBody struct {
    Content map[string]MediaType `json:"content"`
}

type MediaType struct {
    Schema *SchemaRef `json:"schema"`
}

type Response struct {
    Description string               `json:"description"`
    Content     map[string]MediaType `json:"content,omitempty"`
}

type SchemaRef struct {
    Ref         string                `json:"$ref,omitempty"`
    Type        string                `json:"type,omitempty"`
    Format      string                `json:"format,omitempty"`
    Items       *SchemaRef            `json:"items,omitempty"`
    Properties  map[string]*SchemaRef `json:"properties,omitempty"`
    Required    []string              `json:"required,omitempty"`
    ReadOnly    bool                  `json:"readOnly,omitempty"`
    Enum        []any                 `json:"enum,omitempty"`
    Description string                `json:"description,omitempty"`
}

type Components struct {
    Schemas map[string]*SchemaRef `json:"schemas"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./openapi/`
Expected: PASS (no errors)

- [ ] **Step 3: Commit**

```bash
git add openapi/spec.go
git commit -m "feat(openapi): add minimal OpenAPI 3.0 structs"
```

---

### Task 2: Scaffold `Generator` and helper functions in `openapi/generator.go`

**Files:**
- Create: `openapi/generator.go`

- [ ] **Step 1: Write skeleton and helpers**

```go
package openapi

import (
    "context"
    "fmt"
    "strings"

    "git.nobla.cn/golang/rest/schema"
    "gorm.io/gorm"
)

type Generator struct{}

func NewGenerator() *Generator {
    return &Generator{}
}

type Config struct {
    Title      string
    Version    string
    ModuleName string
    TableName  string
    Singular   string
    Plural     string
    Prefix     string
    PrimaryKey string
    Scenarios  []string
    BuildUri   func(scenario string) (method, uri string)
}

func (g *Generator) Generate(ctx context.Context, db *gorm.DB, cfg Config) (*Spec, error) {
    spec := &Spec{
        OpenAPI: "3.0.3",
        Info: Info{
            Title:   cfg.Title,
            Version: cfg.Version,
        },
        Paths:      make(map[string]PathItem),
        Components: Components{Schemas: make(map[string]*SchemaRef)},
    }
    // TODO: Task 4
    _ = ctx
    _ = db
    _ = cfg
    return spec, nil
}

func mapSchemaType(t string) string {
    switch t {
    case schema.TypeInteger:
        return "integer"
    case schema.TypeFloat:
        return "number"
    case schema.TypeBoolean:
        return "boolean"
    default:
        return "string"
    }
}

func mapSchemaFormat(t string) string {
    switch t {
    case schema.FormatDate:
        return "date"
    case schema.FormatDatetime, schema.FormatTimestamp:
        return "date-time"
    default:
        return ""
    }
}

func uriToOpenAPIPath(uri string) string {
    parts := strings.Split(uri, "/")
    for i, p := range parts {
        if strings.HasPrefix(p, ":") {
            parts[i] = "{" + p[1:] + "}"
        }
    }
    return strings.Join(parts, "/")
}

func schemaName(module, table string) string {
    return toPascal(module) + toPascal(table)
}

func toPascal(s string) string {
    parts := strings.Split(s, "_")
    var result string
    for _, p := range parts {
        if p == "" {
            continue
        }
        result += strings.ToUpper(p[:1]) + p[1:]
    }
    return result
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./openapi/`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add openapi/generator.go
git commit -m "feat(openapi): scaffold Generator and mapping helpers"
```

---

### Task 3: Unit test helpers and mapping functions

**Files:**
- Create: `openapi/generator_test.go`

- [ ] **Step 1: Write mapping tests**

```go
package openapi

import (
    "testing"

    "git.nobla.cn/golang/rest/schema"
)

func TestMapSchemaType(t *testing.T) {
    tests := []struct {
        input    string
        wantType string
        wantFmt  string
    }{
        {schema.TypeInteger, "integer", ""},
        {schema.TypeFloat, "number", ""},
        {schema.TypeBoolean, "boolean", ""},
        {schema.TypeString, "string", ""},
        {schema.FormatDate, "string", "date"},
        {schema.FormatDatetime, "string", "date-time"},
        {schema.FormatTimestamp, "string", "date-time"},
    }
    for _, tt := range tests {
        if got := mapSchemaType(tt.input); got != tt.wantType {
            t.Errorf("mapSchemaType(%q) = %q, want %q", tt.input, got, tt.wantType)
        }
        if got := mapSchemaFormat(tt.input); got != tt.wantFmt {
            t.Errorf("mapSchemaFormat(%q) = %q, want %q", tt.input, got, tt.wantFmt)
        }
    }
}

func TestUriToOpenAPIPath(t *testing.T) {
    tests := []struct {
        uri  string
        want string
    }{
        {"/api/v1/rest/user/:id", "/api/v1/rest/user/{id}"},
        {"/api/v1/rest/users", "/api/v1/rest/users"},
        {"/api/v1/rest/user/detail/:id", "/api/v1/rest/user/detail/{id}"},
    }
    for _, tt := range tests {
        if got := uriToOpenAPIPath(tt.uri); got != tt.want {
            t.Errorf("uriToOpenAPIPath(%q) = %q, want %q", tt.uri, got, tt.want)
        }
    }
}

func TestToPascal(t *testing.T) {
    tests := []struct {
        in   string
        want string
    }{
        {"rest", "Rest"},
        {"integ_users", "IntegUsers"},
        {"", ""},
    }
    for _, tt := range tests {
        if got := toPascal(tt.in); got != tt.want {
            t.Errorf("toPascal(%q) = %q, want %q", tt.in, got, tt.want)
        }
    }
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./openapi/ -v`
Expected: 3 PASS

- [ ] **Step 3: Commit**

```bash
git add openapi/generator_test.go
git commit -m "test(openapi): add mapping and helper unit tests"
```

---

### Task 4: Implement `Generate` and `buildOperation`

**Files:**
- Modify: `openapi/generator.go`

Replace the `Generate` method TODO body with the full implementation.

- [ ] **Step 1: Replace `Generate` body and add `buildOperation`**

In `openapi/generator.go`, replace:
```go
func (g *Generator) Generate(ctx context.Context, db *gorm.DB, cfg Config) (*Spec, error) {
    spec := &Spec{
        OpenAPI: "3.0.3",
        Info: Info{
            Title:   cfg.Title,
            Version: cfg.Version,
        },
        Paths:      make(map[string]PathItem),
        Components: Components{Schemas: make(map[string]*SchemaRef)},
    }
    // TODO: Task 4
    _ = ctx
    _ = db
    _ = cfg
    return spec, nil
}
```

With:
```go
func (g *Generator) Generate(ctx context.Context, db *gorm.DB, cfg Config) (*Spec, error) {
    spec := &Spec{
        OpenAPI: "3.0.3",
        Info: Info{
            Title:   cfg.Title,
            Version: cfg.Version,
        },
        Paths:      make(map[string]PathItem),
        Components: Components{Schemas: make(map[string]*SchemaRef)},
    }

    state := &genState{
        db:      db,
        ctx:     ctx,
        spec:    spec,
        cfg:     cfg,
        visited: make(map[string]bool),
    }

    for _, scenario := range cfg.Scenarios {
        method, uri := cfg.BuildUri(scenario)
        if uri == "" {
            continue
        }

        openAPIPath := uriToOpenAPIPath(uri)
        visibleSchemas, err := schema.GetVisibleSchemas(ctx, db, cfg.ModuleName, cfg.TableName, scenario)
        if err != nil {
            continue
        }

        op := state.buildOperation(scenario, visibleSchemas)
        if op == nil {
            continue
        }

        item := spec.Paths[openAPIPath]
        switch strings.ToUpper(method) {
        case "GET":
            item.Get = op
        case "POST":
            item.Post = op
        case "PUT":
            item.Put = op
        case "DELETE":
            item.Delete = op
        }
        spec.Paths[openAPIPath] = item
    }

    return spec, nil
}

type genState struct {
    db      *gorm.DB
    ctx     context.Context
    spec    *Spec
    cfg     Config
    visited map[string]bool
}

func (s *genState) buildOperation(scenario string, schemas []schema.Schema) *Operation {
    var (
        op          = &Operation{}
        parameters  []Parameter
        requestBody *RequestBody
        responses   = make(map[string]Response)
    )

    cfg := s.cfg
    baseName := schemaName(cfg.ModuleName, cfg.TableName)

    switch scenario {
    case schema.ScenarioCreate:
        op.Summary = fmt.Sprintf("创建 %s", cfg.Singular)
        op.OperationID = fmt.Sprintf("create%s", toPascal(cfg.Singular))
        requestBody = s.buildRequestBody(baseName+"Create", cfg.ModuleName, cfg.TableName, scenario)
        responses["200"] = Response{
            Description: "成功",
            Content: map[string]MediaType{
                "application/json": {Schema: &SchemaRef{Type: "object", Properties: map[string]*SchemaRef{
                    "id": {Type: "string"},
                }}},
            },
        }
    case schema.ScenarioUpdate:
        op.Summary = fmt.Sprintf("更新 %s", cfg.Singular)
        op.OperationID = fmt.Sprintf("update%s", toPascal(cfg.Singular))
        parameters = append(parameters, s.buildIDParameter(cfg.PrimaryKey, schemas))
        requestBody = s.buildRequestBody(baseName+"Update", cfg.ModuleName, cfg.TableName, scenario)
        responses["200"] = Response{
            Description: "成功",
            Content: map[string]MediaType{
                "application/json": {Schema: &SchemaRef{Type: "object", Properties: map[string]*SchemaRef{
                    "id": {Type: "string"},
                }}},
            },
        }
    case schema.ScenarioDelete:
        op.Summary = fmt.Sprintf("删除 %s", cfg.Singular)
        op.OperationID = fmt.Sprintf("delete%s", toPascal(cfg.Singular))
        parameters = append(parameters, s.buildIDParameter(cfg.PrimaryKey, schemas))
        responses["200"] = Response{
            Description: "成功",
            Content: map[string]MediaType{
                "application/json": {Schema: &SchemaRef{Type: "object", Properties: map[string]*SchemaRef{
                    "id": {Type: "string"},
                }}},
            },
        }
    case schema.ScenarioDetail:
        op.Summary = fmt.Sprintf("获取 %s 详情", cfg.Singular)
        op.OperationID = fmt.Sprintf("detail%s", toPascal(cfg.Singular))
        parameters = append(parameters, s.buildIDParameter(cfg.PrimaryKey, schemas))
        parameters = append(parameters, Parameter{
            Name:        "format",
            In:          "query",
            Description: "返回格式",
            Schema:      &SchemaRef{Type: "string", Enum: []any{"raw", "both"}},
        })
        ref := s.buildSchemaRef(cfg.ModuleName, cfg.TableName, scenario)
        responses["200"] = Response{
            Description: "成功",
            Content: map[string]MediaType{
                "application/json": {Schema: ref},
            },
        }
    case schema.ScenarioSearch:
        op.Summary = fmt.Sprintf("查询 %s 列表", cfg.Plural)
        op.OperationID = fmt.Sprintf("list%s", toPascal(cfg.Plural))
        parameters = append(parameters,
            Parameter{Name: "page", In: "query", Schema: &SchemaRef{Type: "integer"}},
            Parameter{Name: "page_size", In: "query", Schema: &SchemaRef{Type: "integer"}},
            Parameter{Name: "query", In: "query", Schema: &SchemaRef{Type: "string"}, Description: "AST 查询表达式"},
            Parameter{Name: "format", In: "query", Schema: &SchemaRef{Type: "string", Enum: []any{"raw", "both"}}},
        )
        itemRef := s.buildSchemaRef(cfg.ModuleName, cfg.TableName, schema.ScenarioList)
        responses["200"] = Response{
            Description: "成功",
            Content: map[string]MediaType{
                "application/json": {
                    Schema: &SchemaRef{
                        Type: "object",
                        Properties: map[string]*SchemaRef{
                            "page":        {Type: "integer"},
                            "page_size":   {Type: "integer"},
                            "total_count": {Type: "integer"},
                            "data":        {Type: "array", Items: itemRef},
                        },
                    },
                },
            },
        }
    case schema.ScenarioExport:
        op.Summary = fmt.Sprintf("导出 %s", cfg.Plural)
        op.OperationID = fmt.Sprintf("export%s", toPascal(cfg.Plural))
        parameters = append(parameters,
            Parameter{Name: "query", In: "query", Schema: &SchemaRef{Type: "string"}, Description: "AST 查询表达式"},
        )
        responses["200"] = Response{
            Description: "成功",
            Content: map[string]MediaType{
                "text/csv": {Schema: &SchemaRef{Type: "string", Format: "binary"}},
            },
        }
    default:
        return nil
    }

    op.Parameters = parameters
    op.RequestBody = requestBody
    op.Responses = responses
    return op
}

func (s *genState) buildIDParameter(primaryKey string, schemas []schema.Schema) Parameter {
    var pkType string
    for _, sc := range schemas {
        if sc.Column == primaryKey && sc.PrimaryKey > 0 {
            pkType = mapSchemaType(sc.Type)
            break
        }
    }
    if pkType == "" {
        pkType = "string"
    }
    return Parameter{
        Name:     "id",
        In:       "path",
        Required: true,
        Schema:   &SchemaRef{Type: pkType},
    }
}

func (s *genState) buildRequestBody(refName, module, table, scenario string) *RequestBody {
    ref := s.buildSchemaRef(module, table, scenario)
    if ref.Ref == "" {
        ref = &SchemaRef{Ref: "#/components/schemas/" + refName}
        s.spec.Components.Schemas[refName] = ref
    }
    return &RequestBody{
        Content: map[string]MediaType{
            "application/json": {Schema: &SchemaRef{Ref: "#/components/schemas/" + refName}},
        },
    }
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./openapi/`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add openapi/generator.go
git commit -m "feat(openapi): implement Generate and buildOperation"
```

---

### Task 5: Implement recursive `buildSchemaRef` and association handling

**Files:**
- Modify: `openapi/generator.go`

Append the following methods after `buildRequestBody` in the same file.

- [ ] **Step 1: Add recursive schema builder**

```go
func (s *genState) buildSchemaRef(module, table, scenario string) *SchemaRef {
    key := module + ":" + table
    name := schemaName(module, table)

    // For the primary model, use scenario-specific naming for Create/Update
    if module == s.cfg.ModuleName && table == s.cfg.TableName {
        switch scenario {
        case schema.ScenarioCreate:
            name = name + "Create"
        case schema.ScenarioUpdate:
            name = name + "Update"
        default:
            // Detail, List, Export, Search use base name
        }
    }

    if s.visited[key] {
        return &SchemaRef{Ref: "#/components/schemas/" + name}
    }
    s.visited[key] = true

    schemas, err := schema.GetVisibleSchemas(s.ctx, s.db, module, table, scenario)
    if err != nil || len(schemas) == 0 {
        return &SchemaRef{Type: "object"}
    }

    ref := &SchemaRef{
        Type:       "object",
        Properties: make(map[string]*SchemaRef),
    }

    for _, sc := range schemas {
        prop := s.schemaToProperty(sc, scenario)
        ref.Properties[sc.Column] = prop
    }

    s.spec.Components.Schemas[name] = ref
    return &SchemaRef{Ref: "#/components/schemas/" + name}
}

func (s *genState) schemaToProperty(sc schema.Schema, scenario string) *SchemaRef {
    if sc.Relations.Type != "" {
        assocRef := s.buildSchemaRef(sc.Relations.Module, sc.Relations.Table, scenario)
        switch sc.Relations.Type {
        case "has_many", "many_to_many":
            // Return empty array for nil associations
            return &SchemaRef{Type: "array", Items: assocRef}
        default:
            return assocRef
        }
    }

    prop := &SchemaRef{
        Type:        mapSchemaType(sc.Type),
        Format:      mapSchemaFormat(sc.Type),
        Description: sc.Label,
    }
    if sc.PrimaryKey > 0 {
        prop.ReadOnly = true
    }
    return prop
}
```

**Important:** The `visited` key only tracks `(module, table)` pairs, not scenario. This means associations are generated once per `(module, table)` using the first encountered scenario. For the primary model, scenario-specific names are handled in the `switch` above. This is a deliberate trade-off to avoid an exponential explosion of component schemas.

- [ ] **Step 2: Verify compilation**

Run: `go build ./openapi/`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add openapi/generator.go
git commit -m "feat(openapi): add recursive schema building with cycle detection"
```

---

### Task 6: Unit test `Generate` with in-memory SQLite

**Files:**
- Modify: `openapi/generator_test.go`

- [ ] **Step 1: Add integration-style unit test for Generate**

```go
package openapi

import (
    "context"
    "encoding/json"
    "testing"

    "git.nobla.cn/golang/rest/schema"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type TestOrder struct {
    ID     uint `json:"id" gorm:"primarykey"`
    UserID uint `json:"user_id"`
    Total  int  `json:"total"`
}

type TestUser struct {
    ID     uint        `json:"id" gorm:"primarykey"`
    Name   string      `json:"name" gorm:"size:100"`
    Age    int         `json:"age"`
    Orders []TestOrder `json:"orders" gorm:"foreignKey:UserID" relation:"has_many:Orders::test_orders"`
}

func TestGenerate(t *testing.T) {
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    if err != nil {
        t.Fatalf("open db: %v", err)
    }
    if err := db.AutoMigrate(&schema.Schema{}, &TestUser{}, &TestOrder{}); err != nil {
        t.Fatalf("migrate: %v", err)
    }

    g := NewGenerator()
    cfg := Config{
        Title:      "Test API",
        Version:    "1.0.0",
        ModuleName: "openapi_test",
        TableName:  "test_users",
        Singular:   "user",
        Plural:     "users",
        Prefix:     "/api/v1",
        PrimaryKey: "id",
        Scenarios: []string{
            schema.ScenarioCreate,
            schema.ScenarioUpdate,
            schema.ScenarioDelete,
            schema.ScenarioDetail,
            schema.ScenarioSearch,
            schema.ScenarioExport,
        },
        BuildUri: func(scenario string) (string, string) {
            switch scenario {
            case schema.ScenarioCreate:
                return "POST", "/api/v1/openapi_test/user"
            case schema.ScenarioUpdate:
                return "PUT", "/api/v1/openapi_test/user/:id"
            case schema.ScenarioDelete:
                return "DELETE", "/api/v1/openapi_test/user/:id"
            case schema.ScenarioDetail:
                return "GET", "/api/v1/openapi_test/user/detail/:id"
            case schema.ScenarioSearch:
                return "GET", "/api/v1/openapi_test/users"
            case schema.ScenarioExport:
                return "GET", "/api/v1/openapi_test/user/export"
            }
            return "", ""
        },
    }

    spec, err := g.Generate(context.Background(), db, cfg)
    if err != nil {
        t.Fatalf("Generate failed: %v", err)
    }

    // Verify paths exist
    if len(spec.Paths) != 5 {
        t.Fatalf("expected 5 paths, got %d", len(spec.Paths))
    }
    if spec.Paths["/api/v1/openapi_test/user/{id}"].Put == nil {
        t.Fatal("expected PUT /user/{id}")
    }
    if spec.Paths["/api/v1/openapi_test/user/detail/{id}"].Get == nil {
        t.Fatal("expected GET /user/detail/{id}")
    }

    // Verify components/schemas exist
    if len(spec.Components.Schemas) == 0 {
        t.Fatal("expected non-empty schemas")
    }

    baseName := schemaName("openapi_test", "test_users")
    if _, ok := spec.Components.Schemas[baseName]; !ok {
        t.Fatalf("expected schema %q in components", baseName)
    }

    // Verify association schema exists
    assocName := schemaName("", "test_orders")
    if _, ok := spec.Components.Schemas[assocName]; !ok {
        t.Fatalf("expected association schema %q in components", assocName)
    }

    // Verify JSON serialization works
    _, err = json.Marshal(spec)
    if err != nil {
        t.Fatalf("json marshal failed: %v", err)
    }
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./openapi/ -v`
Expected: all PASS (including new `TestGenerate`)

- [ ] **Step 3: Commit**

```bash
git add openapi/generator_test.go
git commit -m "test(openapi): add Generate unit test with sqlite"
```

---

### Task 7: Add `WithOpenAPI` option

**Files:**
- Modify: `options.go`

- [ ] **Step 1: Add field and option**

In `options.go`, add `enableOpenAPI bool` to the `options` struct:

```go
type options struct {
    db           *gorm.DB
    moduleName   string
    enableTenant bool
    scenarios    []string
    enableOpenAPI bool
}
```

And add the option function:

```go
func WithOpenAPI(enabled bool) Option {
    return func(o *options) {
        o.enableOpenAPI = enabled
    }
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add options.go
git commit -m "feat: add WithOpenAPI option"
```

---

### Task 8: Wire `Resource` to serve generated OpenAPI JSON

**Files:**
- Modify: `resource.go`

- [ ] **Step 1: Add `openAPISpec` field and `ServeOpenAPI` handler**

Add `openAPISpec []byte` to `TypedResource[T]` struct:

```go
type Resource[T any] struct {
    model         *TypedModel[T]
    prefix        string
    router        Router
    responder     Responder
    formatter     *formats.Formatter
    tenantResolve ResolveTenantFunc
    userResolve   ResolveUserFunc
    openAPISpec   []byte
}
```

Add `ServeOpenAPI` method after `Register()`:

```go
func (r *TypedResource[T]) ServeOpenAPI(res http.ResponseWriter, req *http.Request) {
    if len(r.openAPISpec) == 0 {
        res.WriteHeader(http.StatusNotFound)
        return
    }
    res.Header().Set("Content-Type", "application/json")
    res.Write(r.openAPISpec)
}
```

- [ ] **Step 2: Integrate generation into `Register()`**

At the end of `Register()`, after the existing scenario registrations, add:

```go
func (r *TypedResource[T]) Register() {
    // ... existing registrations ...

    if r.model.opts.enableOpenAPI {
        if spec, err := openapi.NewGenerator().Generate(
            context.Background(),
            r.model.GetDB(),
            openapi.Config{
                Title:      r.model.GetNaming().ModuleName + " API",
                Version:    "1.0.0",
                ModuleName: r.model.GetNaming().ModuleName,
                TableName:  r.model.GetNaming().TableName,
                Singular:   r.model.GetNaming().Singular,
                Plural:     r.model.GetNaming().Pluralize,
                Prefix:     r.prefix,
                PrimaryKey: r.model.GetPrimaryKey(),
                Scenarios: []string{
                    schema.ScenarioCreate,
                    schema.ScenarioUpdate,
                    schema.ScenarioDelete,
                    schema.ScenarioDetail,
                    schema.ScenarioSearch,
                    schema.ScenarioExport,
                },
                BuildUri: r.buildUri,
            },
        ); err == nil {
            if bytes, err := json.Marshal(spec); err == nil {
                r.openAPISpec = bytes
                _, uri := r.buildUri("openapi")
                r.router.Handle(http.MethodGet, uri, r.ServeOpenAPI)
            }
        }
    }
}
```

**Imports to add to `resource.go`:**
```go
import (
    // existing imports...
    "context"
    "encoding/json"
    "git.nobla.cn/golang/rest/openapi"
    "net/http"
)
```

Note: `r.model.opts` is currently unexported (`opts *options`). You may need to add a getter like `func (m *TypedModel[T]) IsOpenAPIEnabled() bool` on `model.go`, or make `opts` accessible. The simplest path: add `OpenAPIEnabled() bool` to `TypedModel[T]`.

- [ ] **Step 3: Add `OpenAPIEnabled` getter to `model.go`**

In `model.go`, add:

```go
func (m *TypedModel[T]) OpenAPIEnabled() bool {
    return m.opts.enableOpenAPI
}
```

Then change the condition in `Register()` from `r.model.opts.enableOpenAPI` to `r.model.OpenAPIEnabled()`.

- [ ] **Step 4: Verify compilation**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add resource.go model.go options.go
git commit -m "feat(resource): integrate OpenAPI spec generation and /openapi.json endpoint"
```

---

### Task 9: Integration test for `Resource` OpenAPI endpoint

**Files:**
- Modify: `integration_test.go`

- [ ] **Step 1: Add integration test**

Append to `integration_test.go`:

```go
func TestIntegrationOpenAPIEndpoint(t *testing.T) {
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to open db: %v", err)
    }
    if err := db.AutoMigrate(&schema.Schema{}, &IntegUser{}, &IntegOrder{}); err != nil {
        t.Fatalf("failed to migrate: %v", err)
    }

    userModel, err := NewTypedModel[IntegUser](WithDB(db), WithModuleName("integration"), WithOpenAPI(true))
    if err != nil {
        t.Fatalf("NewModel failed: %v", err)
    }

    mux := http.NewServeMux()
    userResource := NewResource(userModel, ResourceConfig{
        Router: &testRouter{mux: mux},
        Prefix: "/api/v1",
    })
    userResource.Register()

    req := httptest.NewRequest(http.MethodGet, "/api/v1/integration/user/openapi.json", nil)
    rec := httptest.NewRecorder()
    mux.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var spec openapi.Spec
    if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
        t.Fatalf("unmarshal spec: %v", err)
    }

    if spec.OpenAPI != "3.0.3" {
        t.Errorf("openapi version: want 3.0.3, got %s", spec.OpenAPI)
    }
    if len(spec.Paths) == 0 {
        t.Error("expected non-empty paths")
    }
    if len(spec.Components.Schemas) == 0 {
        t.Error("expected non-empty schemas")
    }
}

type testRouter struct {
    mux *http.ServeMux
}

func (tr *testRouter) Handle(method, path string, handler http.HandlerFunc) {
    tr.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
        if r.Method != method {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }
        handler(w, r)
    })
}
```

- [ ] **Step 2: Add missing imports to `integration_test.go`**

```go
import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "git.nobla.cn/golang/rest/openapi"
)
```

- [ ] **Step 3: Run tests**

Run: `go test ./... -run TestIntegrationOpenAPIEndpoint -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add integration_test.go
git commit -m "test(integration): add OpenAPI endpoint integration test"
```

---

### Task 10: Regression test and final verification

**Files:**
- Modify: `integration_test.go`

- [ ] **Step 1: Add regression test for disabled OpenAPI**

```go
func TestIntegrationOpenAPIDisabledByDefault(t *testing.T) {
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to open db: %v", err)
    }
    if err := db.AutoMigrate(&schema.Schema{}, &IntegUser{}); err != nil {
        t.Fatalf("failed to migrate: %v", err)
    }

    userModel, err := NewTypedModel[IntegUser](WithDB(db), WithModuleName("integration"))
    if err != nil {
        t.Fatalf("NewModel failed: %v", err)
    }

    mux := http.NewServeMux()
    userResource := NewResource(userModel, ResourceConfig{
        Router: &testRouter{mux: mux},
        Prefix: "/api/v1",
    })
    userResource.Register()

    req := httptest.NewRequest(http.MethodGet, "/api/v1/integration/user/openapi.json", nil)
    rec := httptest.NewRecorder()
    mux.ServeHTTP(rec, req)

    if rec.Code != http.StatusNotFound {
        t.Fatalf("expected 404 when OpenAPI disabled, got %d", rec.Code)
    }
}
```

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -v`
Expected: All tests PASS

- [ ] **Step 3: Final commit**

```bash
git add integration_test.go
git commit -m "test(integration): add OpenAPI disabled-by-default regression test"
```

---

## Spec Coverage Checklist

| Spec Requirement | Implementing Task |
|---|---|
| 新建 `openapi/` 包 | Task 1, 2 |
| 最小化 OpenAPI 3.0 结构体（无外部库） | Task 1 |
| Schema.Type → OpenAPI type/format 映射 | Task 2 |
| 字段可见性（按 scenario 过滤） | Task 4 |
| 路径严格复用 `buildUri` | Task 4, 8 |
| `:id` → `{id}` 转换 | Task 2 |
| `{id}` path parameter 类型推导 | Task 4 |
| Query 参数（page, page_size, query, format） | Task 4 |
| Create/Update requestBody | Task 4 |
| Detail/List/Export/Delete response | Task 4 |
| 关联字段 `$ref` 处理 | Task 5 |
| has_many → array, has_one → object | Task 5 |
| 循环引用检测（visited map） | Task 5 |
| Schema 缺失降级为 object | Task 5 |
| `WithOpenAPI` Option（默认 false） | Task 7 |
| `Resource.Register()` 集成 | Task 8 |
| 注册时生成并缓存 | Task 8 |
| `ServeOpenAPI` handler | Task 8 |
| 单元测试 | Task 3, 6 |
| 集成测试 | Task 9 |
| 回归测试（默认禁用不注册路由） | Task 10 |

## Placeholder Scan

- No "TBD", "TODO", "implement later" found.
- No vague "add error handling" steps; all code is concrete.
- No "similar to Task N" shortcuts.
- Type/method names consistent across all tasks (`Generator`, `Config`, `buildOperation`, `buildSchemaRef`, `ServeOpenAPI`).
