package validate

import (
	"reflect"
	"regexp"
	"strconv"

	"git.nobla.cn/golang/rest/schema"
	"gorm.io/gorm"
)

const (
	SkipValidations = "validations:skip_validations"
)

var (
	scopeCtxKey    = &validateScope{}
	telephoneRegex = regexp.MustCompile(`^\d{5,20}$`)
)

type (
	validateScope struct {
		DB          *gorm.DB
		Column      string
		MultiDomain bool
		Model       interface{}
	}

	StructError struct {
		Tag     string `json:"tag"`
		Column  string `json:"column"`
		Message string `json:"message"`
	}

	validateRule struct {
		Rule  string
		Value string
		Valid bool
	}
)

func (err *StructError) Error() string {
	return err.Message
}

func newRule(ss ...string) *validateRule {
	v := &validateRule{
		Valid: true,
	}
	if len(ss) == 1 {
		v.Rule = ss[0]
	} else if len(ss) >= 2 {
		v.Rule = ss[0]
		v.Value = ss[1]
	}
	return v
}

// formatError 格式化错误消息
func formatError(rule schema.Rule, scm schema.Schema, tag string) string {
	var s string
	switch tag {
	case "db_unique":
		s = scm.Label + "值已经存在."
	case "required":
		s = scm.Label + "值不能为空."
	case "max":
		if scm.Type == "string" {
			s = scm.Label + "长度不能大于" + strconv.Itoa(rule.Max)
		} else {
			s = scm.Label + "值不能大于" + strconv.Itoa(rule.Max)
		}
	case "min":
		if scm.Type == "string" {
			s = scm.Label + "长度不能小于" + strconv.Itoa(rule.Min)
		} else {
			s = scm.Label + "值不能小于" + strconv.Itoa(rule.Min)
		}
	case "numeric":
		s = scm.Label + "值必须是数字."
	case "boolean":
		s = scm.Label + "值必须是布尔值."
	}
	return s
}

// isEmpty 判断值是否为空
func isEmpty(val any) bool {
	if val == nil {
		return true
	}
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.String, reflect.Array:
		return v.Len() == 0
	case reflect.Map, reflect.Slice:
		return v.IsNil() || v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return reflect.DeepEqual(val, reflect.Zero(v.Type()).Interface())
}
