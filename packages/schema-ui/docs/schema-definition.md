# Schema 定义详解

## 目录

- [Schema 结构](#schema-结构)
- [字段说明](#字段说明)
- [Type 与 Format 对照表](#type-与-format-对照表)
- [Scenarios 场景系统](#scenarios-场景系统)
- [Rules 验证规则](#rules-验证规则)
- [Attributes 扩展属性](#attributes-扩展属性)
- [Live 动态数据](#live-动态数据)
- [与 REST 后端对接](#与-rest-后端对接)
- [示例 Schema](#示例-schema)

---

## Schema 结构

`Schema` 是后端返回的字段元数据定义，描述了一个数据库字段在 UI 中的全部行为。

```typescript
interface Schema {
  id?: number
  created_at?: number
  updated_at?: number
  tenant_id?: string
  module_name: string      // 所属模块
  table_name: string       // 所属表
  enable: number           // 是否启用
  column: string           // 字段名（数据库列名）
  label: string            // 显示标签（UI 上显示的名称）
  type: string             // 数据类型
  format: string           // 显示格式
  native: number           // 是否原生字段
  primary_key: number      // 是否主键（1 = 主键）
  expression: string       // 表达式（计算字段）
  scenarios: string[]      // 适用场景列表
  rules: SchemaRule        // 验证规则
  attributes: SchemaAttribute  // 扩展属性
  relations: Relation      // 关联关系
  position: number         // 显示顺序
}
```

---

## 字段说明

### column

数据库列名，也是 UI 表单字段的 `name` 和表格列的 `prop`。

### label

在表单标签、表格表头中显示的文本。

### type

数据类型，决定后端存储类型和前端基础验证：

| 值 | 说明 |
|----|------|
| `integer` | 整数 |
| `float` | 浮点数 |
| `boolean` | 布尔值 |
| `string` | 字符串 |

### format

显示格式，决定前端渲染的 UI 控件类型：

| 值 | 说明 | 渲染控件 |
|----|------|----------|
| `text` | 文本（搜索场景单行，表单场景多行） | `el-input` / `el-input type="textarea"` |
| `textarea` | 多行文本 | `el-input type="textarea"` |
| `password` | 密码 | `el-input show-password` |
| `dropdown` | 下拉选择 | `el-select` |
| `multiSelect` | 多选下拉 | `el-select multiple` |
| `cascader` | 级联选择 | `el-cascader` |
| `datetime` | 日期时间 | `el-date-picker type="datetime"` |
| `date` | 日期 | `el-date-picker type="date"` |
| `time` | 时间 | `el-time-select` |
| `timestamp` | 时间戳 | `el-date-picker type="datetime"` |
| `boolean` / `bool` | 布尔值 | `el-switch`（表单）/ `el-tag`（表格） |
| `file` | 文件上传 | `el-upload` |
| `integer` | 整数输入 | `el-input v-model.number` |
| `number` | 数字输入 | `el-input v-model.number` |
| `percentage` | 百分比 | `el-input`（输入）/ 自动格式化显示 |
| `duration` | 持续时间（秒） | `el-input`（输入）/ `HH:mm:ss` 格式化显示 |

### primary_key

主键标识，`1` 表示该字段是主键，`0` 表示非主键。

每个表必须有且只有一个主键字段。主键用于：
- 表格行唯一标识
- 编辑时获取记录详情
- 删除时定位记录
- 更新后刷新列表中的对应行

### scenarios

场景数组，决定字段在哪些场景下显示。常见场景：

| 场景 | 说明 |
|------|------|
| `create` | 创建表单 |
| `update` | 编辑表单 |
| `search` | 搜索表单 |
| `list` | 表格列表 |
| `detail` | 详情页面 |
| `export` | 导出数据 |

一个字段可以同时属于多个场景：

```json
{
  "scenarios": ["create", "update", "search", "list"]
}
```

---

## Type 与 Format 对照表

| Type | Format | 后端存储 | 前端控件 |
|------|--------|----------|----------|
| `string` | `text` | VARCHAR | 文本输入框 |
| `string` | `textarea` | TEXT | 多行文本框 |
| `string` | `password` | VARCHAR | 密码输入框 |
| `string` | `dropdown` | VARCHAR | 下拉选择 |
| `integer` | `integer` | INT | 数字输入框 |
| `float` | `float` | FLOAT | 数字输入框 |
| `boolean` | `boolean` | TINYINT | 开关/标签 |
| `string` | `datetime` | DATETIME | 日期时间选择器 |
| `string` | `date` | DATE | 日期选择器 |
| `string` | `time` | TIME | 时间选择器 |
| `string` | `timestamp` | TIMESTAMP | 日期时间选择器 |
| `string` | `file` | VARCHAR | 文件上传 |
| `float` | `percentage` | DECIMAL | 数字输入框（自动格式化显示） |
| `integer` | `duration` | INT | 数字输入框（自动格式化为 HH:mm:ss） |

---

## Scenarios 场景系统

场景系统是本库的核心机制，决定了字段在何时何地显示。

### 场景过滤逻辑

组件内部通过 `scenarios.includes(scenario)` 过滤字段：

```typescript
// SchemaPage.vue
const searchSchemas = computed(() =>
  props.schemas.filter((s) => s.scenarios?.includes('search'))
)

const listSchemas = computed(() =>
  props.schemas.filter((s) => s.scenarios?.includes('list'))
)

const formSchemas = computed(() =>
  props.schemas.filter((s) => s.scenarios?.includes(formScenario.value))
)
```

### 场景设计建议

**搜索场景（`search`）：**
- 用于筛选的字段
- 通常不需要所有字段，只保留有意义的筛选条件
- 布尔值字段在搜索场景渲染为下拉选择（是/否/全部）

**列表场景（`list`）：**
- 表格中显示的字段
- 不宜过多，保持表格可读性
- 可以排序的字段设置 `attributes.sort = true`

**创建/编辑场景（`create`/`update`）：**
- 表单中需要填写或修改的字段
- 主键通常不包含在 `create` 中（自动生成）
- 创建时间和更新时间通常只读或不显示

---

## Rules 验证规则

### 结构

```typescript
interface SchemaRule {
  min: number           // 最小长度/值
  max: number           // 最大长度/值（> 0 时生效）
  type: string          // 数据类型
  unique: boolean       // 是否唯一
  required: string[]    // 必填场景列表
  regular?: string      // 正则表达式
  safe?: boolean        // 安全字段标记。true 时跳过所有表单校验规则生成
}
```

### 必填验证

`required` 数组指定哪些场景下该字段必填：

```json
{
  "rules": {
    "required": ["create", "update"]
  }
}
```

在 `search` 场景下不生成必填规则。

### 长度验证

当 `type === 'string'` 且 `max > 0` 时，生成最大长度验证：

```json
{
  "type": "string",
  "rules": {
    "max": 50
  }
}
```

### 正则验证

```json
{
  "rules": {
    "regular": "^[a-zA-Z0-9]+$"
  }
}
```

---

## Attributes 扩展属性

### visible / invisible

字段可见性控制。

**完全隐藏：**

```json
{
  "attributes": {
    "invisible": true
  }
}
```

设置 `invisible: true` 的字段在任何场景都不显示。

**条件可见：**

```json
{
  "attributes": {
    "visible": [
      {
        "column": "type",
        "values": ["company"]
      }
    ]
  }
}
```

表示：只有当 `type` 字段的值为 `"company"` 时，该字段才可见。

支持多个条件，所有条件必须同时满足：

```json
{
  "attributes": {
    "visible": [
      { "column": "type", "values": ["company"] },
      { "column": "status", "values": ["active"] }
    ]
  }
}
```

### readonly / disable

指定在哪些场景下字段为只读或禁用：

```json
{
  "attributes": {
    "readonly": ["update"],
    "disable": ["create"]
  }
}
```

- `readonly`：字段可见但不可编辑
- `disable`：字段禁用（通常表现为灰色）

### values

枚举值列表，用于下拉选择、单选、标签显示：

```json
{
  "attributes": {
    "values": [
      { "label": "启用", "value": "active", "color": "#67C23A" },
      { "label": "禁用", "value": "inactive", "color": "#F56C6C" }
    ]
  }
}
```

`color` 用于表格中的标签颜色显示。注意：只有当**所有枚举值都设置了 `color`** 时，才会以 tag 样式渲染；否则按普通文本显示。

### sort

是否支持表格排序：

```json
{
  "attributes": {
    "sort": true
  }
}
```

设置为 `true` 后，表格表头会显示排序图标，点击触发 `sortChange` 事件。

### dropdown

下拉框配置：

```json
{
  "attributes": {
    "dropdown": {
      "created": false,        // 是否允许创建新选项
      "filterable": true,      // 是否可搜索
      "multiple": false,       // 是否多选
      "collapse_tags": false,  // 多选时是否折叠标签
      "default_first": false   // 是否默认选中第一项
    }
  }
}
```

### tooltip

字段提示文本，显示为占位符或提示信息：

```json
{
  "attributes": {
    "tooltip": "请输入真实姓名"
  }
}
```

### upload_url

文件上传地址：

```json
{
  "format": "file",
  "attributes": {
    "upload_url": "/api/upload"
  }
}
```

### icon / suffix

输入框前缀图标和后缀文本：

```json
{
  "attributes": {
    "icon": "Search",
    "suffix": "元"
  }
}
```

---

## Live 动态数据

当字段的下拉选项需要从后端动态加载时使用 `live`。

### 结构

```typescript
interface LiveValue {
  enable: boolean       // 是否启用动态加载
  type: string          // 加载类型：dropdown / cascader
  url?: string          // 数据接口地址
  method?: string       // HTTP 方法：GET / POST
  body?: string         // POST 请求体
  content_type?: string // 内容类型
  columns?: string[]    // 级联关联字段
}
```

### 示例

**GET 方式加载：**

```json
{
  "attributes": {
    "live": {
      "enable": true,
      "type": "dropdown",
      "url": "/rest/user/roles",
      "method": "GET"
    }
  }
}
```

**POST 方式加载：**

```json
{
  "attributes": {
    "live": {
      "enable": true,
      "type": "dropdown",
      "url": "/rest/options/search",
      "method": "POST",
      "body": "{\"type\": \"role\"}",
      "content_type": "application/json"
    }
  }
}
```

**级联加载：**

```json
{
  "attributes": {
    "live": {
      "enable": true,
      "type": "cascader",
      "url": "/rest/regions",
      "method": "GET"
    }
  }
}
```

### 加载时机

`CRUD.initialize()` 会在 schema 加载完成后，自动并行拉取所有启用了 `live` 的字段数据。加载结果会写入对应 schema 的 `attributes.values` 中。

---

## 与 REST 后端对接

### Schema 加载接口

**请求：**

```
GET /{apiPrefix}/schema/{module}/{table}
```

**响应：**

```json
[
  {
    "id": 1,
    "module_name": "user",
    "table_name": "admin",
    "column": "id",
    "label": "ID",
    "type": "integer",
    "format": "integer",
    "primary_key": 1,
    "scenarios": ["list", "detail"],
    "rules": { "min": 0, "max": 0, "type": "integer", "unique": true, "required": [] },
    "attributes": { "match": "exactly", "readonly": [], "disable": [], "visible": [], "invisible": false, "sort": true, "values": [], "live": { "enable": false, "type": "" } }
  }
]
```

### 搜索接口

**请求：**

```
GET /{apiPrefix}/{module}/{pluralTable}?page=1&pagesize=15&sort=-created_at&__format=both
```

**响应：**

```json
{
  "page": "1",
  "page_size": "15",
  "total_count": "100",
  "data": [
    { "id": 1, "username": "admin", "status": "active" }
  ]
}
```

### 创建接口

**请求：**

```
POST /{apiPrefix}/{module}/{singularTable}
```

**请求体：** 模型对象

### 更新接口

**请求：**

```
PUT /{apiPrefix}/{module}/{singularTable}/{id}
```

**请求体：** 模型对象

### 删除接口

**请求：**

```
DELETE /{apiPrefix}/{module}/{singularTable}/{id}
```

### 导出接口

**请求：**

```
GET /{apiPrefix}/{module}/{singularTable}/export
```

**响应：** CSV 文件流

---

## 示例 Schema

### 用户管理表

```json
[
  {
    "column": "id",
    "label": "ID",
    "type": "integer",
    "format": "integer",
    "primary_key": 1,
    "scenarios": ["list", "detail"],
    "rules": { "min": 0, "max": 0, "type": "integer", "unique": true, "required": [] },
    "attributes": { "match": "exactly", "readonly": [], "disable": [], "visible": [], "invisible": false, "sort": false, "values": [], "live": { "enable": false, "type": "" } }
  },
  {
    "column": "username",
    "label": "用户名",
    "type": "string",
    "format": "text",
    "primary_key": 0,
    "scenarios": ["create", "update", "search", "list"],
    "rules": { "min": 3, "max": 50, "type": "string", "unique": true, "required": ["create", "update"], "regular": "^[a-zA-Z0-9_]+$" },
    "attributes": { "match": "fuzzy", "readonly": ["update"], "disable": [], "visible": [], "invisible": false, "sort": true, "values": [], "live": { "enable": false, "type": "" } }
  },
  {
    "column": "email",
    "label": "邮箱",
    "type": "string",
    "format": "text",
    "primary_key": 0,
    "scenarios": ["create", "update", "search", "list"],
    "rules": { "min": 0, "max": 100, "type": "string", "unique": true, "required": ["create"], "regular": "^[\\w.-]+@[\\w.-]+\\.\\w+$" },
    "attributes": { "match": "fuzzy", "readonly": [], "disable": [], "visible": [], "invisible": false, "sort": false, "values": [], "live": { "enable": false, "type": "" } }
  },
  {
    "column": "status",
    "label": "状态",
    "type": "string",
    "format": "dropdown",
    "primary_key": 0,
    "scenarios": ["create", "update", "search", "list"],
    "rules": { "min": 0, "max": 0, "type": "string", "unique": false, "required": ["create", "update"] },
    "attributes": {
      "match": "exactly",
      "readonly": [],
      "disable": [],
      "visible": [],
      "invisible": false,
      "sort": true,
      "values": [
        { "label": "启用", "value": "active", "color": "#67C23A" },
        { "label": "禁用", "value": "inactive", "color": "#F56C6C" }
      ],
      "live": { "enable": false, "type": "" }
    }
  },
  {
    "column": "role_id",
    "label": "角色",
    "type": "integer",
    "format": "dropdown",
    "primary_key": 0,
    "scenarios": ["create", "update", "search", "list"],
    "rules": { "min": 0, "max": 0, "type": "integer", "unique": false, "required": ["create", "update"] },
    "attributes": {
      "match": "exactly",
      "readonly": [],
      "disable": [],
      "visible": [],
      "invisible": false,
      "sort": false,
      "values": [],
      "live": {
        "enable": true,
        "type": "dropdown",
        "url": "/rest/user/roles",
        "method": "GET"
      }
    }
  },
  {
    "column": "created_at",
    "label": "创建时间",
    "type": "string",
    "format": "datetime",
    "primary_key": 0,
    "scenarios": ["list", "detail"],
    "rules": { "min": 0, "max": 0, "type": "string", "unique": false, "required": [] },
    "attributes": { "match": "exactly", "readonly": [], "disable": ["create", "update"], "visible": [], "invisible": false, "sort": true, "values": [], "live": { "enable": false, "type": "" } }
  }
]
```
