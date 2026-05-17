# Schema UI 前端组件库设计文档

## 1. 背景与目标

基于 REST 项目（Go 后端）的 schema 定义体系，封装一套独立发布的 Vue 3 + Element Plus 前端组件库。该库以 schema 驱动为核心，让使用方通过声明式配置快速搭建增删改查页面。

**设计原则：**
- 类型定义完全对齐 REST 项目的 `schema` 包（Go 结构体），不兼容历史包袱
- admin 项目的 `components/schema` 仅作为实现参考，提取架构思路而非字段兼容
- 通过可注入配置解耦项目基础设施（HTTP 客户端、权限、路由、国际化）
- 同时提供"一行代码搞定 CRUD"的高阶组件和"完全可控"的底层组件

## 2. 仓库组织方案

**方案：当前项目内建独立 package 目录**

在 `rest/` 仓库下新建 `packages/schema-ui/` 目录，使用 Vite Library Mode 构建为 npm package。

```
rest/
├── go.mod
├── schema/                       # Go 后端 schema 定义（唯一类型标准）
├── packages/
│   └── schema-ui/
│       ├── package.json
│       ├── vite.config.ts        # Vite Library Mode（ESM + CJS + d.ts）
│       ├── tsconfig.json
│       └── src/
│           ├── index.ts          # 入口：导出组件 + Plugin + 类型 + 工具
│           ├── plugin.ts         # Vue Plugin：全局配置注入
│           ├── config.ts         # 配置类型定义 + 全局状态
│           ├── core/             # 纯逻辑，无 Vue / UI 依赖
│           │   ├── constants.ts
│           │   ├── types.ts      # Schema 等接口（对齐 REST Go 结构体）
│           │   ├── scenarios.ts  # 场景判断工具
│           │   ├── codec.ts      # encode / decode
│           │   ├── model.ts      # 模型辅助
│           │   └── form.ts       # 表单规则生成
│           ├── runtime/          # 运行时，依赖 Vue 但不依赖 Element Plus
│           │   ├── config.ts
│           │   ├── crud.ts       # CRUD 类（TypeScript 重构）
│           │   └── useSchemaUI.ts
│           └── ui/               # Vue + Element Plus 组件
│               ├── SchemaForm.vue
│               ├── SchemaGrid.vue
│               ├── SchemaViewer.vue
│               ├── SchemaPage.vue
│               └── parts/
│                   ├── FormItem.vue
│                   ├── Cell.vue
│                   └── Action.vue
```

**构建目标：**
- `format: ['es', 'cjs']` — ESM 给 Vite 项目，CJS 给旧工具链
- `dts: true` — 自动生成 `.d.ts`
- 外部化 `vue` 和 `element-plus`（peerDependencies）

## 3. 类型定义（REST 标准）

所有 TypeScript 接口完全对齐 REST 项目 `schema` 包的 Go 结构体 JSON 序列化后的形状。

```ts
export interface Schema {
  id?: number
  created_at?: number
  updated_at?: number
  tenant_id?: string
  module_name: string
  table_name: string
  enable: number
  column: string
  label: string
  type: string        // integer | float | boolean | string
  format: string      // integer | float | boolean | string | text | dropdown | datetime | date | time | timestamp | password
  native: number
  primary_key: number // 1 = 主键
  expression: string
  scenarios: string[] // create | update | delete | search | export | import | list | detail
  rules: Rule
  attributes: Attribute
  relations: Relation
  position: number
}

export interface Rule {
  min: number
  max: number
  type: string
  unique: boolean
  required: string[]
  regular?: string
  safe?: boolean
}

export interface Attribute {
  match: string
  tag?: string
  default_value: string
  readonly: string[]
  disable: string[]
  visible: VisibleCondition[]
  invisible: boolean
  end_of_now: boolean
  time_search_range: string
  values: EnumValue[]
  live: LiveValue
  upload_url?: string
  icon?: string
  sort: boolean
  suffix?: string
  tooltip?: string
  dropdown?: DropdownOptions
  description?: string
}

export interface EnumValue {
  label: string
  value: string
  color?: string
}

export interface LiveValue {
  enable: boolean
  type: string      // dropdown | cascader
  url?: string
  method?: string
  body?: string
  content_type?: string
  columns?: string[]
}

export interface DropdownOptions {
  created?: boolean
  searchable?: boolean
  filterable?: boolean
  autocomplete?: boolean
  default_first?: boolean
  multiple?: boolean
  collapse_tags?: boolean
}

export interface Relation {
  type: string
  name: string
  module: string
  table: string
}

export interface Model {
  [key: string]: any
}

export interface Action {
  name: string
  label: string
  type?: string
  icon?: string
  round?: boolean
  size?: string
  permission?: string
  selection?: boolean
  hidden?: boolean | ((model: Model) => boolean | Promise<boolean>)
  callback?: (model: Model, schemas?: Schema[], loading?: any) => void
  asyncCallback?: (model: Model, schemas?: Schema[], action?: Action) => Promise<void>
}

export interface Pagination {
  index: number
  size: number
  totalCount: number
}

export interface Sortable {
  column: string
  order: 'ascending' | 'descending'
}

export interface CRUDOptions {
  module?: string
  table?: string
  apiPrefix?: string
  schemas?: Schema[] | Record<string, Schema>
}
```

### 3.1 与 admin 参考项目的关键差异处理

| 功能 | admin 做法 | 新库做法 |
|---|---|---|
| 主键判断 | `schema.attribute.primary_key` / `schema.is_primary_key` | `schema.primary_key === 1`（Schema 根上） |
| 搜索多选 | `schema.attribute.multiple_for_search` | `format === 'multiSelect'` 或 `attributes.dropdown?.multiple` |
| 级联数据 | `schema.attribute.values.children` | 不支持静态 children；级联通过 `live` 动态加载 |
| 错误提示 | `schema.error` 挂载在 schema 上 | 组件内部维护 `fieldErrors: Record<string, string>`，不污染 schema |
| 场景判断 | 数组 `indexOf` | 提供 `Scenarios` 工具类（`has(scenario): boolean`） |

## 4. 配置系统 —— Plugin + Props 混合注入

### 4.1 全局配置接口

```ts
export interface SchemaUIConfig {
  httpClient: {
    get: (url: string, config?: any) => Promise<any>
    post: (url: string, data?: any, config?: any) => Promise<any>
    put: (url: string, data?: any, config?: any) => Promise<any>
    delete: (url: string, data?: any, config?: any) => Promise<any>
  }
  hasPermission?: (permission: string) => boolean
  router?: { push: (to: any) => void }
  i18n?: { t: (key: string, ...args: any[]) => string }
  defaultPageSize?: number
  apiPrefix?: string
  transformRequest?: (config: any) => any
}
```

### 4.2 全局注册

```ts
// main.ts
import { createApp } from 'vue'
import { SchemaUIPlugin } from '@yourscope/schema-ui'
import axiosInstance from './http'
import router from './router'

app.use(SchemaUIPlugin, {
  httpClient: axiosInstance,
  hasPermission: (perm) => useUserStore().hasPermission(perm),
  router: router,
  i18n: { t: (key, ...args) => i18n.global.t(key, args) },
  defaultPageSize: 15
})
```

### 4.3 Props 覆盖机制

高阶组件接收 `config` prop，内部合并逻辑：

```ts
const finalConfig = computed(() => ({
  ...globalConfig,      // Plugin 注入的全局配置
  ...props.config       // 组件级 props 覆盖
}))
```

### 4.4 默认值兜底

- `hasPermission` 未提供 → 默认返回 `true`
- `router` 未提供 → 隐藏需要路由跳转的功能
- `i18n` 未提供 → 返回 key 本身（或预设中文默认文案）
- `defaultPageSize` 未提供 → 默认 `15`

## 5. 核心组件 API

### 5.1 导出清单

```ts
export {
  SchemaUIPlugin,
  SchemaForm,
  SchemaGrid,
  SchemaViewer,
  SchemaPage,
  CRUD,
  useSchemaUI,
  type Schema,
  type Action,
  type CRUDOptions,
  type SchemaUIConfig
}
```

### 5.2 SchemaForm

```vue
<SchemaForm
  :schemas="schemas"
  scenario="create"
  :model="model"
  :actions="actions"
  :grid="true"
  :gridCols="8"
  :inline="false"
  :autoSubmit="false"
  @submit="handleSubmit"
>
  <template #default="{ model, schema }">
    <!-- 自定义特定字段渲染 -->
  </template>
</SchemaForm>
```

| Prop | Type | Default | 说明 |
|---|---|---|---|
| `schemas` | `Schema[]` | 必填 | 字段定义 |
| `scenario` | `string` | `'create'` | 场景 |
| `model` | `Model` | `{}` | 数据模型 |
| `actions` | `Action[]` | `[]` | 底部操作按钮 |
| `grid` | `boolean` | `false` | 是否 grid 布局 |
| `gridCols` | `number` | `0` | grid 列数（0 为自动响应式） |
| `inline` | `boolean` | `false` | 行内表单模式 |
| `labelWidth` | `string` | `''` | 标签宽度（空则自动响应式） |
| `autoSubmit` | `boolean` | `false` | 挂载后自动提交 |

**Events：** `submit(model, schemas)`

**Slots：** `default`, `container`

### 5.3 SchemaGrid

```vue
<SchemaGrid
  :schemas="schemas"
  :models="models"
  scenario="list"
  :actions="rowActions"
  :selection="true"
  :responsive="true"
  :gridProps="{ stripe: true }"
  @selection="handleSelection"
  @sort="handleSort"
/>
```

| Prop | Type | Default | 说明 |
|---|---|---|---|
| `schemas` | `Schema[]` | 必填 | 字段定义 |
| `models` | `Model[]` | `[]` | 数据列表 |
| `scenario` | `string` | `'list'` | 场景 |
| `actions` | `Action[]` | `[]` | 行级操作按钮 |
| `selection` | `boolean` | `true` | 是否显示多选列 |
| `responsive` | `boolean` | `true` | 移动端是否切换为折叠视图 |
| `gridProps` | `object` | `{}` | 透传给 el-table 的属性 |
| `size` | `string` | `''` | 尺寸 |

**Events：** `selection`, `sort`, `dragend`

**关键调整：** 原 admin 通过 `useSystemStore().isMobileView` 判断移动端。解耦后，`responsive` 模式下组件内部通过 `window.matchMedia` 自行判断断点。

### 5.4 SchemaViewer —— 全自动 CRUD 页面

面向"一行代码搞定标准增删改查"的场景。

```vue
<SchemaViewer
  module="system"
  table="user"
  title="用户管理"
  :config="{ apiPrefix: 'rest' }"
  :presetQuery="{ status: 'active' }"
  :rowActions="['edit', 'delete']"
  :batchActions="['export']"
  :defaultSort="'-created_at'"
  :autoFetch="true"
/>
```

| Prop | Type | Default | 说明 |
|---|---|---|---|
| `module` | `string` | 必填 | 模块名 |
| `table` | `string` | 必填 | 表名 |
| `title` | `string` | `''` | 页面标题 |
| `apiPrefix` | `string` | `''` | API 前缀 |
| `config` | `Partial<SchemaUIConfig>` | `{}` | 组件级配置覆盖 |
| `size` | `string` | `''` | 尺寸 |
| `formMode` | `'drawer' \| 'dialog'` | `'dialog'` | 表单弹窗模式 |
| `showHeader` | `boolean` | `true` | 显示标题栏 |
| `showSearch` | `boolean` | `true` | 显示搜索栏 |
| `showToolbar` | `boolean` | `true` | 显示工具栏 |
| `showPagination` | `boolean` | `true` | 显示分页 |
| `readonly` | `boolean` | `false` | 只读模式 |
| `autoFetch` | `boolean` | `true` | 挂载后自动查询 |
| `rowActions` | `Action[]` | 默认 | 表格行操作 |
| `batchActions` | `Action[]` | 默认 | 批量/工具栏操作 |
| `formActions` | `Action[]` | 默认 | 表单操作 |
| `searchActions` | `Action[]` | 默认 | 搜索操作 |
| `defaultSort` | `string` | `''` | 默认排序，如 `-created_at` |
| `presetQuery` | `Record<string, any>` | `{}` | 搜索框初始值 |
| `gridProps` | `object` | `{}` | 透传给 SchemaGrid |
| `formProps` | `object` | `{}` | 透传给 SchemaForm |

**Events：** `ready(crud: CRUD)`

**Slots：** `searchform`, `gridview`, `crudform`, `headerleft`, `headerright`

**内部机制：**
- 挂载时自动调用 `CRUD.initialize()`，通过 `httpClient` 拉取 schema（`GET /:apiPrefix/schema/:module/:table`）
- 搜索、分页、创建、编辑、删除全部自动通过 CRUD 类完成
- 权限判断：调用全局 `hasPermission`，未配置则默认放行
- 路由跳转：如果全局配置了 `router` 且用户有权限，显示"配置"按钮；否则隐藏

### 5.5 SchemaPage —— 底层布局编排

面向需要完全控制数据流的场景。内部**不发任何 HTTP 请求**。

```vue
<SchemaPage
  :schemas="schemas"
  :models="models"
  :pagination="pagination"
  :loading="loading"
  @search="handleSearch"
  @create="handleCreate"
  @edit="handleEdit"
  @delete="handleDelete"
  @pageChange="handlePageChange"
  @sortChange="handleSortChange"
/>
```

| Prop | Type | Default | 说明 |
|---|---|---|---|
| `schemas` | `Schema[]` | 必填 | 字段定义 |
| `models` | `Model[]` | `[]` | 数据列表 |
| `pagination` | `Pagination` | 默认 | 分页信息 |
| `loading` | `boolean` | `false` | 加载状态 |
| `size` | `string` | `''` | 尺寸 |
| `title` | `string` | `''` | 标题 |
| `formMode` | `'drawer' \| 'dialog'` | `'dialog'` | 表单模式 |
| `showHeader` | `boolean` | `true` | 显示标题栏 |
| `showSearch` | `boolean` | `true` | 显示搜索栏 |
| `showToolbar` | `boolean` | `true` | 显示工具栏 |
| `showPagination` | `boolean` | `true` | 显示分页 |
| `readonly` | `boolean` | `false` | 只读模式 |
| `searchActions` | `Action[]` | 默认 | 搜索操作 |
| `rowActions` | `Action[]` | 默认 | 行操作 |
| `batchActions` | `Action[]` | 默认 | 批量操作 |
| `formActions` | `Action[]` | 默认 | 表单操作 |

**Events：** `search`, `create`, `edit`, `delete`, `pageChange`, `sortChange`, `selectionChange`, `formSubmit`

## 6. CRUD 类设计

```ts
class CRUD {
  constructor(options: CRUDOptions & { httpClient: SchemaUIConfig['httpClient'] })

  async initialize(): Promise<Schema[]>       // 拉取 schema + live values
  getSchemas(): Schema[]
  getModels(): Model[]

  // 分页
  setPaginationIndex(index: number): this
  getPaginationIndex(): number
  setPaginationSize(size: number): this
  getPaginationSize(): number
  getPaginationCount(): number
  resetPagination(): this

  // 排序
  setSortable(column: string, order: 'ascending' | 'descending'): this

  // 查询条件
  setQueryParams(qs: Record<string, any>): this
  addQueryParams(k: string, v: any): void
  setFixedQuery(qs: Record<string, any>): this   // 固定条件（每次请求自动带上）
  setPresetQuery(qs: Record<string, any>): this  // 搜索框初始值

  // 模型操作
  async createModel(model: Model): Promise<Model>
  async updateModel(model: Model): Promise<Model>
  async deleteModel(model: Model | string): Promise<any>
  async getModel(qs: Record<string, any>): Promise<Model>
  async searchModel(): Promise<Model[]>
  async deleteModels(data: any[]): Promise<{ total: number; success: number; responses: any[] }>
  async exportModels(): Promise<void>

  // 错误处理
  setColumnError(column: string, error: string): void
  resetError(): void
}
```

**关键调整（vs admin 的 crud.js）：**
- 构造函数必须传入 `httpClient`（从全局配置或 props 获取）
- 移除对 `pluralize`、`query-string` 的外部依赖，使用内部工具函数
- `initialize()` 逻辑保持：优先使用传入的 schemas，否则通过 HTTP 拉取
- 新增 `setFixedQuery()` 方法，替代原 Viewer props 中的 `fixedQueue`

## 7. 查询条件的职责分离

| 功能 | 归属 | 实现方式 |
|---|---|---|
| **搜索框初始值**（用户可见、可改） | 视图层 | `SchemaViewer.presetQuery` prop |
| **固定请求参数**（用户不可见、不可改） | 数据层 | `CRUD.setFixedQuery()` 方法 |
| **隐藏列+固定值** | schema 预处理 | 后端返回 `attributes.invisible: true` + `CRUD.setFixedQuery()` |

**为什么把 `fixedQuery` 从 Viewer props 移除？**

`fixedQuery` 和 UI 完全无关，是请求拦截逻辑。放在 Viewer props 中会导致：
1. 视图组件耦合了数据层行为
2. 动态修改 fixedQuery 需要重渲染整个 Viewer

**使用方式：**

```vue
<!-- 标准页面：presetQuery 影响搜索框初始显示 -->
<SchemaViewer
  module="system"
  table="user"
  :presetQuery="{ status: 'active' }"
/>

<!-- 需要固定隐藏条件的页面：通过 ready 事件配置 CRUD -->
<SchemaViewer
  module="system"
  table="user"
  @ready="(crud) => crud.setFixedQuery({ tenant_id: 'xxx' })"
/>

<!-- 全局固定条件（如每个请求都带 tenant_id）：走全局配置 -->
app.use(SchemaUIPlugin, {
  httpClient: axios,
  transformRequest: (config) => {
    config.params = { ...config.params, tenant_id: 'xxx' }
    return config
  }
})
```

## 8. 依赖解耦策略

| 原耦合点 | admin 实现 | 新库解耦方式 |
|---|---|---|
| HTTP 请求 | `import httpclient from '@/apis/request'` | 通过 `SchemaUIConfig.httpClient` 注入 |
| 权限判断 | `useUserStore().hasPermission()` | 通过 `SchemaUIConfig.hasPermission` 注入 |
| 路由跳转 | `useRouter().push()` | 通过 `SchemaUIConfig.router` 注入 |
| 国际化 | `useI18n().t()` | 通过 `SchemaUIConfig.i18n` 注入 |
| 移动端判断 | `useSystemStore().isMobileView` | 组件内部通过 `matchMedia` 自行判断 |

## 9. 构建与发布

**package.json 关键字段：**

```json
{
  "name": "@yourscope/schema-ui",
  "version": "1.0.0",
  "type": "module",
  "main": "./dist/schema-ui.cjs",
  "module": "./dist/schema-ui.es.js",
  "types": "./dist/index.d.ts",
  "files": ["dist"],
  "peerDependencies": {
    "vue": "^3.3.0",
    "element-plus": "^2.12.0"
  },
  "dependencies": {
    "pluralize": "^8.0.0"
  }
}
```

**发布方式：**
- 代码托管在 Gitea 不影响 npm 发布
- 可以发布到 `registry.npmjs.org`（公开/私有）
- 或发布到 Gitea 内置 Package Registry（1.19+）
- 或组织内部私有 registry（Verdaccio、Nexus）

## 10. 使用示例

### 10.1 最小化使用（全自动）

```vue
<template>
  <SchemaViewer module="system" table="user" title="用户管理" />
</template>

<script setup>
import { SchemaViewer } from '@yourscope/schema-ui'
</script>
```

### 10.2 自定义搜索/行操作

```vue
<template>
  <SchemaViewer
    module="order"
    table="order"
    title="订单管理"
    :presetQuery="{ status: 'pending' }"
    :rowActions="rowActions"
    :batchActions="['export']"
    :defaultSort="'-created_at'"
  />
</template>

<script setup>
import { SchemaViewer } from '@yourscope/schema-ui'

const rowActions = [
  { name: 'edit', label: '编辑', type: 'success' },
  { name: 'detail', label: '详情', type: 'primary', callback: (model) => {
    router.push(`/orders/${model.id}`)
  }}
]
</script>
```

### 10.3 完全自定义数据流（SchemaPage）

```vue
<template>
  <SchemaPage
    :schemas="schemas"
    :models="models"
    :pagination="pagination"
    :loading="loading"
    @search="handleSearch"
    @pageChange="handlePageChange"
  />
</template>

<script setup>
import { ref } from 'vue'
import { SchemaPage } from '@yourscope/schema-ui'

const schemas = ref([])
const models = ref([])
const pagination = ref({ index: 1, size: 15, totalCount: 0 })
const loading = ref(false)

const handleSearch = (queryModel) => {
  loading.value = true
  myCustomApi.search(queryModel).then(res => {
    models.value = res.data
    pagination.value = res.pagination
    loading.value = false
  })
}
</script>
```

### 10.4 独立使用 CRUD 类

```ts
import { CRUD } from '@yourscope/schema-ui'

const crud = new CRUD({
  module: 'system',
  table: 'user',
  httpClient: axiosInstance
})

await crud.initialize()
await crud.searchModel()
console.log(crud.getModels())
```

## 11. 后续扩展点

- **类型自动生成脚本**：后续可编写 Go → TypeScript 的生成工具，从 `rest/schema/*.go` 自动同步类型定义
- **自定义 Widget 注册**：通过 Plugin 提供 `registerWidget(format, component)`，允许用户扩展 FormItem 渲染器
- **Relations 渲染**：REST 的 `relations` 字段目前未在前端使用，后续可扩展关联模型选择器
