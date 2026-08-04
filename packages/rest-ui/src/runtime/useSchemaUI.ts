import { inject } from 'vue'
import { GLOBAL_CONFIG_KEY, type SchemaUIConfig } from '../config'

export function useSchemaUI(): SchemaUIConfig {
  const config = inject<SchemaUIConfig>(GLOBAL_CONFIG_KEY)
  if (!config) {
    throw new Error(
      '[rest-ui] SchemaUIPlugin not installed. Call app.use(SchemaUIPlugin, config) first.'
    )
  }
  return config
}
