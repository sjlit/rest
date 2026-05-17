import type { Model, Schema } from './types'

/**
 * 将模型值编码为提交格式
 */
export function encode(model: Model, schemas: Schema[], scenario: string): Model {
  const result: Model = {}
  for (const key in model) {
    const schema = schemas.find(s => s.column === key)
    let value = model[key]
    if (schema && value !== undefined && value !== null) {
      if (['datetime', 'date', 'timestamp', 'time'].includes(schema.format) && value instanceof Date) {
        value = value.toISOString()
      }
    }
    result[key] = value
  }
  return result
}

/**
 * 将后端返回的模型值解码为表单可用格式
 */
export function decode(model: Model, schemas: Schema[], scenario: string): Model {
  const result: Model = {}
  for (const key in model) {
    const schema = schemas.find(s => s.column === key)
    let value = model[key]
    if (schema && value !== undefined && value !== null) {
      if (['integer'].includes(schema.type) && typeof value === 'string') {
        value = parseInt(value, 10)
      } else if (['float', 'double', 'decimal'].includes(schema.type) && typeof value === 'string') {
        value = parseFloat(value)
      } else if (schema.format === 'boolean' || schema.format === 'bool') {
        value = !!value
      }
    }
    result[key] = value
  }
  return result
}
