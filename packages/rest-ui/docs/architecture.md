# 架构详解

## 目录

- [设计目标](#设计目标)
- [三层架构](#三层架构)
- [数据流](#数据流)
- [配置注入机制](#配置注入机制)
- [组件分层](#组件分层)
- [扩展点](#扩展点)

---

## 设计目标

`@nobla/rest-ui` 的设计目标是让前端开发者能够以最低的成本构建完整的 CRUD 页面。理想情况下，只需指定 `module` 和 `table`，所有 UI 元素（搜索表单、数据表格、分页、创建/编辑弹窗）都能自动渲染。

同时，库也提供底层组件供需要完全控制数据流的场景使用。

核心设计原则：

1. **Schema 驱动**：UI 由后端 schema 定义驱动，而非前端硬编码
2. **配置即代码**：通过 Vue Plugin 注入全局配置，避免每个组件重复传参
3. **渐进式使用**：从全自动到全手动，提供多个抽象层级
4. **零侵入**：不强制修改现有项目架构，可作为独立包引入
5. **类型安全**：完整的 TypeScript 类型定义，与后端结构体对齐

---

## 三层架构

```
├── core/          │  纯逻辑层，零框架依赖
│   ├── types.ts     │  TypeScript 类型定义
│   ├── constants.ts │  业务常量枚举
│   ├── scenarios.ts │  场景工具类
│   ├── codec.ts     │  模型编解码
│   ├── model.ts     │  模型辅助函数
│   └── form.ts      │  表单规则与可见性
│
├── runtime/       │  Vue 运行时层
│   ├── crud.ts      │  CRUD 状态管理类
│   └── useSchemaUI.ts │ 全局配置获取
│
├── ui/            │  UI 组件层（Element Plus）
│   ├── SchemaViewer.vue  │  全自动 CRUD
│   ├── SchemaPage.vue    │  手动编排
│   ├── SchemaGrid.vue    │  响应式表格
│   ├── SchemaForm.vue    │  Schema 表单
│   └── parts/            │  原子组件
│       ├── Action.vue  │  操作按钮
│       ├── Cell.vue    │  表格单元格
│       └── FormItem.vue │  表单控件
│
├── config.ts      │  配置类型定义
└── plugin.ts      │  Vue 插件入口
```

### Core 层

Core 层是整个库的基础，**不依赖 Vue 或 Element Plus**，可以在任何 JavaScript 环境中使用。

**职责：**
- 定义与 REST 后端对齐的数据类型
- 提供业务常量枚举
- 实现模型值的编码/解码（日期、数字、布尔值转换）
- 提供表单验证规则生成器
- 提供字段可见性检查

**设计要点：**
- 所有函数均为纯函数，无副作用
- 输入数据不会被修改，返回新对象
- 不依赖任何全局状态

### Runtime 层

Runtime 层连接 Core 层和 UI 层，提供 Vue 特定的运行时能力。

**职责：**
- `CRUD` 类：封装 HTTP 操作、分页、排序、查询参数管理
- `useSchemaUI`：通过 Vue 的 `provide/inject` 获取全局配置

**设计要点：**
- `CRUD` 类不依赖 Vue 的响应式系统，是一个独立的 TypeScript 类
- `useSchemaUI` 是唯一的 Vue 依赖，用于获取插件注入的配置
- `CRUD` 类的 schema 加载后会自动拉取 `live` 数据（下拉选项）

### UI 层

UI 层构建在 Element Plus 之上，提供 schema 驱动的组件。

**职责：**
- `SchemaViewer`：全自动 CRUD，内部使用 `CRUD` 类
- `SchemaPage`：手动编排，纯展示组件，不发送 HTTP 请求
- `SchemaGrid`：响应式数据表格（桌面端 `el-table` + 移动端折叠面板）
- `SchemaForm`：schema 驱动的表单渲染

**设计要点：**
- 组件通过 `scenarios` 过滤字段，只渲染当前场景需要的字段
- 所有组件都暴露插槽，允许完全自定义渲染
- `SchemaForm` 支持网格布局和行内布局

---

## 数据流

### SchemaViewer 数据流（全自动模式）

```
┌──────────────────────────────────────────────────────────────┐
│                         SchemaViewer                          │
└──────────────────────────────────────────────────────────────┘
                           │
         ┌──────────────┐  │  ┌──────────────┐
         │   CRUD Class   │  │  │  useSchemaUI  │
         └──────────────┘  │  └──────────────┘
                │           │         │
      ┌──────────────────────────────────────────────────────────────┐
      │                     HTTP Client                       │
      └──────────────────────────────────────────────────────────────┘
                           │
      ┌──────────────────────────────────────────────────────────────┐
      │                      REST Backend                       │
      └──────────────────────────────────────────────────────────────┘
```

1. `SchemaViewer` 初始化时创建 `CRUD` 实例
2. `CRUD` 通过 `useSchemaUI()` 获取全局 `httpClient`
3. `CRUD.initialize()` 加载 schema 定义（`GET /rest/schema/{module}/{table}`）
4. `CRUD` 自动识别主键字段（`primary_key === 1`）
5. `CRUD` 拉取所有 `live` 数据（下拉选项）
6. `SchemaViewer` 将 schema 传递给 `SchemaPage`
7. `SchemaPage` 根据 `scenarios` 过滤字段，渲染搜索表单和表格
8. 用户操作（搜索、分页、排序）通过 `CRUD` 发送 HTTP 请求
9. 响应数据更新 `models` 和 `pagination`

### SchemaPage 数据流（手动模式）

```
┌──────────────────────────────────────────────────────────────┐
│                          Parent Component                     │
└──────────────────────────────────────────────────────────────┘
       │    ├───────────────────────────────────────────────┐
       │    │                                              │
       │    │           Props (schemas, models, ...)       │
       │    │                                              │
       │    └───────────────────────────────────────────────┘
       │                           │
       │    ┌───────────────────────────────────────────────┐
       │    │                                              │
       └────┼───┼─────────────────────────────────────────┼───┘
            │   │                                            │
            │   ├───────────────────────────────────────────────┘
            │   │
            │   ├───────────────────────────────────────────────┐
            │   │                                              │
            │   └───────────────────────────────────────────────┘
            │              SchemaPage
            │
            │   ┌───────────────────────────────────────────────┐
            └──┾───────────────────────────────────────────────┘
               │         Events (search, pageChange, ...)      │
               └───────────────────────────────────────────────┘
```

1. 父组件通过 `props` 传入 `schemas`、`models`、`pagination`
2. `SchemaPage` 内部管理搜索模型、表单模型、选中项等状态
3. 用户操作触发 `emit` 事件给父组件
4. 父组件自行处理 HTTP 请求，更新数据后重新传入 `props`

---

## 配置注入机制

本库使用 Vue 3 的 `provide/inject` 机制实现全局配置注入。

```typescript
// plugin.ts
export const SchemaUIPlugin = {
  install(app: App, config: SchemaUIConfig) {
    app.provide(GLOBAL_CONFIG_KEY, config)
  },
}

// useSchemaUI.ts
export function useSchemaUI(): SchemaUIConfig {
  const config = inject<SchemaUIConfig>(GLOBAL_CONFIG_KEY)
  if (!config) {
    throw new Error('[rest-ui] SchemaUIPlugin not installed')
  }
  return config
}
```

### 配置优先级

当同时存在全局配置和组件级配置时，优先级如下：

1. 组件级 `props` 最高（如 `SchemaViewer` 的 `apiPrefix` prop）
2. 全局配置次之（通过 `SchemaUIPlugin` 注入）
3. 默认值最低

例如 `apiPrefix` 的解析逻辑：

```typescript
const apiPrefix = props.apiPrefix || globalConfig.apiPrefix || 'rest'
```

---

## 组件分层

本库提供两个层级的 CRUD 组件：

### 全自动层：SchemaViewer

适合大多数简单 CRUD 场景，一行代码完成完整页面。

**适用场景：**
- 标准的增删改查页面
- 不需要自定义数据处理逻辑
- 快速原型开发

**特点：**
- 内部管理所有 HTTP 请求
- 自动加载 schema、搜索数据、创建/更新/删除
- 暴露 `ready` 事件获取 `CRUD` 实例进行高级操作

### 手动编排层：SchemaPage

适合需要完全控制数据流的场景。

**适用场景：**
- 需要在前端进行数据转换或过滤
- 需要对接非标准 REST API
- 需要与 Vuex/Pinia 等状态管理库集成
- 需要自定义错误处理

**特点：**
- 不发送任何 HTTP 请求
- 所有数据通过 props 传入
- 所有操作通过 events 通知父组件
- 可以脱离 `SchemaUIPlugin` 独立使用

### 原子层：SchemaGrid / SchemaForm

最底层的可复用组件。

**适用场景：**
- 只需要表格或表单，不需要完整页面
- 在自定义布局中使用
- 嵌入到其他组件内部

---

## 扩展点

### 1. 插槽系统

所有组件都提供丰富的插槽：

| 插槽 | 层级 | 说明 |
|------|------|------|
| `searchform` | SchemaPage/SchemaViewer | 自定义搜索表单字段 |
| `gridview` | SchemaPage/SchemaViewer | 自定义表格单元格内容 |
| `crudform` | SchemaPage/SchemaViewer | 自定义表单字段 |
| `headerleft` | SchemaPage/SchemaViewer | 自定义头部左侧 |
| `headerright` | SchemaPage/SchemaViewer | 自定义头部右侧 |
| `container` | SchemaForm | 自定义整个表单容器 |
| `default` | SchemaForm/SchemaGrid | 自定义单个字段/单元格 |

### 2. Action 系统

通过 `Action` 接口定义操作按钮，支持同步和异步回调：

```typescript
interface Action {
  name: string
  label: string
  type?: string
  icon?: string
  permission?: string
  hidden?: boolean | ((model: Model) => boolean | Promise<boolean>)
  callback?: (model: Model, schemas?: Schema[], loading?: any) => void
  asyncCallback?: (model: Model, schemas?: Schema[], action?: Action) => Promise<void>
}
```

### 3. CRUD 类扩展

`CRUD` 类可以独立使用，也可以继承扩展：

```typescript
import { CRUD } from '@nobla/rest-ui'

class CustomCRUD extends CRUD {
  async approveModel(model: Model) {
    // 自定义业务逻辑
    return this.opts.httpClient.post(
      this.__buildUri('approve'),
      model
    )
  }
}
```

### 4. 自定义 HTTP Client

`httpClient` 只需要实现 `get/post/put/delete` 四个方法，可以使用任何 HTTP 库：

```typescript
// 使用 axios
app.use(SchemaUIPlugin, {
  httpClient: axios,
})

// 使用 fetch
app.use(SchemaUIPlugin, {
  httpClient: {
    get: (url, config) => fetch(url, { ...config }).then(r => r.json()),
    post: (url, data) => fetch(url, { method: 'POST', body: JSON.stringify(data) }).then(r => r.json()),
    put: (url, data) => fetch(url, { method: 'PUT', body: JSON.stringify(data) }).then(r => r.json()),
    delete: (url, data) => fetch(url, { method: 'DELETE', body: JSON.stringify(data) }).then(r => r.json()),
  },
})
```
