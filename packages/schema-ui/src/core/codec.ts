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
