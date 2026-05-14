# Preload 关联查询支持设计文档

## 1. 背景与目标

当前框架的 `Model[T]` 仅支持单表 CRUD，无法自动加载关联模型（如 `User` 的 `Orders`）。本设计旨在通过 **Schema 元数据驱动** 的方式，暴露 GORM 的 `Preload` 能力，使所有查询方法（`Detail`/`List`/`Paginate`/`Cursor`）都能自动加载配置的关联数据，并支持**多级嵌套**。

### 1.1 设计原则
- **Schema 驱动**：关联配置集中存储在 `Schema` 元数据中，与字段的 `Rules`、`Attributes` 风格一致
- **零代码侵入**：业务模型无需修改，通过 Schema 表配置即可开关关联加载
- **Formatter 统一处理**：关联数据与普通字段一样经过 `Formatter` 格式化输出
- **嵌套支持**：关联模型自身也可以配置关联，递归自动加载

### 1.2 非目标
- 不支持带条件的 Preload（如 `Preload("Orders", "status = ?", "active")`），后续迭代扩展
- 不支持关联数据的分页/排序，关联数据整体加载
- 不引入独立的关联配置表，保持 Schema 单表管理所有元数据

---

## 2. Schema 结构体扩展

在 `schema/schema.go` 的 `Schema` 结构体中新增 `Relations` 复合字段，与 `Rules Rule`、`Attributes Attribute` 保持一致风格：

```go
type Schema struct {
    // ... 现有字段保持不变
    Relations Relation `json:"relations" gorm:"type:varchar(2048)"`
}

type Relation struct {
    Type   string `json:"type"`   // has_one / has_many / belongs_to / many_to_many
    Name   string `json:"name"`   // GORM 关联字段名，如 "Orders"
    Module string `json:"module"` // 关联目标模块名，用于查找关联模型的 Schema
    Table  string `json:"table"`  // 关联目标表名，用于查找关联模型的 Schema
}
```

`Relation` 需实现 `driver.Valuer` / `sql.Scanner`，通过 JSON 序列化存储到数据库，与 `Rule`、`Attribute`、`Scenarios` 实现方式一致。

**为什么用复合字段而非平铺字段？**
- 关联配置是一个内聚概念，与现有 `Rules`、`Attributes` 风格统一
- 未来扩展（如条件表达式、排序）只需修改 `Relation` 结构体，无需改表结构

---

## 3. 关联配置自动发现

在 `schema/migrate.go` 的 `AutoMigrate` 中，新增 `parseFieldRelations` 函数，解析模型 struct 的 GORM 关联标签，自动写入 `Schema` 记录：

```go
func parseFieldRelations(field *schema.Field) Relation {
    var rel Relation
    // 通过 GORM schema.Field 的 Relationships 信息自动推断
    // 或从自定义 tag "relation" 中解析
    // 例如：relation:"has_many:Orders:order:orders"
    return rel
}
```

如果字段在 `Schema` 表中已存在且 `Relations` 不为空，则保留已有配置（允许人工覆盖自动推断结果）。

---

## 4. 查询层集成（Model[T]）

### 4.1 applyPreloads 方法

在 `model.go` 中新增内部方法，递归构建 GORM 的链式 Preload 路径：

`Detail`、`List`、`Paginate`、`Cursor` 在发起查询前调用此方法：

```go
// 以 Detail 为例
schemas, _ := schema.GetVisibleSchemas(ctx, m.GetDB(), m.naming.ModuleName, m.naming.TableName, schema.ScenarioDetail)
db := m.applyPreloads(ctx, m.GetDB().WithContext(ctx), schemas, schema.ScenarioDetail, make(map[string]bool), "")
err = db.Where(...).First(model).Error
```

### 4.2 嵌套递归实现

`applyPreloads` 通过递归构建 GORM 的链式 Preload 路径来实现嵌套关联加载。

实现策略：

1. 遍历当前模型的可见 Schema，提取 `Relations.Type != ""` 的字段
2. 对每个关联调用 `db.Preload(relation.Name)`
3. 根据 `Relation.Module` + `Relation.Table` 查询关联模型的可见 Schema
4. 对关联模型的 Schema 继续递归，构建子路径（如 `Orders.Items`）
5. 使用 `db.Preload(path)` 一次性注册完整链式路径

```go
func (m *Model[T]) applyPreloads(ctx context.Context, db *gorm.DB, schemas []schema.Schema, scenario string, visited map[string]bool, prefix string) *gorm.DB {
    for _, s := range schemas {
        if s.Relations.Type == "" {
            continue
        }
        path := s.Relations.Name
        if prefix != "" {
            path = prefix + "." + path
        }
        db = db.Preload(path)

        // 循环引用检测
        key := s.Relations.Module + ":" + s.Relations.Table + ":" + path
        if visited[key] {
            continue
        }
        visited[key] = true

        // 加载关联模型的 Schema 并递归
        assocSchemas, _ := schema.GetVisibleSchemas(ctx, m.GetDB(), s.Relations.Module, s.Relations.Table, scenario)
        db = m.applyPreloads(ctx, db, assocSchemas, scenario, visited, path)
    }
    return db
}
```

**Schema 缓存**：使用 `sync.Map` 缓存 `(module, table, scenario) → []Schema`，避免重复查询数据库。

---

## 5. Formatter 递归处理

`Formatter.FormatModel` 当前仅处理一层普通字段，需增强以支持关联字段递归格式化：

```go
func (f *Formatter) FormatModel(ctx context.Context, refValue reflect.Value,
    schemas []schema.Schema, stmt *gorm.Statement, format string) any {

    values := make(map[string]any)
    indirectValue := reflect.Indirect(refValue)

    for _, scm := range schemas {
        if scm.Relations.Type != "" {
            assocValue := f.getModelValue(indirectValue, scm, stmt)
            if assocValue == nil {
                values[scm.Column] = nil
                continue
            }
            // 加载关联模型的 Schema（通过 schema.GetVisibleSchemas）
            assocSchemas, _ := schema.GetVisibleSchemas(ctx, /*db*/, scm.Relations.Module, scm.Relations.Table, schema.ScenarioDetail)
            // 临时解析关联模型的 GORM Statement，用于字段查找
            assocStmt := parseAssocStatement(assocValue)

            assocRef := reflect.ValueOf(assocValue)
            if assocRef.Kind() == reflect.Slice {
                // has_many → 递归 FormatModels
                values[scm.Column] = f.FormatModels(ctx, assocValue, assocSchemas, assocStmt, format)
            } else {
                // has_one / belongs_to → 递归 FormatModel
                values[scm.Column] = f.FormatModel(ctx, assocRef, assocSchemas, assocStmt, format)
            }
        } else {
            // 普通字段：原有逻辑
            values[scm.Column] = f.Format(ctx, scm.Format, f.getModelValue(indirectValue, scm, stmt), modelValue, scm)
        }
    }
    return values
}
```

### 5.1 空数据处理
- `has_many` 关联为空时，返回 `[]`（空切片）
- `has_one` / `belongs_to` 关联为空时，返回 `null`

---

## 6. 数据流

```
HTTP Request
    → Resource.GetRuntimeScope
        → schema.GetVisibleSchemas (当前模型)
            → Model.Detail / List / Paginate / Cursor
                → applyPreloads (递归构建 GORM Preload 链)
                    → GORM Query (自动 JOIN / 子查询加载关联)
                        → Formatter.FormatModel (递归格式化关联数据)
                            → JSON Response
```

---

## 7. 错误处理

| 场景 | 行为 |
|---|---|
| 关联模型的 Schema 未找到 | 跳过该 Preload，记录 warn log，不阻断主查询 |
| 循环引用检测命中 | 跳过该路径，防止栈溢出 |
| 关联字段在 struct 中不存在 | GORM 会返回错误，转为 `ErrUnavailable` 返回 |
| 关联数据为空 | `has_many` 返回 `[]`，`has_one` 返回 `null` |

---

## 8. 与现有代码的集成点

| 文件 | 修改内容 |
|---|---|
| `schema/types.go` | 新增 `Relation` 类型定义，注册到常量/枚举中 |
| `schema/schema.go` | `Schema` 结构体增加 `Relations Relation` 字段 |
| `schema/migrate.go` | 新增 `parseFieldRelations`，在 `AutoMigrate` 中调用 |
| `model.go` | 新增 `applyPreloads` 方法；修改 `Detail`/`List`/`Paginate`/`Cursor` 调用之 |
| `formats/formatter.go` | `FormatModel` 增加关联字段递归处理逻辑 |
| `query/query.go` | 无需修改，GORM 的 `Find`/`Take`/`First` 自动处理 Preload |

---

## 9. 测试策略

- **单元测试**：`applyPreloads` 的 Schema 遍历与递归逻辑
- **集成测试**：创建 `User` + `Order` 模型，配置 Schema 关联，验证 `Detail`/`List` 返回嵌套结构
- **边界测试**：循环引用检测、空关联返回、深层嵌套（3 级以上）
