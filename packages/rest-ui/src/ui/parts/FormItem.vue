<template>
  <template v-if="componentType === 'time'">
    <el-time-select
      v-model="columnValue"
      start="00:00"
      step="00:15"
      end="23:59"
      :disabled="isDisabled || isReadonly"
      :placeholder="placeholder"
      format="HH:mm"
    />
  </template>
  <template v-else-if="componentType === 'date'">
    <el-date-picker
      v-if="isRange"
      v-model="columnValue"
      type="daterange"
      :disabled="isDisabled || isReadonly"
      :editable="false"
      format="YYYY-MM-DD"
      value-format="YYYY-MM-DD"
      :start-placeholder="startPlaceholder"
      :end-placeholder="endPlaceholder"
      :disabled-date="disabledDate"
    />
    <el-date-picker
      v-else
      v-model="columnValue"
      type="date"
      :disabled="isDisabled || isReadonly"
      :editable="false"
      format="YYYY-MM-DD"
      value-format="YYYY-MM-DD"
      :placeholder="placeholder"
      :disabled-date="disabledDate"
    />
  </template>
  <template v-else-if="componentType === 'datetime'">
    <el-date-picker
      v-if="isRange"
      v-model="columnValue"
      type="datetimerange"
      :disabled="isDisabled || isReadonly"
      :editable="false"
      format="YYYY-MM-DD HH:mm:ss"
      value-format="YYYY-MM-DD HH:mm:ss"
      :start-placeholder="startPlaceholder"
      :end-placeholder="endPlaceholder"
      :disabled-date="disabledDate"
    />
    <el-date-picker
      v-else
      v-model="columnValue"
      type="datetime"
      :disabled="isDisabled || isReadonly"
      :editable="false"
      format="YYYY-MM-DD HH:mm:ss"
      value-format="YYYY-MM-DD HH:mm:ss"
      :placeholder="placeholder"
      :disabled-date="disabledDate"
    />
  </template>
  <template v-else-if="componentType === 'dropdown'">
    <el-select
      v-model="columnValue"
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
  <template v-else-if="componentType === 'search_boolean'">
    <el-select v-model="columnValue" clearable>
      <el-option :label="t('boolean.true')" :value="true" />
      <el-option :label="t('boolean.false')" :value="false" />
    </el-select>
  </template>
  <template v-else-if="componentType === 'cascader'">
    <el-cascader
      v-model="columnValue"
      :options="schema.attributes.values"
      :disabled="isDisabled || isReadonly"
      filterable
      clearable
      :placeholder="placeholder"
      :validate-event="false"
    />
  </template>
  <template v-else-if="componentType === 'boolean'">
    <el-switch v-model="columnValue" :disabled="isDisabled || isReadonly" />
  </template>
  <template v-else-if="componentType === 'file'">
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
  <template v-else-if="componentType === 'password'">
    <el-input
      v-model="columnValue"
      :disabled="isDisabled"
      :readonly="isReadonly"
      show-password
      :placeholder="placeholder"
    />
  </template>
  <template v-else-if="componentType === 'multistr'">
    <el-input
      v-model="columnValue"
      type="textarea"
      :disabled="isDisabled"
      :readonly="isReadonly"
      :placeholder="placeholder"
      :maxlength="schema.rules.max > 0 ? schema.rules.max : undefined"
    />
  </template>
  <template v-else-if="componentType === 'number'">
    <el-input
      v-model.number="columnValue"
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
      v-model="columnValue"
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
import type { Schema, Scenario } from '../../core/types'
import { createDefaultTranslator } from '../../core/i18n'

interface Props {
  modelValue: any
  schema: Schema
  scenario?: Scenario
}

const props = defineProps<Props>()
const t = createDefaultTranslator()

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const scenario = computed(() => props.scenario || 'create')
const isSearch = computed(() => scenario.value === 'search')

const columnValue = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

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

const isRange = computed(() => isSearch.value)

const fileList = ref<any[]>([])

function syncFileList() {
  const url = props.modelValue
  if (url) {
    fileList.value = [{ name: String(url).split('/').pop() || url, url }]
  } else {
    fileList.value = []
  }
}

watch(() => props.modelValue, syncFileList, { immediate: true })

function handleUploadSuccess(response: any) {
  const url = response.url || response.data?.url || response
  columnValue.value = url
  fileList.value = [{ name: String(url).split('/').pop() || url, url }]
}

function handleUploadRemove() {
  columnValue.value = ''
  fileList.value = []
}

const placeholder = computed(() => {
  if (props.schema.attributes.tooltip) return props.schema.attributes.tooltip
  if (props.schema.format === 'dropdown') {
    return t('placeholder.select', props.schema.label)
  }
  return t('placeholder.input', props.schema.label)
})

const startPlaceholder = computed(() => t('placeholder.start', props.schema.label))
const endPlaceholder = computed(() => t('placeholder.end', props.schema.label))

function disabledDate(time: Date) {
  if (!props.schema.attributes.end_of_now) return false
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return time.getTime() > today.getTime()
}

const componentType = computed(() => {
  const fmt = props.schema.format
  const tp = props.schema.type

  if (['integer', 'float', 'decimal'].includes(fmt) || ['integer', 'float', 'double'].includes(tp)) {
    return 'number'
  }
  if (['password', 'pass'].includes(fmt)) return 'password'
  if (fmt === 'time') return 'time'
  if (fmt === 'date') return 'date'
  if (['datetime', 'timestamp'].includes(fmt)) return 'datetime'
  if (['dropdown', 'multiSelect'].includes(fmt)) return 'dropdown'
  if (fmt === 'cascader') return 'cascader'
  if (['bool', 'boolean'].includes(fmt)) {
    return isSearch.value ? 'search_boolean' : 'boolean'
  }
  if (fmt === 'file' && !!props.schema.attributes.upload_url) return 'file'
  if (!isSearch.value && fmt === 'text') return 'multistr'
  return 'text'
})
</script>
