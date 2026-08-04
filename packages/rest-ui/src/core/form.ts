import type { Schema, Scenario } from './types'

export interface FieldRules {
  [key: string]: any
}

/**
 * 根据 schema 生成 Element Plus 表单验证规则
 */
export function generateSchemaRule(
  t: (key: string, ...args: any[]) => string,
  schema: Schema,
  scenario: Scenario
): any[] {
  const rules: any[] = []
  const rule = schema.rules

  if (rule.safe) {
    return rules
  }

  if (rule.required && (rule.required as string[]).includes(scenario)) {
    rules.push({
      required: true,
      message: t('validation.required', schema.label),
      trigger: 'blur',
    })
  }

  if (rule.min > 0 && schema.type === 'string') {
    rules.push({
      min: rule.min,
      message: t('validation.min', schema.label, rule.min),
      trigger: 'blur',
    })
  }

  if (rule.max > 0 && schema.type === 'string') {
    rules.push({
      max: rule.max,
      message: t('validation.max', schema.label, rule.max),
      trigger: 'blur',
    })
  }

  if (rule.type) {
    rules.push({
      type: rule.type,
      message: t('validation.type', schema.label),
      trigger: 'blur',
    })
  }

  if (rule.regular) {
    rules.push({
      pattern: new RegExp(rule.regular),
      message: t('validation.pattern', schema.label),
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
    const found = cond.values.some((v) => String(v) === String(modelValue))
    if (!found) {
      return false
    }
  }

  return true
}

/**
 * 生成字段描述文本
 */
export function generateSchemaDescription(
  _t: (key: string, ...args: any[]) => string,
  schema: Schema,
  _scenario: Scenario
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
  _schemas: Schema[]
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

/**
 * 按场景过滤可见的 schema 列表
 */
export function filterByScenario(
  schemas: Schema[],
  scenario: Scenario,
  options?: {
    includeInvisible?: boolean
    visibleCheck?: (schema: Schema) => boolean
  }
): Schema[] {
  return schemas.filter((schema) => {
    if (schema.enable === 0) return false
    if (!Array.isArray(schema.scenarios)) return false
    if (!schema.scenarios.includes(scenario)) return false
    if (!options?.includeInvisible && schema.attributes.invisible) return false
    if (options?.visibleCheck && !options.visibleCheck(schema)) return false
    return true
  })
}
