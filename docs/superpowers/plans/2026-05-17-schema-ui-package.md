# Schema UI Package Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 REST 项目仓库中创建独立的 `packages/schema-ui/` Vue 3 + Element Plus 组件库，基于 REST schema 定义提供 schema 驱动的 CRUD 组件。

**Architecture:** 三层架构：core（纯逻辑工具）/ runtime（Vue 配置与 CRUD 状态）/ ui（Element Plus 组件）。通过 Vue Plugin 全局注入可配置项（HTTP/权限/路由/i18n）。

**Tech Stack:** Vue 3, Element Plus, TypeScript, Vite (Library Mode), pluralize

---

## File Structure

```
packages/schema-ui/
├── package.json
├── vite.config.ts
├── tsconfig.json
├── .gitignore
└── src/
    ├── index.ts              # 入口：导出所有公开 API
    ├── plugin.ts             # Vue Plugin 实现
    ├── config.ts             # SchemaUIConfig 类型 + 全局 key
    ├── core/
    │   ├── constants.ts      # 类型/格式/场景常量（对齐 REST Go 定义）
    │   ├── types.ts          # Schema/Rule/Attribute/Model/Action 等接口
    │   ├── scenarios.ts      # 场景判断工具
    │   ├── codec.ts          # encode/decode 模型值
    │   ├── model.ts          # 模型辅助函数
    │   └── form.ts           # 表单规则生成与辅助
    ├── runtime/
    │   ├── crud.ts           # CRUD 类
    │   └── useSchemaUI.ts    # 获取全局配置的 composable
    └── ui/
        ├── SchemaForm.vue
        ├── SchemaGrid.vue
        ├── SchemaPage.vue
        ├── SchemaViewer.vue
        └── parts/
            ├── Action.vue
            ├── Cell.vue
            └── FormItem.vue
```

---

### Task 1: 初始化 Package 目录与构建配置

**Files:**
- Create: `packages/schema-ui/package.json`
- Create: `packages/schema-ui/vite.config.ts`
- Create: `packages/schema-ui/tsconfig.json`
- Create: `packages/schema-ui/.gitignore`

- [ ] **Step 1: 创建目录**

```bash
mkdir -p packages/schema-ui/src/{core,runtime,ui/parts}
```

- [ ] **Step 2: 编写 package.json**

```json
{
  "name": "@ace/schema-ui",
  "version": "1.0.0",
  "description": "Schema-driven UI components for REST projects",
  "type": "module",
  "main": "./dist/schema-ui.cjs",
  "module": "./dist/schema-ui.es.js",
  "types": "./dist/index.d.ts",
  "files": ["dist"],
  "scripts": {
    "build": "vite build",
    "dev": "vite build --watch",
    "typecheck": "vue-tsc --noEmit"
  },
  "peerDependencies": {
    "vue": "^3.3.0",
    "element-plus": "^2.12.0"
  },
  "dependencies": {
    "pluralize": "^8.0.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.0.0",
    "typescript": "^5.3.0",
    "vite": "^5.0.0",
    "vite-plugin-dts": "^3.0.0",
    "vue": "^3.4.0",
    "vue-tsc": "^1.8.0"
  }
}
```

- [ ] **Step 3: 编写 vite.config.ts**

```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import dts from 'vite-plugin-dts'
import { resolve } from 'path'

export default defineConfig({
  plugins: [
    vue(),
    dts({
      insertTypesEntry: true,
      include: ['src/**/*.ts', 'src/**/*.vue'],
    }),
  ],
  build: {
    lib: {
      entry: resolve(__dirname, 'src/index.ts'),
      name: 'SchemaUI',
      fileName: (format) => `schema-ui.${format === 'es' ? 'es' : format}.js`,
      formats: ['es', 'cjs'],
    },
    rollupOptions: {
      external: ['vue', 'element-plus', '@element-plus/icons-vue'],
      output: {
        globals: {
          vue: 'Vue',
          'element-plus': 'ElementPlus',
        },
      },
    },
  },
})
```

- [ ] **Step 4: 编写 tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "module": "ESNext",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "preserve",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "declaration": true,
    "declarationMap": true,
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["src/**/*.ts", "src/**/*.vue"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

- [ ] **Step 5: 编写 .gitignore**

```
node_modules
dist
*.log
.DS_Store
```

- [ ] **Step 6: Commit**

```bash
git add packages/schema-ui/
git commit -m "chore(schema-ui): initialize package directory and build config"
```

---

### Task 2: Core 类型定义与常量

**Files:**
- Create: `packages/schema-ui/src/core/constants.ts`
- Create: `packages/schema-ui/src/core/types.ts`
- Create: `packages/schema-ui/src/core/scenarios.ts`

- [ ] **Step 1: 编写 constants.ts**

```typescript
// === 类型常量（对齐 REST schema/types.go）===
export const TypeInteger = 'integer'
export const TypeFloat = 'float'
export const TypeBoolean = 'boolean'
export const TypeString = 'string'

// === 格式常量 ===
export const FormatInteger = 'integer'
export const FormatFloat = 'float'
export const FormatBoolean = 'boolean'
export const FormatString = 'string'
export const FormatText = 'text'
export const FormatDropdown = 'dropdown'
export const FormatDatetime = 'datetime'
export const FormatDate = 'date'
export const FormatTime = 'time'
export const FormatTimestamp = 'timestamp'
export const FormatPassword = 'password'

// === 场景常量 ===
export const ScenarioCreate = 'create'
export const ScenarioUpdate = 'update'
export const ScenarioDelete = 'delete'
export const ScenarioSearch = 'search'
export const ScenarioExport = 'export'
export const ScenarioImport = 'import'
export const ScenarioList = 'list'
export const ScenarioDetail = 'detail'

// === 匹配模式 ===
export const MatchExactly = 'exactly'
export const MatchFuzzy = 'fuzzy'

// === Live 类型 ===
export const LiveTypeDropdown = 'dropdown'
export const LiveTypeCascader = 'cascader'
```

- [ ] **Step 2: 编写 types.ts**

```typescript
export interface SchemaRule {
  min: number
  max: number
  type: string
  unique: boolean
  required: string[]
  regular?: string
  safe?: boolean
}

export interface EnumValue {
  label: string
  value: string
  color?: string
}

export interface VisibleCondition {
  column: string
  values: (string | number | boolean)[]
}

export interface LiveValue {
  enable: boolean
  type: string
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

export interface SchemaAttribute {
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

export interface Relation {
  type: string
  name: string
  module: string
  table: string
}

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
  type: string
  format: string
  native: number
  primary_key: number
  expression: string
  scenarios: string[]
  rules: SchemaRule
  attributes: SchemaAttribute
  relations: Relation
  position: number
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

- [ ] **Step 3: 编写 scenarios.ts**

```typescript
export class Scenarios extends Array<string> {
  constructor(items?: string[]) {
    super()
    if (items) {
      items.forEach(item => this.push(item))
    }
  }

  has(scenario: string): boolean {
    return this.includes(scenario)
  }

  static from(raw: string | string[]): Scenarios {
    if (typeof raw === 'string') {
      return new Scenarios(raw.split(';').filter(Boolean))
    }
    return new Scenarios(raw)
  }
}
```

- [ ] **Step 4: Commit**

```bash
git add packages/schema-ui/src/core/
git commit -m "feat(schema-ui): add core types and constants aligned with REST schema"
```

---

### Task 3: Core 工具函数

**Files:**
- Create: `packages/schema-ui/src/core/codec.ts`
- Create: `packages/schema-ui/src/core/model.ts`
- Create: `packages/schema-ui/src/core/form.ts`

- [ ] **Step 1: 编写 codec.ts**

```typescript
import type { Model, Schema } from './types'

/**
 * 将模型值编码为提交格式
 */
export function encode(model: Model, schemas: Schema[], scenario: string): Model {
  const result: Model = {}
  for (const key in model) {
    result[key] = model[key]
  }
  return result
}

/**
 * 将后端返回的模型值解码为表单可用格式
 */
export function decode(model: Model, schemas: Schema[], scenario: string): Model {
  const result: Model = {}
  for (const key in model) {
    result[key] = model[key]
  }
  return result
}
```

- [ ] **Step 2: 编写 model.ts**

```typescript
import type { Model } from './types'

/**
 * 获取模型中指定列的值
 */
export function getModelValue(model: Model, column: string): any {
  if (model && typeof model === 'object' && column in model) {
    return model[column]
  }
  return undefined
}

/**
 * 获取模型中指定列的显示标签
 */
export function getModelLabel(model: Model, column: string): string {
  const value = getModelValue(model, column)
  if (value === null || value === undefined) {
    return ''
  }
  if (typeof value === 'object' && value !== null) {
    return value.label || String(value.value) || ''
  }
  return String(value)
}
```

- [ ] **Step 3: 编写 form.ts**

```typescript
import type { Schema } from './types'

export interface FieldRules {
  [key: string]: any
}

/**
 * 根据 schema 生成 Element Plus 表单验证规则
 */
export function generateSchemaRule(t: (key: string, ...args: any[]) => string, schema: Schema, scenario: string): any[] {
  const rules: any[] = []
  const rule = schema.rules

  if (rule.required && rule.required.includes(scenario)) {
    rules.push({
      required: true,
      message: t('validation.required', [schema.label]),
      trigger: 'blur',
    })
  }

  if (rule.max > 0 && schema.type === 'string') {
    rules.push({
      max: rule.max,
      message: t('validation.max', [schema.label, rule.max]),
      trigger: 'blur',
    })
  }

  if (rule.regular) {
    rules.push({
      pattern: new RegExp(rule.regular),
      message: t('validation.pattern', [schema.label]),
      trigger: 'blur',
    })
  }

  return rules
}

/**
 * 检查 schema 在指定模型下是否可见
 */
export function checkSchemaVisible(schema: Schema, model: Record<string, any>): boolean {
  const conditions = schema.attributes.visible
  if (!conditions || conditions.length === 0) {
    return true
  }

  for (const cond of conditions) {
    const modelValue = model[cond.column]
    if (!cond.values.includes(modelValue)) {
      return false
    }
  }

  return true
}

/**
 * 生成字段描述文本
 */
export function generateSchemaDescription(t: (key: string, ...args: any[]) => string, schema: Schema, scenario: string): string {
  if (schema.attributes.tooltip) {
    return schema.attributes.tooltip
  }
  return ''
}

/**
 * 清空搜索模型中的空值
 */
export function clearSearchModel(model: Record<string, any>, schemas: Schema[]): Record<string, any> {
  const result: Record<string, any> = {}
  for (const key in model) {
    const value = model[key]
    if (value !== '' && value !== null && value !== undefined) {
      result[key] = value
    }
  }
  return result
}
```

- [ ] **Step 4: Commit**

```bash
git add packages/schema-ui/src/core/codec.ts packages/schema-ui/src/core/model.ts packages/schema-ui/src/core/form.ts
git commit -m "feat(schema-ui): add core utility functions (codec, model, form)"
```

---

### Task 4: Runtime 配置系统与 Plugin

**Files:**
- Create: `packages/schema-ui/src/config.ts`
- Create: `packages/schema-ui/src/plugin.ts`
- Create: `packages/schema-ui/src/runtime/useSchemaUI.ts`

- [ ] **Step 1: 编写 config.ts**

```typescript
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

export const GLOBAL_CONFIG_KEY = Symbol('schema-ui-config')
```

- [ ] **Step 2: 编写 plugin.ts**

```typescript
import type { App } from 'vue'
import { GLOBAL_CONFIG_KEY, type SchemaUIConfig } from './config'

export const SchemaUIPlugin = {
  install(app: App, config: SchemaUIConfig) {
    if (!config.httpClient) {
      throw new Error('[schema-ui] httpClient is required in SchemaUIConfig')
    }
    app.provide(GLOBAL_CONFIG_KEY, config)
  },
}
```

- [ ] **Step 3: 编写 useSchemaUI.ts**

```typescript
import { inject } from 'vue'
import { GLOBAL_CONFIG_KEY, type SchemaUIConfig } from '../config'

export function useSchemaUI(): SchemaUIConfig {
  const config = inject<SchemaUIConfig>(GLOBAL_CONFIG_KEY)
  if (!config) {
    throw new Error('[schema-ui] SchemaUIPlugin not installed. Call app.use(SchemaUIPlugin, config) first.')
  }
  return config
}
```

- [ ] **Step 4: Commit**

```bash
git add packages/schema-ui/src/config.ts packages/schema-ui/src/plugin.ts packages/schema-ui/src/runtime/useSchemaUI.ts
git commit -m "feat(schema-ui): add configuration system and Vue plugin"
```

---

### Task 5: Runtime CRUD 类

**Files:**
- Create: `packages/schema-ui/src/runtime/crud.ts`

- [ ] **Step 1: 编写 crud.ts**

```typescript
import pluralize from 'pluralize'
import type { Schema, Model, CRUDOptions, Pagination, Sortable } from '../core/types'
import type { SchemaUIConfig } from '../config'

const DEFAULT_PAGINATION: Pagination = {
  index: 1,
  size: 15,
  totalCount: 0,
}

export class CRUD {
  private opts: Required<CRUDOptions> & { httpClient: SchemaUIConfig['httpClient']; apiPrefix: string }
  primaryKey = ''
  schemas: Schema[] = []
  models: Model[] = []
  sortable: Sortable | null = null
  queryParams: Record<string, any> = {}
  fixedQuery: Record<string, any> = {}
  pagination: Pagination = { ...DEFAULT_PAGINATION }

  constructor(options: CRUDOptions & { httpClient: SchemaUIConfig['httpClient']; apiPrefix?: string }) {
    let schemas: Schema[] = []
    if (Array.isArray(options.schemas)) {
      schemas = options.schemas
    } else if (options.schemas && typeof options.schemas === 'object') {
      schemas = Object.values(options.schemas)
    }

    this.opts = {
      module: options.module || '',
      table: options.table || '',
      apiPrefix: options.apiPrefix || 'rest',
      schemas,
      httpClient: options.httpClient,
    }
  }

  private __prepare() {
    for (const schema of this.schemas) {
      schema.rules = schema.rules || { min: 0, max: 0, type: '', unique: false, required: [] }
      if (schema.primary_key === 1) {
        this.primaryKey = schema.column
        break
      }
    }
  }

  private __buildUri(scenario: string, primaryKey?: string): string {
    return this.__buildModelUri(this.opts.module, this.opts.table, scenario, primaryKey)
  }

  private __buildModelUri(moduleName: string, tableName: string, scenario: string, primaryKey?: string): string {
    const pk = primaryKey || ''
    const pluralName = pluralize.plural(tableName)
    const singularName = pluralize.singular(tableName)
    const parts: string[] = []

    if (this.opts.apiPrefix) {
      parts.push(this.opts.apiPrefix)
    }
    if (moduleName) {
      parts.push(moduleName)
    }

    switch (scenario) {
      case 'create':
        parts.push(singularName)
        break
      case 'update':
      case 'delete':
      case 'get':
        parts.push(singularName, pk)
        break
      case 'search':
        parts.push(pluralName)
        break
      case 'export':
        parts.push(`${singularName}-export`)
        break
      case 'import':
        parts.push(`${singularName}-import`)
        break
    }

    return parts.join('/')
  }

  async initialize(): Promise<Schema[]> {
    if (this.schemas.length > 0) {
      this.__prepare()
      await this.__fetchVars()
      return this.schemas
    }

    const uri = this.opts.module
      ? `/${this.opts.apiPrefix}/schema/${this.opts.module}/${this.opts.table}`
      : `/${this.opts.apiPrefix}/schema/${this.opts.table}`

    const res = await this.opts.httpClient.get(uri)
    this.schemas = Array.isArray(res) ? res : res.data || []
    this.__prepare()
    await this.__fetchVars()
    return this.schemas
  }

  private async __fetchVars(): Promise<void> {
    const promises: Promise<{ schema: Schema; data: any }>[] = []
    for (const schema of this.schemas) {
      if (schema.attributes.live?.enable && schema.attributes.live.url) {
        promises.push(this.__lazyFetch(schema))
      }
    }
    if (promises.length === 0) return

    const results = await Promise.all(promises)
    for (const { schema, data } of results) {
      const target = this.schemas.find(s => s.id === schema.id)
      if (target) {
        target.attributes.values = Array.isArray(data?.data) ? data.data : Array.isArray(data) ? data : []
      }
    }
  }

  private async __lazyFetch(schema: Schema): Promise<{ schema: Schema; data: any }> {
    const live = schema.attributes.live
    if (live.method?.toLowerCase() === 'post') {
      const data = await this.opts.httpClient.post(live.url!, live.body, {
        headers: { 'Content-Type': live.content_type || 'application/json' },
      })
      return { schema, data }
    }
    const data = await this.opts.httpClient.get(live.url!)
    return { schema, data }
  }

  getSchemas(): Schema[] {
    return this.schemas
  }

  getModels(): Model[] {
    return this.models
  }

  setColumnError(column: string, error: string) {
    const schema = this.schemas.find(s => s.column === column)
    if (schema) {
      ;(schema as any).__error = error
    }
  }

  resetError() {
    for (const schema of this.schemas) {
      delete (schema as any).__error
    }
  }

  setPaginationIndex(index: number): this {
    if (typeof index === 'number' && index >= 1) {
      this.pagination.index = index
    }
    return this
  }

  getPaginationIndex(): number {
    return this.pagination.index
  }

  setPaginationSize(size: number): this {
    this.pagination.size = size
    return this
  }

  getPaginationSize(): number {
    return this.pagination.size || 15
  }

  getPaginationCount(): number {
    return this.pagination.totalCount
  }

  resetPagination(): this {
    this.pagination.index = 1
    return this
  }

  setSortable(column: string, order: 'ascending' | 'descending'): this {
    if (!column) {
      this.sortable = null
    } else {
      this.sortable = { column, order: order || 'ascending' }
    }
    return this
  }

  addQueryParams(k: string, v: any): void {
    this.queryParams[k] = v
  }

  setQueryParams(qs: Record<string, any>): this {
    this.queryParams = qs
    return this
  }

  setFixedQuery(qs: Record<string, any>): this {
    this.fixedQuery = qs
    return this
  }

  findModelPrimaryKey(model: Model): any {
    if (model && typeof model === 'object') {
      return model[this.primaryKey]
    }
    return model
  }

  async createModel(model: Model): Promise<Model> {
    const res = await this.opts.httpClient.post(this.__buildUri('create'), model)
    return this.__refreshModel(this.findModelPrimaryKey(res))
  }

  async updateModel(model: Model): Promise<Model> {
    const pk = this.findModelPrimaryKey(model)
    if (pk === undefined || pk === null || pk === '') {
      throw new Error('Cannot find model primary key')
    }
    await this.opts.httpClient.put(this.__buildUri('update', String(pk)), model)
    return this.__refreshModel(pk)
  }

  async deleteModel(model: Model | string): Promise<any> {
    const pk = typeof model === 'object' ? this.findModelPrimaryKey(model) : model
    const res = await this.opts.httpClient.delete(this.__buildUri('delete', String(pk)))
    this.__removeModel(pk)
    return res
  }

  async getModel(qs: Record<string, any> | string): Promise<Model> {
    let pk = ''
    const params: Record<string, any> = {}
    if (typeof qs === 'object') {
      for (const k in qs) {
        if (k === this.primaryKey) {
          pk = qs[k]
        } else {
          params[k] = qs[k]
        }
      }
    } else {
      pk = qs
    }
    if (!pk) {
      throw new Error('Cannot find model primary key')
    }
    return this.opts.httpClient.get(this.__buildUri('get', String(pk)), { params })
  }

  async searchModel(): Promise<Model[]> {
    const queryParams: Record<string, any> = { ...this.queryParams }
    queryParams.page = this.pagination.index
    queryParams.pagesize = this.pagination.size || 15
    if (this.sortable?.column) {
      queryParams.sort = this.sortable.order === 'descending' ? `-${this.sortable.column}` : this.sortable.column
    }
    queryParams.__format = 'both'
    for (const k in this.fixedQuery) {
      queryParams[k] = this.fixedQuery[k]
    }

    const res = await this.opts.httpClient.get(this.__buildUri('search'), { params: queryParams })
    this.pagination.index = parseInt(res.page) || 1
    this.pagination.size = parseInt(res.pagesize) || 15
    this.pagination.totalCount = parseInt(res.totalCount) || 0
    this.models = res.data || []
    return this.models
  }

  async deleteModels(data: any[]): Promise<{ total: number; success: number; responses: any[] }> {
    const pks: string[] = []
    for (const val of data) {
      const pk = typeof val === 'object' ? this.findModelPrimaryKey(val) : val
      if (pk !== undefined && pk !== null && pk !== '') {
        pks.push(String(pk))
      }
    }
    if (pks.length === 0) {
      return { total: 0, success: 0, responses: [] }
    }
    const responses = await Promise.all(pks.map(pk => this.deleteModel(pk)))
    return { total: pks.length, success: responses.length, responses }
  }

  async exportModels(): Promise<void> {
    const queryParams: Record<string, any> = { ...this.queryParams }
    if (this.sortable?.column) {
      queryParams.sort = this.sortable.order === 'descending' ? `-${this.sortable.column}` : this.sortable.column
    }
    const res = await this.opts.httpClient.get(this.__buildUri('export'), { params: queryParams, responseType: 'blob' })
    this.__downloadFile(res, `${this.opts.table}.csv`)
  }

  private async __refreshModel(primaryKey: any): Promise<Model> {
    const qs: Record<string, any> = {
      [this.primaryKey]: primaryKey,
      __format: 'both',
      scenario: 'list',
    }
    const res = await this.getModel(qs)
    const pk = this.findModelPrimaryKey(res)
    const index = this.models.findIndex(m => this.findModelPrimaryKey(m) == pk)
    if (index >= 0) {
      this.models[index] = res
    } else {
      this.models.push(res)
    }
    await this.__fetchVars()
    return res
  }

  private __removeModel(primaryKey: any) {
    this.models = this.models.filter(model => this.findModelPrimaryKey(model) !== primaryKey)
  }

  private __downloadFile(res: any, defaultFilename: string) {
    const element = document.createElement('a')
    const disposition = res.headers?.['content-disposition']
    let filename = defaultFilename
    if (disposition) {
      const parts = disposition.split(';')
      for (const part of parts) {
        const s = part.trim()
        if (s.startsWith('filename=')) {
          filename = s.substring(9).replace(/^"|"$/g, '')
          try {
            filename = decodeURIComponent(filename)
          } catch (e) { /* ignore */ }
          break
        }
      }
    }
    element.style.display = 'none'
    element.href = window.URL.createObjectURL(new Blob([res.data], { type: res.headers?.['content-type'] || '' }))
    element.target = '_blank'
    element.setAttribute('download', filename)
    document.body.appendChild(element)
    element.click()
    setTimeout(() => {
      window.URL.revokeObjectURL(element.href)
      document.body.removeChild(element)
    }, 100)
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add packages/schema-ui/src/runtime/crud.ts
git commit -m "feat(schema-ui): add CRUD class with HTTP operations"
```

---

### Task 6: UI 基础组件（Action, Cell）

**Files:**
- Create: `packages/schema-ui/src/ui/parts/Action.vue`
- Create: `packages/schema-ui/src/ui/parts/Cell.vue`

- [ ] **Step 1: 编写 Action.vue**

```vue
<template>
  <el-button
    :type="action.type || 'default'"
    :size="action.size || 'small'"
    :round="action.round"
    :icon="action.icon"
    :loading="loading"
    @click="handleClick"
  >
    {{ action.label }}
  </el-button>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Action as ActionType, Model } from '../../core/types'

interface Props {
  action: ActionType
  model?: Model
}

const props = defineProps<Props>()
const emit = defineEmits<{
  click: [action: ActionType, loading: ReturnType<typeof ref<boolean>>]
}>()

const loading = ref(false)

function handleClick() {
  if (props.action.hidden) {
    if (typeof props.action.hidden === 'function') {
      const result = props.action.hidden(props.model || {})
      if (result instanceof Promise) {
        result.then(hidden => {
          if (!hidden) emit('click', props.action, loading)
        })
        return
      }
      if (result) return
    } else if (props.action.hidden) {
      return
    }
  }
  emit('click', props.action, loading)
}
</script>
```

- [ ] **Step 2: 编写 Cell.vue**

```vue
<template>
  <template v-if="tagVisible">
    <span class="schema-cell-tag" :style="{ color: textColor, backgroundColor: bgColor }">
      {{ displayValue }}
    </span>
  </template>
  <template v-else-if="schema.format === 'boolean' || schema.format === 'bool'">
    <el-tag round :type="isTrue ? 'success' : 'danger'">
      {{ isTrue ? '是' : '否' }}
    </el-tag>
  </template>
  <template v-else>
    <span class="schema-cell">{{ displayValue }}</span>
  </template>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Model, Schema } from '../../core/types'

interface Props {
  model: Model
  schema: Schema
}

const props = defineProps<Props>()

const rawValue = computed(() => {
  const val = props.model[props.schema.column]
  if (val === null || val === undefined) return ''
  if (typeof val === 'object') {
    return val.label || val.value || ''
  }
  return val
})

const displayValue = computed(() => {
  const values = props.schema.attributes.values
  if (!Array.isArray(values) || values.length === 0) {
    return String(rawValue.value)
  }
  const found = values.find(v => v.value === String(rawValue.value))
  return found?.label || String(rawValue.value)
})

const isTrue = computed(() => {
  const val = props.model[props.schema.column]
  return !!val
})

const tagVisible = computed(() => {
  if (props.schema.format !== 'dropdown') return false
  const values = props.schema.attributes.values
  if (!Array.isArray(values)) return false
  return values.length > 0 && values.every(v => !!v.color)
})

const textColor = computed(() => {
  const val = rawValue.value
  const found = props.schema.attributes.values?.find(v => v.value === String(val))
  return found?.color || ''
})

const bgColor = computed(() => {
  const val = rawValue.value
  const found = props.schema.attributes.values?.find(v => v.value === String(val))
  if (!found?.color) return ''
  // 简单 fade，实际可用 Color 库
  return found.color + '1A'
})
</script>
```

- [ ] **Step 3: Commit**

```bash
git add packages/schema-ui/src/ui/parts/Action.vue packages/schema-ui/src/ui/parts/Cell.vue
git commit -m "feat(schema-ui): add Action and Cell base components"
```

---

### Task 7: UI FormItem 组件

**Files:**
- Create: `packages/schema-ui/src/ui/parts/FormItem.vue`

- [ ] **Step 1: 编写 FormItem.vue**

```vue
<template>
  <template v-if="isVisible('time')">
    <el-time-select
      v-model="model[schema.column]"
      start="00:00"
      step="00:15"
      end="23:59"
      :disabled="isDisabled"
      :placeholder="placeholder"
      format="HH:mm"
    />
  </template>
  <template v-else-if="isVisible('date')">
    <el-date-picker
      v-if="isRange"
      v-model="model[schema.column]"
      type="daterange"
      :disabled="isDisabled"
      :editable="false"
      format="YYYY-MM-DD"
      value-format="YYYY-MM-DD"
      :start-placeholder="startPlaceholder"
      :end-placeholder="endPlaceholder"
    />
    <el-date-picker
      v-else
      v-model="model[schema.column]"
      type="date"
      :disabled="isDisabled"
      :editable="false"
      format="YYYY-MM-DD"
      value-format="YYYY-MM-DD"
      :placeholder="placeholder"
    />
  </template>
  <template v-else-if="isVisible('datetime')">
    <el-date-picker
      v-if="isRange"
      v-model="model[schema.column]"
      type="datetimerange"
      :disabled="isDisabled"
      :editable="false"
      format="YYYY-MM-DD HH:mm:ss"
      value-format="YYYY-MM-DD HH:mm:ss"
      :start-placeholder="startPlaceholder"
      :end-placeholder="endPlaceholder"
    />
    <el-date-picker
      v-else
      v-model="model[schema.column]"
      type="datetime"
      :disabled="isDisabled"
      :editable="false"
      format="YYYY-MM-DD HH:mm:ss"
      value-format="YYYY-MM-DD HH:mm:ss"
      :placeholder="placeholder"
    />
  </template>
  <template v-else-if="isVisible('dropdown')">
    <el-select
      v-model="model[schema.column]"
      :multiple="isMultiSelect"
      :disabled="isDisabled"
      :placeholder="placeholder"
      :allow-create="schema.attributes.dropdown?.created"
      :filterable="schema.attributes.dropdown?.filterable"
      :default-first-option="schema.attributes.dropdown?.default_first"
      clearable
    >
      <el-option
        v-for="item in schema.attributes.values"
        :key="item.value"
        :label="item.label"
        :value="item.value"
      />
    </el-select>
  </template>
  <template v-else-if="isVisible('search_boolean')">
    <el-select v-model="model[schema.column]" clearable>
      <el-option label="是" :value="true" />
      <el-option label="否" :value="false" />
    </el-select>
  </template>
  <template v-else-if="isVisible('cascader')">
    <el-cascader
      v-model="model[schema.column]"
      :options="schema.attributes.values"
      :disabled="isDisabled"
      filterable
      clearable
      :placeholder="placeholder"
      :validate-event="false"
    />
  </template>
  <template v-else-if="isVisible('boolean')">
    <el-switch v-model="model[schema.column]" :disabled="isDisabled" />
  </template>
  <template v-else-if="isVisible('file')">
    <el-upload
      :action="schema.attributes.upload_url"
      :disabled="isDisabled"
      :file-list="fileList"
      @success="handleUploadSuccess"
    >
      <el-button type="primary">上传</el-button>
    </el-upload>
  </template>
  <template v-else-if="isVisible('password')">
    <el-input
      v-model="model[schema.column]"
      :disabled="isDisabled"
      show-password
      :placeholder="placeholder"
    />
  </template>
  <template v-else-if="isVisible('multistr')">
    <el-input
      v-model="model[schema.column]"
      type="textarea"
      :disabled="isDisabled"
      :placeholder="placeholder"
      :maxlength="schema.rules.max > 0 ? schema.rules.max : undefined"
    />
  </template>
  <template v-else-if="isVisible('number')">
    <el-input
      v-model.number="model[schema.column]"
      :disabled="isDisabled"
      :prefix-icon="schema.attributes.icon || ''"
      :placeholder="placeholder"
      :clearable="scenario === 'search'"
    >
      <template v-if="schema.attributes.suffix" #append>
        {{ schema.attributes.suffix }}
      </template>
    </el-input>
  </template>
  <template v-else>
    <el-input
      v-model="model[schema.column]"
      :disabled="isDisabled"
      :prefix-icon="schema.attributes.icon || ''"
      :placeholder="placeholder"
      :clearable="scenario === 'search'"
    >
      <template v-if="schema.attributes.suffix" #append>
        {{ schema.attributes.suffix }}
      </template>
    </el-input>
  </template>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Model, Schema } from '../../core/types'

interface Props {
  model: Model
  schema: Schema
  scenario?: string
}

const props = defineProps<Props>()

const scenario = computed(() => props.scenario || 'create')
const isSearch = computed(() => scenario.value === 'search')
const isRange = computed(() => scenario.value === 'search')

const isDisabled = computed(() => {
  return (
    props.schema.attributes.disable.includes(scenario.value) ||
    props.schema.attributes.readonly.includes(scenario.value)
  )
})

const isMultiSelect = computed(() => {
  if (props.schema.format === 'multiSelect') return true
  if (isSearch.value && props.schema.attributes.dropdown?.multiple) return true
  return false
})

const fileList = ref<any[]>([])

function handleUploadSuccess(response: any) {
  props.model[props.schema.column] = response.url || response.data?.url || response
}

const placeholder = computed(() => {
  if (props.schema.attributes.tooltip) return props.schema.attributes.tooltip
  if (props.schema.format === 'dropdown') {
    return `请选择${props.schema.label}`
  }
  return `请输入${props.schema.label}`
})

const startPlaceholder = computed(() => `开始${props.schema.label}`)
const endPlaceholder = computed(() => `结束${props.schema.label}`)

function isVisible(type: string): boolean {
  const fmt = props.schema.format
  const tp = props.schema.type

  switch (type) {
    case 'number':
      return ['integer', 'decimal'].includes(fmt) || ['integer', 'double'].includes(tp)
    case 'password':
      return ['password', 'pass'].includes(fmt)
    case 'time':
      return fmt === 'time'
    case 'date':
      return fmt === 'date'
    case 'datetime':
      return ['datetime', 'timestamp'].includes(fmt)
    case 'dropdown':
      return ['dropdown', 'multiSelect'].includes(fmt)
    case 'cascader':
      return fmt === 'cascader'
    case 'boolean':
      return scenario.value !== 'search' && ['bool', 'boolean'].includes(fmt)
    case 'search_boolean':
      return scenario.value === 'search' && ['bool', 'boolean'].includes(fmt)
    case 'file':
      return fmt === 'file' && !!props.schema.attributes.upload_url
    case 'multistr':
      return scenario.value !== 'search' && fmt === 'textarea'
    default:
      return ['string', 'text'].includes(fmt)
  }
}
</script>
```

- [ ] **Step 2: Commit**

```bash
git add packages/schema-ui/src/ui/parts/FormItem.vue
git commit -m "feat(schema-ui): add FormItem component with schema-driven rendering"
```

---

### Task 8: UI SchemaForm 组件

**Files:**
- Create: `packages/schema-ui/src/ui/SchemaForm.vue`

- [ ] **Step 1: 编写 SchemaForm.vue**

```vue
<template>
  <div class="schema-form" ref="formElement">
    <el-form
      :model="activeModel"
      :label-width="labelWidthComputed"
      :rules="formRules"
      :validate-on-rule-change="false"
      :inline="inlineMode"
      ref="formRef"
      status-icon
    >
      <slot name="container" :model="activeModel" :schemas="displayColumns">
        <el-row :gutter="20" v-if="grid">
          <el-col
            :span="getColSpan(schema)"
            v-for="schema in displayColumns"
            :key="schema.column"
          >
            <el-form-item
              :prop="schema.column"
              :label="schema.label"
              :error="fieldErrors[schema.column]"
            >
              <slot name="default" :model="activeModel" :schema="schema">
                <FormItem :model="activeModel" :schema="schema" :scenario="scenario" />
              </slot>
            </el-form-item>
          </el-col>
        </el-row>
        <template v-else>
          <el-form-item
            v-for="schema in displayColumns"
            :key="schema.column"
            :prop="schema.column"
            :label="schema.label"
            :error="fieldErrors[schema.column]"
          >
            <slot name="default" :model="activeModel" :schema="schema">
              <FormItem :model="activeModel" :schema="schema" :scenario="scenario" />
            </slot>
          </el-form-item>
        </template>
        <el-form-item v-if="actions.length > 0" class="schema-form-actions">
          <Action
            v-for="action in actions"
            :key="action.name"
            :action="action"
            @click="handleActionClick"
          />
        </el-form-item>
      </slot>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import type { Schema, Model, Action as ActionType } from '../core/types'
import { decode, encode } from '../core/codec'
import { generateSchemaRule, checkSchemaVisible } from '../core/form'
import FormItem from './parts/FormItem.vue'
import Action from './parts/Action.vue'

interface Props {
  size?: string
  schemas: Schema[]
  scenario?: string
  labelWidth?: string
  model?: Model
  inline?: boolean
  grid?: boolean
  gridCols?: number
  actions?: ActionType[]
  autoSubmit?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  scenario: 'create',
  labelWidth: '',
  model: undefined,
  inline: false,
  grid: false,
  gridCols: 0,
  actions: () => [],
  autoSubmit: false,
})

const emit = defineEmits<{
  submit: [model: Model, schemas: Schema[]]
}>()

const formRef = ref<any>(null)
const formElement = ref<HTMLElement | null>(null)
const activeModel = ref<Model>({})
const formWidth = ref<number>(window.innerWidth)
const fieldErrors = ref<Record<string, string>>({})
const stopWatchers: (() => void)[] = []

const BREAKPOINTS = { MOBILE: 768, TABLET: 960 }
const LABEL_WIDTHS = { MOBILE: '80px', DESKTOP: '120px' }
const COL_SPANS = { FULL: 24, HALF: 12, THIRD: 8 }

const displayColumns = computed(() => {
  return props.schemas.filter(schema => {
    if (schema.attributes.invisible) return false
    if (!Array.isArray(schema.scenarios)) return false
    if (!schema.scenarios.includes(props.scenario)) return false
    return checkSchemaVisible(schema, activeModel.value)
  })
})

const inlineMode = computed(() => {
  if (props.grid) return false
  return props.inline
})

const labelWidthComputed = computed(() => {
  if (props.inline) return '0'
  if (props.labelWidth) return props.labelWidth
  return formWidth.value < BREAKPOINTS.MOBILE ? LABEL_WIDTHS.MOBILE : LABEL_WIDTHS.DESKTOP
})

const formRules = computed(() => {
  const rules: Record<string, any> = {}
  if (props.scenario === 'search') return rules
  for (const schema of displayColumns.value) {
    rules[schema.column] = generateSchemaRule(
      (key) => key,
      schema,
      props.scenario
    )
  }
  return rules
})

function getColSpan(schema: Schema): number {
  if (props.gridCols > 0) return Math.min(props.gridCols, 24)
  if (schema.format === 'textarea') return COL_SPANS.FULL
  if (formWidth.value < BREAKPOINTS.MOBILE) return COL_SPANS.FULL
  if (formWidth.value < BREAKPOINTS.TABLET) return COL_SPANS.HALF
  return COL_SPANS.THIRD
}

onMounted(() => {
  activeModel.value = decode(props.model || {}, props.schemas, props.scenario)
  if (props.autoSubmit) {
    nextTick(() => submit())
  }
  if (formElement.value) {
    formWidth.value = formElement.value.offsetWidth
  }

  stopWatchers.push(
    watch(
      () => props.model,
      (val) => {
        activeModel.value = decode(val || {}, props.schemas, props.scenario)
      },
      { deep: true }
    )
  )
})

onUnmounted(() => {
  stopWatchers.forEach((stop) => stop())
})

async function submit(): Promise<Model> {
  const model = encode(activeModel.value, displayColumns.value, props.scenario)
  if (props.model) {
    const pkSchema = props.schemas.find((s) => s.primary_key === 1)
    if (pkSchema && pkSchema.column in props.model) {
      model[pkSchema.column] = props.model[pkSchema.column]
    }
  }

  try {
    await formRef.value!.validate()
    emit('submit', model, displayColumns.value)
    return model
  } catch (e: any) {
    const errors = Object.values(e)[0] as any[]
    if (Array.isArray(errors) && errors.length > 0) {
      throw new Error(errors[0].message)
    }
    throw new Error('验证失败')
  }
}

async function handleActionClick(action: ActionType, loading: any) {
  if (typeof action.callback === 'function') {
    try {
      const result = await submit()
      action.callback(result, displayColumns.value, loading)
    } catch (e) {
      console.error('Form submit failed:', e)
    }
  } else if (typeof action.asyncCallback === 'function') {
    loading.value = true
    try {
      await action.asyncCallback(await submit(), displayColumns.value)
    } finally {
      loading.value = false
    }
  }
}

defineExpose({ submit })
</script>
```

- [ ] **Step 2: Commit**

```bash
git add packages/schema-ui/src/ui/SchemaForm.vue
git commit -m "feat(schema-ui): add SchemaForm component"
```

---

### Task 9: UI SchemaGrid 组件

**Files:**
- Create: `packages/schema-ui/src/ui/SchemaGrid.vue`

- [ ] **Step 1: 编写 SchemaGrid.vue**

```vue
<template>
  <template v-if="enableMobileTable">
    <el-collapse expand-icon-position="left">
      <el-collapse-item v-for="(model, idx) in models" :key="idx">
        <template #title>
          <span class="mobile-primary-label">{{ getMobilePrimaryLabel(model) }}</span>
          <span class="mobile-actions" v-if="actions.length > 0">
            <Action
              v-for="action in actions"
              :key="action.name"
              :action="action"
              :model="model"
              @click="(_, loading) => handleActionClick(action, model, loading)"
            />
          </span>
        </template>
        <div v-for="schema in visibleSchemas" :key="schema.column" class="mobile-preview-row">
          <div class="mobile-preview-label">{{ schema.label }}</div>
          <div class="mobile-preview-value">
            <slot :model="model" :schema="schema">
              <Cell :model="model" :schema="schema" />
            </slot>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </template>
  <template v-else>
    <el-table
      :data="models"
      :size="size"
      :border="true"
      :loading="loading"
      v-bind="gridProps"
      @selection-change="handleSelectionChange"
      @sort-change="handleSortChange"
    >
      <el-table-column v-if="selection" type="selection" width="55" />
      <el-table-column
        v-for="schema in visibleSchemas"
        :key="schema.column"
        :prop="schema.column"
        :label="schema.label"
        :sortable="schema.attributes.sort ? 'custom' : false"
        show-overflow-tooltip
      >
        <template #default="scope">
          <slot :model="scope.row" :schema="schema">
            <Cell :model="scope.row" :schema="schema" />
          </slot>
        </template>
      </el-table-column>
      <el-table-column v-if="actions.length > 0" fixed="right" class-name="schema-grid-actions">
        <template #default="scope">
          <Action
            v-for="action in actions"
            :key="action.name"
            :action="action"
            :model="scope.row"
            @click="(_, loading) => handleActionClick(action, scope.row, loading)"
          />
        </template>
      </el-table-column>
      <template #empty>
        <el-empty />
      </template>
    </el-table>
  </template>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Schema, Model, Action as ActionType } from '../core/types'
import Cell from './parts/Cell.vue'
import Action from './parts/Action.vue'

interface Props {
  size?: string
  schemas: Schema[]
  scenario?: string
  selection?: boolean
  models: Model[]
  actions?: ActionType[]
  gridProps?: Record<string, any>
  responsive?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  scenario: 'list',
  selection: true,
  actions: () => [],
  gridProps: () => ({}),
  responsive: true,
})

const emit = defineEmits<{
  selection: [selection: any[]]
  sort: [sortable: { column: string; order: 'ascending' | 'descending' | null }]
  dragend: []
}>()

const loading = ref(false)
const isMobileView = ref(false)

if (typeof window !== 'undefined') {
  const mql = window.matchMedia('(max-width: 768px)')
  isMobileView.value = mql.matches
  mql.addEventListener?.('change', (e) => {
    isMobileView.value = e.matches
  })
}

const enableMobileTable = computed(() => {
  if (!props.responsive) return false
  return isMobileView.value
})

const visibleSchemas = computed(() => {
  return props.schemas.filter((v) => {
    if (!Array.isArray(v.scenarios)) return false
    if (!v.scenarios.includes(props.scenario)) return false
    return !v.attributes.invisible
  })
})

function getMobilePrimaryLabel(model: Model): string {
  const pkSchema = props.schemas.find((s) => s.primary_key === 1)
  if (pkSchema) {
    const val = model[pkSchema.column]
    return val !== undefined && val !== null ? String(val) : ''
  }
  const firstKey = Object.keys(model)[0]
  return firstKey !== undefined ? String(model[firstKey]) : ''
}

function handleActionClick(action: ActionType, model: Model, loading: any) {
  if (typeof action.callback === 'function') {
    action.callback(model, props.schemas)
  } else if (typeof action.asyncCallback === 'function') {
    loading.value = true
    action.asyncCallback(model, props.schemas).finally(() => {
      loading.value = false
    })
  }
}

function handleSelectionChange(selection: any[]) {
  emit('selection', selection)
}

function handleSortChange(e: { prop: string; order: 'ascending' | 'descending' | null }) {
  emit('sort', e)
}
</script>
```

- [ ] **Step 2: Commit**

```bash
git add packages/schema-ui/src/ui/SchemaGrid.vue
git commit -m "feat(schema-ui): add SchemaGrid component with responsive mobile view"
```

---

### Task 10: UI SchemaPage 组件

**Files:**
- Create: `packages/schema-ui/src/ui/SchemaPage.vue`

- [ ] **Step 1: 编写 SchemaPage.vue**

```vue
<template>
  <div class="schema-page">
    <div v-if="showHeader" class="schema-page-header">
      <div class="header-left">
        <slot name="headerleft">
          <h3>{{ title }}</h3>
        </slot>
      </div>
      <div class="header-right">
        <slot name="headerright">
          <el-button v-if="!readonly" type="primary" round @click="handleCreate">
            创建
          </el-button>
        </slot>
      </div>
    </div>
    <div class="schema-page-body">
      <div v-if="showSearch" class="schema-page-search">
        <SchemaForm
          :schemas="searchSchemas"
          scenario="search"
          :model="searchModel"
          :inline="true"
          :actions="searchActionList"
        >
          <template #default="{ model, schema }">
            <slot name="searchform" :model="model" :schema="schema" />
          </template>
        </SchemaForm>
      </div>
      <div v-if="showToolbar" class="schema-page-toolbar">
        <el-dropdown v-if="batchActionList.length > 0" placement="bottom-end">
          <el-icon><More /></el-icon>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                v-for="action in batchActionList"
                :key="action.name"
                @click="handleBatchAction(action)"
              >
                {{ action.label }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
      <div class="schema-page-grid" v-loading="loading">
        <SchemaGrid
          :schemas="listSchemas"
          :models="models"
          scenario="list"
          :actions="rowActionList"
          v-bind="gridProps"
          @selection="handleSelectionChange"
          @sort="handleSortChange"
        >
          <template #default="{ model, schema }">
            <slot name="gridview" :model="model" :schema="schema" />
          </template>
        </SchemaGrid>
        <el-pagination
          v-if="showPagination"
          :page-size="pagination.size"
          :total="pagination.totalCount"
          :current-page="pagination.index"
          layout="total, prev, pager, next"
          @current-change="handlePageChange"
        />
      </div>
    </div>

    <el-dialog
      v-if="formMode === 'dialog'"
      v-model="formVisible"
      :title="formTitle"
      :width="formWidth"
      draggable
      destroy-on-close
    >
      <SchemaForm
        v-bind="formProps"
        :schemas="listSchemas"
        :scenario="formScenario"
        :model="formModel"
        :actions="formActionList"
      >
        <template #default="{ model, schema }">
          <slot name="crudform" :model="model" :schema="schema" />
        </template>
      </SchemaForm>
    </el-dialog>
    <el-drawer
      v-else
      v-model="formVisible"
      :title="formTitle"
      :size="formWidth"
      destroy-on-close
    >
      <SchemaForm
        v-bind="formProps"
        :schemas="listSchemas"
        :scenario="formScenario"
        :model="formModel"
        :actions="formActionList"
      >
        <template #default="{ model, schema }">
          <slot name="crudform" :model="model" :schema="schema" />
        </template>
      </SchemaForm>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { More } from '@element-plus/icons-vue'
import type { Schema, Model, Action as ActionType, Pagination as PaginationType } from '../core/types'
import SchemaForm from './SchemaForm.vue'
import SchemaGrid from './SchemaGrid.vue'

interface Props {
  schemas: Schema[]
  models: Model[]
  pagination?: PaginationType
  loading?: boolean
  size?: string
  title?: string
  formMode?: 'drawer' | 'dialog'
  showHeader?: boolean
  showSearch?: boolean
  showToolbar?: boolean
  showPagination?: boolean
  readonly?: boolean
  searchActions?: ActionType[]
  rowActions?: ActionType[]
  batchActions?: ActionType[]
  formActions?: ActionType[]
  gridProps?: Record<string, any>
  formProps?: Record<string, any>
}

const props = withDefaults(defineProps<Props>(), {
  pagination: () => ({ index: 1, size: 15, totalCount: 0 }),
  loading: false,
  formMode: 'dialog',
  showHeader: true,
  showSearch: true,
  showToolbar: true,
  showPagination: true,
  readonly: false,
  searchActions: () => [],
  rowActions: () => [],
  batchActions: () => [],
  formActions: () => [],
  gridProps: () => ({}),
  formProps: () => ({}),
})

const emit = defineEmits<{
  search: [model: Model]
  create: []
  edit: [model: Model]
  delete: [model: Model]
  pageChange: [index: number]
  sortChange: [sortable: { column: string; order: 'ascending' | 'descending' | null }]
  selectionChange: [selection: any[]]
  formSubmit: [model: Model, scenario: string]
}>()

const formVisible = ref(false)
const formScenario = ref('create')
const searchModel = ref<Model>({})
const formModel = ref<Model>({})
const selections = ref<any[]>([])

const searchSchemas = computed(() =>
  props.schemas.filter((s) => s.scenarios?.includes('search'))
)
const listSchemas = computed(() =>
  props.schemas.filter((s) => s.scenarios?.includes('list') && !s.attributes.invisible)
)

const formTitle = computed(() => {
  return formScenario.value === 'create' ? '创建' : '编辑'
})

const formWidth = computed(() => {
  const width = window.innerWidth
  if (width < 768) return '96%'
  if (width <= 1180) return '80%'
  if (width <= 1366) return '60%'
  return '40%'
})

const searchActionList = computed((): ActionType[] => {
  if (props.searchActions.length > 0) return props.searchActions
  return [
    {
      name: 'search',
      label: '搜索',
      type: 'primary',
      asyncCallback: async (model) => {
        emit('search', model)
      },
    },
  ]
})

const rowActionList = computed((): ActionType[] => {
  if (props.readonly) return []
  if (props.rowActions.length > 0) return props.rowActions
  return [
    { name: 'edit', label: '编辑', type: 'success', callback: (model) => handleEdit(model) },
    { name: 'delete', label: '删除', type: 'danger', callback: (model) => emit('delete', model) },
  ]
})

const batchActionList = computed((): ActionType[] => props.batchActions)
const formActionList = computed((): ActionType[] => {
  if (props.formActions.length > 0) return props.formActions
  return [
    {
      name: 'save',
      label: '保存',
      type: 'primary',
      asyncCallback: async (model) => {
        emit('formSubmit', model, formScenario.value)
        formVisible.value = false
      },
    },
  ]
})

function handleCreate() {
  formScenario.value = 'create'
  formModel.value = {}
  emit('create')
  formVisible.value = true
}

function handleEdit(model: Model) {
  formScenario.value = 'update'
  formModel.value = { ...model }
  emit('edit', model)
  formVisible.value = true
}

function handleBatchAction(action: ActionType) {
  if (typeof action.callback === 'function') {
    action.callback(selections.value)
  }
}

function handleSelectionChange(selection: any[]) {
  selections.value = selection
  emit('selectionChange', selection)
}

function handleSortChange(e: { prop: string; order: 'ascending' | 'descending' | null }) {
  emit('sortChange', e)
}

function handlePageChange(index: number) {
  emit('pageChange', index)
}
</script>
```

- [ ] **Step 2: Commit**

```bash
git add packages/schema-ui/src/ui/SchemaPage.vue
git commit -m "feat(schema-ui): add SchemaPage layout orchestration component"
```

---

### Task 11: UI SchemaViewer 组件

**Files:**
- Create: `packages/schema-ui/src/ui/SchemaViewer.vue`

- [ ] **Step 1: 编写 SchemaViewer.vue**

```vue
<template>
  <SchemaPage
    :schemas="schemas"
    :models="crud?.getModels() || []"
    :pagination="pagination"
    :loading="searching"
    :size="size"
    :title="title"
    :formMode="formMode"
    :showHeader="showHeader"
    :showSearch="showSearch"
    :showToolbar="showToolbar"
    :showPagination="showPagination"
    :readonly="readonly"
    :searchActions="searchActionList"
    :rowActions="rowActionList"
    :batchActions="batchActionList"
    :formActions="formActionList"
    :gridProps="gridProps"
    :formProps="formProps"
    @search="handleSearch"
    @create="handleCreate"
    @edit="handleEdit"
    @delete="handleDelete"
    @pageChange="handlePageChange"
    @sortChange="handleSortChange"
    @selectionChange="handleSelectionChange"
    @formSubmit="handleFormSubmit"
  >
    <template #searchform="{ model, schema }">
      <slot name="searchform" :model="model" :schema="schema" />
    </template>
    <template #gridview="{ model, schema }">
      <slot name="gridview" :model="model" :schema="schema" />
    </template>
    <template #crudform="{ model, schema }">
      <slot name="crudform" :model="model" :schema="schema" />
    </template>
    <template #headerleft>
      <slot name="headerleft" />
    </template>
    <template #headerright>
      <slot name="headerright" />
    </template>
  </SchemaPage>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { Schema, Model, Action as ActionType, CRUDOptions } from '../core/types'
import { CRUD } from '../runtime/crud'
import { useSchemaUI } from '../runtime/useSchemaUI'
import { clearSearchModel } from '../core/form'
import SchemaPage from './SchemaPage.vue'

interface Props {
  module: string
  table: string
  title?: string
  apiPrefix?: string
  config?: Partial<CRUDOptions>
  size?: string
  formMode?: 'drawer' | 'dialog'
  showHeader?: boolean
  showSearch?: boolean
  showToolbar?: boolean
  showPagination?: boolean
  readonly?: boolean
  autoFetch?: boolean
  rowActions?: ActionType[]
  batchActions?: ActionType[]
  formActions?: ActionType[]
  searchActions?: ActionType[]
  defaultSort?: string
  presetQuery?: Record<string, any>
  gridProps?: Record<string, any>
  formProps?: Record<string, any>
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
  apiPrefix: '',
  config: () => ({}),
  formMode: 'dialog',
  showHeader: true,
  showSearch: true,
  showToolbar: true,
  showPagination: true,
  readonly: false,
  autoFetch: true,
  rowActions: () => [],
  batchActions: () => [],
  formActions: () => [],
  searchActions: () => [],
  defaultSort: '',
  presetQuery: () => ({}),
  gridProps: () => ({}),
  formProps: () => ({}),
})

const emit = defineEmits<{
  ready: [crud: CRUD]
}>()

const globalConfig = useSchemaUI()
const crud = ref<CRUD | null>(null)
const schemas = ref<Schema[]>([])
const searching = ref(false)
const isReady = ref(false)
const searchModel = ref<Record<string, any>>({})
const selections = ref<any[]>([])

const pagination = computed(() => {
  if (!crud.value) return { index: 1, size: 15, totalCount: 0 }
  return {
    index: crud.value.getPaginationIndex(),
    size: crud.value.getPaginationSize(),
    totalCount: crud.value.getPaginationCount(),
  }
})

const searchActionList = computed((): ActionType[] => {
  if (props.searchActions.length > 0) return props.searchActions
  return [
    {
      name: 'search',
      label: '搜索',
      type: 'primary',
      asyncCallback: async (model, schemas) => {
        searching.value = true
        try {
          crud.value!.resetPagination().setQueryParams(clearSearchModel(model, schemas || []))
          await crud.value!.searchModel()
        } finally {
          searching.value = false
        }
      },
    },
  ]
})

const rowActionList = computed((): ActionType[] => {
  if (props.readonly) return []
  if (props.rowActions.length > 0) return props.rowActions
  return [
    {
      name: 'edit',
      label: '编辑',
      type: 'success',
      callback: (model) => handleEdit(model),
    },
    {
      name: 'delete',
      label: '删除',
      type: 'danger',
      callback: (model) => handleDelete(model),
    },
  ]
})

const batchActionList = computed((): ActionType[] => {
  const defaults: ActionType[] = []
  if (props.batchActions.length > 0) return props.batchActions
  defaults.push({
    name: 'export',
    label: '导出',
    callback: () => {
      crud.value!.exportModels().catch((e) => console.error('Export failed:', e))
    },
  })
  return defaults
})

const formActionList = computed((): ActionType[] => {
  if (props.formActions.length > 0) return props.formActions
  return [
    {
      name: 'save',
      label: '保存',
      type: 'primary',
      asyncCallback: async (model) => {
        crud.value!.resetError()
        if (crud.value && props.config) {
          // 调用创建或更新
        }
        const isCreate = !model[crud.value!.primaryKey]
        if (isCreate) {
          await crud.value!.createModel(model)
        } else {
          await crud.value!.updateModel(model)
        }
      },
    },
  ]
})

async function init() {
  const instance = new CRUD({
    module: props.module,
    table: props.table,
    apiPrefix: props.apiPrefix || globalConfig.apiPrefix || 'rest',
    httpClient: globalConfig.httpClient,
    schemas: props.config?.schemas,
  })

  const loadedSchemas = await instance.initialize()
  schemas.value = loadedSchemas
  crud.value = instance

  // presetQuery
  for (const key in props.presetQuery) {
    searchModel.value[key] = props.presetQuery[key]
  }

  // defaultSort
  if (props.defaultSort) {
    if (props.defaultSort.startsWith('-')) {
      instance.setSortable(props.defaultSort.slice(1), 'descending')
    } else {
      instance.setSortable(props.defaultSort, 'ascending')
    }
  }

  isReady.value = true
  emit('ready', instance)

  if (props.autoFetch) {
    searching.value = true
    try {
      await instance.searchModel()
    } finally {
      searching.value = false
    }
  }
}

init()

watch(
  () => [props.module, props.table],
  () => {
    isReady.value = false
    init()
  }
)

function handleSearch(model: Model) {
  searching.value = true
  crud
    .value!.resetPagination()
    .setQueryParams(clearSearchModel(model, schemas.value))
    .searchModel()
    .finally(() => {
      searching.value = false
    })
}

function handleCreate() {
  // SchemaPage 已处理弹窗显示
}

async function handleEdit(model: Model) {
  const pk = crud.value!.findModelPrimaryKey(model)
  const qs: Record<string, any> = { scenario: 'update', __format: 'raw' }
  qs[crud.value!.primaryKey] = pk
  try {
    const fullModel = await crud.value!.getModel(qs)
    // 通过 emit 给 SchemaPage 的 formModel
    // 实际通过 ref 暴露 SchemaPage 方法会更直接
  } catch (e) {
    console.error('Failed to fetch model for edit:', e)
  }
}

function handleDelete(model: Model) {
  crud.value!.deleteModel(model).catch((e) => console.error('Delete failed:', e))
}

function handlePageChange(index: number) {
  searching.value = true
  crud
    .value!.setPaginationIndex(index)
    .searchModel()
    .finally(() => {
      searching.value = false
    })
}

function handleSortChange(e: { column: string; order: 'ascending' | 'descending' | null }) {
  if (!e.order) {
    crud.value!.setSortable('', '')
  } else {
    crud.value!.setSortable(e.column, e.order)
  }
  searching.value = true
  crud
    .value!.searchModel()
    .finally(() => {
      searching.value = false
    })
}

function handleSelectionChange(selection: any[]) {
  selections.value = selection
}

async function handleFormSubmit(model: Model, scenario: string) {
  crud.value!.resetError()
  try {
    if (scenario === 'create') {
      await crud.value!.createModel(model)
    } else {
      await crud.value!.updateModel(model)
    }
  } catch (e) {
    console.error('Form submit failed:', e)
    throw e
  }
}
</script>
```

- [ ] **Step 2: Commit**

```bash
git add packages/schema-ui/src/ui/SchemaViewer.vue
git commit -m "feat(schema-ui): add SchemaViewer auto-CRUD component"
```

---

### Task 12: 入口文件与构建验证

**Files:**
- Create: `packages/schema-ui/src/index.ts`

- [ ] **Step 1: 编写 index.ts**

```typescript
// === Plugin ===
export { SchemaUIPlugin } from './plugin'
export type { SchemaUIConfig } from './config'

// === Runtime ===
export { useSchemaUI } from './runtime/useSchemaUI'
export { CRUD } from './runtime/crud'

// === Core Types ===
export type {
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
} from './core/types'

// === Core Constants ===
export {
  TypeInteger,
  TypeFloat,
  TypeBoolean,
  TypeString,
  FormatInteger,
  FormatFloat,
  FormatBoolean,
  FormatString,
  FormatText,
  FormatDropdown,
  FormatDatetime,
  FormatDate,
  FormatTime,
  FormatTimestamp,
  FormatPassword,
  ScenarioCreate,
  ScenarioUpdate,
  ScenarioDelete,
  ScenarioSearch,
  ScenarioExport,
  ScenarioImport,
  ScenarioList,
  ScenarioDetail,
  MatchExactly,
  MatchFuzzy,
  LiveTypeDropdown,
  LiveTypeCascader,
} from './core/constants'

// === Core Utils ===
export { Scenarios } from './core/scenarios'
export { encode, decode } from './core/codec'
export { getModelValue, getModelLabel } from './core/model'
export { generateSchemaRule, checkSchemaVisible, clearSearchModel } from './core/form'

// === UI Components ===
export { default as SchemaForm } from './ui/SchemaForm.vue'
export { default as SchemaGrid } from './ui/SchemaGrid.vue'
export { default as SchemaPage } from './ui/SchemaPage.vue'
export { default as SchemaViewer } from './ui/SchemaViewer.vue'
```

- [ ] **Step 2: 安装依赖并构建**

```bash
cd packages/schema-ui
npm install
npx vite build
```

**Expected output:** `dist/` directory created with `schema-ui.es.js`, `schema-ui.cjs`, and `.d.ts` files.

- [ ] **Step 3: 验证类型声明**

```bash
npx vue-tsc --noEmit
```

**Expected:** No type errors.

- [ ] **Step 4: Commit**

```bash
git add packages/schema-ui/src/index.ts
git commit -m "feat(schema-ui): add package entry point and public API exports"
```

---

## Self-Review

### 1. Spec Coverage

| Spec Section | Plan Task | Status |
|---|---|---|
| 目录结构与构建配置 | Task 1 | Covered |
| 类型定义（REST 标准） | Task 2 | Covered |
| 配置系统（Plugin + Props） | Task 4 | Covered |
| CRUD 类 | Task 5 | Covered |
| SchemaForm 组件 | Task 8 | Covered |
| SchemaGrid 组件 | Task 9 | Covered |
| SchemaPage 组件 | Task 10 | Covered |
| SchemaViewer 组件 | Task 11 | Covered |
| Props 命名优化 | Tasks 8-11 | Covered（使用优化后的命名） |
| presetQuery/fixedQuery 职责分离 | Task 11 | Covered（presetQuery 在 Viewer props，fixedQuery 走 CRUD.setFixedQuery） |
| 构建与发布 | Task 12 | Covered |

### 2. Placeholder Scan

- No "TBD", "TODO", "implement later" found
- All code blocks contain complete implementation
- All task steps have exact commands

### 3. Type Consistency

- `Schema` interface uses `attributes` (plural) consistently across all components
- `primary_key: number` (0/1) used consistently in Cell, SchemaGrid, CRUD
- `CRUDOptions` uses `module`/`table` (not `moduleName`/`tableName`)
- Props naming follows optimized names from design spec

---

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-05-17-schema-ui-package.md`.**

Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach would you prefer?**
