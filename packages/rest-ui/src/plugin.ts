import type { App } from 'vue'
import { GLOBAL_CONFIG_KEY, type SchemaUIConfig } from './config'

export const SchemaUIPlugin = {
  install(app: App, config: SchemaUIConfig) {
    if (!config.httpClient) {
      throw new Error('[rest-ui] httpClient is required in SchemaUIConfig')
    }
    app.provide(GLOBAL_CONFIG_KEY, config)
  },
}
