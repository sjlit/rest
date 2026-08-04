# @nobla/rest-ui

基于 REST Schema 定义的 Vue 3 + Element Plus 组件库，提供 schema 驱动的自动 CRUD 页面渲染能力。

## 设计理念

本库以 REST 项目的 `schema` 模块定义为唯一类型标准，通过后端返回的 schema 元数据自动渲染搜索表单、数据表格、创建/编辑对话框等 UI 元素，实现"零代码"或"低代码"的 CRUD 页面开发。

### 三层架构

```
core/      -- 纯逻辑层（类型定义、常量、工具函数）
runtime/   -- Vue 运行时层（配置注入、CRUD 状态管理）
ui/        -- UI 组件层（Element Plus 组件封装）
```

- **core** 层不依赖 Vue 或 Element Plus，可在任何 JS 环境使用
- **runtime** 层依赖 Vue 的 `provide/inject`，提供全局配置和 CRUD 状态管理
- **ui** 层依赖 Element Plus，提供可复用的 schema 驱动组件

## 特性

- **Schema 驱动**：基于后端 schema 定义自动渲染表单、表格、CRUD 页面
- **全自动 CRUD**：`SchemaViewer` 组件一行代码完成完整的增删改查页面
- **手动控制**：`SchemaPage` 组件提供底层编排，外部控制数据流
- **响应式设计**：`SchemaGrid` 自动适配移动端（折叠面板）和桌面端（表格）
- **内置样式体系**：基于 Element Plus CSS 变量的自适应阴影、间距系统和响应式断点
- **插件化配置**：通过 Vue Plugin 全局注入 HTTP 客户端、权限、路由、国际化
- **类型安全**：完整的 TypeScript 类型定义，与 REST Go 结构体对齐
- **高度可扩展**：丰富的插槽系统支持自定义搜索表单、表格列、表单字段

## 安装

```bash
npm install @nobla/rest-ui
```

### Peer Dependencies

```bash
npm install vue@^3.3.0 element-plus@^2.12.0 @element-plus/icons-vue@^2.3.0
```

## 快速开始

### 1. 注册插件

在应用入口文件中注册 `SchemaUIPlugin`，传入全局配置：

```typescript
import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import { SchemaUIPlugin } from '@nobla/rest-ui'
import axios from 'axios'
import App from './App.vue'

const app = createApp(App)

app.use(ElementPlus)
app.use(SchemaUIPlugin, {
  // 必填：HTTP 客户端
  httpClient: {
    get: (url, config) => axios.get(url, config),
    post: (url, data, config) => axios.post(url, data, config),
    put: (url, data, config) => axios.put(url, data, config),
    delete: (url, data) => axios.delete(url, { data }),
  },
  // 可选：权限检查
  hasPermission: (permission) => {
    const userPermissions = JSON.parse(localStorage.getItem('permissions') || '[]')
    return userPermissions.includes(permission)
  },
  // 可选：路由跳转
  router: {
    push: (to) => app.config.globalProperties.$router.push(to),
  },
  // 可选：国际化
  i18n: {
    t: (key, ...args) => {
      const messages: Record<string, string> = {
        'validation.required': `${args[0]}不能为空`,
        'validation.max': `${args[0]}不能超过${args[1]}个字符`,
        'validation.pattern': `${args[0]}格式不正确`,
      }
      return messages[key] || key
    },
  },
  // 可选：API 前缀
  apiPrefix: 'rest',
})

app.mount('#app')
```

### 2. 使用 SchemaViewer（全自动 CRUD）

```vue
<template>
  <SchemaViewer module="user" table="admin" title="管理员管理" />
</template>

<script setup lang="ts">
import { SchemaViewer } from '@nobla/rest-ui'
</script>
```

### 3. 使用 SchemaPage（手动控制数据流）

```vue
<template>
  <SchemaPage
    :schemas="schemas"
    :models="models"
    :pagination="pagination"
    title="手动控制示例"
    @search="handleSearch"
    @pageChange="handlePageChange"
    @formSubmit="handleFormSubmit"
  />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { SchemaPage } from '@nobla/rest-ui'
import type { Schema, Model, Pagination } from '@nobla/rest-ui'

const schemas = ref<Schema[]>([])
const models = ref<Model[]>([])
const pagination = ref<Pagination>({ index: 1, size: 15, totalCount: 0 })

async function loadSchemas() {
  // 从后端加载 schema 定义
  const res = await fetch('/rest/schema/user/admin')
  schemas.value = await res.json()
}

async function handleSearch(model: Model) {
  const res = await fetch('/rest/user/admins?' + new URLSearchParams(model))
  const data = await res.json()
  models.value = data.data
  pagination.value = {
    index: parseInt(data.page),
    size: parseInt(data.page_size),
    totalCount: parseInt(data.total_count),
  }
}

function handlePageChange(index: number) {
  pagination.value.index = index
  handleSearch({})
}

function handleFormSubmit(model: Model, scenario: string) {
  if (scenario === 'create') {
    fetch('/rest/user/admin', { method: 'POST', body: JSON.stringify(model) })
  } else {
    fetch(`/rest/user/admin/${model.id}`, { method: 'PUT', body: JSON.stringify(model) })
  }
}

loadSchemas()
</script>
```

---

## 配置系统

### SchemaUIConfig

全局配置通过 `SchemaUIPlugin` 注入，所有字段均为可选，但 `httpClient` 是 `SchemaViewer` 必需的。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `httpClient` | `{ get, post, put, delete }` | 是 | HTTP 客户端，用于 `SchemaViewer` 和 `CRUD` 类 |
| `hasPermission` | `(permission: string) => boolean` | 否 | 权限检查函数 |
| `router` | `{ push: (to: any) => void }` | 否 | 路由对象，用于页面跳转 |
| `i18n` | `{ t: (key: string, ...args: any[]) => string }` | 否 | 国际化翻译函数 |
| `apiPrefix` | `string` | 否 | API 路由前缀（默认 `'rest'`） |

### useSchemaUI

在组件或 composable 中获取全局配置：

```typescript
import { useSchemaUI } from '@nobla/rest-ui'

const config = useSchemaUI()
// config.httpClient, config.i18n, config.hasPermission, ...
```

> **注意**：`useSchemaUI()` 在插件未注册时会抛出错误。如果需要在不注册插件的情况下使用 `SchemaPage` 等纯展示组件，请避免调用此方法。

---

## 核心类型

### Schema

后端 schema 定义的核心接口，与 REST 项目的 Go 结构体对齐。

```typescript
interface Schema {
  id?: number
  created_at?: number
  updated_at?: number
  tenant_id?: string
  module_name: string
  table_name: string
  enable: number
  column: string        // 字段名
  label: string         // 显示标签
  type: string          // 数据类型：integer | float | boolean | string
  format: string        // 显示格式：text | dropdown | datetime | date | time | password | ...
  native: number
  primary_key: number   // 1 = 主键，0 = 非主键
  expression: string
  scenarios: string[]   // 适用场景：create | update | delete | search | list | detail | export
  rules: SchemaRule
  attributes: SchemaAttribute
  relations: Relation
  position: number
}
```

### SchemaRule

字段验证规则。

```typescript
interface SchemaRule {
  min: number
  max: number
  type: string
  unique: boolean
  required: string[]    // 必填场景列表
  regular?: string      // 正则表达式
  safe?: boolean
}
```

### SchemaAttribute

字段扩展属性。

```typescript
interface SchemaAttribute {
  match: string
  tag?: string
  default_value: string
  readonly: string[]    // 只读场景列表
  disable: string[]     // 禁用场景列表
  visible: VisibleCondition[]  // 可见性条件
  invisible: boolean    // 是否完全隐藏
  end_of_now: boolean
  time_search_range: string
  values?: EnumValue[]  // 枚举值列表（下拉选项）
  live: LiveValue       // 动态加载配置
  upload_url?: string   // 文件上传地址
  icon?: string
  sort: boolean         // 是否支持排序
  suffix?: string       // 输入框后缀
  tooltip?: string      // 提示文本
  dropdown?: DropdownOptions  // 下拉框配置
  description?: string
}
```

### Action

操作按钮定义，用于表格行操作、表单操作、搜索操作等。

```typescript
interface Action {
  name: string
  label: string
  type?: string         // Element Plus 按钮类型：primary | success | danger | warning | info
  icon?: string
  round?: boolean
  size?: string
  permission?: string   // 所需权限标识
  selection?: boolean   // 预留字段，当前未生效
  hidden?: boolean | ((model: Model) => boolean | Promise<boolean>)
  callback?: (model: Model, schemas?: Schema[], loading?: any) => void
  asyncCallback?: (model: Model, schemas?: Schema[], action?: Action) => Promise<void>
}
```

### Model

数据模型，任意键值对。

```typescript
interface Model {
  [key: string]: any
}
```

### Pagination

分页信息。

```typescript
interface Pagination {
  index: number         // 当前页码
  size: number          // 每页条数
  totalCount: number    // 总条数
}
```

### CRUDOptions

CRUD 类构造选项。

```typescript
interface CRUDOptions {
  module?: string
  table?: string
  apiPrefix?: string
  schemas?: Schema[] | Record<string, Schema>
  httpClient: SchemaUIConfig['httpClient'] // 必须，用于 HTTP 请求
}
```

---

## 核心常量

### 数据类型常量

```typescript
TypeInteger   = 'integer'
TypeFloat     = 'float'
TypeBoolean   = 'boolean'
TypeString    = 'string'
```

### 格式常量

```typescript
FormatInteger    = 'integer'
FormatFloat      = 'float'
FormatBoolean    = 'boolean'
FormatString     = 'string'
FormatText       = 'text'
FormatDropdown   = 'dropdown'
FormatDatetime   = 'datetime'
FormatDate       = 'date'
FormatTime       = 'time'
FormatTimestamp  = 'timestamp'
FormatPassword   = 'password'
```

### 场景常量

```typescript
ScenarioCreate  = 'create'
ScenarioUpdate  = 'update'
ScenarioDelete  = 'delete'
ScenarioSearch  = 'search'
ScenarioExport  = 'export'
ScenarioList    = 'list'
ScenarioDetail  = 'detail'
```

### 匹配模式

```typescript
MatchExactly  = 'exactly'
MatchFuzzy    = 'fuzzy'
```

### Live 类型

```typescript
LiveTypeDropdown   = 'dropdown'
LiveTypeCascader   = 'cascader'
```

---

## 工具函数

### Scenarios

场景数组的扩展类，提供便捷的 `has()` 方法。

```typescript
import { Scenarios } from '@nobla/rest-ui'

const scenarios = Scenarios.from('create;update;list')
scenarios.has('create')  // true
scenarios.has('delete')  // false
```

### encode / decode

模型值的编码/解码函数，用于表单提交前后的数据转换。

```typescript
import { encode, decode } from '@nobla/rest-ui'

// encode: Date -> 格式字符串（YYYY-MM-DD HH:mm:ss）
const submitModel = encode(model, schemas, 'create')

// decode: string -> number/boolean
const formModel = decode(rawModel, schemas, 'create')
```

转换规则：
- **encode**：`datetime/date/timestamp/time` 格式的 `Date` 对象转为 `YYYY-MM-DD HH:mm:ss` 格式字符串
- **decode**：`integer` 类型的字符串转为整数；`float/double/decimal` 转为浮点数；`boolean` 格式转为布尔值

### getModelValue / getModelLabel

```typescript
import { getModelValue, getModelLabel } from '@nobla/rest-ui'

const value = getModelValue(model, 'status')      // 获取原始值
const label = getModelLabel(model, 'status')      // 获取显示标签（支持枚举映射）
```

### generateSchemaRule

根据 schema 生成 Element Plus 表单验证规则。

```typescript
import { generateSchemaRule } from '@nobla/rest-ui'

const rules = generateSchemaRule(
  (key, args) => `${args[0]}不能为空`,  // 翻译函数
  schema,
  'create'  // 场景
)
// 返回 Element Plus 的 rules 数组
```

### checkSchemaVisible

检查字段在指定模型下是否可见（根据 `visible` 条件）。

```typescript
import { checkSchemaVisible } from '@nobla/rest-ui'

const isVisible = checkSchemaVisible(schema, model)
```

### clearSearchModel

清除搜索模型中的空值（`''`、`null`、`undefined`）。

```typescript
import { clearSearchModel } from '@nobla/rest-ui'

const cleanQuery = clearSearchModel(searchModel, schemas)
```

---

## 组件 API

### SchemaViewer

全自动 CRUD 组件，内部管理所有 HTTP 请求，一行代码完成完整页面。

#### Props

| Prop | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `module` | `string` | - | 模块名 |
| `table` | `string` | - | 表名 |
| `title` | `string` | `''` | 页面标题 |
| `apiPrefix` | `string` | `''` | API 前缀，默认使用全局配置 |
| `config` | `Partial<CRUDOptions>` | `{}` | CRUD 配置覆盖 |
| `size` | `string` | - | 预留字段，当前未生效 |
| `formMode` | `'drawer' \| 'dialog'` | `'dialog'` | 表单弹窗模式 |
| `showHeader` | `boolean` | `true` | 是否显示头部 |
| `showSearch` | `boolean` | `true` | 是否显示搜索栏 |
| `showToolbar` | `boolean` | `true` | 是否显示工具栏 |
| `showPagination` | `boolean` | `true` | 是否显示分页 |
| `readonly` | `boolean` | `false` | 只读模式（隐藏编辑/删除/创建按钮） |
| `autoFetch` | `boolean` | `true` | 初始化时是否自动加载数据 |
| `rowActions` | `Action[]` | `[]` | 自定义行操作按钮 |
| `batchActions` | `Action[]` | `[]` | 自定义批量操作按钮 |
| `formActions` | `Action[]` | `[]` | 自定义表单操作按钮 |
| `searchActions` | `Action[]` | `[]` | 自定义搜索操作按钮 |
| `defaultSort` | `string` | `''` | 默认排序字段，前缀 `-` 表示降序 |
| `presetQuery` | `Record<string, any>` | `{}` | 预设查询参数 |
| `gridProps` | `Record<string, any>` | `{}` | 传递给 SchemaGrid 的额外属性 |
| `formProps` | `Record<string, any>` | `{}` | 传递给 SchemaForm 的额外属性 |

#### Events

| Event | 参数 | 说明 |
|-------|------|------|
| `ready` | `(crud: CRUD)` | CRUD 实例初始化完成 |

#### Slots

| Slot | 参数 | 说明 |
|------|------|------|
| `searchform` | `{ model, schema }` | 自定义搜索表单字段 |
| `gridview` | `{ model, schema }` | 自定义表格列内容 |
| `crudform` | `{ model, schema }` | 自定义表单字段 |
| `headerleft` | - | 自定义头部左侧 |
| `headerright` | - | 自定义头部右侧 |

#### 使用示例

```vue
<template>
  <SchemaViewer
    module="order"
    table="order"
    title="订单管理"
    :readonly="false"
    :autoFetch="true"
    defaultSort="-created_at"
    :presetQuery="{ status: 'pending' }"
    :rowActions="customRowActions"
    @ready="onReady"
  >
    <template #gridview="{ model, schema }">
      <span v-if="schema.column === 'status'" :class="`status-${model.status}`">
        {{ statusMap[model.status] }}
      </span>
    </template>
  </SchemaViewer>
</template>

<script setup lang="ts">
import { SchemaViewer } from '@nobla/rest-ui'
import type { CRUD, Action } from '@nobla/rest-ui'

const customRowActions: Action[] = [
  {
    name: 'detail',
    label: '详情',
    type: 'primary',
    callback: (model) => {
      console.log('View detail:', model)
    },
  },
]

function onReady(crud: CRUD) {
  console.log('CRUD ready:', crud.getSchemas())
}
</script>
```

---

### SchemaPage

底层编排组件，组合搜索栏 + 表格 + 分页 + 弹窗/抽屉，不发送 HTTP 请求，所有数据流通过 props/events 控制。

#### Props

| Prop | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `schemas` | `Schema[]` | - | Schema 定义数组 |
| `models` | `Model[]` | - | 数据列表 |
| `pagination` | `Pagination` | `{ index: 1, size: 15, totalCount: 0 }` | 分页信息 |
| `loading` | `boolean` | `false` | 加载状态 |
| `size` | `string` | - | 预留字段，当前未生效 |
| `title` | `string` | - | 页面标题 |
| `formMode` | `'drawer' \| 'dialog'` | `'dialog'` | 表单弹窗模式 |
| `showHeader` | `boolean` | `true` | 是否显示头部 |
| `showSearch` | `boolean` | `true` | 是否显示搜索栏 |
| `showToolbar` | `boolean` | `true` | 是否显示工具栏 |
| `showPagination` | `boolean` | `true` | 是否显示分页 |
| `readonly` | `boolean` | `false` | 只读模式 |
| `searchActions` | `Action[]` | `[]` | 搜索操作按钮 |
| `rowActions` | `Action[]` | `[]` | 行操作按钮 |
| `batchActions` | `Action[]` | `[]` | 批量操作按钮 |
| `formActions` | `Action[]` | `[]` | 表单操作按钮 |
| `gridProps` | `Record<string, any>` | `{}` | 传递给 SchemaGrid 的属性 |
| `formProps` | `Record<string, any>` | `{}` | 传递给 SchemaForm 的属性 |

#### Events

| Event | 参数 | 说明 |
|-------|------|------|
| `search` | `(model: Model)` | 搜索提交 |
| `create` | - | 点击创建按钮 |
| `edit` | `(model: Model)` | 点击编辑按钮 |
| `delete` | `(model: Model)` | 点击删除按钮 |
| `pageChange` | `(index: number)` | 分页切换 |
| `sortChange` | `{ column, order }` | 排序变化 |
| `selectionChange` | `(selection: any[])` | 选中项变化 |
| `formSubmit` | `(model: Model, scenario: string)` | 表单提交 |

#### Slots

与 `SchemaViewer` 相同：`searchform`、`gridview`、`crudform`、`headerleft`、`headerright`。

#### 使用示例

```vue
<template>
  <SchemaPage
    :schemas="schemas"
    :models="models"
    :pagination="pagination"
    :loading="loading"
    title="手动数据流示例"
    @search="handleSearch"
    @pageChange="handlePageChange"
    @formSubmit="handleFormSubmit"
    @delete="handleDelete"
  />
</template>
```

---

### SchemaGrid

响应式数据表格组件，桌面端使用 `el-table`，移动端（< 768px）自动切换为折叠面板视图。

#### Props

| Prop | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `schemas` | `Schema[]` | - | Schema 定义数组 |
| `models` | `Model[]` | - | 数据列表 |
| `scenario` | `string` | `'list'` | 场景 |
| `size` | `string` | - | 预留字段，当前未生效 |
| `selection` | `boolean` | `true` | 是否显示多选列 |
| `actions` | `Action[]` | `[]` | 行操作按钮 |
| `gridProps` | `Record<string, any>` | `{}` | 传递给 `el-table` 的属性 |
| `responsive` | `boolean` | `true` | 是否启用响应式 |
| `loading` | `boolean` | `false` | 加载状态 |

#### Events

| Event | 参数 | 说明 |
|-------|------|------|
| `selection` | `(selection: any[])` | 多选变化 |
| `sort` | `{ column, order }` | 排序变化 |

#### Slots

| Slot | 参数 | 说明 |
|------|------|------|
| `default` | `{ model, schema }` | 自定义单元格内容 |

---

### SchemaForm

Schema 驱动的表单组件，支持网格布局、行内布局、响应式断点和场景过滤。

#### Props

| Prop | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `schemas` | `Schema[]` | - | Schema 定义数组 |
| `model` | `Model` | `undefined` | 初始模型数据 |
| `scenario` | `string` | `'create'` | 场景 |
| `size` | `string` | - | 预留字段，当前未生效 |
| `labelWidth` | `string` | `''` | 标签宽度 |
| `inline` | `boolean` | `false` | 行内表单模式 |
| `grid` | `boolean` | `false` | 网格布局模式 |
| `gridCols` | `number` | `0` | 网格列数（0 表示自动） |
| `actions` | `Action[]` | `[]` | 表单操作按钮 |
| `autoSubmit` | `boolean` | `false` | 挂载后自动提交 |

#### Events

| Event | 参数 | 说明 |
|-------|------|------|
| `submit` | `(model: Model, schemas: Schema[])` | 表单验证通过并提交 |

#### Slots

| Slot | 参数 | 说明 |
|------|------|------|
| `container` | `{ model, schemas }` | 自定义整个表单容器 |
| `default` | `{ model, schema }` | 自定义单个表单字段 |

#### Expose

| 方法 | 返回 | 说明 |
|------|------|------|
| `submit` | `Promise<Model>` | 手动触发提交和验证 |

---

## CRUD 类

`CRUD` 类封装了完整的 RESTful HTTP 操作，包括 schema 加载、数据搜索、增删改查、导出等功能。

### 构造函数

```typescript
const crud = new CRUD({
  module: 'user',
  table: 'admin',
  apiPrefix: 'rest',
  httpClient: axiosInstance,
  schemas: preloadedSchemas,  // 可选：预加载 schema
})
```

### 属性

| 属性 | 类型 | 说明 |
|------|------|------|
| `primaryKey` | `string` | 主键字段名 |
| `schemas` | `Schema[]` | 当前 schema 列表 |
| `models` | `Model[]` | 当前数据列表 |
| `sortable` | `Sortable \| null` | 当前排序 |
| `queryParams` | `Record<string, any>` | 查询参数 |
| `fixedQuery` | `Record<string, any>` | 固定查询参数（每次请求自动附加） |
| `pagination` | `Pagination` | 分页信息 |
| `fieldErrors` | `Record<string, string>` | 字段错误信息 |

### 方法

| 方法 | 参数 | 返回 | 说明 |
|------|------|------|------|
| `initialize()` | - | `Promise<Schema[]>` | 初始化：加载 schema、识别主键、拉取 live 数据 |
| `getSchemas()` | - | `Schema[]` | 获取 schema 列表 |
| `getModels()` | - | `Model[]` | 获取数据列表 |
| `setColumnError(column, error)` | `(string, string)` | `void` | 设置字段错误 |
| `resetError()` | - | `void` | 重置所有字段错误 |
| `getFieldErrors()` | - | `Record<string, string>` | 获取字段错误映射 |
| `setPaginationIndex(index)` | `number` | `this` | 设置当前页 |
| `getPaginationIndex()` | - | `number` | 获取当前页 |
| `setPaginationSize(size)` | `number` | `this` | 设置每页条数 |
| `getPaginationSize()` | - | `number` | 获取每页条数 |
| `getPaginationCount()` | - | `number` | 获取总条数 |
| `resetPagination()` | - | `this` | 重置到第一页 |
| `setSortable(column, order)` | `(string, 'ascending' \| 'descending')` | `this` | 设置排序 |
| `addQueryParams(k, v)` | `(string, any)` | `void` | 添加查询参数 |
| `setQueryParams(qs)` | `Record<string, any>` | `this` | 设置查询参数 |
| `setFixedQuery(qs)` | `Record<string, any>` | `this` | 设置固定查询参数 |
| `findModelPrimaryKey(model)` | `Model` | `any` | 获取模型主键值 |
| `createModel(model)` | `Model` | `Promise<Model>` | 创建记录 |
| `updateModel(model)` | `Model` | `Promise<Model>` | 更新记录 |
| `deleteModel(model)` | `Model \| string` | `Promise<any>` | 删除记录 |
| `getModel(qs)` | `Record<string, any> \| string` | `Promise<Model>` | 获取单条记录详情 |
| `searchModel()` | - | `Promise<Model[]>` | 搜索记录 |
| `deleteModels(data)` | `any[]` | `Promise<{ total, success, responses }>` | 批量删除 |
| `exportModels()` | - | `Promise<void>` | 导出数据（自动下载 CSV） |

### URI 构建规则

CRUD 类使用 `pluralize` 库根据 `module`、`table` 和 `scenario` 自动构建 RESTful URI：

| Scenario | URI 示例（module=user, table=admin） |
|----------|--------------------------------------|
| `create` | `/rest/user/admin` |
| `update` | `/rest/user/admin/123` |
| `delete` | `/rest/user/admin/123` |
| `get` | `/rest/user/admin/detail/123` |
| `search` | `/rest/user/admins` |
| `export` | `/rest/user/admin/export` |

Schema 加载 URI：
- 有 module：`GET /rest/schema/{module}/{table}`
- 无 module：`GET /rest/schema/{table}`

---

## 进阶示例

### 自定义搜索表单

```vue
<template>
  <SchemaViewer module="order" table="order">
    <template #searchform="{ model, schema }">
      <el-date-picker
        v-if="schema.column === 'date_range'"
        v-model="model.date_range"
        type="daterange"
      />
    </template>
  </SchemaViewer>
</template>
```

### 自定义表格列

```vue
<template>
  <SchemaViewer module="user" table="user">
    <template #gridview="{ model, schema }">
      <el-avatar v-if="schema.column === 'avatar'" :src="model.avatar" />
      <el-tag v-else-if="schema.column === 'role'" :type="roleType(model.role)">
        {{ model.role }}
      </el-tag>
    </template>
  </SchemaViewer>
</template>
```

### 自定义表单字段

```vue
<template>
  <SchemaViewer module="article" table="article">
    <template #crudform="{ model, schema }">
      <RichEditor
        v-if="schema.column === 'content'"
        v-model="model.content"
      />
    </template>
  </SchemaViewer>
</template>
```

### 批量操作

```vue
<script setup lang="ts">
import { SchemaViewer } from '@nobla/rest-ui'
import type { Action } from '@nobla/rest-ui'

const batchActions: Action[] = [
  {
    name: 'batchDelete',
    label: '批量删除',
    type: 'danger',
    callback: (selections) => {
      console.log('Batch delete:', selections)
    },
  },
  {
    name: 'batchExport',
    label: '批量导出',
    callback: (selections) => {
      console.log('Batch export:', selections)
    },
  },
]
</script>

<template>
  <SchemaViewer module="user" table="user" :batchActions="batchActions" />
</template>
```

### 权限控制

```vue
<script setup lang="ts">
const rowActions: Action[] = [
  {
    name: 'edit',
    label: '编辑',
    type: 'success',
    permission: 'user:edit',
    callback: (model) => { /* ... */ },
  },
  {
    name: 'delete',
    label: '删除',
    type: 'danger',
    permission: 'user:delete',
    hidden: (model) => model.status === 'locked',
    callback: (model) => { /* ... */ },
  },
]
</script>
```

### 手动数据流（完全自定义）

```vue
<template>
  <SchemaPage
    :schemas="schemas"
    :models="models"
    :pagination="pagination"
    :loading="loading"
    :rowActions="rowActions"
    @search="onSearch"
    @pageChange="onPageChange"
    @formSubmit="onFormSubmit"
    @delete="onDelete"
  />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { SchemaPage, CRUD } from '@nobla/rest-ui'
import type { Schema, Model, Pagination, Action } from '@nobla/rest-ui'

const schemas = ref<Schema[]>([])
const models = ref<Model[]>([])
const pagination = ref<Pagination>({ index: 1, size: 15, totalCount: 0 })
const loading = ref(false)

// 使用 CRUD 类辅助 URI 构建
const crud = new CRUD({
  module: 'custom',
  table: 'data',
  httpClient: { /* ... */ },
})

async function loadData() {
  loading.value = true
  await crud.initialize()
  const result = await crud.searchModel()
  models.value = result
  pagination.value = { ...crud.pagination }
  loading.value = false
}

function onSearch(model: Model) {
  crud.setQueryParams(model).resetPagination()
  loadData()
}

function onPageChange(index: number) {
  crud.setPaginationIndex(index)
  loadData()
}

function onFormSubmit(model: Model, scenario: string) {
  if (scenario === 'create') {
    crud.createModel(model).then(loadData)
  } else {
    crud.updateModel(model).then(loadData)
  }
}

function onDelete(model: Model) {
  crud.deleteModel(model).then(loadData)
}

const rowActions: Action[] = [
  { name: 'detail', label: '详情', callback: (model) => openDetail(model) },
]
</script>
```

### 固定查询条件

```vue
<script setup lang="ts">
import { SchemaViewer } from '@nobla/rest-ui'
import { onMounted, ref } from 'vue'
import type { CRUD } from '@nobla/rest-ui'

const viewerRef = ref<InstanceType<typeof SchemaViewer> | null>(null)

function onReady(crud: CRUD) {
  // 设置固定查询条件：只显示未删除的数据
  crud.setFixedQuery({ deleted_at: null })
  crud.searchModel()
}
</script>

<template>
  <SchemaViewer
    module="article"
    table="article"
    :autoFetch="false"
    @ready="onReady"
  />
</template>
```

### 动态切换 module/table

```vue
<template>
  <SchemaViewer :module="currentModule" :table="currentTable" />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { SchemaViewer } from '@nobla/rest-ui'

const currentModule = ref('user')
const currentTable = ref('admin')

// 切换时会自动重新初始化
function switchTable(module: string, table: string) {
  currentModule.value = module
  currentTable.value = table
}
</script>
```

---

## 与 REST 后端对接

### Schema 接口约定

后端需要提供以下接口返回 schema 定义：

```
GET /{apiPrefix}/schema/{module}/{table}
GET /{apiPrefix}/schema/{table}
```

响应格式：

```json
[
  {
    "id": 1,
    "module_name": "user",
    "table_name": "admin",
    "column": "username",
    "label": "用户名",
    "type": "string",
    "format": "text",
    "primary_key": 0,
    "scenarios": ["create", "update", "search", "list"],
    "rules": {
      "min": 0,
      "max": 50,
      "type": "string",
      "unique": true,
      "required": ["create", "update"]
    },
    "attributes": {
      "match": "fuzzy",
      "readonly": [],
      "disable": [],
      "visible": [],
      "invisible": false,
      "sort": true,
      "values": [],
      "live": { "enable": false, "type": "" }
    }
  }
]
```

### 搜索接口约定

```
GET /{apiPrefix}/{module}/{pluralTable}?page=1&pagesize=15&sort=-created_at&__format=both
```

响应格式：

```json
{
  "page": "1",
  "page_size": "15",
  "total_count": "100",
  "data": [{ ... }]
}
```

### Live 数据加载

当 schema 的 `attributes.live.enable` 为 `true` 时，CRUD 类会自动请求 `live.url` 加载下拉选项：

```json
{
  "attributes": {
    "live": {
      "enable": true,
      "type": "dropdown",
      "url": "/rest/user/roles",
      "method": "GET"
    }
  }
}
```

---

## 开发指南

### 本地开发

```bash
cd packages/rest-ui
npm install
npm run dev    # 监听模式构建
npm test       # 运行单元测试（node:test，无额外依赖）
```

### 在独立 Vue 项目中本地引用

如果你有一个**独立的 Vue 前端项目**想要在开发时引用本库，推荐以下两种方式：

#### 方式一：Vite Alias 指向源码（推荐，支持热更新）

在消费项目的 `vite.config.ts` 中配置路径别名，直接指向 `@nobla/rest-ui` 的源码入口：

```typescript
import { defineConfig } from 'vite'
import { resolve } from 'path'

export default defineConfig({
  resolve: {
    alias: {
      '@nobla/rest-ui': resolve(__dirname, '/absolute/path/to/rest/packages/rest-ui/src/index.ts'),
    },
  },
})
```

**步骤**：
1. 在消费项目中安装本库的 `peerDependencies`：
   ```bash
   npm install vue@^3.3.0 element-plus@^2.12.0 @element-plus/icons-vue@^2.3.0
   ```
2. 在 `vite.config.ts` 添加上述 `alias` 配置（路径替换为你本地的实际绝对路径）。
3. 启动消费项目，修改 `rest-ui` 源码即可实时热更新。

> **注意**：此方式要求消费项目具备编译 `.vue` 单文件组件的能力（已安装 `@vitejs/plugin-vue`）。

#### 方式二：本地路径安装 + Watch 构建

如果你不方便修改消费项目的 Vite 配置，可以通过 `file:` 协议直接安装本地包：

```bash
# 在消费项目中执行
npm install /absolute/path/to/rest/packages/rest-ui
```

`package.json` 会自动添加：
```json
"dependencies": {
  "@nobla/rest-ui": "file:/absolute/path/to/rest/packages/rest-ui"
}
```

**步骤**：
1. 在消费项目中执行上述 `npm install` 命令。
2. 确保消费项目已安装 `peerDependencies`（`vue`、`element-plus`、`@element-plus/icons-vue`）。
3. 在 `rest-ui` 目录启动监听构建：
   ```bash
   cd packages/rest-ui
   npm run dev   # vite build --watch，自动更新 dist/
   ```
4. 消费项目刷新页面即可获得最新构建产物。

> **提示**：`npm run dev` 会监听源码变化并自动重新打包到 `dist/`。由于 `file:` 链接指向包的根目录，引用的是 `dist/` 下的构建产物，因此必须保持 watch 构建处于运行状态。

### 构建

```bash
npm run build  # 输出到 dist/ 目录
```

输出文件：
- `dist/rest-ui.es.js` — ESM 格式
- `dist/rest-ui.cjs` — CJS 格式
- `dist/style.css` — 组件样式（自动随 JS 注入，也可单独引用）
- `dist/index.d.ts` — TypeScript 类型声明

单独引用样式（可选；组件样式默认已随 JS 自动注入）：

```typescript
import '@nobla/rest-ui/style.css'
// 或
import '@nobla/rest-ui/dist/style.css'
```

### 类型检查

```bash
npm run typecheck  # 使用 vue-tsc
```

---

## 类型声明

本库使用 TypeScript 编写，提供完整的类型声明。主要导出类型：

```typescript
import type {
  Schema,
  SchemaRule,
  SchemaAttribute,
  EnumValue,
  VisibleCondition,
  LiveValue,
  DropdownOptions,
  Relation,
  Model,
  Action,
  Pagination,
  Sortable,
  CRUDOptions,
  SchemaUIConfig,
} from '@nobla/rest-ui'
```

---

## 更新日志

### 1.0.0

- 初始版本发布
- Schema 驱动的 CRUD 组件库
- Vue 3 + Element Plus + TypeScript 支持
- 全自动 CRUD（SchemaViewer）和手动编排（SchemaPage）两种模式
- CRUD 类封装完整的 RESTful HTTP 操作
- 响应式移动端适配
- 插件化配置系统
