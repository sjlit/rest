<template>
  <template v-if="isReady">
    <SchemaPage ref="schemaPageRef" :schemas="schemas" :models="crud?.getModels() || []" :pagination="pagination"
      :loading="searching" :size="size" :title="title" :formMode="formMode" :showHeader="showHeader"
      :showSearch="showSearch" :showToolbar="showToolbar" :showPagination="showPagination" :readonly="readonly"
      :searchActions="searchActionList" :rowActions="rowActionList" :batchActions="batchActionList"
      :formActions="formActionList" :gridProps="gridProps" :formProps="formProps" :presetQuery="presetQuery"
      :formErrors="crud?.getFieldErrors() || {}"
      @search="handleSearch" @create="handleCreate" @delete="handleDelete" @pageChange="handlePageChange"
      @sortChange="handleSortChange" @formSubmit="handleFormSubmit">
      <template #searchform="{ model, schema }">
        <slot name="searchform" :model="model" :schema="schema" />
      </template>
      <template #gridview="{ model, schema }">
        <slot name="gridview" :model="model" :schema="schema" />
      </template>
      <template #crudform="{ model, schema }">
        <slot name="crudform" :model="model" :schema="schema" />
      </template>
      <template #headerleft>
        <slot name="headerleft" />
      </template>
      <template #headerright>
        <slot name="headerright" />
      </template>
    </SchemaPage>
  </template>
  <template v-else></template>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import type { Schema, Model, Action as ActionType, CRUDOptions, Scenario } from '../core/types'
import { CRUD } from '../runtime/crud'
import { useSchemaUI } from '../runtime/useSchemaUI'
import { clearSearchModel } from '../core/form'
import { createDefaultTranslator } from '../core/i18n'
import { Delete, EditPen, View } from '@element-plus/icons-vue'
import SchemaPage from './SchemaPage.vue'

interface Props {
  module: string
  table: string
  title?: string
  apiPrefix?: string
  config?: Partial<CRUDOptions>
  size?: string
  formMode?: 'drawer' | 'dialog'
  showHeader?: boolean
  showSearch?: boolean
  showToolbar?: boolean
  showPagination?: boolean
  readonly?: boolean
  autoFetch?: boolean
  rowActions?: ActionType[]
  batchActions?: ActionType[]
  formActions?: ActionType[]
  searchActions?: ActionType[]
  defaultSort?: string
  presetQuery?: Record<string, any>
  gridProps?: Record<string, any>
  formProps?: Record<string, any>
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
  apiPrefix: '',
  config: () => ({}),
  formMode: 'dialog',
  showHeader: true,
  showSearch: true,
  showToolbar: true,
  showPagination: true,
  readonly: false,
  autoFetch: true,
  rowActions: () => [],
  batchActions: () => [],
  formActions: () => [],
  searchActions: () => [],
  defaultSort: '',
  presetQuery: () => ({}),
  gridProps: () => ({}),
  formProps: () => ({}),
})

const emit = defineEmits<{
  ready: [crud: CRUD]
}>()

const globalConfig = useSchemaUI()
const t = (key: string, ...args: any[]) => {
  const fn = globalConfig?.i18n?.t || createDefaultTranslator()
  return fn(key, ...args)
}
const crud = ref<CRUD | null>(null)
const schemas = ref<Schema[]>([])
const searching = ref(false)
const isReady = ref(false)
const schemaPageRef = ref<any>(null)

const pagination = computed(() => {
  if (!crud.value) return { index: 1, size: 15, totalCount: 0 }
  return {
    index: crud.value.getPaginationIndex(),
    size: crud.value.getPaginationSize(),
    totalCount: crud.value.getPaginationCount(),
  }
})

const searchActionList = computed((): ActionType[] => {
  if (props.searchActions.length > 0) return props.searchActions
  return [
    {
      name: 'search',
      label: t('action.search'),
      type: 'primary',
      asyncCallback: async (model, schemas) => {
        searching.value = true
        try {
          crud.value!.resetPagination().setQueryParams(clearSearchModel(model, schemas || []))
          await crud.value!.searchModel()
        } finally {
          searching.value = false
        }
      },
    },
  ]
})

const rowActionList = computed((): ActionType[] => {
  if (props.readonly) return []
  if (props.rowActions.length > 0) return props.rowActions
  return [
    {
      name: 'view',
      label: t('action.view'),
      icon: View,
      type: 'info',
      asyncCallback: async (model) => handleView(model),
    },
    {
      name: 'edit',
      label: t('action.edit'),
      icon: EditPen,
      type: 'success',
      asyncCallback: async (model) => handleEdit(model),
    },
    {
      name: 'delete',
      label: t('action.delete'),
      icon: Delete,
      type: 'danger',
      callback: (model) => handleDelete(model),
    },
  ]
})

const batchActionList = computed((): ActionType[] => {
  if (props.batchActions.length > 0) return props.batchActions
  const defaults: ActionType[] = []
  defaults.push({
    name: 'export',
    label: t('action.export'),
    callback: () => {
      crud.value!.exportModels().catch((e) => console.error('Export failed:', e))
    },
  })
  return defaults
})

const formActionList = computed((): ActionType[] => {
  // Pass through user-provided formActions only.
  // When empty, SchemaPage uses its built-in save action which emits
  // 'formSubmit' — handled by handleFormSubmit below.
  return props.formActions
})

async function init() {
  const instance = new CRUD({
    module: props.module,
    table: props.table,
    apiPrefix: props.apiPrefix || globalConfig.apiPrefix || 'rest',
    httpClient: globalConfig.httpClient,
    schemas: props.config?.schemas,
  })

  const loadedSchemas = await instance.initialize()
  schemas.value = loadedSchemas
  crud.value = instance

  // Use fixedQuery for presetQuery so they survive setQueryParams during search
  if (Object.keys(props.presetQuery).length > 0) {
    instance.setFixedQuery(props.presetQuery)
  }

  if (props.defaultSort) {
    if (props.defaultSort.startsWith('-')) {
      instance.setSortable(props.defaultSort.slice(1), 'descending')
    } else {
      instance.setSortable(props.defaultSort, 'ascending')
    }
  }

  isReady.value = true
  emit('ready', instance)

  if (props.autoFetch) {
    searching.value = true
    try {
      await instance.searchModel()
    } finally {
      searching.value = false
    }
  }
}

init().catch((err) => {
  console.error('[SchemaViewer] init failed:', err)
})

watch(
  () => [props.module, props.table],
  () => {
    isReady.value = false
    crud.value = null
    schemas.value = []
    init().catch((err) => {
      console.error('[SchemaViewer] init failed:', err)
    })
  }
)

function handleSearch(model: Model) {
  searching.value = true
  crud
    .value!.resetPagination()
    .setQueryParams(clearSearchModel(model, schemas.value))
    .searchModel()
    .finally(() => {
      searching.value = false
    })
}

function handleCreate() {
  // SchemaPage handles dialog display
}

async function handleView(model: Model) {
  try {
    const pk = crud.value!.findModelPrimaryKey(model)
    if (pk !== undefined && pk !== null && pk !== '') {
      const detail = await crud.value!.getModel(String(pk))
      schemaPageRef.value?.openDetail(detail)
    } else {
      schemaPageRef.value?.openDetail(model)
    }
  } catch (e) {
    console.error('Failed to fetch detail:', e)
    schemaPageRef.value?.openDetail(model)
  }
}

async function handleEdit(model: Model) {
  try {
    const pk = crud.value!.findModelPrimaryKey(model)
    if (pk !== undefined && pk !== null && pk !== '') {
      const detail = await crud.value!.getModel(String(pk))
      schemaPageRef.value?.openEdit(detail)
    } else {
      schemaPageRef.value?.openEdit(model)
    }
  } catch (e) {
    console.error('Failed to fetch detail:', e)
    schemaPageRef.value?.openEdit(model)
  }
}

function handleDelete(model: Model) {
  ElMessageBox.confirm('确认删除该数据?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(() => {
      crud.value!.deleteModel(model).catch((e) => console.error('Delete failed:', e))
    })
    .catch(() => { })
}

function handlePageChange(index: number) {
  searching.value = true
  crud
    .value!.setPaginationIndex(index)
    .searchModel()
    .finally(() => {
      searching.value = false
    })
}

function handleSortChange(e: { column: string; order: 'ascending' | 'descending' | null }) {
  if (!e.order) {
    crud.value!.setSortable('', 'ascending')
  } else {
    crud.value!.setSortable(e.column, e.order)
  }
  searching.value = true
  crud
    .value!.searchModel()
    .finally(() => {
      searching.value = false
    })
}

async function handleFormSubmit(model: Model, scenario: Scenario) {
  crud.value!.resetError()
  try {
    if (scenario === 'create') {
      await crud.value!.createModel(model)
    } else {
      await crud.value!.updateModel(model)
    }
    schemaPageRef.value?.closeForm()
    schemaPageRef.value?.resolveFormSubmit()
  } catch (e: any) {
    console.error('Form submit failed:', e)
    // Surface error to the form via fieldErrors
    const message = e?.message || e?.response?.data?.message || '保存失败'
    crud.value!.setColumnError('*', message)
    schemaPageRef.value?.rejectFormSubmit(e)
  }
}
</script>
