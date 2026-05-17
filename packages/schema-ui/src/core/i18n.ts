export const defaultMessages: Record<string, string | ((...args: string[]) => string)> = {
  'validation.required': (label: string) => `${label}不能为空`,
  'validation.min': (label: string, min: string) => `${label}不能少于${min}个字符`,
  'validation.max': (label: string, max: string) => `${label}不能超过${max}个字符`,
  'validation.pattern': (label: string) => `${label}格式不正确`,
  'validation.type': (label: string) => `${label}格式类型不正确`,
}

export function createDefaultTranslator(): (key: string, args?: string[]) => string {
  return (key: string, args?: string[]) => {
    const message = defaultMessages[key]
    if (typeof message === 'function') {
      return message(...(args || []))
    }
    return message || key
  }
}
