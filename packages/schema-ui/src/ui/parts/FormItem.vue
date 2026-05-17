<template>
  <template v-if="isVisible('time')">
    <el-time-select
      v-model="model[schema.column]"
      start="00:00"
      step="00:15"
      end="23:59"
      :disabled="isDisabled"
      :readonly="isReadonly"
      :placeholder="placeholder"
      format="HH:mm"
    />
  </template>
  <template v-else-if="isVisible('date')">
    <el-date-picker
      v-if="isRange"
      v-model="model[schema.column]"
      type="daterange"
      :disabled="isDisabled"
      :readonly="isReadonly"
      :editable="false"
      format="YYYY-MM-DD"
      value-format="YYYY-MM-DD"
      :start-placeholder="startPlaceholder"
      :end-placeholder="endPlaceholder"
      :disabled-date="disabledDate"
    />
    <el-date-picker
      v-else
      v-model="model[schema.column]"
      type="date"
      :disabled="isDisabled"
      :readonly="isReadonly"
      :editable="false"
      format="YYYY-MM-DD"
      value-format="YYYY-MM-DD"
      :placeholder="placeholder"
      :disabled-date="disabledDate"
    />
  </template>
  <template v-else-if="isVisible('datetime')">
    <el-date-picker
      v-if="isRange"
      v-model="model[schema.column]"
      type="datetimerange"
      :disabled="isDisabled"
      :readonly="isReadonly"
      :editable="false"
      format="YYYY-MM-DD HH:mm:ss"
      value-format="YYYY-MM-DD HH:mm:ss"
      :start-placeholder="startPlaceholder"
      :end-placeholder="endPlaceholder"
      :disabled-date="disabledDate"
    />
    <el-date-picker
      v-else
      v-model="model[schema.column]"
      type="datetime"
      :disabled="isDisabled"
      :readonly="isReadonly"
      :editable="false"
      format="YYYY-MM-DD HH:mm:ss"
      value-format="YYYY-MM-DD HH:mm:ss"
      :placeholder="placeholder"
      :disabled-date="disabledDate"
    />
  </template>
  <template v-else-if="isVisible('dropdown')">
    <el-select
      v-model="model[schema.column]"
      :multiple="isMultiSelect"
      :disabled="isDisabled || isReadonly"
      :placeholder="placeholder"
      :allow-create="schema.attributes.dropdown?.created"
      :filterable="schema.attributes.dropdown?.filterable"
      :default-first-option="schema.attributes.dropdown?.default_first"
      clearable
    >
      <el-option
        v-for="item in schema.attributes.values"
        :key="item.value"
        :label="item.label"
        :value="item.value"
      />
    </el-select>
  </template>
  <template v-else-if="isVisible('search_boolean')">
    <el-select v-model="model[schema.column]" clearable>
      <el-option label="是" :value="true" />
      <el-option label="否" :value="false" />
    </el-select>
  </template>
  <template v-else-if="isVisible('cascader')">
    <el-cascader
      v-model="model[schema.column]"
      :options="schema.attributes.values"
      :disabled="isDisabled || isReadonly"
      filterable
      clearable
      :placeholder="placeholder"
      :validate-event="false"
    />
  </template>
  <template v-else-if="isVisible('boolean')">
    <el-switch v-model="model[schema.column]" :disabled="isDisabled || isReadonly" />
  </template>
  <template v-else-if="isVisible('file')">
    <el-upload
      :action="schema.attributes.upload_url"
      :disabled="isDisabled || isReadonly"
      :file-list="fileList"
      @success="handleUploadSuccess"
      @remove="handleUploadRemove"
    >
      <el-button type="primary">上传</el-button>
    </el-upload>
  </template>
  <template v-else-if="isVisible('password')">
    <el-input
      v-model="model[schema.column]"
      :disabled="isDisabled"
      :readonly="isReadonly"
      show-password
      :placeholder="placeholder"
    />
  </template>
  <template v-else-if="isVisible('multistr')">
    <el-input
      v-model="model[schema.column]"
      type="textarea"
      :disabled="isDisabled"
      :readonly="isReadonly"
      :placeholder="placeholder"
      :maxlength="schema.rules.max > 0 ? schema.rules.max : undefined"
    />
  </template>
  <template v-else-if="isVisible('number')">
    <el-input
      v-model.number="model[schema.column]"
      :disabled="isDisabled"
      :readonly="isReadonly"
      :prefix-icon="schema.attributes.icon || ''"
      :placeholder="placeholder"
      :clearable="scenario === 'search'"
    >
      <template v-if="schema.attributes.suffix" #append>
        {{ schema.attributes.suffix }}
      </template>
    </el-input>
  </template>
  <template v-else>
    <el-input
      v-model="model[schema.column]"
      :disabled="isDisabled"
      :readonly="isReadonly"
      :prefix-icon="schema.attributes.icon || ''"
      :placeholder="placeholder"
      :clearable="scenario === 'search'"
    >
      <template v-if="schema.attributes.suffix" #append>
        {{ schema.attributes.suffix }}
      </template>
    </el-input>
  </template>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Model, Schema } from '../../core/types'

interface Props {
  model: Model
  schema: Schema
  scenario?: string
}

const props = defineProps<Props>()

const scenario = computed(() => props.scenario || 'create')
const isSearch = computed(() => scenario.value === 'search')
const isRange = computed(() => scenario.value === 'search')

const isDisabled = computed(() => {
  return props.schema.attributes.disable.includes(scenario.value)
})

const isReadonly = computed(() => {
  return (
    !isDisabled.value &&
    props.schema.attributes.readonly.includes(scenario.value)
  )
})

const isMultiSelect = computed(() => {
  if (props.schema.format === 'multiSelect') return true
  if (props.schema.attributes.dropdown?.multiple) return true
  return false
})

const fileList = ref<any[]>([])

function syncFileList() {
  const url = props.model[props.schema.column]
  if (url) {
    fileList.value = [{ name: String(url).split('/').pop() || url, url }]
  } else {
    fileList.value = []
  }
}

watch(() => props.model[props.schema.column], syncFileList, { immediate: true })

function handleUploadSuccess(response: any) {
  const url = response.url || response.data?.url || response
  props.model[props.schema.column] = url
  fileList.value = [{ name: String(url).split('/').pop() || url, url }]
}

function handleUploadRemove() {
  props.model[props.schema.column] = ''
  fileList.value = []
}

const placeholder = computed(() => {
  if (props.schema.attributes.tooltip) return props.schema.attributes.tooltip
  if (props.schema.format === 'dropdown') {
    return `请选择${props.schema.label}`
  }
  return `请输入${props.schema.label}`
})

const startPlaceholder = computed(() => `开始${props.schema.label}`)
const endPlaceholder = computed(() => `结束${props.schema.label}`)

function disabledDate(time: Date) {
  if (!props.schema.attributes.end_of_now) return false
  return time.getTime() > Date.now()
}

function isVisible(type: string): boolean {
  const fmt = props.schema.format
  const tp = props.schema.type

  switch (type) {
    case 'number':
      return ['integer', 'float', 'decimal'].includes(fmt) || ['integer', 'float', 'double'].includes(tp)
    case 'password':
      return ['password', 'pass'].includes(fmt)
    case 'time':
      return fmt === 'time'
    case 'date':
      return fmt === 'date'
    case 'datetime':
      return ['datetime', 'timestamp'].includes(fmt)
    case 'dropdown':
      return ['dropdown', 'multiSelect'].includes(fmt)
    case 'cascader':
      return fmt === 'cascader'
    case 'boolean':
      return scenario.value !== 'search' && ['bool', 'boolean'].includes(fmt)
    case 'search_boolean':
      return scenario.value === 'search' && ['bool', 'boolean'].includes(fmt)
    case 'file':
      return fmt === 'file' && !!props.schema.attributes.upload_url
    case 'multistr':
      return scenario.value !== 'search' && fmt === 'text'
    default:
      return ['string', 'text'].includes(fmt)
  }
}
</script>
