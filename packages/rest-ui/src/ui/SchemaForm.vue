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
                <FormItem v-model="activeModel[schema.column]" :schema="schema" :scenario="scenario" />
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
              <FormItem v-model="activeModel[schema.column]" :schema="schema" :scenario="scenario" />
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
import { ref, computed, watch, onMounted, onUnmounted, nextTick, inject } from 'vue'
import type { Schema, Model, Action as ActionType, Scenario } from '../core/types'
import type { SchemaUIConfig } from '../config'
import { decode, encode } from '../core/codec'
import { generateSchemaRule, checkSchemaVisible, filterByScenario } from '../core/form'
import { createDefaultTranslator } from '../core/i18n'
import { GLOBAL_CONFIG_KEY } from '../config'
import FormItem from './parts/FormItem.vue'
import Action from './parts/Action.vue'
import '../styles/form.scss'

interface Props {
  size?: string
  schemas: Schema[]
  scenario?: Scenario
  labelWidth?: string
  model?: Model
  inline?: boolean
  grid?: boolean
  gridCols?: number
  actions?: ActionType[]
  autoSubmit?: boolean
  errors?: Record<string, string>
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
  errors: () => ({}),
})

const formRef = ref<any>(null)
const formElement = ref<HTMLElement | null>(null)
const activeModel = ref<Model>({})
const formWidth = ref<number>(1200)
const fieldErrors = ref<Record<string, string>>({})
const stopWatchers: (() => void)[] = []
let resizeHandler: (() => void) | null = null

const BREAKPOINTS = { MOBILE: 768, TABLET: 960 }
const LABEL_WIDTHS = { MOBILE: '80px', DESKTOP: '120px' }
const COL_SPANS = { FULL: 24, HALF: 12, THIRD: 8 }

const displayColumns = computed(() => {
  return filterByScenario(props.schemas, props.scenario, {
    visibleCheck: (schema) => checkSchemaVisible(schema, activeModel.value),
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

const globalConfig = inject<SchemaUIConfig | null>(GLOBAL_CONFIG_KEY, null)

const formRules = computed(() => {
  const rules: Record<string, any> = {}
  if (props.scenario === 'search') return rules
  const t = globalConfig?.i18n?.t || createDefaultTranslator()
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

function applyDefaults(model: Model) {
  for (const schema of props.schemas) {
    const defaultValue = schema.attributes.default_value
    if (defaultValue !== undefined && defaultValue !== null && defaultValue !== '') {
      if (model[schema.column] === undefined || model[schema.column] === null || model[schema.column] === '') {
        model[schema.column] = defaultValue
      }
    }
  }
}

onMounted(() => {
  activeModel.value = decode(props.model || {}, props.schemas, props.scenario)
  applyDefaults(activeModel.value)
  if (props.autoSubmit) {
    nextTick(() => submit())
  }
  if (formElement.value) {
    formWidth.value = formElement.value.offsetWidth
  }

  resizeHandler = () => {
    if (formElement.value) {
      formWidth.value = formElement.value.offsetWidth
    }
  }
  window.addEventListener('resize', resizeHandler)

  stopWatchers.push(
    watch(
      () => props.model,
      (val) => {
        activeModel.value = decode(val || {}, props.schemas, props.scenario)
        applyDefaults(activeModel.value)
      },
      { deep: true }
    )
  )

  // Sync external errors prop to internal fieldErrors
  stopWatchers.push(
    watch(
      () => props.errors,
      (val) => {
        fieldErrors.value = { ...val }
      },
      { deep: true, immediate: true }
    )
  )
})

onUnmounted(() => {
  stopWatchers.forEach((stop) => stop())
  if (resizeHandler) {
    window.removeEventListener('resize', resizeHandler)
  }
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
