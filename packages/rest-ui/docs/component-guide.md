# 组件使用指南

## 目录

- [SchemaViewer 全自动 CRUD](#schemaviewer-全自动-crud)
- [SchemaPage 手动编排](#schemapage-手动编排)
- [SchemaGrid 响应式表格](#schemagrid-响应式表格)
- [SchemaForm Schema 驱动表单](#schemaform-schema-驱动表单)
- [插槽使用详解](#插槽使用详解)
- [Action 按钮系统](#action-按钮系统)

---

## SchemaViewer 全自动 CRUD

`SchemaViewer` 是本库最高层级的组件，只需提供 `module` 和 `table`，即可自动生成完整的 CRUD 页面。

### 基础用法

```vue
<template>
  <SchemaViewer module="user" table="admin" title="管理员管理" />
</template>

<script setup lang="ts">
import { SchemaViewer } from '@nobla/rest-ui'
</script>
```

### 生命周期

```
挂载
  │
  ├─── 创建 CRUD 实例
  ├─── 调用 initialize() 加载 schema
  ├─── 拉取 live 数据
  ├─── 应用 presetQuery（附加到查询参数，但不在搜索表单中显示）
  ├─── 应用 defaultSort
  ├─── 触发 ready 事件
  └─── 如 autoFetch=true，自动搜索数据
```

### Props 详解

#### module / table

模块名和表名，用于构建 API 路径和加载 schema。

```vue
<SchemaViewer module="order" table="order" />
```

对应的 API 路径：
- Schema：`GET /rest/schema/order/order`
- 搜索：`GET /rest/order/orders`
- 创建：`POST /rest/order/order`
- 更新：`PUT /rest/order/order/{id}`

#### apiPrefix

覆盖全局配置的 API 前缀：

```vue
<SchemaViewer module="user" table="admin" apiPrefix="api/v2" />
```

#### config

传递额外的 CRUD 配置，如预加载的 schemas：

```vue
<SchemaViewer
  module="user"
  table="admin"
  :config="{ schemas: preloadedSchemas }"
/>
```

当传入 `schemas` 时，`SchemaViewer` 不会从后端加载 schema 定义。

#### formMode

表单弹窗模式：

```vue
<!-- 对话框模式（默认） -->
<SchemaViewer module="user" table="admin" formMode="dialog" />

<!-- 抽屉模式 -->
<SchemaViewer module="user" table="admin" formMode="drawer" />
```

#### showHeader / showSearch / showToolbar / showPagination

控制页面各区域的显示：

```vue
<SchemaViewer
  module="user"
  table="admin"
  :showHeader="true"
  :showSearch="true"
  :showToolbar="true"
  :showPagination="true"
/>
```

#### readonly

只读模式，隐藏所有编辑按钮：

```vue
<SchemaViewer module="user" table="admin" :readonly="true" />
```

#### autoFetch

控制初始化时是否自动加载数据：

```vue
<!-- 初始化时自动加载（默认） -->
<SchemaViewer module="user" table="admin" :autoFetch="true" />

<!-- 初始化时不加载，等待用户搜索 -->
<SchemaViewer module="user" table="admin" :autoFetch="false" />
```

#### defaultSort

设置默认排序字段：

```vue
<!-- 按 created_at 降序 -->
<SchemaViewer module="user" table="admin" defaultSort="-created_at" />

<!-- 按 username 升序 -->
<SchemaViewer module="user" table="admin" defaultSort="username" />
```

#### presetQuery

预设查询条件：

```vue
<SchemaViewer
  module="order"
  table="order"
  :presetQuery="{ status: 'pending', type: 'normal' }"
/>
```

预设条件会附加到每次搜索请求中。

#### gridProps / formProps

向底层组件传递额外属性：

```vue
<SchemaViewer
  module="user"
  table="admin"
  :gridProps="{ stripe: true, height: '500px' }"
  :formProps="{ labelWidth: '0' }"
/>
```

### 事件

#### ready

CRUD 实例初始化完成后触发：

```vue
<template>
  <SchemaViewer module="user" table="admin" @ready="onReady" />
</template>

<script setup lang="ts">
import { SchemaViewer } from '@nobla/rest-ui'
import type { CRUD } from '@nobla/rest-ui'

function onReady(crud: CRUD) {
  console.log('主键字段：', crud.primaryKey)
  console.log('Schema 列表：', crud.getSchemas())
  
  // 设置固定查询条件
  crud.setFixedQuery({ tenant_id: '123' })
  
  // 手动触发搜索
  crud.searchModel()
}
</script>
```

---

## SchemaPage 手动编排

`SchemaPage` 是纯展示组件，不发送任何 HTTP 请求。所有数据通过 props 传入，所有操作通过 events 通知父组件。

除了表单弹窗/抽屉外，`SchemaPage` 还内置了详情查看功能（通过 `el-descriptions` 渲染），可通过 `openDetail` 方法打开。

### 何时使用 SchemaPage

1. **需要自定义数据源**
   - 数据来自 Vuex/Pinia
   - 数据需要在前端转换
   - 需要对接非标准 REST API

2. **需要完全控制数据流**
   - 自定义错误处理
   - 自定义加载状态
   - 自定义缓存策略

3. **不需要 HTTP 功能**
   - 纯展示页面
   - 数据已在父组件中准备好

### 基础用法

```vue
<template>
  <SchemaPage
    :schemas="schemas"
    :models="models"
    :pagination="pagination"
    :loading="loading"
    title="用户列表"
    @search="handleSearch"
    @pageChange="handlePageChange"
    @formSubmit="handleFormSubmit"
    @delete="handleDelete"
  />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { SchemaPage } from '@nobla/rest-ui'
import type { Schema, Model, Pagination } from '@nobla/rest-ui'

const schemas = ref<Schema[]>([])
const models = ref<Model[]>([])
const pagination = ref<Pagination>({ index: 1, size: 15, totalCount: 0 })
const loading = ref(false)

async function loadSchemas() {
  const res = await fetch('/rest/schema/user/admin')
  schemas.value = await res.json()
}

async function handleSearch(model: Model) {
  loading.value = true
  const query = new URLSearchParams({
    ...model,
    page: String(pagination.value.index),
    page_size: String(pagination.value.size),
  })
  const res = await fetch(`/rest/user/admins?${query}`)
  const data = await res.json()
  models.value = data.data
  pagination.value = {
    index: parseInt(data.page),
    size: parseInt(data.page_size),
    totalCount: parseInt(data.total_count),
  }
  loading.value = false
}

function handlePageChange(index: number) {
  pagination.value.index = index
  handleSearch({})
}

async function handleFormSubmit(model: Model, scenario: string) {
  if (scenario === 'create') {
    await fetch('/rest/admin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(model),
    })
  } else {
    await fetch(`/rest/admin/${model.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(model),
    })
  }
  handleSearch({})
}

async function handleDelete(model: Model) {
  await fetch(`/rest/admin/${model.id}`, { method: 'DELETE' })
  handleSearch({})
}

loadSchemas()
</script>
```

### 暴露方法

`SchemaPage` 通过 `ref` 暴露了以下方法，供父组件调用：

| 方法 | 参数 | 说明 |
|------|------|------|
| `openEdit(model)` | `Model` | 打开编辑弹窗/抽屉 |
| `openDetail(model)` | `Model` | 打开详情弹窗/抽屉 |
| `closeForm()` | - | 关闭表单弹窗/抽屉 |

```vue
<template>
  <SchemaPage ref="pageRef" :schemas="schemas" :models="models" ... />
</template>

<script setup>
const pageRef = ref(null)

// 打开编辑
pageRef.value?.openEdit(model)

// 打开详情
pageRef.value?.openDetail(model)

// 关闭表单
pageRef.value?.closeForm()
</script>
```

### 结合 CRUD 类使用

虽然 `SchemaPage` 不强制使用 `CRUD` 类，但可以利用 `CRUD` 的 URI 构建和辅助方法：

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { SchemaPage, CRUD, useSchemaUI } from '@nobla/rest-ui'
import type { Schema, Model, Pagination } from '@nobla/rest-ui'

const config = useSchemaUI()

const crud = new CRUD({
  module: 'user',
  table: 'admin',
  httpClient: config.httpClient,
})

const schemas = ref<Schema[]>([])
const models = ref<Model[]>([])
const pagination = ref<Pagination>({ index: 1, size: 15, totalCount: 0 })
const loading = ref(false)

async function init() {
  schemas.value = await crud.initialize()
  await handleSearch({})
}

async function handleSearch(model: Model) {
  loading.value = true
  crud.setQueryParams(model)
  await crud.searchModel()
  models.value = crud.getModels()
  pagination.value = { ...crud.pagination }
  loading.value = false
}

init()
</script>
```

---

## SchemaGrid 响应式表格

`SchemaGrid` 根据屏幕宽度自动切换显示模式：

- **桌面端**（>= 768px）：`el-table` 表格
- **移动端**（< 768px）：`el-collapse` 折叠面板

### 基础用法

```vue
<template>
  <SchemaGrid
    :schemas="schemas"
    :models="models"
    :actions="actions"
    @selection="handleSelection"
    @sort="handleSort"
  />
</template>

<script setup lang="ts">
import { SchemaGrid } from '@nobla/rest-ui'
import type { Schema, Model, Action } from '@nobla/rest-ui'

const schemas: Schema[] = [/* ... */]
const models: Model[] = [/* ... */]

const actions: Action[] = [
  { name: 'edit', label: '编辑', type: 'primary', callback: (model) => console.log(model) },
]

function handleSelection(selection: any[]) {
  console.log('Selected:', selection)
}

function handleSort({ column, order }: { column: string; order: string | null }) {
  console.log('Sort:', column, order)
}
</script>
```

### 属性

| 属性 | 说明 |
|------|------|
| `schemas` | Schema 定义数组 |
| `models` | 数据列表 |
| `scenario` | 场景（默认 `list`） |
| `selection` | 是否显示多选列（默认 `true`） |
| `actions` | 行操作按钮 |
| `responsive` | 是否响应式（默认 `true`） |
| `loading` | 加载状态 |

### 关闭响应式

```vue
<SchemaGrid :schemas="schemas" :models="models" :responsive="false" />
```

### 自定义单元格

```vue
<SchemaGrid :schemas="schemas" :models="models">
  <template #default="{ model, schema }">
    <span v-if="schema.column === 'status'">
      <el-tag :type="model.status === 'active' ? 'success' : 'danger'">
        {{ model.status }}
      </el-tag>
    </span>
  </template>
</SchemaGrid>
```

---

## SchemaForm Schema 驱动表单

`SchemaForm` 根据 schema 定义自动渲染表单字段。

### 基础用法

```vue
<template>
  <SchemaForm
    :schemas="schemas"
    scenario="create"
    :model="initialModel"
    @submit="handleSubmit"
  />
</template>

<script setup lang="ts">
import { SchemaForm } from '@nobla/rest-ui'
import type { Schema, Model } from '@nobla/rest-ui'

const schemas: Schema[] = [/* ... */]
const initialModel: Model = { status: 'active' }

function handleSubmit(model: Model, displaySchemas: Schema[]) {
  console.log('Submit:', model)
}
</script>
```

### 布局模式

**行内布局（搜索表单）：**

```vue
<SchemaForm
  :schemas="schemas"
  scenario="search"
  :inline="true"
  :actions="searchActions"
/>
```

**网格布局：**

```vue
<SchemaForm
  :schemas="schemas"
  scenario="create"
  :grid="true"
  :gridCols="12"
/>
```

`gridCols` 指定每列占用的栅格数（Element Plus 的 `el-col span`）。

### 手动提交

通过 `ref` 调用 `submit` 方法：

```vue
<template>
  <SchemaForm ref="formRef" :schemas="schemas" />
  <el-button @click="handleSubmit">提交</el-button>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { SchemaForm } from '@nobla/rest-ui'

const formRef = ref<InstanceType<typeof SchemaForm> | null>(null)

async function handleSubmit() {
  try {
    const model = await formRef.value?.submit()
    console.log('Submitted:', model)
  } catch (e) {
    console.error('Validation failed:', e)
  }
}
</script>
```

---

## 插槽使用详解

### searchform 插槽

自定义搜索表单字段的渲染：

```vue
<SchemaViewer module="user" table="admin">
  <template #searchform="{ model, schema }">
    <!-- 自定义日期范围选择 -->
    <el-date-picker
      v-if="schema.column === 'date_range'"
      v-model="model.date_range"
      type="daterange"
      range-separator="至"
      start-placeholder="开始日期"
      end-placeholder="结束日期"
    />
    
    <!-- 自定义级联选择 -->
    <el-cascader
      v-else-if="schema.column === 'region'"
      v-model="model.region"
      :options="regionOptions"
    />
  </template>
</SchemaViewer>
```

**注意：** 使用 `searchform` 插槽时，只有匹配自定义条件的字段会被替换，其余字段仍使用默认渲染。

### gridview 插槽

自定义表格单元格内容：

```vue
<SchemaViewer module="user" table="admin">
  <template #gridview="{ model, schema }">
    <!-- 头像 -->
    <el-avatar
      v-if="schema.column === 'avatar'"
      :src="model.avatar"
      :size="40"
    />
    
    <!-- 状态标签 -->
    <el-tag
      v-else-if="schema.column === 'status'"
      :type="model.status === 'active' ? 'success' : 'danger'"
    >
      {{ statusText[model.status] }}
    </el-tag>
    
    <!-- 进度条 -->
    <el-progress
      v-else-if="schema.column === 'progress'"
      :percentage="model.progress"
    />
    
    <!-- 链接 -->
    <el-link
      v-else-if="schema.column === 'website'"
      :href="model.website"
      target="_blank"
      type="primary"
    >
      {{ model.website }}
    </el-link>
  </template>
</SchemaViewer>
```

### crudform 插槽

自定义表单字段：

```vue
<SchemaViewer module="article" table="article">
  <template #crudform="{ model, schema }">
    <!-- 富文本编辑器 -->
    <RichEditor
      v-if="schema.column === 'content'"
      v-model="model.content"
    />
    
    <!-- 图片上传 -->
    <ImageUploader
      v-else-if="schema.column === 'cover'"
      v-model="model.cover"
      :limit="5"
    />
    
    <!-- 代码编辑器 -->
    <CodeEditor
      v-else-if="schema.column === 'code'"
      v-model="model.code"
      language="javascript"
    />
  </template>
</SchemaViewer>
```

### headerleft / headerright 插槽

自定义页面头部：

```vue
<SchemaViewer module="user" table="admin">
  <template #headerleft>
    <div class="custom-header">
      <h3>管理员管理</h3>
      <el-tag type="info">共 {{ total }} 人</el-tag>
    </div>
  </template>
  
  <template #headerright>
    <el-button type="success" @click="handleImport">导入</el-button>
    <el-button type="warning" @click="handleExport">导出</el-button>
    <el-button type="primary" @click="handleCreate">创建</el-button>
  </template>
</SchemaViewer>
```

---

## Action 按钮系统

Action 是本库统一的按钮定义接口，用于搜索操作、行操作、批量操作和表单操作。

### 基础 Action

```typescript
import type { Action } from '@nobla/rest-ui'

const action: Action = {
  name: 'edit',        // 唯一标识
  label: '编辑',       // 显示文本
  type: 'primary',     // 按钮类型
  icon: 'Edit',        // 图标（Element Plus 图标名）
  size: 'small',       // 按钮尺寸
  callback: (model, schemas) => {
    console.log('Edit:', model)
  },
}
```

### 异步 Action

需要显示加载状态的异步操作：

```typescript
const action: Action = {
  name: 'approve',
  label: '审批通过',
  type: 'success',
  asyncCallback: async (model, schemas) => {
    await fetch(`/api/approve/${model.id}`, { method: 'POST' })
  },
}
```

异步 Action 会自动管理按钮的 `loading` 状态。

### 权限控制

```typescript
const action: Action = {
  name: 'delete',
  label: '删除',
  type: 'danger',
  permission: 'user:delete',
  callback: (model) => deleteUser(model.id),
}
```

当配置了 `hasPermission` 全局函数时，没有权限的用户不会看到该按钮。

### 动态显示

根据模型数据动态决定是否显示：

```typescript
const action: Action = {
  name: 'activate',
  label: '激活',
  type: 'success',
  hidden: (model) => model.status !== 'inactive',
  callback: (model) => activateUser(model.id),
}
```

支持同步和异步判断：

```typescript
const action: Action = {
  name: 'edit',
  label: '编辑',
  hidden: async (model) => {
    const result = await checkEditable(model.id)
    return !result.editable
  },
}
```

### 行操作

```vue
<script setup lang="ts">
const rowActions: Action[] = [
  {
    name: 'detail',
    label: '详情',
    type: 'primary',
    callback: (model) => router.push(`/users/${model.id}`),
  },
  {
    name: 'edit',
    label: '编辑',
    type: 'success',
    permission: 'user:edit',
    callback: (model) => openEditDialog(model),
  },
  {
    name: 'delete',
    label: '删除',
    type: 'danger',
    permission: 'user:delete',
    hidden: (model) => model.is_super,
    callback: (model) => confirmDelete(model),
  },
]
</script>

<template>
  <SchemaViewer module="user" table="admin" :rowActions="rowActions" />
</template>
```

### 批量操作

批量操作接收选中项数组。注意：批量操作**只支持 `callback`**，不支持 `asyncCallback`。

```typescript
const batchActions: Action[] = [
  {
    name: 'batchDelete',
    label: '批量删除',
    type: 'danger',
    callback: (selections) => {
      console.log('Delete:', selections)
    },
  },
  {
    name: 'batchExport',
    label: '批量导出 Excel',
    callback: (selections) => {
      exportToExcel(selections)
    },
  },
  {
    name: 'batchUpdateStatus',
    label: '批量更新状态',
    type: 'primary',
    callback: (selections) => {
      updateStatus(selections.map(s => s.id), 'active')
    },
  },
]
```

### 搜索操作

自定义搜索按钮：

```typescript
const searchActions: Action[] = [
  {
    name: 'search',
    label: '搜索',
    type: 'primary',
    asyncCallback: async (model, schemas) => {
      // 自定义搜索逻辑
      await performSearch(model)
    },
  },
  {
    name: 'reset',
    label: '重置',
    callback: (model) => {
      Object.keys(model).forEach(key => delete model[key])
    },
  },
]
```

### 表单操作

```typescript
const formActions: Action[] = [
  {
    name: 'save',
    label: '保存',
    type: 'primary',
    asyncCallback: async (model) => {
      await saveModel(model)
    },
  },
  {
    name: 'saveAndContinue',
    label: '保存并继续',
    type: 'success',
    asyncCallback: async (model) => {
      await saveModel(model)
      // 不关闭弹窗，清空表单继续创建
    },
  },
]
```
