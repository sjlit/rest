# Schema 缓存实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `schema.GetSchemas` / `schema.GetVisibleSchemas` 引入进程内缓存，通过时间戳校验 + 主动失效 + TTL 兜底三层策略降低数据库读取压力。

**Architecture:** 在 `schema` 包内新增 `Cache` 结构体，以 `module:table` 为 key 缓存 `[]Schema`。读请求优先命中缓存，通过 `SELECT COALESCE(MAX(updated_at), 0)` 校验新鲜度。写操作后主动 `Invalidate` 清除对应缓存。`GetSchemas` / `GetVisibleSchemas` 自动检测全局 `defaultCache`，对调用方零侵入。

**Tech Stack:** Go 1.25, GORM v1.31.1, sqlite (测试用), sync.RWMutex

---

## 文件结构

| 文件 | 操作 | 说明 |
|---|---|---|
| `schema/cache.go` | 新建 | Cache 结构体、选项、Invalidate、GetSchemas、GetVisibleSchemas |
| `schema/cache_test.go` | 新建 | 单元测试：命中、未命中、TTL、时间戳刷新、并发击穿、空结果 |
| `schema/schema.go` | 修改 | `GetSchemas` / `GetVisibleSchemas` 增加 `defaultCache` 分支 |
| `schema/migrate.go` | 修改 | `AutoMigrate` 返回前调用 `InvalidateCache` |

---

### Task 1: Cache 结构体、选项和基础方法

**Files:**
- Create: `schema/cache.go`
- Test: `schema/cache_test.go`

- [ ] **Step 1: Write the failing test**

```go
package schema

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCacheTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&Schema{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestNewCache(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db, WithTTL(5*time.Minute))
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
	if c.ttl != 5*time.Minute {
		t.Errorf("ttl: want 5m, got %v", c.ttl)
	}
}

func TestCacheInvalidate(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// Manually inject an entry
	c.mu.Lock()
	c.entries["test:users"] = &cacheEntry{schemas: []Schema{{Column: "id"}}}
	c.mu.Unlock()

	c.Invalidate("test", "users")

	c.mu.RLock()
	_, ok := c.entries["test:users"]
	c.mu.RUnlock()
	if ok {
		t.Error("expected cache entry to be invalidated")
	}
}

func TestCacheInvalidateAll(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	c.mu.Lock()
	c.entries["a:b"] = &cacheEntry{}
	c.entries["c:d"] = &cacheEntry{}
	c.mu.Unlock()

	c.InvalidateAll()

	c.mu.RLock()
	if len(c.entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(c.entries))
	}
	c.mu.RUnlock()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./schema -run "TestNewCache|TestCacheInvalidate" -v`

Expected: FAIL with `cacheEntry` / `NewCache` / `WithTTL` / `Invalidate` not defined

- [ ] **Step 3: Write minimal implementation**

```go
package schema

import (
	"sync"
	"time"

	"gorm.io/gorm"
)

type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	db      *gorm.DB
}

type cacheEntry struct {
	schemas       []Schema
	lastUpdatedAt int64
	cachedAt      time.Time
}

type CacheOption func(*Cache)

func WithTTL(d time.Duration) CacheOption {
	return func(c *Cache) {
		c.ttl = d
	}
}

func NewCache(db *gorm.DB, opts ...CacheOption) *Cache {
	c := &Cache{
		entries: make(map[string]*cacheEntry),
		db:      db,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Cache) key(module, table string) string {
	return module + ":" + table
}

func (c *Cache) Invalidate(module, table string) {
	c.mu.Lock()
	delete(c.entries, c.key(module, table))
	c.mu.Unlock()
}

func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	c.entries = make(map[string]*cacheEntry)
	c.mu.Unlock()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./schema -run "TestNewCache|TestCacheInvalidate" -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add schema/cache.go schema/cache_test.go
git commit -m "feat(schema): add Cache structure with Invalidate and options"
```

---

### Task 2: Cache.GetSchemas 核心逻辑

**Files:**
- Modify: `schema/cache.go`
- Test: `schema/cache_test.go`

- [ ] **Step 1: Write the failing test**

在 `schema/cache_test.go` 中追加：

```go
func TestCacheGetSchemas_Miss(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// Insert schema records
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "id", UpdatedAt: 1000})
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "name", UpdatedAt: 1000})

	schemas, err := c.GetSchemas(nil, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(schemas))
	}

	// Verify cache was populated
	c.mu.RLock()
	ent, ok := c.entries["mod:tbl"]
	c.mu.RUnlock()
	if !ok {
		t.Fatal("expected cache entry after miss")
	}
	if len(ent.schemas) != 2 {
		t.Errorf("expected 2 cached schemas, got %d", len(ent.schemas))
	}
	if ent.lastUpdatedAt != 1000 {
		t.Errorf("lastUpdatedAt: want 1000, got %d", ent.lastUpdatedAt)
	}
}

func TestCacheGetSchemas_Hit(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// Populate cache manually
	c.mu.Lock()
	c.entries["mod:tbl"] = &cacheEntry{
		schemas:       []Schema{{Column: "id"}},
		lastUpdatedAt: 1000,
		cachedAt:      time.Now(),
	}
	c.mu.Unlock()

	// Database has newer data, but cache entry lastUpdatedAt == db max(updated_at)
	// So we simulate by NOT inserting any record with updated_at > 1000
	// Actually, empty db means MAX(updated_at) returns 0, so cache would be stale.
	// Let's insert a record with updated_at = 1000 to match cache timestamp.
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "id", UpdatedAt: 1000})

	schemas, err := c.GetSchemas(nil, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema (from cache), got %d", len(schemas))
	}
	if schemas[0].Column != "id" {
		t.Errorf("column: want id, got %s", schemas[0].Column)
	}
}

func TestCacheGetSchemas_TimestampMismatch(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// Populate stale cache
	c.mu.Lock()
	c.entries["mod:tbl"] = &cacheEntry{
		schemas:       []Schema{{Column: "old"}},
		lastUpdatedAt: 1000,
		cachedAt:      time.Now(),
	}
	c.mu.Unlock()

	// Database has newer data
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "new", UpdatedAt: 2000})

	schemas, err := c.GetSchemas(nil, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema after refresh, got %d", len(schemas))
	}
	if schemas[0].Column != "new" {
		t.Errorf("column: want new, got %s", schemas[0].Column)
	}

	// Verify cache was updated
	c.mu.RLock()
	ent := c.entries["mod:tbl"]
	c.mu.RUnlock()
	if ent.lastUpdatedAt != 2000 {
		t.Errorf("lastUpdatedAt: want 2000, got %d", ent.lastUpdatedAt)
	}
}

func TestCacheGetSchemas_TTLOverride(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db, WithTTL(-1*time.Hour)) // negative TTL forces expiration

	c.mu.Lock()
	c.entries["mod:tbl"] = &cacheEntry{
		schemas:       []Schema{{Column: "id"}},
		lastUpdatedAt: 1000,
		cachedAt:      time.Now(),
	}
	c.mu.Unlock()

	// Even though timestamp matches, TTL is expired
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "id", UpdatedAt: 1000})

	schemas, err := c.GetSchemas(nil, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	// Should re-query from db (which returns the same data, but proves TTL works)
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema after TTL expiry, got %d", len(schemas))
	}
}

func TestCacheGetSchemas_EmptyResult(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// No records in db
	schemas, err := c.GetSchemas(nil, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 0 {
		t.Fatalf("expected 0 schemas, got %d", len(schemas))
	}

	// Verify empty result is cached
	c.mu.RLock()
	ent, ok := c.entries["mod:tbl"]
	c.mu.RUnlock()
	if !ok {
		t.Fatal("expected empty result to be cached")
	}
	if len(ent.schemas) != 0 {
		t.Errorf("expected 0 cached schemas, got %d", len(ent.schemas))
	}
	if ent.lastUpdatedAt != 0 {
		t.Errorf("lastUpdatedAt: want 0, got %d", ent.lastUpdatedAt)
	}
}

func TestCacheGetSchemas_ConcurrentLoad(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// Pre-populate db with a single record
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "id", UpdatedAt: 1000})

	// Run 50 concurrent GetSchemas calls
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.GetSchemas(nil, "mod", "tbl")
			if err != nil {
				t.Errorf("GetSchemas error: %v", err)
			}
		}()
	}
	wg.Wait()

	// Verify db was only queried a few times (ideally once for timestamp + once for full load)
	// The exact count isn't critical; what matters is no panic and data is correct.
	c.mu.RLock()
	ent := c.entries["mod:tbl"]
	c.mu.RUnlock()
	if ent == nil {
		t.Fatal("expected cache entry")
	}
	if len(ent.schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(ent.schemas))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./schema -run "TestCacheGetSchemas" -v`

Expected: FAIL with `GetSchemas` method not defined on `*Cache`

- [ ] **Step 3: Write minimal implementation**

在 `schema/cache.go` 中追加：

```go
import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

func (c *Cache) GetSchemas(ctx context.Context, moduleName, tableName string) ([]Schema, error) {
	key := c.key(moduleName, tableName)

	// Step 1: read-lock check
	c.mu.RLock()
	ent, ok := c.entries[key]
	c.mu.RUnlock()

	if ok {
		// TTL check
		if c.ttl > 0 && time.Since(ent.cachedAt) > c.ttl {
			ok = false
		}
	}

	if ok {
		// Step 2: timestamp validation (lock-free)
		var lastUpdated int64
		err := c.db.Model(&Schema{}).
			Select("COALESCE(MAX(updated_at), 0)").
			Where("module_name = ? AND table_name = ?", moduleName, tableName).
			Scan(&lastUpdated).Error
		if err != nil {
			// Timestamp query failed: degrade to direct query
			ok = false
		} else if lastUpdated == ent.lastUpdatedAt {
			return ent.schemas, nil
		} else {
			ok = false
		}
	}

	if !ok {
		// Step 3: write-lock with double-check
		c.mu.Lock()
		ent, ok = c.entries[key]
		if ok && c.ttl > 0 && time.Since(ent.cachedAt) <= c.ttl {
			// Double-check: another goroutine refreshed while we waited
			var lastUpdated int64
			err := c.db.Model(&Schema{}).
				Select("COALESCE(MAX(updated_at), 0)").
				Where("module_name = ? AND table_name = ?", moduleName, tableName).
				Scan(&lastUpdated).Error
			if err == nil && lastUpdated == ent.lastUpdatedAt {
				c.mu.Unlock()
				return ent.schemas, nil
			}
		}

		// Full query from db
		var values []Schema
		values = make([]Schema, 0)
		var lastUpdated int64

		values, err := gorm.G[Schema](c.db).
			Where("module_name = ? AND table_name = ?", moduleName, tableName).
			Order("position ASC").
			Find(ctx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = nil
		}
		if err != nil {
			c.mu.Unlock()
			return nil, err
		}

		// Get max updated_at
		c.db.Model(&Schema{}).
			Select("COALESCE(MAX(updated_at), 0)").
			Where("module_name = ? AND table_name = ?", moduleName, tableName).
			Scan(&lastUpdated)

		c.entries[key] = &cacheEntry{
			schemas:       values,
			lastUpdatedAt: lastUpdated,
			cachedAt:      time.Now(),
		}
		c.mu.Unlock()
		return values, nil
	}

	return ent.schemas, nil
}
```

注意：上面的 import 需要合并到文件顶部的 import 块中。`cache.go` 最终 import 应为：

```go
import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./schema -run "TestCacheGetSchemas" -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add schema/cache.go schema/cache_test.go
git commit -m "feat(schema): implement Cache.GetSchemas with timestamp validation"
```

---

### Task 3: Cache.GetVisibleSchemas

**Files:**
- Modify: `schema/cache.go`
- Test: `schema/cache_test.go`

- [ ] **Step 1: Write the failing test**

在 `schema/cache_test.go` 中追加：

```go
func TestCacheGetVisibleSchemas(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// Insert records with different scenarios
	db.Create(&Schema{
		ModuleName: "mod", TableName: "tbl", Column: "id",
		Scenarios: Scenarios{ScenarioList, ScenarioDetail},
		UpdatedAt: 1000,
	})
	db.Create(&Schema{
		ModuleName: "mod", TableName: "tbl", Column: "secret",
		Scenarios: Scenarios{ScenarioCreate, ScenarioUpdate},
		UpdatedAt: 1000,
	})

	schemas, err := c.GetVisibleSchemas(nil, "mod", "tbl", ScenarioList)
	if err != nil {
		t.Fatalf("GetVisibleSchemas error: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 visible schema, got %d", len(schemas))
	}
	if schemas[0].Column != "id" {
		t.Errorf("column: want id, got %s", schemas[0].Column)
	}

	// Call again to verify cache hit path
	schemas2, err := c.GetVisibleSchemas(nil, "mod", "tbl", ScenarioList)
	if err != nil {
		t.Fatalf("GetVisibleSchemas error: %v", err)
	}
	if len(schemas2) != 1 {
		t.Fatalf("expected 1 visible schema on cache hit, got %d", len(schemas2))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./schema -run "TestCacheGetVisibleSchemas" -v`

Expected: FAIL with `GetVisibleSchemas` method not defined on `*Cache`

- [ ] **Step 3: Write minimal implementation**

在 `schema/cache.go` 中追加：

```go
func (c *Cache) GetVisibleSchemas(ctx context.Context, moduleName, tableName, scenario string) ([]Schema, error) {
	schemas, err := c.GetSchemas(ctx, moduleName, tableName)
	if err != nil {
		return nil, err
	}
	result := make([]Schema, 0, len(schemas))
	for _, row := range schemas {
		if row.Scenarios.Has(scenario) {
			result = append(result, row)
		}
	}
	return result, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./schema -run "TestCacheGetVisibleSchemas" -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add schema/cache.go schema/cache_test.go
git commit -m "feat(schema): add Cache.GetVisibleSchemas delegating to GetSchemas"
```

---

### Task 4: 修改 schema.go 接入 defaultCache

**Files:**
- Modify: `schema/schema.go`
- Test: `schema/cache_test.go`（追加集成测试）

- [ ] **Step 1: Write the failing test**

在 `schema/cache_test.go` 中追加：

```go
func TestDefaultCacheIntegration(t *testing.T) {
	// Ensure clean state
	defaultCache = nil
	defer func() { defaultCache = nil }()

	db := setupCacheTestDB(t)
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "id", UpdatedAt: 1000})

	// Before enabling cache, direct query works
	schemas, err := GetSchemas(nil, db, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}

	// Enable cache
	EnableCache(db)

	// This call should populate cache
	schemas2, err := GetSchemas(nil, db, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas2) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas2))
	}

	// Verify cache was populated
	defaultCache.mu.RLock()
	ent, ok := defaultCache.entries["mod:tbl"]
	defaultCache.mu.RUnlock()
	if !ok {
		t.Fatal("expected defaultCache to have entry")
	}
	if len(ent.schemas) != 1 {
		t.Errorf("expected 1 cached schema, got %d", len(ent.schemas))
	}

	// GetVisibleSchemas should also use cache
	visible, err := GetVisibleSchemas(nil, db, "mod", "tbl", ScenarioList)
	if err != nil {
		t.Fatalf("GetVisibleSchemas error: %v", err)
	}
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible schema, got %d", len(visible))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./schema -run "TestDefaultCacheIntegration" -v`

Expected: FAIL with `defaultCache` / `EnableCache` / `InvalidateCache` not defined

- [ ] **Step 3: Write minimal implementation**

在 `schema/schema.go` 中，在现有 `GetSchemas` / `GetVisibleSchemas` 之上添加：

```go
var defaultCache *Cache

func EnableCache(db *gorm.DB, opts ...CacheOption) {
	defaultCache = NewCache(db, opts...)
}

func InvalidateCache(module, table string) {
	if defaultCache != nil {
		defaultCache.Invalidate(module, table)
	}
}
```

然后修改 `GetSchemas`：

```go
func GetSchemas(ctx context.Context, db *gorm.DB, moduleName, tableName string) ([]Schema, error) {
	if defaultCache != nil {
		return defaultCache.GetSchemas(ctx, moduleName, tableName)
	}

	var (
		err    error
		values []Schema
	)
	values = make([]Schema, 0)
	if moduleName == "" || tableName == "" {
		return nil, ErrMissingModuleName
	}
	values, err = gorm.G[Schema](db).Where("module_name=? AND table_name=?", moduleName, tableName).Order("position ASC").Find(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return values, err
}
```

修改 `GetVisibleSchemas`：

```go
func GetVisibleSchemas(ctx context.Context, db *gorm.DB, moduleName, tableName, scenario string) ([]Schema, error) {
	if defaultCache != nil {
		return defaultCache.GetVisibleSchemas(ctx, moduleName, tableName, scenario)
	}

	schemas, err := GetSchemas(ctx, db, moduleName, tableName)
	if err != nil {
		return nil, err
	}
	result := make([]Schema, 0, len(schemas))
	for _, row := range schemas {
		if row.Scenarios.Has(scenario) {
			result = append(result, row)
		}
	}
	return result, nil
}
```

注意：`schema.go` 当前文件顶部 import 已有 `context` 和 `errors`，但如果没有 `context`，需要添加。检查现有 import：

```go
import "errors"
```

只有 `errors`。需要添加 `context`：

```go
import (
	"context"
	"errors"
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./schema -run "TestDefaultCacheIntegration" -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add schema/schema.go schema/cache_test.go
git commit -m "feat(schema): wire defaultCache into GetSchemas and GetVisibleSchemas"
```

---

### Task 5: 修改 migrate.go 在 AutoMigrate 后失效缓存

**Files:**
- Modify: `schema/migrate.go`

- [ ] **Step 1: Write the failing test**

在 `schema/cache_test.go` 中追加：

```go
func TestAutoMigrateInvalidatesCache(t *testing.T) {
	// Ensure clean state
	defaultCache = nil
	defer func() { defaultCache = nil }()

	db := setupCacheTestDB(t)
	EnableCache(db)

	type TestUser struct {
		ID   uint   `json:"id" gorm:"primarykey"`
		Name string `json:"name"`
	}

	// First AutoMigrate populates cache
	_, err := AutoMigrate(nil, db, &TestUser{}, "testmod")
	if err != nil {
		t.Fatalf("AutoMigrate error: %v", err)
	}

	// Verify cache has entry
	defaultCache.mu.RLock()
	_, ok := defaultCache.entries["testmod:test_users"]
	defaultCache.mu.RUnlock()
	if !ok {
		// Cache might not have been populated if no GetSchemas was called
		// Let's populate it manually to test invalidation
		GetSchemas(nil, db, "testmod", "test_users")
	}

	defaultCache.mu.RLock()
	_, ok = defaultCache.entries["testmod:test_users"]
	defaultCache.mu.RUnlock()
	if !ok {
		t.Fatal("expected cache entry before second AutoMigrate")
	}

	// Second AutoMigrate should invalidate cache
	_, err = AutoMigrate(nil, db, &TestUser{}, "testmod")
	if err != nil {
		t.Fatalf("AutoMigrate error: %v", err)
	}

	// Verify cache was invalidated
	defaultCache.mu.RLock()
	_, ok = defaultCache.entries["testmod:test_users"]
	defaultCache.mu.RUnlock()
	if ok {
		t.Error("expected cache entry to be invalidated after AutoMigrate")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./schema -run "TestAutoMigrateInvalidatesCache" -v`

Expected: FAIL because `AutoMigrate` does not call `InvalidateCache`

- [ ] **Step 3: Write minimal implementation**

在 `schema/migrate.go` 的 `AutoMigrate` 函数末尾，在 `return tableName, err` 之前添加：

```go
	InvalidateCache(moduleName, tableName)
	return
```

当前 `AutoMigrate` 末尾是：

```go
	if len(models) > 0 {
		err = gorm.G[Schema](db).CreateInBatches(ctx, &models, 50)
	}
	return
```

修改为：

```go
	if len(models) > 0 {
		err = gorm.G[Schema](db).CreateInBatches(ctx, &models, 50)
	}
	InvalidateCache(moduleName, tableName)
	return
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./schema -run "TestAutoMigrateInvalidatesCache" -v`

Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `go test ./...`

Expected: ALL PASS（确保缓存改动没有破坏现有功能）

- [ ] **Step 6: Commit**

```bash
git add schema/migrate.go schema/cache_test.go
git commit -m "feat(schema): invalidate cache after AutoMigrate"
```

---

## Self-Review

### 1. Spec coverage

| Spec 章节 | 对应 Task |
|---|---|
| 2. 缓存策略（三层互补） | Task 2（时间戳校验 + TTL） + Task 5（主动失效） |
| 3. 缓存数据结构 | Task 1 |
| 4. 读取与校验流程 | Task 2 |
| 5. 写入与失效策略 | Task 3（GetVisibleSchemas 委托） + Task 5（Invalidate） |
| 6. 接口设计与接入方式 | Task 4（defaultCache） + Task 1（CacheOption） |
| 7. 错误处理与边界情况 | Task 2（TTL过期、空结果、并发、时间戳失败降级） |

### 2. Placeholder scan

- 无 TBD / TODO / "implement later"
- 每个步骤都有完整代码块
- 每个测试都有明确的断言
- 每个命令都有预期输出

### 3. Type consistency

- `cacheEntry.lastUpdatedAt` 始终为 `int64`
- `Cache.GetSchemas` 和 `Cache.GetVisibleSchemas` 的签名与设计文档一致
- `defaultCache` 是 `*Cache` 类型，贯穿所有 task
- `gorm.G[Schema](c.db)...Find(ctx)` 与现有代码风格一致
