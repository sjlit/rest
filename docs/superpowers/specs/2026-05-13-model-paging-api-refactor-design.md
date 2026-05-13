# Model[T] 分页 API 重构设计

## 背景

当前 `Model[T].Find` 方法存在两个根本问题：

1. **强制 Count + All 双重查询**：即使调用方不需要总数，也会执行一次 `COUNT(*)`，大数据量下性能差
2. **副作用污染 Builder**：`Find` 内部直接修改 `queryBuilder` 的 `Offset/Limit`，对调用方不可见但危险

## 目标

- 移除 `Find`，替换为职责清晰的独立方法
- 支持三种分页模式：`List`（偏移分页）、`Paginate`（精确分页）、`Cursor`（游标分页）
- 消除 `Builder` 的副作用

## 设计决策

### 兼容性策略

v3 分支允许 **Breaking Change**，直接删除 `Find`。

### Builder 副作用修复策略

保持 `Builder` 可变语义不变，在执行层内部通过 `QuerySpec.Clone()` 深拷贝隔离修改。

理由：
- 改动面最小，不破坏现有 Builder 的链式调用习惯
- 副作用问题在执行层（`Query.Count/compile`）彻底解决
- 所有现有 Builder 测试无需改动

---

## API 设计

### 新增类型

```go
// PageResult 精确分页结果
package rest

type PageResult[T any] struct {
    Page       int    `json:"page"`        // 当前页码（从1开始）
    PageSize   int    `json:"page_size"`
    TotalCount int64  `json:"total_count"` // 精确总数
    TotalPages int    `json:"total_pages"` // 计算得出
    Data       []*T   `json:"data"`
}

// CursorResult 游标分页结果
type CursorResult[T any] struct {
    Data       []*T   `json:"data"`
    NextCursor string `json:"next_cursor,omitempty"` // 空表示无更多数据
    HasMore    bool   `json:"has_more"`
}
```

### Model[T] 新方法

```go
// List — 纯数据查询，不执行 Count
func (m *Model[T]) List(ctx context.Context, offset, limit int, qb *query.Builder) ([]*T, error)

// Paginate — 精确分页，内部自动 Count
func (m *Model[T]) Paginate(ctx context.Context, page, size int, qb *query.Builder) (*PageResult[T], error)

// Cursor — 游标分页，高性能，无总数
func (m *Model[T]) Cursor(ctx context.Context, cursor string, limit int, qb *query.Builder) (*CursorResult[T], error)

// Count — 独立计数
func (m *Model[T]) Count(ctx context.Context, qb *query.Builder) (int64, error)
```

### 删除的旧方法

```go
// 删除 Find
func (m *Model[T]) Find(ctx context.Context, offset, limit int, queryBuilder *query.Builder) (totalCount int64, values []*T, err error)
```

---

## 各方法行为定义

### List

- 参数校验：`offset < 0` 归一化为 `0`，`limit <= 0` 归一化为 `0`（表示无限制，由调用方控制）
- 不复用 `Find` 的旧逻辑（旧逻辑会强制 Count）
- 内部直接调用 `query.New(...).All()`

### Paginate

- 参数校验：`page < 1` 归一化为 `1`，`size <= 0` 归一化为 `20`
- 内部执行流程：
  1. `Count(ctx, qb)` 获取总数
  2. `List(ctx, (page-1)*size, size, qb)` 获取数据
  3. 组装 `PageResult`，计算 `TotalPages = ceil(TotalCount / size)`

### Cursor

- `cursor` 为空字符串时，表示从第一条开始
- `cursor` 格式：`base64(strconv.Itoa(offset))`
- 内部执行流程：
  1. 解码 `cursor` 得到 `offset`
  2. `List(ctx, offset, limit+1, qb)` 查询比实际需要多一条
  3. 判断 `hasMore = len(data) > limit`
  4. 如果有更多数据，`data = data[:limit]`，并生成 `NextCursor = base64(offset + limit)`

### Count

- 内部 Clone `qb.Spec()`，将 `Offset/Limit` 清零后执行 `query.New(...).Count()`
- 不修改原始 `Builder`

---

## Query 层改动

### QuerySpec.Clone

在 `query/ast.go` 新增：

```go
func (s QuerySpec) Clone() QuerySpec {
    cloned := QuerySpec{
        Source: s.Source,
        Limit:  s.Limit,
        Offset: s.Offset,
    }
    if len(s.Selects) > 0 {
        cloned.Selects = make([]Expr, len(s.Selects))
        copy(cloned.Selects, s.Selects)
    }
    if len(s.Joins) > 0 {
        cloned.Joins = make([]Join, len(s.Joins))
        copy(cloned.Joins, s.Joins)
    }
    if len(s.Where) > 0 {
        cloned.Where = make([]Clause, len(s.Where))
        copy(cloned.Where, s.Where)
    }
    if len(s.GroupBy) > 0 {
        cloned.GroupBy = make([]string, len(s.GroupBy))
        copy(cloned.GroupBy, s.GroupBy)
    }
    if len(s.Having) > 0 {
        cloned.Having = make([]Clause, len(s.Having))
        copy(cloned.Having, s.Having)
    }
    if len(s.OrderBy) > 0 {
        cloned.OrderBy = make([]Order, len(s.OrderBy))
        copy(cloned.OrderBy, s.OrderBy)
    }
    return cloned
}
```

### Query.compile 重构

在 `query/query.go`：

1. 保留 `compile(ctx)` 作为公开调用入口（供 `One/All/Page` 使用）
2. 新增 `compileWithSpec(ctx, spec)` 作为内部方法（供 `Count` 使用）
3. `Count` 方法改为：

```go
func (q *Query) Count(ctx context.Context) (int64, error) {
    spec := q.builder.Spec().Clone()
    spec.Offset = 0
    spec.Limit = 0

    db, err := q.compileWithSpec(ctx, spec)
    if err != nil {
        return 0, err
    }
    var count int64
    err = db.Count(&count).Error
    return count, err
}
```

---

## Resource 层适配

`resource.go` 的 `Search` 方法改为调用 `Paginate`：

```go
func (r *Resource[T]) Search(res http.ResponseWriter, req *http.Request) {
    // ... 前置逻辑不变 ...

    queryBuilder := r.buildQuery(req, schemas)

    result, err := r.model.Paginate(req.Context(), pageIndex+1, pageSize, queryBuilder)
    if err != nil {
        r.Respond(res, req, ErrUnavailable)
        return
    }

    if r.formatter != nil {
        formatted := r.formatter.FormatModels(req.Context(), result.Data, schemas, r.model.GetDB().Statement, "")
        // 构造新的响应结构，包含分页元信息
    }

    r.Respond(res, req, result)
}
```

---

## 迁移路径

现有用户从 `Find` 迁移：

```go
// 旧代码
 total, users, err := model.Find(ctx, 0, 10, qb)

// 新代码 — 需要精确分页
 result, err := model.Paginate(ctx, 1, 10, qb)
 // result.TotalCount, result.Data

// 新代码 — 只需要数据
 users, err := model.List(ctx, 0, 10, qb)

// 新代码 — 手动 Count + List
 total, err := model.Count(ctx, qb)
 users, err := model.List(ctx, 0, 10, qb)
```

---

## 影响范围

| 文件 | 改动 |
|------|------|
| `rest/types.go` | 新增 `PageResult[T]`、`CursorResult[T]` |
| `rest/model.go` | 删除 `Find`，新增 `List/Paginate/Cursor/Count` |
| `query/ast.go` | 新增 `QuerySpec.Clone()` |
| `query/query.go` | `Count` 改为内部 Clone，`compile` 拆出 `compileWithSpec` |
| `rest/resource.go` | `Search` 改为调用 `Paginate` |
| `query/query_test.go` | 删除 `TestPage` 对旧逻辑的依赖，改为测试新 API |

---

## 边界情况

1. **Cursor 为空**：`decodeCursor("")` 返回 `0`，从第一条开始
2. **Cursor 非法**：返回错误，由调用方处理
3. **Paginate size 过大**：不额外限制，由调用方或上层中间件控制
4. **List limit 为 0**：表示无 limit，查询全部（GORM 默认行为）
5. **Count 时 Builder 已有 Limit**：Clone 后清零，不影响原始 Builder 的 Limit
