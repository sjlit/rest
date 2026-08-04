<template>
  <template v-if="enableMobileTable">
    <el-collapse expand-icon-position="left">
      <el-collapse-item v-for="(model, idx) in models" :key="idx">
        <template #title>
          <span class="mobile-primary-label">{{ getMobilePrimaryLabel(model) }}</span>
          <span class="mobile-actions" v-if="actions.length > 0">
            <Action v-for="action in actions" :key="action.name" :action="action" :model="model"
              @click="(_, loading) => handleActionClick(action, model, loading)" />
          </span>
        </template>
        <div v-for="schema in visibleSchemas" :key="schema.column" class="mobile-preview-row">
          <div class="mobile-preview-label">{{ schema.label }}</div>
          <div class="mobile-preview-value">
            <slot :model="model" :schema="schema">
              <Cell :model="model" :schema="schema" />
            </slot>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </template>
  <template v-else>
    <el-table :data="models" :size="size" :border="true" :loading="loading" v-bind="gridProps"
      @selection-change="handleSelectionChange" @sort-change="handleSortChange">
      <el-table-column v-if="selection" type="selection" width="55" />
      <el-table-column v-for="schema in visibleSchemas" :key="schema.column" :prop="schema.column" :label="schema.label"
        :sortable="schema.attributes.sort ? 'custom' : false" show-overflow-tooltip>
        <template #default="scope">
          <template v-if="scope">
            <slot :model="scope.row" :schema="schema">
              <Cell :model="scope.row" :schema="schema" />
            </slot>
          </template>
        </template>
      </el-table-column>
      <el-table-column v-if="actions.length > 0" fixed="right" class-name="schema-grid-actions">
        <template #default="scope">
          <template v-if="scope">
            <Action v-for="action in actions" :key="action.name" :action="action" :model="scope.row"
              @click="(_, loading) => handleActionClick(action, scope.row, loading)" />
          </template>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty />
      </template>
    </el-table>
  </template>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import type { Schema, Model, Action as ActionType, Scenario } from '../core/types'
import { filterByScenario } from '../core/form'
import Cell from './parts/Cell.vue'
import Action from './parts/Action.vue'
import '../styles/grid.scss'

interface Props {
  size?: string
  schemas: Schema[]
  scenario?: Scenario
  selection?: boolean
  models: Model[]
  actions?: ActionType[]
  gridProps?: Record<string, any>
  responsive?: boolean
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  scenario: 'list',
  selection: true,
  actions: () => [],
  gridProps: () => ({}),
  responsive: true,
  loading: false,
})

const emit = defineEmits<{
  selection: [selection: any[]]
  sort: [sortable: { column: string; order: 'ascending' | 'descending' | null }]
}>()

const isMobileView = ref(false)

let mql: MediaQueryList | null = null
let mediaListener: ((e: MediaQueryListEvent) => void) | null = null

onMounted(() => {
  mql = window.matchMedia('(max-width: 768px)')
  isMobileView.value = mql.matches
  mediaListener = (e) => { isMobileView.value = e.matches }
  mql.addEventListener?.('change', mediaListener)
})

onUnmounted(() => {
  if (mql && mediaListener) {
    mql.removeEventListener?.('change', mediaListener)
  }
})

const enableMobileTable = computed(() => {
  if (!props.responsive) return false
  return isMobileView.value
})

const visibleSchemas = computed(() => filterByScenario(props.schemas, props.scenario))

function getMobilePrimaryLabel(model: Model): string {
  const pkSchema = props.schemas.find((s) => s.primary_key === 1)
  if (pkSchema) {
    const val = model[pkSchema.column]
    return val !== undefined && val !== null ? String(val) : ''
  }
  const firstKey = Object.keys(model)[0]
  return firstKey !== undefined ? String(model[firstKey]) : ''
}

function handleActionClick(action: ActionType, model: Model, loading: any) {
  if (typeof action.callback === 'function') {
    action.callback(model, props.schemas)
  } else if (typeof action.asyncCallback === 'function') {
    loading.value = true
    action.asyncCallback(model, props.schemas).finally(() => {
      loading.value = false
    })
  }
}

function handleSelectionChange(selection: any[]) {
  emit('selection', selection)
}

function handleSortChange(e: { prop: string; order: 'ascending' | 'descending' | null }) {
  emit('sort', { column: e.prop, order: e.order })
}
</script>
