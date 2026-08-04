# API 文档自动生成设计文档

## 1. 背景与目标

本设计旨在为 `rest` 框架的每个 `TypedResource[T]` 自动生成符合 **OpenAPI 3.0** 规范的 JSON 文档，使前端开发者、测试人员及 API 消费者能够直接通过运行时 HTTP 端点获取准确的 API 契约。

### 1.1 设计原则

- **Schema 驱动**：API 文档的字段、类型、描述、关联关系全部来源于 `schema.Schema` 元数据，与运行时行为完全一致。
- **零代码侵入**：业务模型无需添加任何 tag，通过配置开关即可开启或关闭文档生成。
- **Resource 级独立文档**：每个 Resource 拥有独立的 `openapi.json`，不耦合其他模块。
- **运行时可用**：文档通过 HTTP 端点实时访问，而非构建时静态文件。

### 1.2 非目标

- 不内置 Swagger UI（仅输出 JSON Spec，由消费者自行导入 Postman/Apifox/Swagger Editor）。
- 不支持开发者通过 struct tag 自定义文档内容（纯自动推导，后续迭代可扩展）。
- 不生成 YAML 格式（仅 JSON，YAML 为后续可扩展项）。

---

## 2. 方案选择

**采用方案 B：Schema 元数据驱动（Schema-Metadata-Driven）**

从 `schema.GetVisibleSchemas` 读取每个 Resource 的字段元数据，结合 `Resource` 的路由信息生成 OpenAPI。

- 与框架"Schema 驱动"的核心理念完全一致。
- 生成的文档准确反映运行时行为（字段标签、字段类型、关联关系、场景开关等）。
- 直接复用现有 `schema.GetVisibleSchemas` 接口，实现聚焦、可靠。

---

## 3. `openapi` 包结构

新增独立的 `openapi/` 子包，职责单一：将框架元数据序列化为 OpenAPI 3.0 JSON。

### 3.1 核心结构

```go
// openapi/spec.go —— 最小化的 OpenAPI 3.0 对象，仅用于序列化
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

// ... Schema, Parameter, Operation 等最小定义
```

```go
// openapi/generator.go —— 生成器入口
type Generator struct{}

func (g *Generator) Generate(
    module, table, prefix string,
    scenarios map[string][]schema.Schema,
) (*Spec, error)
```

`Generate` 接收模块名、表名、路由前缀，以及按 scenario 分组的所有可见 Schema，输出可直接序列化为 JSON 的 `*Spec`。

**设计要点**：不引入第三方 OpenAPI 库（如 kin-openapi），避免增加依赖；只声明生成所需的最小字段集合。

---

## 4. Schema → OpenAPI 类型映射

| Schema.Type | OpenAPI type | OpenAPI format | 说明 |
|---|---|---|---|
| `string` | `string` | — | 基础字符串 |
| `integer` | `integer` | `int64` | 整型 |
| `decimal` | `number` | `double` | 浮点 |
| `date` | `string` | `date` | 日期 |
| `time` | `string` | — | 时间（OpenAPI 无标准 time 格式） |
| `datetime` / `timestamp` | `string` | `date-time` | 日期时间 |
| `duration` | `string` | — | 时长 |
| `dropdown` | `string` | — | 下拉选项，值类型统一按 string 处理 |
| `percentage` | `number` | — | 百分比 |

### 4.1 关联字段映射

- `has_one` / `belongs_to` → `{ "$ref": "#/components/schemas/{Module}{Table}" }`
- `has_many` / `many_to_many` → `{ "type": "array", "items": { "$ref": "#/components/schemas/{Module}{Table}" } }`

### 4.2 字段可见性

根据 `schema.Scenarios` 决定字段出现在哪些 operation 中。例如 `ScenarioCreate` 不可见的字段不会出现在 `POST` 的 requestBody 里。

---

## 5. Resource 路由 → OpenAPI Paths

OpenAPI 的 `paths` 键直接复用 `Resource.buildUri` 返回的 URI，确保文档路径与运行时注册的路由**严格一致**。`Generator` 接收 `(scenario string) → (method, uri)` 的映射结果，而非硬编码前缀。

以 `prefix="/api/v1", module="rest", singular="user", pluralize="users"` 为例：

| 方法 | 路径 | Operation | 说明 |
|---|---|---|---|
| `POST` | `/api/v1/rest/user` | `create` | 创建 |
| `PUT` | `/api/v1/rest/user/{id}` | `update` | 更新 |
| `DELETE` | `/api/v1/rest/user/{id}` | `delete` | 删除 |
| `GET` | `/api/v1/rest/users` | `list` | 查询列表 |
| `GET` | `/api/v1/rest/user/detail/{id}` | `detail` | 详情 |
| `GET` | `/api/v1/rest/user/export` | `export` | 导出 |

### 5.1 参数定义

- **`{id}`**（path parameter）：从主键字段的 `Schema.Type` 自动推导类型（如 `string` / `integer`），标记 `required: true`。
- **Query 参数**（仅 `list` / `export`）：
  - `page`: `integer`，分页页码
  - `size`: `integer`，每页条数
  - `query`: `string`，AST 查询表达式（直接透传 `query.Builder` 的输入）
  - `format`: `string`，`enum: [raw, both]`，控制返回格式（仅 `list` / `detail`）

### 5.2 Schema 引用规则

在 `components/schemas` 中按 scenario 生成独立的命名 Schema，确保 API 消费者看到的是精确的可用字段：

- `{Module}{SingularTable}` —— `Detail` / `List` / `Export` 响应体引用（包含对应 scenario 可见字段）
- `{Module}{SingularTable}Create` —— `create` 的 requestBody
- `{Module}{SingularTable}Update` —— `update` 的 requestBody

### 5.3 关联模型的处理

关联字段（`Relations.Type != ""`）在 Schema 中通过 `$ref` 指向目标模型的 Schema。如果关联模型自身也有关联，其 Schema 同样会被递归生成并注册到 `components/schemas` 中，且通过 `visited` 机制去重，避免循环引用。

### 5.4 Response 定义

- `list`: `{ total: integer, data: array<{Module}{SingularTable}> }`
- `create`/`update`/`detail`: 直接返回对应 Schema
- `delete`: 204 No Content 或标准成功响应
- `export`: `text/csv` 格式

---

## 6. 集成点与配置开关

### 6.1 配置 API

在 `options.go` 中新增：

```go
func WithOpenAPI(enabled bool) Option
```

默认值为 `false`，保持零侵入。

### 6.2 Resource 集成

`Resource.Register()` 在最后阶段，若启用 OpenAPI，则执行：

1. 调用 `openapi.NewGenerator().Generate(...)`，传入当前 Resource 的 `module/table/prefix` 和各 scenario 的 `[]schema.Schema`。
2. 将生成的 `*openapi.Spec` 一次性序列化为 `[]byte`，缓存在 `Resource` 实例中。
3. 注册路由：`GET /{prefix}/openapi.json`，直接返回缓存的 JSON，Content-Type 为 `application/json`。

```go
// resource.go Register() 中
if r.opts.enableOpenAPI {
    // 一次性生成，运行时无 DB 查询
    r.openAPISpec = generateAndCache(r.module, r.table, r.prefix, r.db)
    r.router.GET(r.prefix+"/openapi.json", r.serveOpenAPI)
}
```

**为什么注册时生成并缓存？**

- Schema 元数据在运行时不频繁变化，一次生成即可。
- 避免每次请求 `/openapi.json` 都触发数据库查询（即使有 `sync.Map` 缓存，`GetVisibleSchemas` 仍有上下文开销）。

### 6.3 错误处理

- 生成失败（如 Schema 未找到）时记录 Warn 日志，`Register()` 不返回错误、不阻断主流程，只是该 Resource 不暴露 `/openapi.json`。
- 关联模型的 Schema 缺失时，对应关联字段退化为 `type: object`（或 `array`），保证文档仍可渲染。

---

## 7. 边界情况

| 场景 | 行为 |
|---|---|
| **关联循环引用** | `Generator` 内部维护 `visited map[string]bool`，以 `Module:Table` 为 key，已访问的关联跳过，防止无限递归和栈溢出。 |
| **关联模型 Schema 缺失** | 若 `schema.GetVisibleSchemas` 返回空或错误，该关联字段在文档中退化为 `type: object`（`has_many` 则为 `type: array, items: {type: object}`），不阻断整体生成。 |
| **主键类型推断失败** | `{id}` path parameter 默认回退为 `type: string`，保证文档可用。 |
| **空 Schema 列表** | 若某 scenario 下无任何可见字段（极端情况），对应 operation 的 requestBody/response 生成空对象 `{}`。 |
| **JSON 序列化失败** | `Register()` 中生成失败仅记录 warn，不 panic，该 Resource 不暴露 `/openapi.json`。 |

---

## 8. 测试策略

1. **单元测试**：`openapi` 包独立测试，验证：
   - `Schema.Type` → OpenAPI `type/format` 的映射矩阵
   - 循环引用场景下 `visited` 去重逻辑
   - 关联字段正确生成 `$ref` 或 `array` + `$ref`
2. **集成测试**：基于现有集成测试框架，创建 `User` + `Order` Resource，启用 `WithOpenAPI(true)`，启动 HTTP 服务器，请求 `/users/openapi.json`，验证返回的 JSON 包含预期的 paths、components 和字段描述。
3. **回归测试**：确认 `WithOpenAPI(false)`（默认）时，没有任何额外路由被注册。

---

## 9. 与现有代码的集成点

| 文件 | 修改内容 |
|---|---|
| `openapi/spec.go` | 新增最小化的 OpenAPI 3.0 结构体定义 |
| `openapi/generator.go` | 新增 `Generator` 及 `Generate` 方法，实现递归 Schema 到 Spec 的转换 |
| `options.go` | 新增 `enableOpenAPI` 字段与 `WithOpenAPI` Option |
| `resource.go` | `Resource` 结构体增加 `openAPISpec []byte`；`Register()` 末尾条件注册 `/openapi.json` 路由 |

---

## 10. 数据流

```
应用启动
  → NewTypedModel[T](WithOpenAPI(true))
      → Resource.Register()
          → schema.GetVisibleSchemas (按 scenario 分组查询)
              → openapi.Generator.Generate()
                  → 递归构建 Spec (paths + components/schemas)
                      → json.Marshal 缓存到 Resource.openAPISpec
                          → 注册 GET /{prefix}/openapi.json
                              → HTTP 请求 /users/openapi.json
                                  → 直接返回缓存的 JSON Spec
```
