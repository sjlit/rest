# Schema 缓存设计文档

## 1. 背景与目标

当前框架的 `schema.GetSchemas` 和 `schema.GetVisibleSchemas` 每次调用都会执行全量 `SELECT *` 查询数据库。在 `Resource` 的查询构建、OpenAPI 生成、数据格式化等多个环节，同一 `module+table` 组合的 schema 被反复读取，给数据库带来不必要的压力。

本设计旨在为 schema 查询层引入**进程内缓存**，在保证数据一致性的前提下，显著降低数据库读取压力。

### 1.1 设计原则

- **对调用方零侵入**：`GetSchemas` / `GetVisibleSchemas` 保持现有签名，自动使用缓存（如果已启用）
- **最终一致性**：后台更新 schema 后，下一次读请求即可感知并刷新
- **降级安全**：任何缓存环节失败都不阻断正常服务，自动降级为直接查库
- **单实例友好**：利用进程内内存，无需引入 Redis 等外部依赖

### 1.2 非目标

- 不缓存派生数据（如按 scenario 过滤后的结果），仅缓存原始 `[]Schema`
- 不引入分布式缓存或多节点同步机制
- 不做 LRU 或容量限制（key 空间受限于表数量，通常为几十到几百个）

---

## 2. 缓存策略

采用**三层互补策略**：

| 策略 | 作用 | 说明 |
|---|---|---|
| 时间戳校验（主） | 检测数据是否变更 | 每次读缓存前执行 `SELECT MAX(updated_at)`，与缓存记录的时间戳比对 |
| 主动失效（优） | 减少时间戳查询 | 写操作后调用 `Invalidate`，直接清除对应缓存 |
| TTL 兜底（辅） | 防止永久脏数据 | 可选配置，缓存条目超过 TTL 强制视为过期 |

**为什么不用纯 TTL？** 后台动态更新 schema 后，纯 TTL 会导致数据在 TTL 到期前不一致。时间戳校验能在下一次读请求时立即感知变更。

**为什么不用纯主动失效？** 需要修改所有写 schema 的代码，侵入性大。时间戳校验作为兜底，即使遗漏主动失效也能保证最终一致。

---

## 3. 缓存数据结构

新增 `schema/cache.go`：

```go
type Cache struct {
    mu      sync.RWMutex
    entries map[string]*cacheEntry
    ttl     time.Duration // 0 表示不启用 TTL
    db      *gorm.DB
}

type cacheEntry struct {
    schemas       []Schema
    lastUpdatedAt int64     // 该 module+table 下 MAX(updated_at)
    cachedAt      time.Time // 用于 TTL 兜底判断
}
```

**Key 规则**：`moduleName + ":" + tableName`，如 `"order:orders"`。

**复制策略**：`Schema` 结构体内部无指针字段，缓存返回时切片浅拷贝安全，无需深拷贝。

---

## 4. 读取与校验流程

```
1. 生成 key = module + ":" + table
2. 读锁获取 cacheEntry
   - 未命中       → 跳到步骤 4
   - TTL 过期     → 跳到步骤 4
   - 命中且未过期 → 释放读锁，进入步骤 3
3. 时间戳校验（无锁状态，不阻塞其他读）
   SELECT COALESCE(MAX(updated_at), 0) FROM schema
   WHERE module_name = ? AND table_name = ?
   - 结果 == entry.lastUpdatedAt → 返回缓存的 schemas
   - 结果 != entry.lastUpdatedAt → 进入步骤 4
4. 获取写锁，double-check 缓存是否已被其他 goroutine 刷新
   - 如果已被刷新且时间戳仍有效 → 返回
   - 否则执行原查询：SELECT * FROM schema WHERE ...
     同时计算 MAX(updated_at)
     写入缓存，释放写锁，返回结果
```

**并发设计要点**：
- 时间戳校验故意在**无锁状态**下执行，避免轻量查询阻塞并发读
- 只有回源查全量数据时才获取写锁
- double-check 防止缓存击穿时多个 goroutine 重复查询数据库

---

## 5. 写入与失效策略

### 5.1 主动失效

```go
func (c *Cache) Invalidate(module, table string)
func (c *Cache) InvalidateAll()
```

**调用时机**：
- `AutoMigrate` 完成后，清除对应 `module+table` 缓存
- 后台管理接口修改 schema 后，清除对应缓存
- 主动失效是性能优化，非强制性（时间戳校验会兜底）

### 5.2 被动失效

如果写操作遗漏了主动失效，下一次读请求会通过 `MAX(updated_at)` 时间戳校验自动发现不一致并刷新缓存。

### 5.3 TTL 兜底

通过 `WithTTL(duration)` 选项启用。缓存条目超过 TTL 后强制视为过期，回源查库。防止极端情况下（如数据库时间戳异常）永久读取脏数据。

---

## 6. 接口设计与接入方式

### 6.1 包级默认缓存

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

### 6.2 现有函数改造

`GetSchemas` 和 `GetVisibleSchemas` 内部优先检查 `defaultCache`：

```go
func GetSchemas(ctx context.Context, db *gorm.DB, moduleName, tableName string) ([]Schema, error) {
    if defaultCache != nil {
        return defaultCache.GetSchemas(ctx, moduleName, tableName)
    }
    // 原逻辑不变
    ...
}
```

### 6.3 Cache 构造与方法

```go
func NewCache(db *gorm.DB, opts ...CacheOption) *Cache
func (c *Cache) GetSchemas(ctx context.Context, module, table string) ([]Schema, error)
func (c *Cache) GetVisibleSchemas(ctx context.Context, module, table, scenario string) ([]Schema, error)
func (c *Cache) Invalidate(module, table string)
func (c *Cache) InvalidateAll()
```

### 6.4 启用示例

```go
schema.EnableCache(db, schema.WithTTL(10*time.Minute))
```

### 6.5 AutoMigrate 集成

在 `AutoMigrate` 返回前调用 `InvalidateCache(moduleName, tableName)`。

---

## 7. 错误处理与边界情况

| 场景 | 行为 |
|---|---|
| 时间戳查询失败 | 降级为直接查询全量数据，不阻塞请求 |
| 全量查询失败 | 返回错误，不写入缓存 |
| 数据库无 schema 记录（RecordNotFound） | 缓存空结果（`lastUpdatedAt = 0`），避免重复查询 |
| 并发缓存击穿 | 写锁 + double-check，仅一个 goroutine 回源 |
| 缓存内存增长 | key 空间受限于表数量，不限制容量 |
| Schema 结构体未来增加指针字段 | 需调整为深拷贝，当前版本浅拷贝安全 |

---

## 8. 与现有代码的集成点

| 文件 | 修改内容 |
|---|---|
| `schema/cache.go` | 新增文件：Cache 结构体、时间戳校验、读取流程、失效逻辑 |
| `schema/migrate.go` | `AutoMigrate` 返回前调用 `InvalidateCache` |
| `schema/schema.go` | `GetSchemas` / `GetVisibleSchemas` 增加 defaultCache 分支 |
| `resource.go` / `model.go` / `openapi` 等 | **无需修改**，自动享受缓存收益 |

---

## 9. 测试策略

- **单元测试**：Cache 的命中、未命中、TTL 过期、时间戳不一致刷新、并发安全（double-check）
- **集成测试**：启用缓存后，验证 `Resource` 的 List/Detail/Search 等功能行为一致
- **边界测试**：空结果缓存、时间戳查询失败降级、Invalidate 后下一次读取正确刷新
