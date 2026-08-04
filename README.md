# REST

一个基于 GORM 的 Go 语言 RESTful 数据层框架，提供泛型模型驱动、动态 Schema 管理、类型安全查询构建等能力，帮助快速构建可配置、可扩展的数据访问层。

---

## 特性

- **泛型 TypedModel[T]** — 通过泛型为任意结构体一键生成类型安全的 CRUD 接口（Create / Update / Delete / Detail / Search）
- **动态 Model** — 模型类型在运行时才确定时使用（批量实例化、动态发现、混合类型集合），CRUD 结果通过类型断言取回
- **动态 Schema 管理** — 基于结构体标签自动生成字段配置，支持场景化字段控制（创建、更新、列表、详情、搜索、导出等）
- **类型安全查询构建器** — 支持 WHERE、JOIN、GROUP BY、HAVING、ORDER BY、分页、子查询（IN / EXISTS）、聚合函数（COUNT / SUM / AVG / MAX / MIN）
- **数据格式化器** — 内置多种字段格式（日期、时间、百分比、时长、下拉选项等），支持自定义扩展
- **多租户支持** — 内置 `tenant_id` 字段隔离机制
- **生命周期钩子系统** — 支持全局 any 注册与局部泛型注册，覆盖 Before/After Create/Update/Delete 和 AfterSaved，事务内阻断 + 事务外副作用
- **运行时作用域** — 通过 Context 传递模块、表、场景等运行时信息，便于插件和钩子扩展
- **SQL 注入防护** — 查询构建器内置字段名校验和非法字符过滤

---

## 快速开始

### 安装

```bash
go get git.nobla.cn/golang/rest
```

### 基础示例

```go
package main

import (
    "context"
    "fmt"

    "git.nobla.cn/golang/rest"
    "git.nobla.cn/golang/rest/query"
    "git.nobla.cn/golang/rest/schema"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type User struct {
    ID        uint64 `gorm:"primaryKey" json:"id"`
    Name      string `gorm:"size:120" json:"name" comment:"姓名"`
    Email     string `gorm:"size:255" json:"email" comment:"邮箱"`
    Age       int    `json:"age" comment:"年龄"`
    CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func main() {
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

    ctx := context.Background()

    // 创建泛型模型
    model, err := rest.NewTypedModel[User](
        rest.WithDB(db),
        rest.WithModuleName("user"),
    )
    if err != nil {
        panic(err)
    }

    // 创建记录
    user := User{Name: "Alice", Email: "alice@example.com", Age: 30}
    diff, err := model.Create(ctx, &user)
    fmt.Println("Created:", diff)

    // 分页搜索记录
    q := query.NewBuilder().
        Where("age", query.OpGte, 18).
        OrderBy("created_at", "DESC")

    total, users, err := model.Paginate(ctx, 0, 10, q)
    fmt.Printf("Total: %d, Users: %+v\n", total, users)
}
```

---

## 项目结构

```
.
├── rest.go                  # 包入口
├── model.go                 # 动态 Model 定义与 CRUD 操作（非泛型核心）
├── model_typed.go           # 泛型 TypedModel[T] 类型安全包装
├── resource.go              # 动态 Resource 定义与 HTTP 处理
├── resource_typed.go        # 泛型 TypedResource[T] 类型安全包装
├── hook.go                  # 生命周期钩子注册系统
├── options.go               # 模型配置选项
├── scope.go                 # 运行时作用域（Context 传递）
├── tenant.go                # 多租户常量
├── types.go                 # 公共类型与错误定义
├── schema/                  # Schema 管理与迁移
│   ├── schema.go            # Schema 结构体定义
│   ├── migrate.go           # 自动迁移与字段解析
│   ├── scenarios.go         # 场景枚举与序列化
│   ├── rule.go              # 字段校验规则
│   ├── attribute.go         # 字段属性与枚举值
│   └── types.go             # 常量与枚举定义
├── query/                   # 查询构建器
│   ├── ast.go               # AST 数据类型（QuerySpec、Expr、Clause）
│   ├── builder.go           # 流式查询构建 API
│   ├── validator.go         # 查询规范校验（防注入）
│   ├── compiler.go          # AST 编译为 GORM 查询
│   └── query.go             # 查询执行器（Count / One / All / Page）
├── formats/                 # 数据格式化器
│   ├── formatter.go         # Formatter 注册与管理
│   └── builtin.go           # 内置格式化函数
├── plugins/                 # 插件系统（预留扩展接口）
└── internal/inflector/      # 单复数转换工具
```

---

## TypedModel[T] CRUD 操作

```go
// 创建
model.Create(ctx, &user)

// 更新（返回变更字段 diff）
model.Update(ctx, primaryKey, &user)

// 删除
model.Delete(ctx, primaryKey)

// 详情
model.Detail(ctx, primaryKey)

// 分页搜索
total, list, err := model.Paginate(ctx, page, size, queryBuilder)

// 游标分页
next, hasMore, list, err := model.Cursor(ctx, cursor, limit, queryBuilder)
```

### 动态 Model（运行时确定类型）

当模型类型在编译期无法确定（批量实例化、从配置/插件动态发现、混合类型集合）时，使用动态 Model：

```go
// 实例（值或指针）或 reflect.Type 均可作为模型
userModel, _ := rest.NewModel(&User{}, rest.WithDB(db), rest.WithModuleName("user"))
orderModel, _ := rest.NewModel(reflect.TypeOf(Order{}), rest.WithDB(db), rest.WithModuleName("order"))

// 批量实例化：循环注册任意多个模型
models := []any{&User{}, &Order{}, &Product{}}
for _, m := range models {
    model, err := rest.NewModel(m, rest.WithDB(db), rest.WithModuleName("shop"))
    if err != nil {
        panic(err)
    }
    rest.NewResource(model, rest.ResourceConfig{Router: router, Prefix: "/api/v1"}).Register()
}

// CRUD 结果通过类型断言取回
v, _ := userModel.Detail(ctx, 1)
user := v.(*User)

list, _ := userModel.List(ctx, 0, 10, nil)
users := list.([]*User)

// 动态模型的钩子使用 any 签名
userModel.RegisterBeforeCreate(func(ctx context.Context, db *gorm.DB, model any) error {
    if u, ok := model.(*User); ok && u.Email == "" {
        return errors.New("email required")
    }
    return nil
})
```

### 配置选项

| 选项 | 说明 |
|------|------|
| `WithDB(db)` | 绑定 GORM 数据库连接 |
| `WithModuleName(name)` | 设置模块名称（用于 Schema 隔离） |
| `WithTenant()` | 启用多租户模式 |
| `WithScenarios(...)` | 限制模型可用的场景 |

---

## HTTP Resource

将 `TypedModel[T]` 包装为 HTTP Resource，自动生成 RESTful 路由：

```go
model, _ := rest.NewTypedModel[User](rest.WithDB(db), rest.WithModuleName("user"))

userResource := rest.NewTypedResource(model, rest.ResourceConfig{
    Router:    router,         // 路由引擎
    Prefix:    "/api/v1",      // URL 前缀
    Formatter: formatter,      // 可选：数据格式化器
})
userResource.Register() // 自动注册 Create / Update / Delete / Detail / Search / Export / OpenAPI 路由
```

如果想合并「构造 Model」与「包装为 Resource」两步，可使用便捷函数 `NewTypedResourceWithOptions`，错误会一并返回：

```go
userResource, err := rest.NewTypedResourceWithOptions[User](rest.ResourceConfig{
    Router:    router,
    Prefix:    "/api/v1",
    Formatter: formatter,
},
    rest.WithDB(db),
    rest.WithModuleName("user"),
)
if err != nil {
    panic(err)
}
userResource.Register()
```

动态 Resource 同样可用，`NewResource` / `NewResourceWithOptions` 以模型实例（或 `reflect.Type`）为首参：

```go
res := rest.NewResource(model, rest.ResourceConfig{Router: router, Prefix: "/api/v1"})
res.Register()

res, err := rest.NewResourceWithOptions(&User{}, rest.ResourceConfig{Router: router, Prefix: "/api/v1"},
    rest.WithDB(db), rest.WithModuleName("user"))
```

`ResourceConfig` 字段：

| 字段 | 说明 |
|------|------|
| `Router` | 路由引擎（需实现 `Router` 接口） |
| `Prefix` | API URL 前缀 |
| `Responder` | 自定义响应处理器 |
| `Formatter` | 数据格式化器 |
| `TenantResolve` | 多租户解析函数 |
| `UserResolve` | 用户解析函数 |

### 路由注册守卫

- 当 `Formatter == nil` 时，**Export 路由不会被注册**（避免运行期出现 500 兜底）。
- `Register()` 是幂等的（基于 `atomic.Bool`），多次调用安全。
- 主键解析（`findPrimaryKey`）按 URL 段位匹配 `:id`，对多余前缀段、尾斜杠等都有回退。

---

## 生命周期钩子

支持注册式（非侵入式）生命周期钩子，**全局 any 注册**（所有模型共享）与**局部泛型注册**（单实例类型安全）双层设计。

### 支持的钩子阶段

| 阶段 | 执行时机 | 返回值 | 说明 |
|------|----------|--------|------|
| `BeforeCreate` | 事务内，Create 之前 | `error` | 失败阻断创建 |
| `AfterCreate` | 事务外，Create 之后 | 无 | 副作用，失败不阻断 |
| `BeforeUpdate` | 事务内，Update 之前 | `error` | 失败阻断更新 |
| `AfterUpdate` | 事务外，Update 之后 | 无 | 副作用，失败不阻断 |
| `AfterSaved` | 事务外，Create/Update 共用 | 无 | 副作用，失败不阻断 |
| `BeforeDelete` | 事务内，Delete 之前 | `error` | 失败阻断删除 |
| `AfterDelete` | 事务外，Delete 之后 | 无 | 副作用，失败不阻断 |

### 全局注册（any，所有模型共享）

```go
// 所有模型创建后都记录审计日志
rest.RegisterAfterCreate(func(ctx context.Context, db *gorm.DB, model any, diff []*rest.DiffAttr) {
    log.Printf("[AUDIT] %T created, diff: %d", model, len(diff))
})

// 所有模型删除前检查依赖
rest.RegisterBeforeDelete(func(ctx context.Context, db *gorm.DB, model any) error {
    switch m := model.(type) {
    case *User:
        if hasOrders(db, m.ID) {
            return errors.New("user has orders")
        }
    }
    return nil
})
```

### 局部注册（泛型，类型安全）

```go
model, _ := rest.NewTypedModel[User](rest.WithDB(db))

// 只有这个 model 实例会触发
model.RegisterAfterCreate(func(ctx context.Context, db *gorm.DB, u *User, diff []*rest.DiffAttr) {
    go notification.Send("user_created", u.ID)
})

model.RegisterBeforeUpdate(func(ctx context.Context, db *gorm.DB, u *User) error {
    if u.Email == "" {
        return errors.New("email required")
    }
    return nil
})
```

### 执行顺序

每个操作的执行顺序固定为：**全局钩子 → 局部钩子 → 接口式钩子**。

以 `Create` 为例：
1. 全局 `BeforeCreate` → 局部 `BeforeCreate`
2. GORM `tx.Create(model)`
3. 全局 `AfterCreate` → 局部 `AfterCreate`
4. 全局 `AfterSaved` → 局部 `AfterSaved`
5. 接口式 `AfterCreated` / `AfterSaved`（兼容现有接口）

### 与现有接口式钩子共存

注册式钩子与原有接口式钩子（`AfterCreated` / `AfterUpdated` / `AfterDeleted` / `AfterSaved`）**完全兼容**，两者可同时使用。

---

## 查询构建器

### 基本查询

```go
q := query.NewBuilder().
    Where("status", query.OpEq, "active").
    Where("age", query.OpGt, 18).
    OrderBy("created_at", "DESC").
    Limit(20).
    Offset(0)
```

### 模糊搜索

```go
q := query.NewBuilder().
    Like("name", "alice", query.LikeContains)
```

### 组合条件

```go
q := query.NewBuilder().
    WhereGroup(func(b *query.Builder) {
        b.Where("status", query.OpEq, "active").
          Where("age", query.OpGte, 18)
    }).
    OrWhereGroup(func(b *query.Builder) {
        b.Where("role", query.OpEq, "admin")
    })
```

### JOIN 查询

```go
q := query.NewBuilder().
    Select(query.Field("users.name"), query.Field("orders.amount")).
    From("users").
    LeftJoin("orders", "orders.user_id = users.id", nil).
    Where("orders.status", query.OpEq, "paid")
```

### 子查询

```go
// IN 子查询
q := query.NewBuilder().
    Where("user_id", query.OpIn, query.Subquery(func(sb *query.Builder) {
        sb.Select(query.Field("user_id")).From("orders").Where("amount", query.OpGt, 100)
    }))

// EXISTS
q := query.NewBuilder().
    WhereExists(func(sb *query.Builder) {
        sb.Select(query.Raw("1")).From("orders").
          Where("orders.user_id", query.OpEq, query.Raw("users.id"))
    })
```

### 聚合与 HAVING

```go
q := query.NewBuilder().
    Select(query.Field("status"), query.Sum("amount").As("total")).
    From("orders").
    GroupBy("status").
    Having("total", query.OpGt, 1000)
```

### 分页查询

```go
q := query.New(db, model, builder)
total, err := q.Count(ctx)
err = q.All(ctx, &results)
total, err = q.Page(ctx, 1, 20, &results)  // 第1页，每页20条
```

---

## Schema 字段标签

在模型结构体上添加标签，自动驱动 Schema 生成与行为控制：

```go
type Product struct {
    ID          uint64 `gorm:"primaryKey" json:"id"`
    Name        string `gorm:"size:120" json:"name" comment:"商品名称"`
    Price       float64 `json:"price" comment:"价格" format:"decimal"`
    Status      int     `json:"status" comment:"状态" enum:"0:禁用#red;1:启用#green"`
    CategoryID  uint64  `json:"category_id" comment:"分类" live:"url:/api/categories;method:GET;type:dropdown;columns:id,name"`
    IsVIP       bool    `json:"is_vip" comment:"VIP专属" props:"tag:el-switch"`
    Description string  `gorm:"size:4096" json:"description" comment:"描述" scenarios:"create;update;detail"`
    CreatedAt   int64   `json:"created_at" gorm:"autoCreateTime"`
}
```

### 常用标签

| 标签 | 说明 | 示例 |
|------|------|------|
| `comment` | 字段显示名称 | `comment:"用户名"` |
| `format` | 字段格式 | `format:"datetime"`、`format:"percentage"` |
| `enum` | 下拉枚举值 | `enum:"0:禁用#red;1:启用#green"` |
| `scenarios` | 可用场景 | `scenarios:"create;update;detail"` |
| `rule` | 校验规则 | `rule:"required:create,update;unique"` |
| `props` | 附加属性 | `props:"icon:User;match:exactly"` |
| `live` | 远程加载配置 | `live:"url:/api/list;method:GET;columns:id,name"` |
| `virtual` | 虚拟字段（非数据库列） | `virtual:""` |

### 内置场景

- `create` — 创建
- `update` — 更新
- `delete` — 删除
- `search` — 搜索条件
- `list` — 列表展示
- `detail` — 详情展示
- `export` — 导出
- `import` — 导入

---

## 数据格式化器

内置格式器支持：

| 格式 | 说明 |
|------|------|
| `string` | 字符串 |
| `integer` | 整数 |
| `decimal` | 小数 |
| `date` | 日期（2006-01-02） |
| `time` | 时间（15:04:05） |
| `datetime` | 日期时间 |
| `timestamp` | 时间戳转日期时间 |
| `percentage` | 百分比 |
| `duration` | 时长（HH:MM:SS） |
| `dropdown` | 下拉选项转标签 |

自定义格式器：

```go
f := formats.DefaultFormatter()
f.Register("myFormat", func(ctx context.Context, value any, model any, scm schema.Schema) any {
    return fmt.Sprintf("custom:%v", value)
})
```

---

## 运行时作用域

通过 Context 传递运行时信息，便于在 GORM Hook 或插件中访问：

```go
scope := rest.RuntimeScopeFromContext(ctx)
if scope != nil {
    fmt.Println(scope.ModuleName)  // 模块名
    fmt.Println(scope.TableName)   // 表名
    fmt.Println(scope.Scenario)    // 当前场景
    fmt.Println(scope.Schemas)     // 可见字段 Schema 列表
}
```

---

## 多租户

启用多租户后，模型自动在 Schema 和查询中注入 `tenant_id` 隔离条件：

```go
model, _ := rest.NewTypedModel[User](
    rest.WithDB(db),
    rest.WithModuleName("user"),
    rest.WithTenant(),  // 启用多租户
)
```

---

## 错误定义

```go
rest.ErrPermissionDenied   // 4003 — 无场景权限
rest.ErrRecordNotFound     // 4004 — 记录不存在
rest.ErrPayloadInvalid     // 1001 — 参数无效
rest.ErrCreateFailed       // 6001 — 创建失败
rest.ErrUpdateFailed       // 6002 — 更新失败
rest.ErrDeleteFailed       // 6003 — 删除失败
rest.ErrUnavailable        // 1003 — 内部错误
```

### 错误响应契约

所有 handler（`Create` / `Update` / `Delete` / `Detail` / `Search` / `Export`）的失败响应统一为 `application/json`：

| 触发条件              | HTTP 状态 | `Error.Code` | 含义 |
|-----------------------|-----------|--------------|------|
| `ErrPermissionDenied` | 403       | 4003         | 场景未启用 |
| `ErrRecordNotFound`   | 404       | 4004         | 记录不存在；`Detail` 也会把 `gorm.ErrRecordNotFound` 归一到这个 |
| `ErrPayloadInvalid`   | 400       | 1001         | 请求体无法解析 |
| `ErrCreateFailed`     | 500       | 6001         | 创建失败 |
| `ErrUpdateFailed`     | 500       | 6002         | 更新失败 |
| `ErrDeleteFailed`     | 500       | 6003         | 删除失败 |
| `ErrUnavailable`      | 500       | 1003         | 其它内部错误 |
| 其它 `error`          | 500       | 0            | 非框架错误；body 仅含 `reason` |

错误响应体：

```json
{ "code": 4004, "reason": "record not found" }
```

`OpenApi` 端点无论成功失败均返回 `Content-Type: application/json`，失败时响应体格式同上。

> 若配置了自定义 `Responder`，错误码映射由 Responder 自行负责，默认 `Respond` 行为不再生效。

---

## 依赖

- [GORM v2](https://gorm.io/) — ORM 框架
- [SQLite Driver](https://github.com/go-gorm/sqlite) — 测试与开发用数据库驱动

---

## 许可证

MIT
