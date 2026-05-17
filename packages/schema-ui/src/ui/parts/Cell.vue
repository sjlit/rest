<template>
  <template v-if="tagVisible">
    <span class="schema-cell-tag" :style="{ color: textColor, backgroundColor: bgColor }">
      {{ displayValue }}
    </span>
  </template>
  <template v-else-if="schema.format === 'boolean' || schema.format === 'bool'">
    <el-tag round :type="isTrue ? 'success' : 'danger'">
      {{ isTrue ? '是' : '否' }}
    </el-tag>
  </template>
  <template v-else>
    <span class="schema-cell">{{ displayValue }}</span>
  </template>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Model, Schema } from '../../core/types'

interface Props {
  model: Model
  schema: Schema
}

const props = defineProps<Props>()

const rawValue = computed(() => {
  const val = props.model[props.schema.column]
  if (val === null || val === undefined) return ''
  if (typeof val === 'object') {
    return val.label || val.value || ''
  }
  return val
})

const displayValue = computed(() => {
  const values = props.schema.attributes.values
  if (!Array.isArray(values) || values.length === 0) {
    return String(rawValue.value)
  }
  const found = values.find((v) => v.value === String(rawValue.value))
  return found?.label || String(rawValue.value)
})

const isTrue = computed(() => {
  const val = props.model[props.schema.column]
  return !!val
})

const tagVisible = computed(() => {
  if (props.schema.format !== 'dropdown') return false
  const values = props.schema.attributes.values
  if (!Array.isArray(values)) return false
  return values.length > 0 && values.every((v) => !!v.color)
})

const textColor = computed(() => {
  const val = rawValue.value
  const found = props.schema.attributes.values?.find((v) => v.value === String(val))
  return found?.color || ''
})

const bgColor = computed(() => {
  const val = rawValue.value
  const found = props.schema.attributes.values?.find((v) => v.value === String(val))
  if (!found?.color) return ''
  return found.color + '1A'
})
</script>
