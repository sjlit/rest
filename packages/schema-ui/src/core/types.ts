import type { Component } from 'vue'

export type Scenario = 'create' | 'update' | 'delete' | 'search' | 'list' | 'detail' | 'export'

export interface SchemaRule {
  min: number
  max: number
  type: 'string' | 'number' | 'integer' | 'float' | 'boolean' | 'array' | 'object' | 'date' | 'email' | 'url' | string
  unique: boolean
  required: Scenario[]
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
  match: 'exactly' | 'fuzzy' | string
  tag?: string
  default_value: string
  readonly: Scenario[]
  disable: Scenario[]
  visible: VisibleCondition[]
  invisible: boolean
  end_of_now: boolean
  time_search_range: string
  values?: EnumValue[]
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
  scenarios: Scenario[]
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
  icon?: string | Component
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
  httpClient: {
    get: (url: string, config?: any) => Promise<any>
    post: (url: string, data?: any, config?: any) => Promise<any>
    put: (url: string, data?: any, config?: any) => Promise<any>
    delete: (url: string, data?: any, config?: any) => Promise<any>
  }
  schemas?: Schema[] | Record<string, Schema>
}
