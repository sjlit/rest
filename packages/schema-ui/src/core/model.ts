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
