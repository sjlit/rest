<template>
  <div class="schema-form" ref="formElement">
    <el-form
      :model="activeModel"
      :label-width="labelWidthComputed"
      :rules="formRules"
      :validate-on-rule-change="false"
      :inline="inlineMode"
      ref="formRef"
      status-icon
    >
      <slot name="container" :model="activeModel" :schemas="displayColumns">
        <el-row :gutter="20" v-if="grid">
          <el-col
            :span="getColSpan(schema)"
            v-for="schema in displayColumns"
            :key="schema.column"
          >
            <el-form-item
              :prop="schema.column"
              :label="schema.label"
              :error="fieldErrors[schema.column]"
            >
              <slot name="default" :model="activeModel" :schema="schema">
                <FormItem :model="activeModel" :schema="schema" :scenario="scenario" />
              </slot>
            </el-form-item>
          </el-col>
        </el-row>
        <template v-else>
          <el-form-item
            v-for="schema in displayColumns"
            :key="schema.column"
            :prop="schema.column"
            :label="schema.label"
            :error="fieldErrors[schema.column]"
          >
            <slot name="default" :model="activeModel" :schema="schema">
              <FormItem :model="activeModel" :schema="schema" :scenario="scenario" />
            </slot>
          </el-form-item>
        </template>
        <el-form-item v-if="actions.length > 0" class="schema-form-actions">
          <Action
            v-for="action in actions"
            :key="action.name"
            :action="action"
            @click="handleActionClick"
          />
        </el-form-item>
      </slot>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import type { Schema, Model, Action as ActionType } from '../core/types'
import { decode, encode } from '../core/codec'
import { generateSchemaRule, checkSchemaVisible } from '../core/form'
import { useSchemaUI } from '../runtime/useSchemaUI'
import FormItem from './parts/FormItem.vue'
import Action from './parts/Action.vue'

interface Props {
  size?: string
  schemas: Schema[]
  scenario?: string
  labelWidth?: string
  model?: Model
  inline?: boolean
  grid?: boolean
  gridCols?: number
  actions?: ActionType[]
  autoSubmit?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  scenario: 'create',
  labelWidth: '',
  model: undefined,
  inline: false,
  grid: false,
  gridCols: 0,
  actions: () => [],
  autoSubmit: false,
})

const emit = defineEmits<{
  submit: [model: Model, schemas: Schema[]]
}>()

const formRef = ref<any>(null)
const formElement = ref<HTMLElement | null>(null)
const activeModel = ref<Model>({})
const formWidth = ref<number>(typeof window !== 'undefined' ? window.innerWidth : 1200)
const fieldErrors = ref<Record<string, string>>({})
const stopWatchers: (() => void)[] = []

const BREAKPOINTS = { MOBILE: 768, TABLET: 960 }
const LABEL_WIDTHS = { MOBILE: '80px', DESKTOP: '120px' }
const COL_SPANS = { FULL: 24, HALF: 12, THIRD: 8 }

const displayColumns = computed(() => {
  return props.schemas.filter(schema => {
    if (schema.attributes.invisible) return false
    if (!Array.isArray(schema.scenarios)) return false
    if (!schema.scenarios.includes(props.scenario)) return false
    return checkSchemaVisible(schema, activeModel.value)
  })
})

const inlineMode = computed(() => {
  if (props.grid) return false
  return props.inline
})

const labelWidthComputed = computed(() => {
  if (props.inline) return '0'
  if (props.labelWidth) return props.labelWidth
  return formWidth.value < BREAKPOINTS.MOBILE ? LABEL_WIDTHS.MOBILE : LABEL_WIDTHS.DESKTOP
})

let globalConfig: any = null
try {
  globalConfig = useSchemaUI()
} catch { /* SchemaPage may be used without plugin */ }

const formRules = computed(() => {
  const rules: Record<string, any> = {}
  if (props.scenario === 'search') return rules
  const t = globalConfig?.i18n?.t || ((key: string, args?: any[]) => {
    const messages: Record<string, string> = {
      'validation.required': args ? `${args[0]}不能为空` : '必填项',
      'validation.max': args ? `${args[0]}不能超过${args[1]}个字符` : '超出最大长度',
      'validation.pattern': args ? `${args[0]}格式不正确` : '格式不正确',
    }
    return messages[key] || key
  })
  for (const schema of displayColumns.value) {
    rules[schema.column] = generateSchemaRule(t, schema, props.scenario)
  }
  return rules
})

function getColSpan(schema: Schema): number {
  if (props.gridCols > 0) return Math.min(props.gridCols, 24)
  if (schema.format === 'textarea') return COL_SPANS.FULL
  if (formWidth.value < BREAKPOINTS.MOBILE) return COL_SPANS.FULL
  if (formWidth.value < BREAKPOINTS.TABLET) return COL_SPANS.HALF
  return COL_SPANS.THIRD
}

onMounted(() => {
  activeModel.value = decode(props.model || {}, props.schemas, props.scenario)
  if (props.autoSubmit) {
    nextTick(() => submit())
  }
  if (formElement.value) {
    formWidth.value = formElement.value.offsetWidth
  }

  stopWatchers.push(
    watch(
      () => props.model,
      (val) => {
        activeModel.value = decode(val || {}, props.schemas, props.scenario)
      },
      { deep: true }
    )
  )
})

onUnmounted(() => {
  stopWatchers.forEach((stop) => stop())
})

async function submit(): Promise<Model> {
  const model = encode(activeModel.value, displayColumns.value, props.scenario)
  if (props.model) {
    const pkSchema = props.schemas.find((s) => s.primary_key === 1)
    if (pkSchema && pkSchema.column in props.model) {
      model[pkSchema.column] = props.model[pkSchema.column]
    }
  }

  try {
    await formRef.value!.validate()
    emit('submit', model, displayColumns.value)
    return model
  } catch (e: any) {
    if (e && typeof e === 'object') {
      const fields = Object.values(e)
      for (const field of fields) {
        if (Array.isArray(field) && field.length > 0 && field[0]?.message) {
          throw new Error(field[0].message)
        }
      }
    }
    throw new Error('验证失败')
  }
}

async function handleActionClick(action: ActionType, loading: any) {
  if (typeof action.callback === 'function') {
    try {
      const result = await submit()
      action.callback(result, displayColumns.value, loading)
    } catch (e) {
      console.error('Form submit failed:', e)
    }
  } else if (typeof action.asyncCallback === 'function') {
    loading.value = true
    try {
      await action.asyncCallback(await submit(), displayColumns.value)
    } finally {
      loading.value = false
    }
  }
}

defineExpose({ submit })
</script>
