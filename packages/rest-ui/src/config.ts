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

export const GLOBAL_CONFIG_KEY = Symbol('rest-ui-config')
