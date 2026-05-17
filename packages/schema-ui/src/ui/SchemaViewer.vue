<template>
  <SchemaPage
    :schemas="schemas"
    :models="crud?.getModels() || []"
    :pagination="pagination"
    :loading="searching"
    :size="size"
    :title="title"
    :formMode="formMode"
    :showHeader="showHeader"
    :showSearch="showSearch"
    :showToolbar="showToolbar"
    :showPagination="showPagination"
    :readonly="readonly"
    :searchActions="searchActionList"
    :rowActions="rowActionList"
    :batchActions="batchActionList"
    :formActions="formActionList"
    :gridProps="gridProps"
    :formProps="formProps"
    :presetQuery="presetQuery"
    @search="handleSearch"
    @create="handleCreate"
    @edit="handleEdit"
    @delete="handleDelete"
    @pageChange="handlePageChange"
    @sortChange="handleSortChange"
    @selectionChange="handleSelectionChange"
    @formSubmit="handleFormSubmit"
  >
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

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { Schema, Model, Action as ActionType, CRUDOptions } from '../core/types'
import { CRUD } from '../runtime/crud'
import { useSchemaUI } from '../runtime/useSchemaUI'
import { clearSearchModel } from '../core/form'
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
const crud = ref<CRUD | null>(null)
const schemas = ref<Schema[]>([])
const searching = ref(false)
const isReady = ref(false)
const selections = ref<any[]>([])

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
      label: '搜索',
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
      name: 'edit',
      label: '编辑',
      type: 'success',
      callback: (model) => handleEdit(model),
    },
    {
      name: 'delete',
      label: '删除',
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
    label: '导出',
    callback: () => {
      crud.value!.exportModels().catch((e) => console.error('Export failed:', e))
    },
  })
  return defaults
})

const formActionList = computed((): ActionType[] => {
  if (props.formActions.length > 0) return props.formActions
  return [
    {
      name: 'save',
      label: '保存',
      type: 'primary',
      asyncCallback: async (model) => {
        crud.value!.resetError()
        const isCreate = !model[crud.value!.primaryKey]
        if (isCreate) {
          await crud.value!.createModel(model)
        } else {
          await crud.value!.updateModel(model)
        }
      },
    },
  ]
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

  for (const key in props.presetQuery) {
    instance.addQueryParams(key, props.presetQuery[key])
  }
  // TODO: presetQuery values should also be visible in SchemaPage's search form.
  // SchemaPage currently does not expose a way to pre-populate searchModel.

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

function handleEdit(model: Model) {
  // SchemaPage handles dialog display with the row model
  // Full model fetch would require exposing setFormModel on SchemaPage via template ref
}

function handleDelete(model: Model) {
  crud.value!.deleteModel(model).catch((e) => console.error('Delete failed:', e))
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

function handleSelectionChange(selection: any[]) {
  selections.value = selection
}

async function handleFormSubmit(model: Model, scenario: string) {
  crud.value!.resetError()
  try {
    if (scenario === 'create') {
      await crud.value!.createModel(model)
    } else {
      await crud.value!.updateModel(model)
    }
  } catch (e) {
    console.error('Form submit failed:', e)
    throw e
  }
}
</script>
