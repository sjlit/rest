package schema

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
)

const (
	TypeInteger = "integer"
	TypeFloat   = "float"
	TypeBoolean = "boolean"
	TypeString  = "string"
)

const (
	FormatInteger   = "integer"
	FormatFloat     = "float"
	FormatBoolean   = "boolean"
	FormatString    = "string"
	FormatText      = "text"
	FormatDropdown  = "dropdown"
	FormatDatetime  = "datetime"
	FormatDate      = "date"
	FormatTime      = "time"
	FormatTimestamp = "timestamp"
	FormatPassword  = "password"
)

const (
	ScenarioCreate = "create"
	ScenarioUpdate = "update"
	ScenarioDelete = "delete"
	ScenarioSearch = "search"
	ScenarioExport = "export"
	ScenarioImport = "import"
	ScenarioList   = "list"
	ScenarioDetail = "detail"
)

// IsValidScenario 报告 s 是否为已识别的 scenario 常量。
// 用于在 HTTP query 参数等外部入口校验传入值，未命中时调用方应回退到默认 scenario。
func IsValidScenario(s string) bool {
	switch s {
	case ScenarioCreate, ScenarioUpdate, ScenarioDelete,
		ScenarioSearch, ScenarioExport, ScenarioImport,
		ScenarioList, ScenarioDetail:
		return true
	}
	return false
}

const (
	MatchExactly = "exactly" //精确匹配
	MatchFuzzy   = "fuzzy"   //模糊匹配
)

const (
	LiveTypeDropdown = "dropdown"
	LiveTypeCascader = "cascader"
)

var (
	ErrMissingModuleName = errors.New("module name required")
)

var (
	matchEnums           = []string{MatchExactly, MatchFuzzy}
	readonlyScenario     = []string{ScenarioCreate, ScenarioUpdate}
	requiredScenario     = []string{ScenarioCreate, ScenarioUpdate}
	allowMethods         = []string{http.MethodPut, http.MethodPost}
	timeSearchRangeEnums = []string{"minute", "hour", "day", "week", "month", "year"}
)

type Relation struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Module string `json:"module"`
	Table  string `json:"table"`
}

func (r *Relation) Scan(value any) error {
	if value == nil {
		return nil
	}
	switch s := value.(type) {
	case string:
		return json.Unmarshal([]byte(s), r)
	case []byte:
		return json.Unmarshal(s, r)
	}
	return ErrUnsupportType
}

func (r Relation) Value() (driver.Value, error) {
	return json.Marshal(r)
}
