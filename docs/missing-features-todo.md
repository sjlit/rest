# 项目功能差距分析

> 对比当前项目 (`/mobe/workspace/rest`) 与参考项目 (`/mobe/reference/rest`) 的功能差异。
> 当前项目为 v3 重构分支，采用泛型 `Model[T]` + AST-based Query 架构；参考项目为旧版本，功能更完整。

---

## 核心机制层

- [ ] **生命周期钩子系统 (hook.go)**
  - 当前 `Model[T]` 的 `Create`/`Update`/`Delete` 中未调用任何钩子
  - 缺少：`BeforeCreate`, `AfterCreate`, `BeforeUpdate`, `AfterUpdate`, `BeforeSave`, `AfterSave`, `BeforeDelete`, `AfterDelete`
  - 缺少：`AfterExport`, `AfterImport`（导出/导入后钩子）
  - 缺少：`ActiveQuery` 接口（`BeforeQuery`/`AfterQuery`，在 `condition.go` 中使用）
  - 需要适配泛型 `Model[T]` 架构

- [ ] **HTTP 请求条件构建 (condition.go)**
  - 缺少 `BuildConditions` 函数：从 HTTP 请求解析查询条件（JSON body + query string）
  - 支持操作符：`BETWEEN`, `>`, `>=`, `<`, `<=`, `LIKE`, `IN`
  - 支持 `sort` 参数解析（`+field`/`-field`）

- [ ] **工具函数 (utils.go)**
  - 缺少：`hasToken`（逗号/分号分隔的 token 匹配）
  - 缺少：`isEmpty`（完整的 reflect 空值判断，当前 `rest.go` 中的 `IsEmpty` 不完整）
  - 缺少：`recursiveTier`（层级数据递归构建）

- [ ] **全局模型注册与管理**
  - 缺少：`modelEntities` 全局注册表
  - 缺少：`GetModels()` 获取所有已注册模型
  - 缺少：`Init()` 全局初始化入口
  - 缺少：`AutoMigrate()` 自动迁移并注册路由（当前只有 `NewModel`）

- [ ] **Schema 克隆功能**
  - 缺少：`CloneSchemas()` 跨域克隆 schema 数据

- [ ] **查询辅助函数**
  - 缺少：`ModelTypes[T]()` / `ModelTiers[T]()` 键值对/层级数据查询
  - 缺少：`GetFieldValue()` / `SetFieldValue()` / `SafeSetFileValue()` 反射辅助

---

## 插件系统 (plugins/)

- [ ] **缓存插件 (`plugins/cache/cache.go`)**
  - GORM 查询回调插件，基于 `xxhash` + JSON 序列化缓存查询结果
  - 支持通过 `gorm:cache_duration` 控制缓存时长

- [ ] **身份/主键生成插件 (`plugins/identity/`)**
  - `engine.go`：ID 生成引擎接口（`XidEngine`, `UUIDEngine`, `SnowflakeEngine`）
  - `identified.go`：GORM Create 回调，自动为字符串主键生成 ID

- [ ] **分片插件 (`plugins/sharding/`)**
  - `sharding.go`：GORM 分表插件，支持 Create/Update/Delete/Query 拦截
  - `types.go`：分片规则接口（`Model.ShardingRule()`, `ShardingTable()`, `ShardingTables()`）
  - `condition.go`, `scope.go`：分片条件解析
  - 支持 `datetime`/`hash` 分片策略
  - 支持多表 `UNION ALL` 查询重写

- [ ] **验证插件 (`plugins/validate/`)**
  - `validation.go`：基于 `go-playground/validator/v10` 的 GORM 验证插件
  - `types.go`：验证规则、错误格式化、空值判断
  - 自定义验证器：`telephone`, `db_unique`（数据库唯一性校验）
  - 支持字段可见性条件校验（`isVisible`）
  - 支持按场景（create/update）的 required 校验

---

## Model / Resource 功能层

- [ ] **导入/导出功能**
  - `Export`：CSV 导出，支持 schema 筛选、报表模式
  - `Import`：CSV 导入，支持模板下载、批量插入、失败记录文件、异步处理
  - 当前 `resource.go` 中 Export 被注释掉了

- [ ] **View / Detail HTTP 端点增强**
  - 当前 `resource.go` 已注册 Detail，但参考项目 `model.go` 的 `View` 方法支持 `?scenario=` 参数切换 schema

- [ ] **报表 (Reporter) 支持**
  - 缺少 `Reporter` 接口及聚合查询支持（`GROUP BY`, `HAVING`, 自定义 COUNT）
  - `buildReporterCountColumns` / `buildReporterQueryColumns`

- [ ] **权限系统**
  - 缺少 `PermissionChecker` 接口及集成
  - 缺少 `Permission()` 方法生成权限标识（`module:model:scenario`）
  - 当前 `Model[T]` 只有简单的 scenario 白名单检查

- [ ] **ValueLookup 系统**
  - 缺少 `ValueLookupFunc`：从 HTTP 请求提取 `domain`、`user` 等运行时值
  - 当前 `RuntimeScope` 缺少 `Domain`、`User` 字段

- [ ] **HttpWriter / HttpRouter 抽象**
  - 参考项目有标准的 `Success`/`Failure` 响应格式（`ListResponse`, `CreateResponse` 等）
  - 当前项目 `Responder` 接口过于简单，没有统一错误码体系

- [ ] **模型生命周期接口**
  - `afterCreated` / `afterUpdated` / `afterDeleted` / `afterSaved`
  - `FormatModel` 接口（模型自定义格式化）

- [ ] **Tabler 动态表名支持**
  - 参考项目支持模型实现 `Tabler` 接口动态切换表名（在 Create/Update/Delete/Search 中）

---

## Options / 配置层

- [ ] **更多 Option**
  - `WithUriPrefix` / `urlPrefix`
  - `WithoutDomain` / `disableDomain`
  - `WithHttpRouter`（当前是 `WithRouter` 但类型不同）
  - `WithHttpWriter`（当前是 `WithResponder`）
  - `WithPermissionChecker`
  - `WithValueLookup`
  - `WithResourceDirectory`
  - `WithFormatter`（当前有但选项类型不同）

---

## Query 层差异（架构重写中）

当前项目的 `query/` 包是全新 AST-based 架构，功能上已经覆盖了参考项目 query 的大部分能力，但仍有以下差异：

- [ ] **FilterWhere**（条件为空时自动跳过）
  - 参考项目有 `AndFilterWhere` / `OrFilterWhere`，当前 Builder 只有 `Where`
  - `resource.go` 的 `buildQuery` 中当 `formValue == ""` 时直接 `continue`，这实际上已实现了 filter 逻辑

- [ ] **排序参数解析**
  - 参考项目 `condition.go` 支持 `sort=-name,+age` 格式
  - 当前 `resource.go` 的 `Search` 中没有解析 sort 参数

---

## 其他

- [ ] **response 类型**
  - 参考项目有完整的响应结构体：`ListResponse`, `CreateResponse`, `UpdateResponse`, `DeleteResponse`, `ImportResponse`
  - 当前项目只有 `SearchResult`, `CreateResult`, `UpdateResult`, `DeletedResult`

- [ ] **tenant.go**
  - 当前项目只有 48 字节的占位文件，参考项目没有此文件（可能 tenant 逻辑在 `disableDomain` 和 `domain` 字段中）

---

## 建议优先级

### 高优先级（核心功能）

1. 钩子系统 (hook.go)
2. 验证插件 (plugins/validate)
3. 身份插件 (plugins/identity)
4. 条件构建 (condition.go) 适配 query.Builder
5. 权限系统 + ValueLookup

### 中优先级（扩展功能）

6. 导入/导出
7. 缓存插件
8. 分片插件
9. HttpWriter 标准化响应格式

### 低优先级（工具/辅助）

10. utils.go 工具函数
11. 全局模型注册 / CloneSchemas
12. ModelTypes / ModelTiers 辅助函数
