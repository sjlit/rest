# CRUD 类使用指南

## 目录

- [概述](#概述)
- [构造函数](#构造函数)
- [属性](#属性)
- [方法详解](#方法详解)
- [URI 构建规则](#uri-构建规则)
- [生命周期](#生命周期)
- [使用示例](#使用示例)
- [扩展 CRUD 类](#扩展-crud-类)

---

## 概述

`CRUD` 类是本库的 HTTP 操作核心，封装了与 REST 后端交互的完整逻辑，包括：

- Schema 加载与缓存
- Live 数据（下拉选项）自动拉取
- 主键自动识别
- 分页、排序、查询参数管理
- 创建、读取、更新、删除操作
- 批量删除和数据导出
- 字段错误管理

`CRUD` 类是纯 TypeScript 类，**不依赖 Vue 响应式系统**，可以在任何 JavaScript/TypeScript 环境中使用。

---

## 构造函数

```typescript
import { CRUD } from '@nobla/rest-ui'

const crud = new CRUD({
  module: 'user',           // 模块名
  table: 'admin',           // 表名
  apiPrefix: 'rest',        // API 前缀（可选，默认 'rest'）
  httpClient: axios,        // HTTP 客户端（必须）
  schemas: preloadedSchemas // 预加载的 schema（可选）
})
```

### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `module` | `string` | 否 | 模块名，用于构建 URI |
| `table` | `string` | 否 | 表名，用于构建 URI 和 schema 加载 |
| `apiPrefix` | `string` | 否 | API 前缀，默认 `'rest'` |
| `httpClient` | `object` | 是 | HTTP 客户端，需实现 `get/post/put/delete` |
| `schemas` | `Schema[] \| Record<string, Schema>` | 否 | 预加载的 schema，传入后不会从后端加载 |

---

## 属性

| 属性 | 类型 | 说明 |
|------|------|------|
| `primaryKey` | `string` | 主键字段名，初始化后自动识别 |
| `schemas` | `Schema[]` | 当前 schema 列表 |
| `models` | `Model[]` | 当前数据列表（最后一次搜索的结果） |
| `sortable` | `Sortable \| null` | 当前排序设置 |
| `queryParams` | `Record<string, any>` | 当前查询参数 |
| `fixedQuery` | `Record<string, any>` | 固定查询参数（每次请求自动附加） |
| `pagination` | `Pagination` | 分页信息 |
| `fieldErrors` | `Record<string, string>` | 字段错误映射 |

---

## 方法详解

### initialize(): Promise<Schema[]>

初始化 CRUD 实例，加载 schema 定义。

```typescript
const schemas = await crud.initialize()
```

**逻辑：**
1. 如果构造时传入了 `schemas`，直接使用（会克隆防止外部修改）
2. 否则从后端加载：`GET /{apiPrefix}/schema/{module}/{table}`
3. 识别主键字段（`primary_key === 1`）
4. 并行拉取所有启用了 `live` 的字段数据

**返回值：** 加载完成的 schema 数组

---

### getSchemas(): Schema[]

获取当前 schema 列表。

```typescript
const schemas = crud.getSchemas()
```

---

### getModels(): Model[]

获取当前数据列表。

```typescript
const models = crud.getModels()
```

---

### 错误管理

#### setColumnError(column: string, error: string): void

设置字段错误信息。

```typescript
crud.setColumnError('email', '邮箱格式不正确')
```

#### resetError(): void

清空所有字段错误。

```typescript
crud.resetError()
```

#### getFieldErrors(): Record<string, string>

获取所有字段错误。

```typescript
const errors = crud.getFieldErrors()
// { email: '邮箱格式不正确', username: '用户名已存在' }
```

---

### 分页管理

#### setPaginationIndex(index: number): this

设置当前页码（从 1 开始）。

```typescript
crud.setPaginationIndex(2)
```

#### getPaginationIndex(): number

获取当前页码。

```typescript
const page = crud.getPaginationIndex() // 2
```

#### setPaginationSize(size: number): this

设置每页条数。

```typescript
crud.setPaginationSize(20)
```

#### getPaginationSize(): number

获取每页条数。

```typescript
const size = crud.getPaginationSize() // 20
```

#### getPaginationCount(): number

获取总条数。

```typescript
const total = crud.getPaginationCount() // 100
```

#### resetPagination(): this

重置到第一页。

```typescript
crud.resetPagination()
```

---

### 排序管理

#### setSortable(column: string, order: 'ascending' | 'descending'): this

设置排序字段和方向。

```typescript
// 按 created_at 降序
crud.setSortable('created_at', 'descending')

// 清除排序
crud.setSortable('', 'ascending')
```

---

### 查询参数管理

#### addQueryParams(k: string, v: any): void

添加单个查询参数。

```typescript
crud.addQueryParams('status', 'active')
crud.addQueryParams('type', 'admin')
```

#### setQueryParams(qs: Record<string, any>): this

设置完整查询参数（覆盖已有）。

```typescript
crud.setQueryParams({ status: 'active', type: 'admin' })
```

#### setFixedQuery(qs: Record<string, any>): this

设置固定查询参数。固定参数会在每次搜索请求中自动附加，且不会被 `setQueryParams` 覆盖。

```typescript
// 设置租户 ID 为固定参数
crud.setFixedQuery({ tenant_id: '123' })

// 搜索时会自动附加 tenant_id
await crud.searchModel()
// 实际请求：GET /rest/user/admins?...&tenant_id=123
```

适用场景：
- 多租户系统中固定 tenant_id
- 固定数据范围（如只显示未删除数据）
- 默认筛选条件

---

### 模型操作

#### findModelPrimaryKey(model: Model): any

获取模型的主键值。

```typescript
const pk = crud.findModelPrimaryKey({ id: 123, name: 'test' })
// 123
```

---

#### createModel(model: Model): Promise<Model>

创建记录。

```typescript
const newModel = await crud.createModel({
  username: 'newuser',
  email: 'new@example.com',
  status: 'active',
})
```

**逻辑：**
1. 发送 `POST` 请求到创建 URI
2. 从响应中提取主键值
3. 调用 `__refreshModel` 获取完整数据
4. 更新本地 `models` 数组

**返回值：** 创建后的完整模型（包含后端生成的字段如 `id`、`created_at`）

---

#### updateModel(model: Model): Promise<Model>

更新记录。

```typescript
const updatedModel = await crud.updateModel({
  id: 123,
  username: 'updated',
  status: 'inactive',
})
```

**逻辑：**
1. 从模型中提取主键值
2. 发送 `PUT` 请求到更新 URI
3. 调用 `__refreshModel` 获取最新数据
4. 更新本地 `models` 数组中的对应记录

---

#### deleteModel(model: Model | string): Promise<any>

删除记录。

```typescript
// 通过模型删除
await crud.deleteModel({ id: 123, ... })

// 通过主键删除
await crud.deleteModel('123')
```

**逻辑：**
1. 从模型中提取主键值（或直接使用传入的主键）
2. 发送 `DELETE` 请求
3. 从本地 `models` 数组中移除对应记录

---

#### getModel(qs: Record<string, any> | string): Promise<Model>

获取单条记录详情。

```typescript
// 通过主键获取
const model = await crud.getModel('123')

// 通过查询参数获取（包含额外参数）
const model = await crud.getModel({
  id: 123,
  scenario: 'update',
  __format: 'raw',
})
```

---

#### searchModel(): Promise<Model[]>

搜索记录。

```typescript
// 设置查询条件
crud.setQueryParams({ status: 'active' })
crud.setSortable('created_at', 'descending')
crud.setPaginationIndex(1)

// 执行搜索
const models = await crud.searchModel()
```

**请求参数：**
- `page`: 当前页码
- `page_size`: 每页条数
- `sort`: 排序字段（前缀 `-` 表示降序）
- `__format`: `'both'`（固定值）
- 所有 `queryParams` 中的参数
- 所有 `fixedQuery` 中的参数

**响应处理：**
- 更新 `pagination.index` / `pagination.size` / `pagination.totalCount`
- 更新 `models` 数组

---

#### deleteModels(data: any[]): Promise<{ total: number; success: number; responses: any[] }>

批量删除。

```typescript
const result = await crud.deleteModels([
  { id: 1 },
  { id: 2 },
  { id: 3 },
])

console.log(result.total)     // 3
console.log(result.success)   // 3
console.log(result.responses) // 各条删除的响应
```

---

#### exportModels(): Promise<void>

导出数据。

```typescript
await crud.exportModels()
```

**逻辑：**
1. 发送 `GET` 请求到导出 URI
2. 使用当前查询参数和排序设置
3. 响应类型为 `blob`
4. 自动解析 `Content-Disposition` 头获取文件名
5. 触发浏览器下载

---

## URI 构建规则

`CRUD` 类使用 `pluralize` 库根据 `module`、`table` 和 `scenario` 自动构建 RESTful URI。

### 构建规则

| Scenario | 构建规则 | 示例（module=user, table=admin） |
|----------|----------|--------------------------------|
| `create` | `/{prefix}/{module}/{singular}` | `/rest/user/admin` |
| `update` | `/{prefix}/{module}/{singular}/{pk}` | `/rest/user/admin/123` |
| `delete` | `/{prefix}/{module}/{singular}/{pk}` | `/rest/user/admin/123` |
| `get` | `/{prefix}/{module}/{singular}/detail/{pk}` | `/rest/user/admin/detail/123` |
| `search` | `/{prefix}/{module}/{plural}` | `/rest/user/admins` |
| `export` | `/{prefix}/{module}/{singular}/export` | `/rest/user/admin/export` |

### 特殊情况

- 无 module：`/{prefix}/{table}`（如 `/rest/admin`）
- 无 apiPrefix：`/{module}/{table}`（如 `/user/admins`）

---

## 生命周期

```
new CRUD(options)
  │
  ├─── 解析 options（module, table, apiPrefix, httpClient, schemas）
  ├─── 克隆 schemas（防止外部修改）
  │
  └─── initialize()
         │
         ├─── 传入 schemas? 使用传入的 : 从后端加载
         ├─── __prepare() - 识别主键
         └─── __fetchVars() - 拉取 live 数据
  │
  ├─── searchModel() - 搜索数据
  ├─── createModel() - 创建记录
  ├─── updateModel() - 更新记录
  ├─── deleteModel() - 删除记录
  └─── exportModels() - 导出数据
```

---

## 使用示例

### 基础 CRUD 操作

```typescript
import { CRUD } from '@nobla/rest-ui'

const crud = new CRUD({
  module: 'user',
  table: 'admin',
  httpClient: axios,
})

async function init() {
  // 加载 schema
  await crud.initialize()
  
  // 搜索
  await crud.searchModel()
  console.log('数据：', crud.getModels())
  console.log('分页：', crud.pagination)
}

async function create() {
  const newModel = await crud.createModel({
    username: 'newadmin',
    email: 'new@example.com',
  })
  console.log('创建成功：', newModel)
}

async function update() {
  const updatedModel = await crud.updateModel({
    id: 1,
    username: 'updated',
  })
  console.log('更新成功：', updatedModel)
}

async function remove() {
  await crud.deleteModel('1')
  console.log('删除成功')
}

async function searchWithFilter() {
  crud.setQueryParams({ status: 'active', role: 'admin' })
  crud.setSortable('created_at', 'descending')
  crud.resetPagination()
  
  const models = await crud.searchModel()
  console.log('筛选结果：', models)
}

init()
```

### 固定查询条件

```typescript
const crud = new CRUD({
  module: 'article',
  table: 'article',
  httpClient: axios,
})

await crud.initialize()

// 设置固定查询：只显示未删除的文章
crud.setFixedQuery({ deleted_at: null })

// 用户搜索时自动附加 deleted_at=null
await crud.searchModel()

// 切换筛选条件时，fixedQuery 保持不变
crud.setQueryParams({ category: 'tech' })
await crud.searchModel()
// 实际请求包含：category=tech&deleted_at=null
```

### 分页浏览

```typescript
async function goToPage(page: number) {
  crud.setPaginationIndex(page)
  await crud.searchModel()
}

async function changePageSize(size: number) {
  crud.setPaginationSize(size)
  crud.resetPagination()
  await crud.searchModel()
}

// 第一页
await goToPage(1)

// 第二页
await goToPage(2)

// 每页 50 条
await changePageSize(50)
```

### 排序

```typescript
// 按创建时间降序
crud.setSortable('created_at', 'descending')
await crud.searchModel()

// 按用户名升序
crud.setSortable('username', 'ascending')
await crud.searchModel()

// 清除排序
crud.setSortable('', 'ascending')
await crud.searchModel()
```

---

## 扩展 CRUD 类

可以通过继承 `CRUD` 类来添加自定义业务方法。

### 自定义业务方法

```typescript
import { CRUD } from '@nobla/rest-ui'
import type { Model } from '@nobla/rest-ui'

class OrderCRUD extends CRUD {
  // 审批订单
  async approveOrder(orderId: string): Promise<Model> {
    const uri = this.getSchemas().length > 0
      ? `/${this.opts.apiPrefix}/${this.opts.module}/${this.opts.table}/${orderId}/approve`
      : ''
    const res = await this.opts.httpClient.post(uri)
    return res
  }
  
  // 取消订单
  async cancelOrder(orderId: string, reason: string): Promise<Model> {
    const uri = this.getSchemas().length > 0
      ? `/${this.opts.apiPrefix}/${this.opts.module}/${this.opts.table}/${orderId}/cancel`
      : ''
    const res = await this.opts.httpClient.post(uri, { reason })
    return res
  }
  
  // 发货
  async shipOrder(orderId: string, trackingNumber: string): Promise<Model> {
    const uri = this.getSchemas().length > 0
      ? `/${this.opts.apiPrefix}/${this.opts.module}/${this.opts.table}/${orderId}/ship`
      : ''
    const res = await this.opts.httpClient.post(uri, { tracking_number: trackingNumber })
    return res
  }
}

// 使用
const orderCrud = new OrderCRUD({
  module: 'shop',
  table: 'order',
  httpClient: axios,
})

await orderCrud.initialize()
await orderCrud.approveOrder('123')
await orderCrud.shipOrder('123', 'SF123456')
```

### 自定义 URI 构建

```typescript
class CustomCRUD extends CRUD {
  // 注意：__buildModelUri 是 private 方法，无法直接覆盖。
  // 如需自定义 URI，建议在子类中重写 searchModel/createModel/updateModel/deleteModel 等方法，
  // 或在构造时通过外部配置控制路径。
  // 以下为示例思路（非直接覆盖）：
  private __customBuildUri(
    moduleName: string,
    tableName: string,
    scenario: string,
    primaryKey?: string
  ): string {
    // 自定义 URI 格式
    const pk = primaryKey || ''
    switch (scenario) {
      case 'search':
        return `/api/${moduleName}/${tableName}/list`
      case 'create':
        return `/api/${moduleName}/${tableName}/add`
      case 'update':
        return `/api/${moduleName}/${tableName}/update/${pk}`
      case 'delete':
        return `/api/${moduleName}/${tableName}/remove/${pk}`
      default:
        // 注意：super.__buildModelUri 无法调用（private），
        // 实际开发中应手动拼接路径或避免继承方式自定义 URI
        return `/api/${moduleName}/${tableName}/${scenario}`
    }
  }
}
```

### 自定义响应处理

```typescript
class CustomCRUD extends CRUD {
  async searchModel(): Promise<Model[]> {
    // 自定义查询参数构建
    const queryParams = { ...this.queryParams }
    queryParams._t = Date.now() // 添加时间戳防止缓存
    
    // 调用父类方法但使用自定义参数
    const res = await this.opts.httpClient.get(
      this.__buildUri('search'),
      { params: queryParams }
    )
    
    // 自定义响应解析
    this.pagination.index = (parseInt(res.page) || 0) + 1
    this.pagination.size = parseInt(res.page_size) || 15
    this.pagination.totalCount = parseInt(res.total_count) || 0
    this.models = res.data || []
    
    return this.models
  }
}
```
