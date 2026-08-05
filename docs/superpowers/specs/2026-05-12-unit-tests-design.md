# 单元测试完善设计文档

## 背景

当前项目 `git.nobla.cn/golang/rest/v3` 是一个基于 GORM 的 Go REST 框架/库，除 `query` 包外，其余包均缺失单元测试。本设计聚焦**易于纯单元测试覆盖**的部分，不涉及需要深度依赖 GORM 数据库操作的复杂场景。

## 范围

### 包含（纯单元测试，零数据库依赖）

| 包 | 源文件 | 测试文件 | 测试目标 |
|---|---|---|---|
| `formats` | `builtin.go` | `builtin_test.go` | 所有 format 函数 |
| `internal/inflector` | `inflector.go` | `inflector_test.go` | 单复数、命名转换函数 |
| `schema` | `migrate.go` | `migrate_test.go` | 所有 `parseField*` 解析函数 |
| `schema` | `scenarios.go` | `scenarios_test.go` | `Scenarios.Has`、`Scan`、`Value` |
| `schema` | `rule.go` | `rule_test.go` | `Rule.Scan`、`Rule.Value` |
| `schema` | `attribute.go` | `attribute_test.go` | `Attribute.Scan`、`Attribute.Value` |

### 不包含（本次迭代）

- `rest` 包的 `TypedModel[T]` CRUD 方法（深度依赖 `*gorm.DB` 链式调用）
- `schema` 包的 `GetSchemas`、`GetVisibleSchemas`、`AutoMigrate`（需要真实数据库连接）
- `plugins` 包（当前为空）

## 测试策略

统一使用 Go 标准表格驱动测试，格式如下：

```go
func TestXxx(t *testing.T) {
    tests := []struct {
        name     string
        input    any
        want     any
        wantErr  bool
    }{
        // cases...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := fn(tt.input)
            if got != tt.want {
                t.Errorf("fn() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

- 零外部测试库依赖
- `schema/migrate_test.go` 为**同包测试**（`package schema`），以便访问非导出的 `parseField*` 函数
- 其余测试文件为外部测试包或同包测试均可

## 各包测试详请

### 1. formats 包（`formats/builtin_test.go`）

测试 `builtin.go` 中所有 format 函数：

- **`stringFormat`**：int、float、bool、nil、struct 输入的字符串化结果
- **`integerFormat`**：float64/int/uint/string/[]byte 的转换，不可转换值的回退
- **`decimalFormat`**：各类型到 float64 的转换及精度保持
- **`dateFormat`**：`time.Time`、`sql.NullTime`、int64 时间戳的 `"2006-01-02"` 格式化，不可转换值返回 `""`
- **`timeFormat`**：同上，格式为 `"15:04:05"`
- **`datetimeFormat`**：同上，格式为 `"2006-01-02 15:04:05"`
- **`percentageFormat`**：值 <=1 时乘 100 加 `%`，>1 时直接加 `%`，不可转换值返回 `"0.00%"`
- **`durationFormat`**：0 秒、<60 秒、跨小时、跨天的 `"HH:MM:SS"` 格式化
- **`dropdownFormat`**：枚举值匹配返回 label，未匹配返回原值，nil values 处理

### 2. internal/inflector 包（`internal/inflector/inflector_test.go`）

测试导出函数：

- **`Pluralize`**：规则复数（`user` -> `users`）、不规则（`child` -> `children`, `person` -> `people`）、不变词（`fish` -> `fish`, `news` -> `news`）
- **`Singularize`**：规则（`users` -> `user`）、不规则（`children` -> `child`）、不变词
- **`Camelize`**：`send_email` -> `SendEmail`、`a_b_c` -> `ABC`、空字符串、无下划线
- **`Camel2id`**：`SendEmail` -> `send_email`、`UserID` -> `user_id`、纯小写、全大写缩写
- **`Camel2words`**：`send_email` -> `Send Email`、空字符串

### 3. schema 包解析函数（`schema/migrate_test.go`）

同包测试，直接调用非导出函数：

- **`parseFieldName`**：`user_name` -> `User Name`、`age` -> `Age`、多个下划线
- **`parseFieldNative`**：字段 tag 含 `virtual` -> `0`，不含 -> `1`
- **`parseFieldType`**：bool -> `boolean`、int 系列 -> `integer`、float 系列 -> `float`、默认 -> `string`、tag 覆盖 `type:"xxx"`
- **`parseFieldFormat`**：
  - 各 reflect.Kind 到 format 的映射
  - `enum` tag 触发 `dropdown`
  - `CreatedAt`/`UpdatedAt` 触发 `timestamp`/`datetime`
  - 含 `pass` 的字段名触发 `password`
  - `time.Time` 触发 `datetime`
  - `field.Size >= 1024` 触发 `text`
- **`parseFieldRules`**：
  - `rule:"required"` -> Required 含所有场景
  - `rule:"required:create"` -> Required 仅含 create
  - `rule:"unique"` -> Unique = true
  - `rule:"safe"` -> Safe = true
  - `rule:"regexp:^[a-z]+$"` -> Regular 设置
  - PrimaryKey 字段 -> Unique = true
- **`parseFieldScenario`**：
  - 无 tag 时：PrimaryKey -> `list,detail,export`
  - `CreatedAt`/`UpdatedAt` -> `list`
  - `DeletedAt`/`Namespace` -> `[]`
  - index < 10 的常规字段 -> 全场景
  - index >= 10 的常规字段 -> 部分场景
  - `scenarios:"create;update"` tag 覆盖
- **`parseFieldAttributes`**：
  - `props:"icon:xxx"` -> Icon 设置
  - `props:"match:exactly"` -> Match 设置
  - `props:"end_of_now:true"` -> EndOfNow = true
  - `live:"method:POST;url:/api"` -> Live 结构填充
  - `dropdown:"created;filterable"` -> DropdownOptions 设置
  - `condition:"status:1,2"` -> Visible 条件
  - `enum:"1:一#red;2:二"` -> Values 枚举
- **`parseFieldPosition`**：
  - `position:"50"` tag -> 50
  - 无 tag -> index + 100

### 4. schema 包小函数

**`scenarios_test.go`**：
- `Scenarios.Has`：包含/不包含场景
- `Scenarios.Scan` / `Scenarios.Value`：JSON 往返、nil 输入、string 输入、[]byte 输入、非法类型返回 `ErrUnsupportType`

**`rule_test.go`**：
- `Rule.Scan` / `Rule.Value`：JSON 往返、nil 输入、空 JSON、非法类型

**`attribute_test.go`**：
- `Attribute.Scan` / `Attribute.Value`：JSON 往返、nil 输入、空 JSON、非法类型

## 测试执行

```bash
go test ./formats/... ./internal/inflector/... ./schema/...
```

## 排除项说明

以下部分本次不覆盖，原因均为**深度依赖 `*gorm.DB` 链式调用或数据库连接**：

- `rest` 包：`NewModel`、`Create`、`Update`、`Delete`、`Detail`、`Search` 等方法
- `schema` 包：`GetSchemas`、`GetVisibleSchemas`、`AutoMigrate`
- `query` 包：已有 `query_test.go`（集成测试风格）
- `plugins` 包：当前为空，无测试内容
