# 单元测试完善 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `formats`、`internal/inflector`、`schema` 包补充纯单元测试，覆盖所有不依赖数据库的函数。

**Architecture:** 每个被测包创建一个 `_test.go` 文件，使用标准 Go 表格驱动测试。`schema/migrate_test.go` 为同包测试以访问非导出函数，其余为外部测试包。`schema` 解析测试使用 `gorm.io/gorm/schema.Parse` 从结构体生成 `*schema.Field`。

**Tech Stack:** Go 1.25, GORM v2, `testing` 标准库

---

## 文件结构

| 文件 | 操作 | 说明 |
|---|---|---|
| `formats/builtin_test.go` | 创建 | 测试 `stringFormat` ~ `dropdownFormat` 9 个函数 |
| `internal/inflector/inflector_test.go` | 创建 | 测试 `Pluralize`, `Singularize`, `Camelize`, `Camel2id`, `Camel2words` |
| `schema/rule_test.go` | 创建 | 测试 `Rule.Scan` / `Rule.Value` |
| `schema/attribute_test.go` | 创建 | 测试 `Attribute.Scan` / `Attribute.Value` |
| `schema/scenarios_test.go` | 创建 | 测试 `Scenarios.Has`, `Scenarios.Scan`, `Scenarios.Value` |
| `schema/migrate_test.go` | 创建 | 同包测试 `parseFieldName` ~ `parseFieldPosition` 8 个函数 |

---

### Task 1: `formats/builtin_test.go` — 基础 format 函数

**Files:**
- Create: `formats/builtin_test.go`

- [ ] **Step 1: 编写 `stringFormat` 和 `integerFormat` 测试**

```go
package formats

import (
	"context"
	"testing"

	"github.com/sjlit/rest/v3/schema"
)

func TestStringFormat(t *testing.T) {
	ctx := context.Background()
	scm := schema.Schema{}
	tests := []struct {
		name  string
		value any
		want  any
	}{
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
		{"nil", nil, "<nil>"},
		{"string", "hello", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringFormat(ctx, tt.value, nil, scm)
			if got != tt.want {
				t.Errorf("stringFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntegerFormat(t *testing.T) {
	ctx := context.Background()
	scm := schema.Schema{}
	tests := []struct {
		name  string
		value any
		want  any
	}{
		{"float64", 3.14, 3},
		{"int", 42, 42},
		{"uint", uint(10), 10},
		{"string", "99", 99},
		{"bytes", []byte("77"), 77},
		{"invalid", "abc", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := integerFormat(ctx, tt.value, nil, scm)
			if got != tt.want {
				t.Errorf("integerFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: 编写 `decimalFormat` 和 `dateFormat` 测试**

在 `formats/builtin_test.go` 中追加：

```go
func TestDecimalFormat(t *testing.T) {
	ctx := context.Background()
	scm := schema.Schema{}
	tests := []struct {
		name  string
		value any
		want  any
	}{
		{"float64", 3.14, 3.14},
		{"int", 42, 42.0},
		{"string", "2.5", 2.5},
		{"bytes", []byte("1.5"), 1.5},
		{"invalid", "abc", 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decimalFormat(ctx, tt.value, nil, scm)
			if got != tt.want {
				t.Errorf("decimalFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDateFormat(t *testing.T) {
	ctx := context.Background()
	scm := schema.Schema{}
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name  string
		value any
		want  any
	}{
		{"time.Time", tm, "2024-01-15"},
		{"int64", tm.Unix(), "2024-01-15"},
		{"invalid", "hello", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dateFormat(ctx, tt.value, nil, scm)
			if got != tt.want {
				t.Errorf("dateFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

注意需要在文件顶部追加 `time` 和 `database/sql` 的 import（Step 5 中完整 import 已处理）。

- [ ] **Step 3: 编写 `timeFormat`、`datetimeFormat`、`percentageFormat`、`durationFormat`、`dropdownFormat` 测试**

在 `formats/builtin_test.go` 中追加：

```go
func TestTimeFormat(t *testing.T) {
	ctx := context.Background()
	scm := schema.Schema{}
	tm := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name  string
		value any
		want  any
	}{
		{"time.Time", tm, "10:30:45"},
		{"int64", tm.Unix(), "10:30:45"},
		{"invalid", "hello", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeFormat(ctx, tt.value, nil, scm)
			if got != tt.want {
				t.Errorf("timeFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDatetimeFormat(t *testing.T) {
	ctx := context.Background()
	scm := schema.Schema{}
	tm := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name  string
		value any
		want  any
	}{
		{"time.Time", tm, "2024-01-15 10:30:45"},
		{"int64", tm.Unix(), "2024-01-15 10:30:45"},
		{"invalid", "hello", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := datetimeFormat(ctx, tt.value, nil, scm)
			if got != tt.want {
				t.Errorf("datetimeFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPercentageFormat(t *testing.T) {
	ctx := context.Background()
	scm := schema.Schema{}
	tests := []struct {
		name  string
		value any
		want  any
	}{
		{"less_than_one", 0.15, "15.00%"},
		{"greater_than_one", 15.0, "15.00%"},
		{"invalid", "abc", "0.00%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentageFormat(ctx, tt.value, nil, scm)
			if got != tt.want {
				t.Errorf("percentageFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDurationFormat(t *testing.T) {
	ctx := context.Background()
	scm := schema.Schema{}
	tests := []struct {
		name  string
		value any
		want  any
	}{
		{"zero", 0, "00:00:00"},
		{"seconds_only", 45, "00:00:45"},
		{"minutes", 125, "00:02:05"},
		{"hours", 3665, "01:01:05"},
		{"days", 90061, "25:01:01"},
		{"invalid", "abc", "00:00:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := durationFormat(ctx, tt.value, nil, scm)
			if got != tt.want {
				t.Errorf("durationFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDropdownFormat(t *testing.T) {
	ctx := context.Background()
	scm := schema.Schema{
		Attributes: schema.Attribute{
			Values: []schema.EnumValue{
				{Value: "1", Label: "启用"},
				{Value: "2", Label: "禁用"},
			},
		},
	}
	tests := []struct {
		name  string
		value any
		want  any
	}{
		{"matched", "1", "启用"},
		{"matched_2", "2", "禁用"},
		{"not_matched", "3", "3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dropdownFormat(ctx, tt.value, nil, scm)
			if got != tt.want {
				t.Errorf("dropdownFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 4: 确保完整 import 正确**

`formats/builtin_test.go` 完整 imports 应为：

```go
import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/sjlit/rest/v3/schema"
)
```

`database/sql` 是因为 `dateFormat`/`timeFormat`/`datetimeFormat` 支持 `*sql.NullTime` 类型，虽然当前测试用例没用到，但为完整性保留 import。

- [ ] **Step 5: 运行测试**

Run:
```bash
cd /root/workspaces/go/rest && go test ./formats/... -v
```

Expected: 全部 PASS

- [ ] **Step 6: Commit**

```bash
git add formats/builtin_test.go
git commit -m "test(formats): add unit tests for all builtin format functions"
```

---

### Task 2: `internal/inflector/inflector_test.go` — 单复数与命名转换

**Files:**
- Create: `internal/inflector/inflector_test.go`

- [ ] **Step 1: 编写 `Pluralize` 和 `Singularize` 测试**

```go
package inflector

import "testing"

func TestPluralize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user", "users"},
		{"child", "children"},
		{"person", "people"},
		{"ox", "oxen"},
		{"fish", "fish"},
		{"news", "news"},
		{"status", "statuses"},
		{"quiz", "quizzes"},
		{"mouse", "mice"},
		{"vertex", "vertices"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Pluralize(tt.input)
			if got != tt.want {
				t.Errorf("Pluralize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", "user"},
		{"children", "child"},
		{"people", "person"},
		{"oxen", "ox"},
		{"fish", "fish"},
		{"news", "news"},
		{"statuses", "status"},
		{"quizzes", "quiz"},
		{"mice", "mouse"},
		{"vertices", "vertex"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Singularize(tt.input)
			if got != tt.want {
				t.Errorf("Singularize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: 编写 `Camelize`、`Camel2id`、`Camel2words` 测试**

在文件中追加：

```go
func TestCamelize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"send_email", "SendEmail"},
		{"a_b_c", "ABC"},
		{"user", "User"},
		{"", ""},
		{"alreadyCamel", "Alreadycamel"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Camelize(tt.input)
			if got != tt.want {
				t.Errorf("Camelize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCamel2id(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SendEmail", "send_email"},
		{"UserID", "user_id"},
		{"HTTPSConnection", "https_connection"},
		{"already_lower", "already_lower"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Camel2id(tt.input)
			if got != tt.want {
				t.Errorf("Camel2id(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCamel2words(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"send_email", "Send Email"},
		{"user", "User"},
		{"a_b_c", "A B C"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Camel2words(tt.input)
			if got != tt.want {
				t.Errorf("Camel2words(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 3: 运行测试**

Run:
```bash
cd /root/workspaces/go/rest && go test ./internal/inflector/... -v
```

Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add internal/inflector/inflector_test.go
git commit -m "test(inflector): add unit tests for pluralize, singularize, and naming converters"
```

---

### Task 3: `schema/rule_test.go` — Rule 的 Scan/Value

**Files:**
- Create: `schema/rule_test.go`

- [ ] **Step 1: 编写 `Rule.Scan` 和 `Rule.Value` 测试**

```go
package schema

import (
	"encoding/json"
	"testing"
)

func TestRuleScanValueRoundTrip(t *testing.T) {
	original := Rule{
		Min:      1,
		Max:      100,
		Type:     "string",
		Unique:   true,
		Required: []string{"create"},
		Regular:  "^[a-z]+$",
	}

	v, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}

	var scanned Rule
	if err := scanned.Scan(v); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if scanned.Min != original.Min || scanned.Max != original.Max || scanned.Type != original.Type {
		t.Errorf("round-trip failed: got %+v, want %+v", scanned, original)
	}
	if !scanned.Unique || len(scanned.Required) != 1 || scanned.Required[0] != "create" {
		t.Errorf("round-trip failed for complex fields: got %+v", scanned)
	}
}

func TestRuleScanNil(t *testing.T) {
	var r Rule
	if err := r.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) error: %v", err)
	}
	if r.Min != 0 || r.Type != "" || r.Unique {
		t.Error("expected zero value after nil scan")
	}
}

func TestRuleScanString(t *testing.T) {
	var r Rule
	data := `{"min":5,"max":50,"type":"int"}`
	if err := r.Scan(data); err != nil {
		t.Fatalf("Scan(string) error: %v", err)
	}
	if r.Min != 5 || r.Max != 50 || r.Type != "int" {
		t.Errorf("unexpected result: %+v", r)
	}
}

func TestRuleScanBytes(t *testing.T) {
	var r Rule
	data := []byte(`{"unique":true}`)
	if err := r.Scan(data); err != nil {
		t.Fatalf("Scan([]byte) error: %v", err)
	}
	if !r.Unique {
		t.Error("expected Unique=true")
	}
}

func TestRuleScanUnsupportedType(t *testing.T) {
	var r Rule
	if err := r.Scan(123); err != ErrUnsupportType {
		t.Errorf("expected ErrUnsupportType, got %v", err)
	}
}
```

- [ ] **Step 2: 运行测试**

Run:
```bash
cd /root/workspaces/go/rest && go test ./schema/... -run TestRule -v
```

Expected: 全部 PASS

- [ ] **Step 3: Commit**

```bash
git add schema/rule_test.go
git commit -m "test(schema): add unit tests for Rule Scan/Value"
```

---

### Task 4: `schema/attribute_test.go` — Attribute 的 Scan/Value

**Files:**
- Create: `schema/attribute_test.go`

- [ ] **Step 1: 编写 `Attribute.Scan` 和 `Attribute.Value` 测试**

```go
package schema

import (
	"testing"
)

func TestAttributeScanValueRoundTrip(t *testing.T) {
	original := Attribute{
		Tag:          "input",
		DefaultValue: "default",
		Readonly:     []string{"create"},
		Invisible:    true,
		Sort:         true,
		Values: []EnumValue{
			{Value: "1", Label: "一", Color: "#red"},
		},
	}

	v, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}

	var scanned Attribute
	if err := scanned.Scan(v); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if scanned.Tag != original.Tag || scanned.DefaultValue != original.DefaultValue {
		t.Errorf("round-trip failed: got %+v, want %+v", scanned, original)
	}
	if !scanned.Invisible || !scanned.Sort {
		t.Error("expected boolean fields to survive round-trip")
	}
	if len(scanned.Values) != 1 || scanned.Values[0].Label != "一" {
		t.Errorf("unexpected values: %+v", scanned.Values)
	}
}

func TestAttributeScanNil(t *testing.T) {
	var a Attribute
	if err := a.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) error: %v", err)
	}
	if a.Tag != "" || a.DefaultValue != "" || a.Invisible {
		t.Error("expected zero value after nil scan")
	}
}

func TestAttributeScanString(t *testing.T) {
	var a Attribute
	data := `{"tag":"textarea","default_value":"hello"}`
	if err := a.Scan(data); err != nil {
		t.Fatalf("Scan(string) error: %v", err)
	}
	if a.Tag != "textarea" || a.DefaultValue != "hello" {
		t.Errorf("unexpected result: %+v", a)
	}
}

func TestAttributeScanBytes(t *testing.T) {
	var a Attribute
	data := []byte(`{"sort":true}`)
	if err := a.Scan(data); err != nil {
		t.Fatalf("Scan([]byte) error: %v", err)
	}
	if !a.Sort {
		t.Error("expected Sort=true")
	}
}

func TestAttributeScanUnsupportedType(t *testing.T) {
	var a Attribute
	if err := a.Scan(456); err != ErrUnsupportType {
		t.Errorf("expected ErrUnsupportType, got %v", err)
	}
}
```

- [ ] **Step 2: 运行测试**

Run:
```bash
cd /root/workspaces/go/rest && go test ./schema/... -run TestAttribute -v
```

Expected: 全部 PASS

- [ ] **Step 3: Commit**

```bash
git add schema/attribute_test.go
git commit -m "test(schema): add unit tests for Attribute Scan/Value"
```

---

### Task 5: `schema/scenarios_test.go` — Scenarios 的 Has/Scan/Value

**Files:**
- Create: `schema/scenarios_test.go`

- [ ] **Step 1: 编写 `Scenarios.Has`、`Scenarios.Scan`、`Scenarios.Value` 测试**

```go
package schema

import (
	"encoding/json"
	"testing"
)

func TestScenariosHas(t *testing.T) {
	s := Scenarios{"create", "update", "list"}
	if !s.Has("create") {
		t.Error("expected Has(create) = true")
	}
	if !s.Has("list") {
		t.Error("expected Has(list) = true")
	}
	if s.Has("delete") {
		t.Error("expected Has(delete) = false")
	}
	if s.Has("") {
		t.Error("expected Has('') = false")
	}
}

func TestScenariosValue(t *testing.T) {
	s := Scenarios{"create", "update"}
	v, err := s.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}
	want, _ := json.Marshal([]string{"create", "update"})
	if string(v.([]byte)) != string(want) {
		t.Errorf("got %s, want %s", v, want)
	}
}

func TestScenariosScanNil(t *testing.T) {
	var s Scenarios
	if err := s.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) error: %v", err)
	}
	if len(s) != 0 {
		t.Errorf("expected empty, got %v", s)
	}
}

func TestScenariosScanString(t *testing.T) {
	var s Scenarios
	data := `["create","update","delete"]`
	if err := s.Scan(data); err != nil {
		t.Fatalf("Scan(string) error: %v", err)
	}
	if len(s) != 3 || !s.Has("delete") {
		t.Errorf("unexpected scenarios: %v", s)
	}
}

func TestScenariosScanBytes(t *testing.T) {
	var s Scenarios
	data := []byte(`["detail"]`)
	if err := s.Scan(data); err != nil {
		t.Fatalf("Scan([]byte) error: %v", err)
	}
	if len(s) != 1 || !s.Has("detail") {
		t.Errorf("unexpected scenarios: %v", s)
	}
}

func TestScenariosScanUnsupportedType(t *testing.T) {
	var s Scenarios
	if err := s.Scan(789); err != ErrUnsupportType {
		t.Errorf("expected ErrUnsupportType, got %v", err)
	}
}
```

- [ ] **Step 2: 运行测试**

Run:
```bash
cd /root/workspaces/go/rest && go test ./schema/... -run TestScenarios -v
```

Expected: 全部 PASS

- [ ] **Step 3: Commit**

```bash
git add schema/scenarios_test.go
git commit -m "test(schema): add unit tests for Scenarios Has/Scan/Value"
```

---

### Task 6: `schema/migrate_test.go` — parseFieldName / parseFieldNative / parseFieldType

**Files:**
- Create: `schema/migrate_test.go`

- [ ] **Step 1: 编写 helper 和 `parseFieldName` 测试**

```go
package schema

import (
	"sync"
	"testing"
	"time"

	gormSchema "gorm.io/gorm/schema"
)

func mustParseTestSchema(t *testing.T) *gormSchema.Schema {
	t.Helper()
	type TestModel struct {
		ID          uint      `gorm:"primaryKey"`
		Name        string    `gorm:"column:name;type:varchar(100);comment:名称;rule:required;scenarios:create;update;props:icon:user"`
		Age         int       `gorm:"column:age;type:int;comment:年龄"`
		Email       string    `gorm:"column:email;virtual"`
		Status      string    `gorm:"column:status;enum:1:启用#green;2:禁用#red"`
		Position    int       `gorm:"column:position;position:50"`
		Password    string    `gorm:"column:password"`
		Score       float64   `gorm:"column:score;type:decimal"`
		Active      bool      `gorm:"column:active"`
		Description string    `gorm:"size:2048"`
		CreatedAt   time.Time
		UpdatedAt   time.Time
	}
	s, err := gormSchema.Parse(&TestModel{}, &sync.Map{}, gormSchema.NamingStrategy{})
	if err != nil {
		t.Fatalf("schema.Parse failed: %v", err)
	}
	return s
}

func TestParseFieldName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user_name", "User Name"},
		{"age", "Age"},
		{"a_b_c", "A B C"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseFieldName(tt.input)
			if got != tt.want {
				t.Errorf("parseFieldName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: 编写 `parseFieldNative` 和 `parseFieldType` 测试**

在文件中追加：

```go
func TestParseFieldNative(t *testing.T) {
	s := mustParseTestSchema(t)
	tests := []struct {
		fieldName string
		want      uint8
	}{
		{"Name", 1},
		{"Email", 0},
	}
	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			f := s.LookUpField(tt.fieldName)
			if f == nil {
				t.Fatalf("field %s not found", tt.fieldName)
			}
			got := parseFieldNative(f)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseFieldType(t *testing.T) {
	s := mustParseTestSchema(t)
	tests := []struct {
		fieldName string
		want      string
	}{
		{"Name", "string"},
		{"Age", "integer"},
		{"Score", "float"},
		{"ID", "integer"},
		{"Active", "boolean"},
	}
	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			f := s.LookUpField(tt.fieldName)
			if f == nil {
				t.Fatalf("field %s not found", tt.fieldName)
			}
			got := parseFieldType(f)
			if got != tt.want {
				t.Errorf("parseFieldType(%s) = %q, want %q", tt.fieldName, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 3: 运行测试**

Run:
```bash
cd /root/workspaces/go/rest && go test ./schema/... -run "TestParseFieldName|TestParseFieldNative|TestParseFieldType" -v
```

Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add schema/migrate_test.go
git commit -m "test(schema): add unit tests for parseFieldName, parseFieldNative, parseFieldType"
```

---

### Task 7: `schema/migrate_test.go` — parseFieldFormat / parseFieldRules

**Files:**
- Modify: `schema/migrate_test.go`

- [ ] **Step 1: 追加 `parseFieldFormat` 测试**

```go
func TestParseFieldFormat(t *testing.T) {
	s := mustParseTestSchema(t)
	tests := []struct {
		fieldName string
		want      string
	}{
		{"Name", "string"},
		{"Status", "dropdown"},
		{"Password", "password"},
		{"CreatedAt", "datetime"},
		{"UpdatedAt", "datetime"},
		{"Description", "text"},
		{"Score", "float"},
		{"Age", "integer"},
		{"Active", "boolean"},
	}
	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			f := s.LookUpField(tt.fieldName)
			if f == nil {
				t.Fatalf("field %s not found", tt.fieldName)
			}
			got := parseFieldFormat(f)
			if got != tt.want {
				t.Errorf("parseFieldFormat(%s) = %q, want %q", tt.fieldName, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: 追加 `parseFieldRules` 测试**

```go
func TestParseFieldRules(t *testing.T) {
	s := mustParseTestSchema(t)

	// Name has rule:"required"
	f := s.LookUpField("Name")
	if f == nil {
		t.Fatal("field Name not found")
	}
	r := parseFieldRules(f)
	if len(r.Required) == 0 {
		t.Error("expected Required to be set for Name")
	}
	if !r.Unique {
		t.Error("expected Name to have Unique from PrimaryKey")
	}

	// ID is PrimaryKey
	f = s.LookUpField("ID")
	if f == nil {
		t.Fatal("field ID not found")
	}
	r = parseFieldRules(f)
	if !r.Unique {
		t.Error("expected ID to have Unique=true")
	}
}
```

- [ ] **Step 3: 运行测试**

Run:
```bash
cd /root/workspaces/go/rest && go test ./schema/... -run "TestParseFieldFormat|TestParseFieldRules" -v
```

Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add schema/migrate_test.go
git commit -m "test(schema): add unit tests for parseFieldFormat and parseFieldRules"
```

---

### Task 8: `schema/migrate_test.go` — parseFieldScenario / parseFieldAttributes / parseFieldPosition

**Files:**
- Modify: `schema/migrate_test.go`

- [ ] **Step 1: 追加 `parseFieldScenario` 测试**

```go
func TestParseFieldScenario(t *testing.T) {
	s := mustParseTestSchema(t)
	tests := []struct {
		fieldName string
		index     int
		want      Scenarios
	}{
		{"ID", 0, Scenarios{ScenarioList, ScenarioDetail, ScenarioExport}},
		{"CreatedAt", 0, Scenarios{ScenarioList}},
		{"Name", 1, Scenarios{ScenarioSearch, ScenarioList, ScenarioCreate, ScenarioUpdate, ScenarioDetail, ScenarioExport}},
	}
	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			f := s.LookUpField(tt.fieldName)
			if f == nil {
				t.Fatalf("field %s not found", tt.fieldName)
			}
			got := parseFieldScenario(tt.index, f)
			if len(got) != len(tt.want) {
				t.Errorf("parseFieldScenario(%s) = %v, want %v", tt.fieldName, got, tt.want)
			}
			for _, w := range tt.want {
				if !got.Has(w) {
					t.Errorf("expected scenario %s not found in %v", w, got)
				}
			}
		})
	}
}
```

- [ ] **Step 2: 追加 `parseFieldAttributes` 测试**

```go
func TestParseFieldAttributes(t *testing.T) {
	s := mustParseTestSchema(t)

	// Name has props:icon:user
	f := s.LookUpField("Name")
	if f == nil {
		t.Fatal("field Name not found")
	}
	attr := parseFieldAttributes(f)
	if attr.Icon != "user" {
		t.Errorf("expected Icon=user, got %s", attr.Icon)
	}
	if attr.Match != MatchFuzzy {
		t.Errorf("expected Match=fuzzy, got %s", attr.Match)
	}

	// Status has enum
	f = s.LookUpField("Status")
	if f == nil {
		t.Fatal("field Status not found")
	}
	attr = parseFieldAttributes(f)
	if len(attr.Values) != 2 {
		t.Errorf("expected 2 enum values, got %d", len(attr.Values))
	}
	if attr.Values[0].Label != "启用" || attr.Values[0].Color != "#green" {
		t.Errorf("unexpected first enum value: %+v", attr.Values[0])
	}
	if attr.Match != MatchExactly {
		t.Errorf("expected Match=exactly for enum field, got %s", attr.Match)
	}
}
```

- [ ] **Step 3: 追加 `parseFieldPosition` 测试**

```go
func TestParseFieldPosition(t *testing.T) {
	s := mustParseTestSchema(t)
	tests := []struct {
		fieldName string
		index     int
		want      int
	}{
		{"Position", 10, 50},
		{"Age", 2, 102},
		{"ID", 0, 100},
	}
	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			f := s.LookUpField(tt.fieldName)
			if f == nil {
				t.Fatalf("field %s not found", tt.fieldName)
			}
			got := parseFieldPosition(f, tt.index)
			if got != tt.want {
				t.Errorf("parseFieldPosition(%s, %d) = %d, want %d", tt.fieldName, tt.index, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 4: 运行全部 schema 测试**

Run:
```bash
cd /root/workspaces/go/rest && go test ./schema/... -v
```

Expected: 全部 PASS（包括 migrate_test.go、rule_test.go、attribute_test.go、scenarios_test.go）

- [ ] **Step 5: Commit**

```bash
git add schema/migrate_test.go
git commit -m "test(schema): add unit tests for parseFieldScenario, parseFieldAttributes, parseFieldPosition"
```

---

## 自检清单

**1. Spec 覆盖检查：**

| Spec 要求 | 对应任务 |
|---|---|
| `formats` 9 个 format 函数 | Task 1 |
| `inflector` 5 个导出函数 | Task 2 |
| `schema` `Rule.Scan/Value` | Task 3 |
| `schema` `Attribute.Scan/Value` | Task 4 |
| `schema` `Scenarios.Has/Scan/Value` | Task 5 |
| `schema` `parseFieldName` | Task 6 |
| `schema` `parseFieldNative` | Task 6 |
| `schema` `parseFieldType` | Task 6 |
| `schema` `parseFieldFormat` | Task 7 |
| `schema` `parseFieldRules` | Task 7 |
| `schema` `parseFieldScenario` | Task 8 |
| `schema` `parseFieldAttributes` | Task 8 |
| `schema` `parseFieldPosition` | Task 8 |

全部覆盖，无遗漏。

**2. Placeholder 扫描：** 无 TBD、TODO、"implement later"。每个步骤均含完整代码和命令。

**3. 类型一致性：** 所有测试中使用的类型名、方法签名与源码一致。`schema.Field` 通过 `gorm.io/gorm/schema.Parse` 生成，`LookUpField` 获取字段。
