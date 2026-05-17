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
            创建
          </el-button>
        </slot>
      </div>
    </div>
    <div class="schema-page-body">
      <div v-if="showSearch" class="schema-page-search">
        <SchemaForm
          :schemas="searchSchemas"
          scenario="search"
          :model="searchModel"
          :inline="true"
          :actions="searchActionList"
        >
          <template #default="{ model, schema }">
            <slot name="searchform" :model="model" :schema="schema" />
          </template>
        </SchemaForm>
      </div>
      <div v-if="showToolbar" class="schema-page-toolbar">
        <el-dropdown v-if="batchActionList.length > 0" placement="bottom-end">
          <el-icon><More /></el-icon>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                v-for="action in batchActionList"
                :key="action.name"
                @click="handleBatchAction(action)"
              >
                {{ action.label }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
      <div class="schema-page-grid" v-loading="loading">
        <SchemaGrid
          :schemas="listSchemas"
          :models="models"
          scenario="list"
          :actions="rowActionList"
          v-bind="gridProps"
          @selection="handleSelectionChange"
          @sort="handleSortChange"
        >
          <template #default="{ model, schema }">
            <slot name="gridview" :model="model" :schema="schema" />
          </template>
        </SchemaGrid>
        <el-pagination
          v-if="showPagination"
          :page-size="pagination.size"
          :total="pagination.totalCount"
          :current-page="pagination.index"
          layout="total, prev, pager, next"
          @current-change="handlePageChange"
        />
      </div>
    </div>

    <el-dialog
      v-if="formMode === 'dialog'"
      v-model="formVisible"
      :title="formTitle"
      :width="formWidth"
      draggable
      destroy-on-close
    >
      <SchemaForm
        v-bind="formProps"
        :schemas="formSchemas"
        :scenario="formScenario"
        :model="formModel"
        :actions="formActionList"
      >
        <template #default="{ model, schema }">
          <slot name="crudform" :model="model" :schema="schema" />
        </template>
      </SchemaForm>
    </el-dialog>
    <el-drawer
      v-else
      v-model="formVisible"
      :title="formTitle"
      :size="formWidth"
      destroy-on-close
    >
      <SchemaForm
        v-bind="formProps"
        :schemas="formSchemas"
        :scenario="formScenario"
        :model="formModel"
        :actions="formActionList"
      >
        <template #default="{ model, schema }">
          <slot name="crudform" :model="model" :schema="schema" />
        </template>
      </SchemaForm>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { More } from '@element-plus/icons-vue'
import type { Schema, Model, Action as ActionType, Pagination as PaginationType } from '../core/types'
import SchemaForm from './SchemaForm.vue'
import SchemaGrid from './SchemaGrid.vue'

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
})

const emit = defineEmits<{
  search: [model: Model]
  create: []
  edit: [model: Model]
  delete: [model: Model]
  pageChange: [index: number]
  sortChange: [sortable: { column: string; order: 'ascending' | 'descending' | null }]
  selectionChange: [selection: any[]]
  formSubmit: [model: Model, scenario: string]
}>()

const formVisible = ref(false)
const formScenario = ref('create')
const searchModel = ref<Model>({})
const formModel = ref<Model>({})
const selections = ref<any[]>([])

const searchSchemas = computed(() =>
  props.schemas.filter((s) => s.scenarios?.includes('search'))
)
const listSchemas = computed(() =>
  props.schemas.filter((s) => s.scenarios?.includes('list') && !s.attributes.invisible)
)
const formSchemas = computed(() =>
  props.schemas.filter((s) => s.scenarios?.includes(formScenario.value) && !s.attributes.invisible)
)

const formTitle = computed(() => {
  return formScenario.value === 'create' ? '创建' : '编辑'
})

const formWidth = computed(() => {
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
      label: '搜索',
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
    { name: 'edit', label: '编辑', type: 'success', callback: (model) => handleEdit(model) },
    { name: 'delete', label: '删除', type: 'danger', callback: (model) => emit('delete', model) },
  ]
})

const batchActionList = computed((): ActionType[] => props.batchActions)
const formActionList = computed((): ActionType[] => {
  if (props.formActions.length > 0) return props.formActions
  return [
    {
      name: 'save',
      label: '保存',
      type: 'primary',
      asyncCallback: async (model) => {
        emit('formSubmit', model, formScenario.value)
        formVisible.value = false
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
  formModel.value = { ...model }
  emit('edit', model)
  formVisible.value = true
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
</script>
