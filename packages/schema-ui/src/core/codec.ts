import type { Model, Schema, Scenario } from './types'

/**
 * 将模型值编码为提交格式
 */
export function encode(model: Model, schemas: Schema[], _scenario: Scenario): Model {
  const result: Model = {}
  for (const key in model) {
    const schema = schemas.find(s => s.column === key)
    let value = model[key]
    if (schema && value !== undefined && value !== null) {
      if (['datetime', 'date', 'timestamp', 'time'].includes(schema.format) && value instanceof Date) {
        const pad = (n: number) => String(n).padStart(2, '0')
        const y = value.getFullYear()
        const m = pad(value.getMonth() + 1)
        const d = pad(value.getDate())
        const h = pad(value.getHours())
        const min = pad(value.getMinutes())
        const s = pad(value.getSeconds())
        if (schema.format === 'date') {
          value = `${y}-${m}-${d}`
        } else if (schema.format === 'time') {
          value = `${h}:${min}:${s}`
        } else {
          value = `${y}-${m}-${d} ${h}:${min}:${s}`
        }
      }
    }
    result[key] = value
  }
  return result
}

/**
 * 将后端返回的模型值解码为表单可用格式
 */
export function decode(model: Model, schemas: Schema[], _scenario: Scenario): Model {
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
