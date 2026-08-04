export const defaultMessages: Record<string, string | ((...args: string[]) => string)> = {
  'validation.required': (label: string) => `${label}不能为空`,
  'validation.min': (label: string, min: string) => `${label}不能少于${min}个字符`,
  'validation.max': (label: string, max: string) => `${label}不能超过${max}个字符`,
  'validation.pattern': (label: string) => `${label}格式不正确`,
  'validation.type': (label: string) => `${label}格式类型不正确`,

  'action.search': '搜索',
  'action.create': '创建',
  'action.edit': '编辑',
  'action.delete': '删除',
  'action.view': '详情',
  'action.save': '保存',
  'action.export': '导出',
  'action.confirm': '确定',
  'action.cancel': '取消',

  'boolean.true': '是',
  'boolean.false': '否',

  'form.create': '创建',
  'form.edit': '编辑',
  'form.detail': '详情',

  'placeholder.select': (label: string) => `请选择${label}`,
  'placeholder.input': (label: string) => `请输入${label}`,
  'placeholder.start': (label: string) => `开始${label}`,
  'placeholder.end': (label: string) => `结束${label}`,
}

export function createDefaultTranslator(): (key: string, ...args: any[]) => string {
  return (key: string, ...args: any[]) => {
    const message = defaultMessages[key]
    if (typeof message === 'function') {
      return message(...args)
    }
    return message || key
  }
}
