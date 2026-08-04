import pluralize from 'pluralize'
import type { Schema, Model, CRUDOptions, Pagination, Sortable, Scenario } from '../core/types'

type UriScenario = Scenario | 'get'

const DEFAULT_PAGINATION: Pagination = {
  index: 1,
  size: 15,
  totalCount: 0,
}

export class CRUD {
  private opts: Required<CRUDOptions>
  primaryKey = ''
  schemas: Schema[] = []
  models: Model[] = []
  sortable: Sortable | null = null
  queryParams: Record<string, any> = {}
  fixedQuery: Record<string, any> = {}
  pagination: Pagination = { ...DEFAULT_PAGINATION }

  constructor(options: CRUDOptions) {
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
      httpClient: options.httpClient,
      schemas: schemas.map(s => ({ ...s, rules: s.rules ? { ...s.rules } : { min: 0, max: 0, type: '', unique: false, required: [] } })),
    }
  }

  private __prepare() {
    for (const schema of this.schemas) {
      if (schema.primary_key === 1) {
        this.primaryKey = schema.column
        break
      }
    }
  }

  private __buildUri(scenario: UriScenario, primaryKey?: string): string {
    return this.__buildModelUri(this.opts.module, this.opts.table, scenario, primaryKey)
  }

  private __buildModelUri(moduleName: string, tableName: string, scenario: UriScenario, primaryKey?: string): string {
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
      case 'get':
        parts.push(singularName, 'detail', pk)
        break
      case 'update':
      case 'delete':
        parts.push(singularName, pk)
        break
      case 'search':
        parts.push(pluralName)
        break
      case 'export':
        parts.push(singularName, 'export')
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
    const rawSchemas: Schema[] = Array.isArray(res) ? res : res.data || []
    this.schemas = rawSchemas.map(s => ({ ...s, rules: s.rules ? { ...s.rules } : { min: 0, max: 0, type: '', unique: false, required: [] } }))
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

  fieldErrors: Record<string, string> = {}

  setColumnError(column: string, error: string) {
    this.fieldErrors[column] = error
  }

  resetError() {
    this.fieldErrors = {}
  }

  getFieldErrors(): Record<string, string> {
    return this.fieldErrors
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

  addQueryParams(k: string, v: any): this {
    this.queryParams[k] = v
    return this
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
      const value = model[this.primaryKey]
      // 兼容后端 __format=both 返回的 { label, value } 单元格结构，解包取原始值
      if (value && typeof value === 'object' && 'value' in value) {
        return value.value
      }
      return value
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
    queryParams.page_size = this.pagination.size || 15
    if (this.sortable?.column) {
      queryParams.sort = this.sortable.order === 'descending' ? `-${this.sortable.column}` : this.sortable.column
    }
    queryParams.__format = 'both'
    for (const k in this.fixedQuery) {
      queryParams[k] = this.fixedQuery[k]
    }

    // Convert date range arrays to comma-separated strings for backend compatibility
    for (const key in queryParams) {
      const value = queryParams[key]
      if (
        Array.isArray(value) &&
        value.length === 2 &&
        value[0] !== '' && value[0] !== null && value[0] !== undefined &&
        value[1] !== '' && value[1] !== null && value[1] !== undefined
      ) {
        const scm = this.schemas.find(s => s.column === key)
        if (scm && ['datetime', 'date', 'timestamp', 'time'].includes(scm.format)) {
          queryParams[key] = `${value[0]},${value[1]}`
        }
      }
    }

    const res = await this.opts.httpClient.get(this.__buildUri('search'), { params: queryParams })
    this.pagination.index = (parseInt(res.page) || 0) + 1
    this.pagination.size = parseInt(res.page_size) || 15
    this.pagination.totalCount = parseInt(res.total_count) || 0
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
    queryParams.__format = 'both'
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
    const index = this.models.findIndex(m => this.findModelPrimaryKey(m) === pk)
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
