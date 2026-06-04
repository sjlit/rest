<template>
  <div class="schema-page">
    <div v-if="showHeader" class="schema-page-header">
      <div class="header-left">
        <slot name="headerleft">
          <h3>{{ title }}</h3>
        </slot>
      </div>
      <div class="header-right">
        <slot name="headerright">
          <el-button v-if="!readonly" type="primary" round @click="handleCreate">
            {{ t('action.create') }}
          </el-button>
        </slot>
      </div>
    </div>
    <div class="schema-page-body">
      <div v-if="showSearch" class="schema-page-search">
        <SchemaForm :schemas="searchSchemas" scenario="search" :model="searchModel" :inline="true"
          :actions="searchActionList">
          <template #default="{ model, schema }">
            <slot name="searchform" :model="model" :schema="schema" />
          </template>
        </SchemaForm>
      </div>
      <div v-if="showToolbar" class="schema-page-toolbar">
        <el-dropdown v-if="batchActionList.length > 0" placement="bottom-end">
          <el-icon>
            <More />
          </el-icon>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-for="action in batchActionList" :key="action.name" @click="handleBatchAction(action)">
                {{ action.label }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
      <div class="schema-page-grid" v-loading="loading">
        <SchemaGrid :schemas="listSchemas" :models="models" scenario="list" :actions="rowActionList" v-bind="gridProps"
          @selection="handleSelectionChange" @sort="handleSortChange">
          <template #default="{ model, schema }">
            <slot name="gridview" :model="model" :schema="schema" />
          </template>
        </SchemaGrid>
        <div v-if="showPagination" class="schema-page-pagination">
          <el-pagination :page-size="pagination.size" :total="pagination.totalCount"
            :current-page="pagination.index" layout="total, prev, pager, next" @current-change="handlePageChange" />
        </div>
      </div>
    </div>

    <el-dialog v-if="formMode === 'dialog'" v-model="formVisible" :title="formTitle" :width="formWidth" draggable
      destroy-on-close>
      <SchemaForm v-bind="formProps" :schemas="formSchemas" :scenario="formScenario" :model="formModel"
        :actions="formActionList" :errors="formErrors">
        <template #default="{ model, schema }">
          <slot name="crudform" :model="model" :schema="schema" />
        </template>
      </SchemaForm>
    </el-dialog>
    <el-drawer v-else v-model="formVisible" :title="formTitle" :size="formWidth" destroy-on-close>
      <SchemaForm v-bind="formProps" :schemas="formSchemas" :scenario="formScenario" :model="formModel"
        :actions="formActionList" :errors="formErrors">
        <template #default="{ model, schema }">
          <slot name="crudform" :model="model" :schema="schema" />
        </template>
      </SchemaForm>
    </el-drawer>

    <el-dialog v-if="formMode === 'dialog'" v-model="detailVisible" :title="detailTitle" :width="formWidth" draggable
      destroy-on-close>
      <el-descriptions :column="1" border>
        <el-descriptions-item v-for="schema in detailSchemas" :key="schema.column" :label="schema.label">
          <Cell :model="detailModel" :schema="schema" />
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
    <el-drawer v-else v-model="detailVisible" :title="detailTitle" :size="formWidth" destroy-on-close>
      <el-descriptions :column="1" border>
        <el-descriptions-item v-for="schema in detailSchemas" :key="schema.column" :label="schema.label">
          <Cell :model="detailModel" :schema="schema" />
        </el-descriptions-item>
      </el-descriptions>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, inject } from 'vue'
import { More } from '@element-plus/icons-vue'
import type { Schema, Model, Action as ActionType, Pagination as PaginationType, Scenario } from '../core/types'
import type { SchemaUIConfig } from '../config'
import { filterByScenario } from '../core/form'
import { createDefaultTranslator } from '../core/i18n'
import { GLOBAL_CONFIG_KEY } from '../config'
import SchemaForm from './SchemaForm.vue'
import SchemaGrid from './SchemaGrid.vue'
import Cell from './parts/Cell.vue'
import '../styles/page.scss'

interface Props {
  schemas: Schema[]
  models: Model[]
  pagination?: PaginationType
  loading?: boolean
  size?: string
  title?: string
  formMode?: 'drawer' | 'dialog'
  showHeader?: boolean
  showSearch?: boolean
  showToolbar?: boolean
  showPagination?: boolean
  readonly?: boolean
  searchActions?: ActionType[]
  rowActions?: ActionType[]
  batchActions?: ActionType[]
  formActions?: ActionType[]
  gridProps?: Record<string, any>
  formProps?: Record<string, any>
  presetQuery?: Record<string, any>
  formErrors?: Record<string, string>
}

const props = withDefaults(defineProps<Props>(), {
  pagination: () => ({ index: 1, size: 15, totalCount: 0 }),
  loading: false,
  formMode: 'dialog',
  showHeader: true,
  showSearch: true,
  showToolbar: true,
  showPagination: true,
  readonly: false,
  searchActions: () => [],
  rowActions: () => [],
  batchActions: () => [],
  formActions: () => [],
  gridProps: () => ({}),
  formProps: () => ({}),
  presetQuery: () => ({}),
  formErrors: () => ({}),
})

const emit = defineEmits<{
  search: [model: Model]
  create: []
  edit: [model: Model]
  delete: [model: Model]
  pageChange: [index: number]
  sortChange: [sortable: { column: string; order: 'ascending' | 'descending' | null }]
  selectionChange: [selection: any[]]
  formSubmit: [model: Model, scenario: Scenario]
}>()

const globalConfig = inject<SchemaUIConfig | null>(GLOBAL_CONFIG_KEY, null)

const t = (key: string, ...args: any[]) => {
  const fn = globalConfig?.i18n?.t || createDefaultTranslator()
  return fn(key, ...args)
}

const formVisible = ref(false)
const formScenario = ref<Scenario>('create')
const searchModel = ref<Model>({ ...props.presetQuery })
const formModel = ref<Model>({})
const selections = ref<any[]>([])
const detailVisible = ref(false)
const detailModel = ref<Model>({})
const formSubmitting = ref(false)

watch(() => props.presetQuery, (val) => {
  searchModel.value = { ...val }
}, { deep: true })

const searchSchemas = computed(() =>
  filterByScenario(props.schemas, 'search', { includeInvisible: true })
)
const listSchemas = computed(() =>
  filterByScenario(props.schemas, 'list')
)
const formSchemas = computed(() =>
  filterByScenario(props.schemas, formScenario.value)
)
const detailSchemas = computed(() =>
  filterByScenario(props.schemas, 'detail')
)

const formTitle = computed(() => {
  return formScenario.value === 'create' ? t('form.create') : t('form.edit')
})

const detailTitle = computed(() => t('form.detail'))

const formWidth = computed(() => {
  if (typeof window === 'undefined') return '60%'
  const width = window.innerWidth
  if (width < 768) return '96%'
  if (width <= 1180) return '80%'
  if (width <= 1366) return '60%'
  return '40%'
})

const searchActionList = computed((): ActionType[] => {
  if (props.searchActions.length > 0) return props.searchActions
  return [
    {
      name: 'search',
      label: t('action.search'),
      type: 'primary',
      asyncCallback: async (model) => {
        emit('search', model)
      },
    },
  ]
})

const rowActionList = computed((): ActionType[] => {
  if (props.readonly) return []
  if (props.rowActions.length > 0) return props.rowActions
  return [
    { name: 'edit', label: t('action.edit'), type: 'success', callback: (model) => handleEdit(model) },
    { name: 'delete', label: t('action.delete'), type: 'danger', callback: (model) => emit('delete', model) },
  ]
})

const batchActionList = computed((): ActionType[] => props.batchActions)

// Promise-based submit mechanism: the save action's asyncCallback returns
// a Promise that resolves when the parent (SchemaViewer) completes the API call.
let formSubmitResolve: (() => void) | null = null
let formSubmitReject: ((err: any) => void) | null = null

function resolveFormSubmit() {
  formSubmitResolve?.()
  formSubmitResolve = null
  formSubmitReject = null
}

function rejectFormSubmit(err: any) {
  formSubmitReject?.(err)
  formSubmitResolve = null
  formSubmitReject = null
}

const formActionList = computed((): ActionType[] => {
  if (props.formActions.length > 0) {
    // External formActions are used as-is; the parent component
    // (e.g. SchemaViewer) handles save logic and form closing.
    return props.formActions
  }
  return [
    {
      name: 'save',
      label: t('action.save'),
      type: 'primary',
      asyncCallback: async (model) => {
        formSubmitting.value = true
        emit('formSubmit', model, formScenario.value)
        // Wait for the parent to resolve/reject (via resolveFormSubmit/rejectFormSubmit)
        return new Promise<void>((resolve, reject) => {
          formSubmitResolve = () => {
            formSubmitting.value = false
            resolve()
          }
          formSubmitReject = (err: any) => {
            formSubmitting.value = false
            reject(err)
          }
        })
      },
    },
  ]
})

function handleCreate() {
  formScenario.value = 'create'
  formModel.value = {}
  emit('create')
  formVisible.value = true
}

function handleEdit(model: Model) {
  formScenario.value = 'update'
  formModel.value = typeof structuredClone === 'function'
    ? structuredClone(model)
    : JSON.parse(JSON.stringify(model))
  emit('edit', model)
  formVisible.value = true
}

function handleDetail(model: Model) {
  detailModel.value = typeof structuredClone === 'function'
    ? structuredClone(model)
    : JSON.parse(JSON.stringify(model))
  detailVisible.value = true
}

function closeForm() {
  formVisible.value = false
}

function handleBatchAction(action: ActionType) {
  if (typeof action.callback === 'function') {
    action.callback(selections.value)
  }
}

function handleSelectionChange(selection: any[]) {
  selections.value = selection
  emit('selectionChange', selection)
}

function handleSortChange(e: { column: string; order: 'ascending' | 'descending' | null }) {
  emit('sortChange', e)
}

function handlePageChange(index: number) {
  emit('pageChange', index)
}

defineExpose({
  openEdit: handleEdit,
  openDetail: handleDetail,
  closeForm,
  formSubmitting,
  resolveFormSubmit,
  rejectFormSubmit,
})
</script>
