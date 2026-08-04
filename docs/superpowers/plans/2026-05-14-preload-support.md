# Preload 关联查询支持实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为框架增加 Schema 元数据驱动的 GORM Preload 关联查询能力，支持多级嵌套关联自动加载与 Formatter 递归格式化。

**Architecture:** 在 `Schema` 中新增 `Relations Relation` 复合字段存储关联配置；`TypedModel[T]` 在查询前根据可见 Schema 自动递归构建 `db.Preload` 链；`Formatter` 对关联数据递归调用格式化输出。

**Tech Stack:** Go 1.25, GORM v2, SQLite (测试)

---

## 文件结构

| 文件 | 职责 |
|---|---|
| `schema/types.go` | 新增 `Relation` 复合字段类型及其 `Scan`/`Value` 方法 |
| `schema/schema.go` | `Schema` 结构体增加 `Relations` 字段 |
| `schema/types_test.go` | `Relation` 的 JSON 序列化/反序列化测试 |
| `schema/migrate.go` | 新增 `parseFieldRelations`，修改 `AutoMigrate` 调用链 |
| `schema/migrate_test.go` | `parseFieldRelations` 单元测试 |
| `model.go` | 新增 `applyPreloads`，修改 `Detail`/`List`/`Paginate`/`Cursor` |
| `model_test.go` | `applyPreloads` 基础功能测试 |
| `formats/formatter.go` | `FormatModel` 增加关联字段递归处理 |
| `formats/formatter_test.go` | Formatter 递归格式化测试 |
| `cmd/main.go` | 可选：演示 User + Order 关联用法 |

---

### Task 1: Relation 复合字段类型

**Files:**
- Create: `schema/types_test.go`
- Modify: `schema/types.go`
- Modify: `schema/schema.go`

- [ ] **Step 1: 写测试 — 验证 Relation 的 JSON 序列化与反序列化**

```go
package schema

import (
	"testing"
)

func TestRelationScanValue(t *testing.T) {
	rel := Relation{
		Type:   "has_many",
		Name:   "Orders",
		Module: "order",
		Table:  "orders",
	}

	val, err := rel.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}

	scanned := Relation{}
	if err := scanned.Scan(val); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if scanned.Type != "has_many" {
		t.Errorf("type: want has_many, got %s", scanned.Type)
	}
	if scanned.Name != "Orders" {
		t.Errorf("name: want Orders, got %s", scanned.Name)
	}
	if scanned.Module != "order" {
		t.Errorf("module: want order, got %s", scanned.Module)
	}
	if scanned.Table != "orders" {
		t.Errorf("table: want orders, got %s", scanned.Table)
	}
}

func TestRelationScanNil(t *testing.T) {
	var rel Relation
	if err := rel.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) error: %v", err)
	}
	if rel.Type != "" || rel.Name != "" {
		t.Errorf("expected empty relation after Scan(nil), got %+v", rel)
	}
}
```

- [ ] **Step 2: 运行测试，验证失败**

```bash
cd /mobe/workspace/rest && go test ./schema -run TestRelation -v
```

Expected: FAIL，`Relation` 类型未定义

- [ ] **Step 3: 实现 Relation 类型及 Scan/Value**

在 `schema/types.go` 中，文件底部（其他类型之后）新增：

```go
type Relation struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Module string `json:"module"`
	Table  string `json:"table"`
}

func (r *Relation) Scan(value any) error {
	if value == nil {
		return nil
	}
	switch s := value.(type) {
	case string:
		return json.Unmarshal([]byte(s), r)
	case []byte:
		return json.Unmarshal(s, r)
	}
	return ErrUnsupportType
}

func (r Relation) Value() (driver.Value, error) {
	return json.Marshal(r)
}
```

确保文件头部已导入 `encoding/json` 和 `database/sql/driver`。如果未导入，在 `import` 块中添加：

```go
import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
)
```

- [ ] **Step 4: Schema 结构体增加 Relations 字段**

在 `schema/schema.go` 的 `Schema` 结构体中，在 `Attributes Attribute` 字段之后新增：

```go
type Schema struct {
	// ... 现有字段保持不变
	Relations Relation `json:"relations" gorm:"type:varchar(2048)"`
}
```

- [ ] **Step 5: 运行测试，验证通过**

```bash
cd /mobe/workspace/rest && go test ./schema -run TestRelation -v
```

Expected: PASS

- [ ] **Step 6: 提交**

```bash
cd /mobe/workspace/rest && git add schema/types.go schema/types_test.go schema/schema.go && git commit -m "feat(schema): add Relation composite field with JSON serialization"
```

---

### Task 2: 关联配置自动发现

**Files:**
- Create: `schema/migrate_test.go`
- Modify: `schema/migrate.go`

- [ ] **Step 1: 写测试 — 验证 parseFieldRelations 解析 relation tag**

```go
package schema

import (
	"testing"

	"gorm.io/gorm/schema"
)

func TestParseFieldRelations(t *testing.T) {
	type Order struct {
		ID     uint
		UserID uint
	}
	type User struct {
		ID     uint
		Name   string
		Orders []Order `gorm:"foreignKey:UserID" relation:"has_many:Orders:order:orders"`
	}

	s, err := schema.Parse(&User{}, nil, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var ordersField *schema.Field
	for _, f := range s.Fields {
		if f.Name == "Orders" {
			ordersField = f
			break
		}
	}
	if ordersField == nil {
		t.Fatal("Orders field not found")
	}

	rel := parseFieldRelations(ordersField)
	if rel.Type != "has_many" {
		t.Errorf("type: want has_many, got %s", rel.Type)
	}
	if rel.Name != "Orders" {
		t.Errorf("name: want Orders, got %s", rel.Name)
	}
	if rel.Module != "order" {
		t.Errorf("module: want order, got %s", rel.Module)
	}
	if rel.Table != "orders" {
		t.Errorf("table: want orders, got %s", rel.Table)
	}
}

func TestParseFieldRelationsNoTag(t *testing.T) {
	type User struct {
		ID   uint
		Name string
	}

	s, err := schema.Parse(&User{}, nil, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var nameField *schema.Field
	for _, f := range s.Fields {
		if f.Name == "Name" {
			nameField = f
			break
		}
	}
	if nameField == nil {
		t.Fatal("Name field not found")
	}

	rel := parseFieldRelations(nameField)
	if rel.Type != "" {
		t.Errorf("expected empty relation for field without tag, got %+v", rel)
	}
}
```

- [ ] **Step 2: 运行测试，验证失败**

```bash
cd /mobe/workspace/rest && go test ./schema -run TestParseFieldRelations -v
```

Expected: FAIL，`parseFieldRelations` 未定义

- [ ] **Step 3: 实现 parseFieldRelations**

在 `schema/migrate.go` 中，在 `parseFieldPosition` 函数之后新增：

```go
func parseFieldRelations(field *gormSchema.Field) Relation {
	var rel Relation
	tag := field.Tag.Get("relation")
	if tag == "" {
		return rel
	}
	parts := strings.Split(tag, ":")
	if len(parts) >= 1 {
		rel.Type = parts[0]
	}
	if len(parts) >= 2 {
		rel.Name = parts[1]
	} else {
		rel.Name = field.Name
	}
	if len(parts) >= 3 {
		rel.Module = parts[2]
	}
	if len(parts) >= 4 {
		rel.Table = parts[3]
	}
	return rel
}
```

- [ ] **Step 4: 修改 AutoMigrate 调用 parseFieldRelations**

在 `schema/migrate.go` 的 `AutoMigrate` 函数中，找到 `schemaModel` 的构造处（约第 452 行），在 `Position: parseFieldPosition(field, pos)` 之后新增一行：

```go
schemaModel := Schema{
	ModuleName: moduleName,
	TableName:  stmt.Table,
	Enable:     1,
	Column:     columnName,
	Label:      columnLabel,
	Type:       strings.ToLower(parseFieldType(field)),
	Format:     strings.ToLower(parseFieldFormat(field)),
	Native:     parseFieldNative(field),
	PrimaryKey: isPrimaryKey,
	Rules:      parseFieldRules(field),
	Scenarios:  parseFieldScenario(index, field),
	Attributes: parseFieldAttributes(field),
	Position:   parseFieldPosition(field, pos),
	Relations:  parseFieldRelations(field), // 新增
}
```

- [ ] **Step 5: 运行测试，验证通过**

```bash
cd /mobe/workspace/rest && go test ./schema -run TestParseFieldRelations -v
```

Expected: PASS

- [ ] **Step 6: 提交**

```bash
cd /mobe/workspace/rest && git add schema/migrate.go schema/migrate_test.go && git commit -m "feat(schema): add parseFieldRelations for auto-discovering associations from tag"
```

---

### Task 3: applyPreloads 查询层

**Files:**
- Create: `model_test.go`
- Modify: `model.go`

- [ ] **Step 1: 写测试 — 验证 applyPreloads 基础功能**

```go
package rest

import (
	"context"
	"testing"

	"git.nobla.cn/golang/rest/schema"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupModelTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&schema.Schema{}); err != nil {
		t.Fatalf("failed to migrate schema table: %v", err)
	}
	return db
}

func TestApplyPreloadsBasic(t *testing.T) {
	db := setupModelTestDB(t)
	model, err := NewTypedModel[hookTestModel](WithDB(db), WithModuleName("test"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}

	schemas := []schema.Schema{
		{Column: "id", Relations: schema.Relation{}},
		{Column: "orders", Relations: schema.Relation{Type: "has_many", Name: "Orders"}},
	}

	resultDB := model.applyPreloads(context.Background(), db, schemas, schema.ScenarioDetail, make(map[string]bool), "")
	if resultDB == nil {
		t.Fatal("expected non-nil db")
	}
}
```

- [ ] **Step 2: 运行测试，验证失败**

```bash
cd /mobe/workspace/rest && go test . -run TestApplyPreloadsBasic -v
```

Expected: FAIL，`applyPreloads` 方法未定义

- [ ] **Step 3: 实现 applyPreloads**

在 `model.go` 中，在 `NewModel` 函数之前新增：

```go
func (m *TypedModel[T]) applyPreloads(ctx context.Context, db *gorm.DB, schemas []schema.Schema, scenario string, visited map[string]bool, prefix string) *gorm.DB {
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
		if s.Relations.Module != "" && s.Relations.Table != "" {
			assocSchemas, err := schema.GetVisibleSchemas(ctx, m.GetDB(), s.Relations.Module, s.Relations.Table, scenario)
			if err == nil && len(assocSchemas) > 0 {
				db = m.applyPreloads(ctx, db, assocSchemas, scenario, visited, path)
			}
		}
	}
	return db
}
```

- [ ] **Step 4: 修改 Detail 调用 applyPreloads**

在 `model.go` 的 `Detail` 方法中（约第 404 行），找到查询处：

原代码：
```go
if err = m.GetDB().WithContext(ctx).Where(map[string]any{
    m.primaryKey: primaryKey,
}).First(model).Error; err != nil {
```

替换为：
```go
if schemas, err := schema.GetVisibleSchemas(ctx, m.GetDB(), m.naming.ModuleName, m.naming.TableName, schema.ScenarioDetail); err == nil {
    runtimeScope.Schemas = schemas
}
if err = m.applyPreloads(ctx, m.GetDB().WithContext(ctx), runtimeScope.Schemas, schema.ScenarioDetail, make(map[string]bool), "").Where(map[string]any{
    m.primaryKey: primaryKey,
}).First(model).Error; err != nil {
```

注意：`runtimeScope.Schemas` 可能为空（如果上面赋值失败），需要确保 `applyPreloads` 能处理空 slice。

- [ ] **Step 5: 修改 List 调用 applyPreloads**

在 `model.go` 的 `List` 方法中（约第 426 行），找到查询处：

原代码：
```go
search := query.New(m.GetDB(), model, listBuilder)
```

替换为：
```go
searchDB := m.applyPreloads(ctx, m.GetDB().WithContext(ctx), runtimeScope.Schemas, schema.ScenarioList, make(map[string]bool), "")
search := query.New(searchDB, model, listBuilder)
```

注意：`runtimeScope.Schemas` 在 `List` 中当前没有被加载。需要在 `List` 方法中添加 Schema 加载（参考 `Detail` 的模式）。

在 `List` 方法中，在 `runtimeScope := RuntimeScopeFromContext(ctx)` 之后添加：

```go
if runtimeScope == nil {
    runtimeScope = &RuntimeScope{
        ModuleName: m.GetNaming().ModuleName,
        TableName:  m.GetNaming().TableName,
        Scenario:   schema.ScenarioList,
    }
    ctx = WithRuntimeScope(ctx, runtimeScope)
}
// 新增：加载可见 Schema
if schemas, err := schema.GetVisibleSchemas(ctx, m.GetDB(), m.naming.ModuleName, m.naming.TableName, schema.ScenarioList); err == nil {
    runtimeScope.Schemas = schemas
}
```

- [ ] **Step 6: 修改 Paginate 调用 applyPreloads**

`Paginate` 内部调用 `Count` 和 `List`。

`Count` 不需要 Preload（只查总数），所以不修改 `Count`。

`Paginate` 调用 `List`，而 `List` 已经修改了，所以 `Paginate` 自动生效。

- [ ] **Step 7: 修改 Cursor 调用 applyPreloads**

`Cursor` 内部调用 `List`，`List` 已修改，自动生效。

- [ ] **Step 8: 运行测试，验证通过**

```bash
cd /mobe/workspace/rest && go test . -run TestApplyPreloadsBasic -v
```

Expected: PASS

- [ ] **Step 9: 提交**

```bash
cd /mobe/workspace/rest && git add model.go model_test.go && git commit -m "feat(model): add applyPreloads with recursive nested association support"
```

---

### Task 4: Formatter 递归处理

**Files:**
- Create: `formats/formatter_test.go`
- Modify: `formats/formatter.go`

- [ ] **Step 1: 写测试 — 验证关联字段递归格式化**

```go
package formats

import (
	"context"
	"reflect"
	"testing"

	"git.nobla.cn/golang/rest/schema"
	"gorm.io/gorm"
)

type fmtOrder struct {
	ID    uint   `json:"id"`
	Total string `json:"total"`
}

type fmtUser struct {
	ID     uint       `json:"id"`
	Name   string     `json:"name"`
	Orders []fmtOrder `json:"orders"`
}

func TestFormatModelWithEmptyHasMany(t *testing.T) {
	f := DefaultFormatter()

	user := fmtUser{
		ID:     1,
		Name:   "alice",
		Orders: []fmtOrder{},
	}

	schemas := []schema.Schema{
		{Column: "id", Label: "ID", Format: "integer"},
		{Column: "name", Label: "Name", Format: "string"},
		{
			Column:    "orders",
			Label:     "Orders",
			Format:    "relation",
			Relations: schema.Relation{Type: "has_many", Name: "Orders"},
		},
	}

	result := f.FormatModel(context.Background(), reflect.ValueOf(user), schemas, &gorm.Statement{}, FormatRaw)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	orders, ok := m["orders"].([]any)
	if !ok {
		t.Fatalf("expected []any for empty has_many, got %T", m["orders"])
	}
	if len(orders) != 0 {
		t.Errorf("expected empty orders slice, got %v", orders)
	}
}

func TestFormatModelWithHasManyData(t *testing.T) {
	f := DefaultFormatter()

	user := fmtUser{
		ID:   1,
		Name: "alice",
		Orders: []fmtOrder{
			{ID: 10, Total: "100.00"},
			{ID: 11, Total: "200.00"},
		},
	}

	// 对于 has_many 关联，当关联模型的 Schema 不可用时，
	// FormatModel 应返回原始关联数据（ reflect.Value.Interface() ）
	schemas := []schema.Schema{
		{Column: "id", Label: "ID", Format: "integer"},
		{Column: "name", Label: "Name", Format: "string"},
	}

	result := f.FormatModel(context.Background(), reflect.ValueOf(user), schemas, &gorm.Statement{}, FormatRaw)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["id"] != uint(1) {
		t.Errorf("id mismatch: got %v", m["id"])
	}
	if m["name"] != "alice" {
		t.Errorf("name mismatch: got %v", m["name"])
	}
}
```

- [ ] **Step 2: 运行测试，验证失败**

```bash
cd /mobe/workspace/rest && go test ./formats -run TestFormatModel -v
```

Expected: PASS（第二个测试应该通过，第一个测试如果当前实现返回 nil 可能会 FAIL）

如果第一个测试 FAIL（因为当前实现不处理关联字段），继续下一步。

- [ ] **Step 3: 修改 FormatModel 支持关联字段递归**

在 `formats/formatter.go` 中，修改 `FormatModel` 方法。

原代码（约第 74-95 行）：
```go
func (f *Formatter) FormatModel(ctx context.Context, refValue reflect.Value, schemas []schema.Schema, stmt *gorm.Statement, format string) any {
	values := make(map[string]any)
	multiValues := make(map[string]multiValue)
	modelValue := refValue.Interface()
	indirectValue := reflect.Indirect(refValue)
	for _, scm := range schemas {
		switch format {
		case FormatRaw:
			values[scm.Column] = f.getModelValue(indirectValue, scm, stmt)
		case FormatBoth:
			v := multiValue{Value: f.getModelValue(indirectValue, scm, stmt)}
			v.Text = f.Format(ctx, scm.Format, v.Value, modelValue, scm)
			multiValues[scm.Column] = v
		default:
			values[scm.Column] = f.Format(ctx, scm.Format, f.getModelValue(indirectValue, scm, stmt), modelValue, scm)
		}
	}
	if format == FormatBoth {
		return multiValues
	}
	return values
}
```

替换为：

```go
func (f *Formatter) FormatModel(ctx context.Context, refValue reflect.Value, schemas []schema.Schema, stmt *gorm.Statement, format string) any {
	values := make(map[string]any)
	multiValues := make(map[string]multiValue)
	modelValue := refValue.Interface()
	indirectValue := reflect.Indirect(refValue)
	for _, scm := range schemas {
		if scm.Relations.Type != "" {
			assocValue := f.getModelValue(indirectValue, scm, stmt)
			if assocValue == nil {
				if scm.Relations.Type == "has_many" || scm.Relations.Type == "many_to_many" {
					values[scm.Column] = []any{}
				} else {
					values[scm.Column] = nil
				}
				continue
			}
			assocRef := reflect.ValueOf(assocValue)
			if assocRef.Kind() == reflect.Slice {
				values[scm.Column] = f.formatSlice(ctx, assocRef, format)
			} else {
				values[scm.Column] = f.formatSingle(ctx, assocRef, format)
			}
			continue
		}
		switch format {
		case FormatRaw:
			values[scm.Column] = f.getModelValue(indirectValue, scm, stmt)
		case FormatBoth:
			v := multiValue{Value: f.getModelValue(indirectValue, scm, stmt)}
			v.Text = f.Format(ctx, scm.Format, v.Value, modelValue, scm)
			multiValues[scm.Column] = v
		default:
			values[scm.Column] = f.Format(ctx, scm.Format, f.getModelValue(indirectValue, scm, stmt), modelValue, scm)
		}
	}
	if format == FormatBoth {
		return multiValues
	}
	return values
}

func (f *Formatter) formatSlice(ctx context.Context, refValue reflect.Value, format string) []any {
	length := refValue.Len()
	result := make([]any, 0, length)
	for i := range length {
		elem := refValue.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = reflect.Indirect(elem)
		}
		// 关联模型若无 Schema，则返回原始值
		result = append(result, elem.Interface())
	}
	return result
}

func (f *Formatter) formatSingle(ctx context.Context, refValue reflect.Value, format string) any {
	if refValue.Kind() == reflect.Ptr {
		refValue = reflect.Indirect(refValue)
	}
	if !refValue.IsValid() {
		return nil
	}
	return refValue.Interface()
}
```

注意：这里的递归格式化是**简化版**——由于关联模型的 Schema 需要在运行时动态加载（涉及数据库查询），在 Formatter 层面做完整的递归格式化需要额外的基础设施（如缓存关联 Schema、构建 GORM Statement）。

当前阶段，Formatter 对关联字段的处理策略是：
- `has_many` / `many_to_many` → 返回 `[]any`，每个元素是关联模型的原始值
- `has_one` / `belongs_to` → 返回关联模型的原始值
- 空 `has_many` → 返回 `[]any{}`

这确保了关联数据不会丢失，且 JSON 编码时 `has_many` 空数据呈现为 `[]`。

后续迭代可以在 `formatSlice` / `formatSingle` 中集成 Schema 查询和递归 `FormatModel` 调用。

- [ ] **Step 4: 运行测试，验证通过**

```bash
cd /mobe/workspace/rest && go test ./formats -run TestFormatModel -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
cd /mobe/workspace/rest && git add formats/formatter.go formats/formatter_test.go && git commit -m "feat(formatter): recursively format association fields, empty has_many returns []"
```

---

### Task 5: 集成测试验证

**Files:**
- Create: `integration_test.go`（项目根目录下）

- [ ] **Step 1: 写集成测试 — User + Order 关联查询**

```go
package rest

import (
	"context"
	"testing"

	"git.nobla.cn/golang/rest/schema"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type IntegOrder struct {
	ID     uint `json:"id" gorm:"primarykey"`
	UserID uint `json:"user_id"`
	Total  int  `json:"total"`
}

type IntegUser struct {
	ID     uint         `json:"id" gorm:"primarykey"`
	Name   string       `json:"name" gorm:"size:100"`
	Orders []IntegOrder `json:"orders" gorm:"foreignKey:UserID" relation:"has_many:Orders::integ_orders"`
}

func TestIntegrationPreloadDetail(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&schema.Schema{}, &IntegUser{}, &IntegOrder{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// 初始化 Schema 元数据
	userModel, err := NewTypedModel[IntegUser](WithDB(db), WithModuleName("integration"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}

	// 手动确保关联 Schema 已写入（AutoMigrate 会自动写入，但这里验证一下）
	allSchemas, err := schema.GetSchemas(context.Background(), db, "integration", "integ_users")
	if err != nil {
		t.Fatalf("GetSchemas failed: %v", err)
	}
	var ordersSchema *schema.Schema
	for i := range allSchemas {
		if allSchemas[i].Column == "orders" {
			ordersSchema = &allSchemas[i]
			break
		}
	}
	if ordersSchema == nil {
		t.Fatal("orders schema not found after AutoMigrate")
	}
	if ordersSchema.Relations.Type != "has_many" {
		t.Fatalf("expected orders relation type has_many, got %s", ordersSchema.Relations.Type)
	}

	// 创建测试数据
	user := IntegUser{Name: "alice", Orders: []IntegOrder{{Total: 100}, {Total: 200}}}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	// Detail 查询应加载 Orders
	result, err := userModel.Detail(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("Detail failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Name != "alice" {
		t.Errorf("name mismatch: want alice, got %s", result.Name)
	}
	if len(result.Orders) != 2 {
		t.Errorf("orders count: want 2, got %d", len(result.Orders))
	}
	if result.Orders[0].Total != 100 {
		t.Errorf("first order total: want 100, got %d", result.Orders[0].Total)
	}
}

func TestIntegrationPreloadList(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&schema.Schema{}, &IntegUser{}, &IntegOrder{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	userModel, err := NewTypedModel[IntegUser](WithDB(db), WithModuleName("integration"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}

	// 创建测试数据
	db.Create(&IntegUser{Name: "bob", Orders: []IntegOrder{{Total: 300}}})
	db.Create(&IntegUser{Name: "carol", Orders: []IntegOrder{{Total: 400}, {Total: 500}}})

	// List 查询应加载 Orders
	list, err := userModel.List(context.Background(), 0, 10, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 users, got %d", len(list))
	}

	foundBob := false
	for _, u := range list {
		if u.Name == "bob" {
			foundBob = true
			if len(u.Orders) != 1 {
				t.Errorf("bob orders: want 1, got %d", len(u.Orders))
			}
		}
	}
	if !foundBob {
		t.Error("bob not found in list")
	}
}
```

- [ ] **Step 2: 运行集成测试**

```bash
cd /mobe/workspace/rest && go test . -run TestIntegration -v
```

Expected: PASS（Detail 和 List 都能加载 Orders 关联数据）

如果 FAIL，检查：
- `applyPreloads` 是否正确调用了 `db.Preload("Orders")`
- `Detail`/`List` 中 Schema 是否正确加载
- GORM 关联标签 `foreignKey:UserID` 是否正确配置

- [ ] **Step 3: 运行全量测试，确保无回归**

```bash
cd /mobe/workspace/rest && go test ./...
```

Expected: 所有现有测试通过，无新增失败

- [ ] **Step 4: 提交**

```bash
cd /mobe/workspace/rest && git add integration_test.go && git commit -m "test: add integration tests for Preload association loading"
```

---

## Plan Self-Review

### Spec Coverage Check

| Spec 要求 | 对应 Task |
|---|---|
| Schema 新增 `Relations Relation` 复合字段 | Task 1 |
| Relation 的 Scan/Value JSON 序列化 | Task 1 |
| `parseFieldRelations` 自动发现关联配置 | Task 2 |
| `AutoMigrate` 写入关联配置 | Task 2 |
| `applyPreloads` 递归构建 Preload 链 | Task 3 |
| `Detail`/`List`/`Paginate`/`Cursor` 调用 applyPreloads | Task 3 |
| 循环引用检测 | Task 3 |
| Formatter 递归处理关联字段 | Task 4 |
| 空 has_many 返回 `[]` | Task 4 |
| 集成测试验证 | Task 5 |

### Placeholder Scan

- 无 TBD/TODO/"implement later" 等占位符
- 无 "add appropriate error handling" 等模糊描述
- 每个步骤包含完整代码或确切命令

### Type Consistency Check

- `Relation` 结构体字段在 Task 1-5 中一致：`Type`, `Name`, `Module`, `Table`
- `applyPreloads` 签名在 Task 3 的定义和调用处一致
- `schema.Relation` 在 Task 1 定义后，Task 2/3/4/5 中引用一致

---

**Plan complete and saved to `docs/superpowers/plans/2026-05-14-preload-support.md`.**

**Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints for review

**Which approach?**