// === Plugin ===
export { SchemaUIPlugin } from './plugin'
export type { SchemaUIConfig } from './config'

// === Runtime ===
export { useSchemaUI } from './runtime/useSchemaUI'
export { CRUD } from './runtime/crud'

// === Core Types ===
export type {
  Schema,
  SchemaRule,
  SchemaAttribute,
  EnumValue,
  VisibleCondition,
  LiveValue,
  DropdownOptions,
  Relation,
  Model,
  Action,
  Pagination,
  Sortable,
  CRUDOptions,
} from './core/types'

// === Core Constants ===
export {
  TypeInteger,
  TypeFloat,
  TypeBoolean,
  TypeString,
  FormatInteger,
  FormatFloat,
  FormatBoolean,
  FormatString,
  FormatText,
  FormatDropdown,
  FormatDatetime,
  FormatDate,
  FormatTime,
  FormatTimestamp,
  FormatPassword,
  ScenarioCreate,
  ScenarioUpdate,
  ScenarioDelete,
  ScenarioSearch,
  ScenarioExport,
  ScenarioList,
  ScenarioDetail,
  MatchExactly,
  MatchFuzzy,
  LiveTypeDropdown,
  LiveTypeCascader,
} from './core/constants'

// === Core Utils ===
export { Scenarios } from './core/scenarios'
export { encode, decode } from './core/codec'
export { getModelValue, getModelLabel } from './core/model'
export { generateSchemaRule, checkSchemaVisible, clearSearchModel, filterByScenario } from './core/form'
export { createDefaultTranslator } from './core/i18n'

// === UI Components ===
export { default as SchemaForm } from './ui/SchemaForm.vue'
export { default as SchemaGrid } from './ui/SchemaGrid.vue'
export { default as SchemaPage } from './ui/SchemaPage.vue'
export { default as SchemaViewer } from './ui/SchemaViewer.vue'
