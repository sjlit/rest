import type { Schema } from './types'

export interface FieldRules {
  [key: string]: any
}

/**
 * 根据 schema 生成 Element Plus 表单验证规则
 */
export function generateSchemaRule(
  t: (key: string, ...args: any[]) => string,
  schema: Schema,
  scenario: string
): any[] {
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
export function generateSchemaDescription(
  t: (key: string, ...args: any[]) => string,
  schema: Schema,
  scenario: string
): string {
  if (schema.attributes.tooltip) {
    return schema.attributes.tooltip
  }
  return ''
}

/**
 * 清空搜索模型中的空值
 */
export function clearSearchModel(
  model: Record<string, any>,
  schemas: Schema[]
): Record<string, any> {
  const result: Record<string, any> = {}
  for (const key in model) {
    const value = model[key]
    if (value !== '' && value !== null && value !== undefined) {
      result[key] = value
    }
  }
  return result
}
