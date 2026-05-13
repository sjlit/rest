package validate

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"git.nobla.cn/golang/rest"
	"git.nobla.cn/golang/rest/schema"
	validator "github.com/go-playground/validator/v10"
	"gorm.io/gorm"
	gschema "gorm.io/gorm/schema"
)

type Validate struct {
	validator *validator.Validate
}

func (validate *Validate) Name() string {
	return "gorm:validate"
}

func (validate *Validate) telephoneValidate(ctx context.Context, fl validator.FieldLevel) bool {
	val := fmt.Sprint(fl.Field().Interface())
	return telephoneRegex.MatchString(val)
}

func (validate *Validate) uniqueValidate(ctx context.Context, fl validator.FieldLevel) bool {
	var (
		scope           *validateScope
		ok              bool
		count           int64
		field           *gschema.Field
		primaryKeyValue reflect.Value
	)
	val := fl.Field().Interface()
	if scope, ok = ctx.Value(scopeCtxKey).(*validateScope); !ok {
		return true
	}
	if len(scope.DB.Statement.Schema.PrimaryFields) > 0 {
		field = scope.DB.Statement.Schema.PrimaryFields[0]
		primaryKeyValue = reflect.Indirect(reflect.ValueOf(scope.Model))
		for _, n := range field.BindNames {
			primaryKeyValue = primaryKeyValue.FieldByName(n)
		}
	}
	sess := scope.DB.Session(&gorm.Session{NewDB: true})
	if primaryKeyValue.IsValid() && !primaryKeyValue.IsZero() && field != nil {
		sess.Model(scope.Model).Where(scope.Column+"=? AND "+field.DBName+" != ?", val, primaryKeyValue.Interface()).Count(&count)
	} else {
		sess.Model(scope.Model).Where(scope.Column+"=?", val).Count(&count)
	}
	if count > 0 {
		return false
	}
	return true
}

func (validate *Validate) grantRules(scm schema.Schema, scenario string, rule schema.Rule) []*validateRule {
	rules := make([]*validateRule, 0, 5)
	if rule.Min != 0 {
		rules = append(rules, newRule("min", strconv.Itoa(rule.Min)))
	}
	if rule.Max != 0 {
		rules = append(rules, newRule("max", strconv.Itoa(rule.Max)))
	}
	//主键不做唯一判断
	if rule.Unique && scm.PrimaryKey == 0 {
		rules = append(rules, newRule("db_unique"))
	}
	if rule.Type != "" {
		rules = append(rules, newRule(rule.Type))
	}
	if len(rule.Required) > 0 {
		if slices.Contains(rule.Required, scenario) {
			rules = append(rules, newRule("required"))
		}
	}
	return rules
}

func (validate *Validate) buildRules(rs []*validateRule) string {
	var sb strings.Builder
	for _, r := range rs {
		if !r.Valid {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString(",")
		}
		if r.Value == "" {
			sb.WriteString(r.Rule)
		} else {
			sb.WriteString(r.Rule + "=" + r.Value)
		}
	}
	return sb.String()
}

func (validate *Validate) findRule(name string, rules []*validateRule) *validateRule {
	for _, r := range rules {
		if r.Rule == name {
			return r
		}
	}
	return nil
}

func (validate *Validate) Initialize(db *gorm.DB) (err error) {
	validate.validator = validator.New()
	if err = db.Callback().Create().Before("gorm:before_create").Register("rest_validate_create", validate.Validate); err != nil {
		return
	}
	if err = db.Callback().Update().Before("gorm:before_update").Register("rest_validate_update", validate.Validate); err != nil {
		return
	}
	if err = validate.validator.RegisterValidationCtx("telephone", validate.telephoneValidate); err != nil {
		return
	}
	if err = validate.validator.RegisterValidationCtx("db_unique", validate.uniqueValidate); err != nil {
		return
	}
	return
}

func (validate *Validate) inArray(v any, vs []any) bool {
	sv := fmt.Sprint(v)
	for _, s := range vs {
		if fmt.Sprint(s) == sv {
			return true
		}
	}
	return false
}

// isVisible 判断字段是否需要显示
func (validate *Validate) isVisible(stmt *gorm.Statement, scm schema.Schema) bool {
	if len(scm.Attributes.Visible) <= 0 {
		return true
	}
	for _, row := range scm.Attributes.Visible {
		if len(row.Values) == 0 {
			continue
		}
		targetField := stmt.Schema.LookUpField(row.Column)
		if targetField == nil {
			continue
		}
		targetValue := stmt.ReflectValue.FieldByName(targetField.Name)
		if !targetValue.IsValid() {
			return false
		}
		if !validate.inArray(targetValue.Interface(), row.Values) {
			return false
		}
	}
	return true
}

// Validate 校验字段
func (validate *Validate) Validate(db *gorm.DB) {
	var (
		err          error
		rules        []*validateRule
		stmt         *gorm.Statement
		skipValidate bool
		multiDomain  bool
		value        reflect.Value
		runtimeScope *rest.RuntimeScope
	)
	if result, ok := db.Get(SkipValidations); ok && result.(bool) {
		return
	}
	stmt = db.Statement
	if stmt.Model == nil {
		return
	}
	if db.Statement.Context == nil {
		return
	}
	if runtimeScope = rest.RuntimeScopeFromContext(db.Statement.Context); runtimeScope == nil {
		return
	}
	//if not current table, skip it
	if runtimeScope.TableName != stmt.Table {
		return
	}
	if runtimeScope.Schemas == nil {
		return
	}
	if stmt.Schema.LookUpField("domain") != nil {
		multiDomain = true
	}
	for _, row := range runtimeScope.Schemas {
		//如果字段隐藏，那么就不进行校验
		if !validate.isVisible(stmt, row) {
			continue
		}
		if rules = validate.grantRules(row, runtimeScope.Scenario, row.Rules); len(rules) <= 0 {
			continue
		}
		field := stmt.Schema.LookUpField(row.Column)
		if field == nil {
			continue
		}
		value = stmt.ReflectValue.FieldByName(field.Name)
		if !value.IsValid() {
			continue
		}
		skipValidate = false
		if r := validate.findRule("required", rules); r != nil {
			if value.Interface() != nil {
				vType := reflect.ValueOf(value.Interface())
				switch vType.Kind() {
				case reflect.Bool:
					skipValidate = true
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					skipValidate = true
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					skipValidate = true
				case reflect.Float32, reflect.Float64:
					skipValidate = true
				default:
					skipValidate = false
				}
			}
			if skipValidate {
				r.Valid = false
			}
		} else {
			if isEmpty(value.Interface()) {
				continue
			}
		}
		ctx := context.WithValue(db.Statement.Context, scopeCtxKey, &validateScope{
			DB:          db,
			Column:      row.Column,
			Model:       stmt.Model,
			MultiDomain: multiDomain,
		})
		if err = validate.validator.VarCtx(ctx, value.Interface(), validate.buildRules(rules)); err != nil {
			if errs, ok := err.(validator.ValidationErrors); ok {
				for _, e := range errs {
					_ = db.AddError(&StructError{
						Tag:     e.Tag(),
						Column:  row.Column,
						Message: formatError(row.Rules, row, e.Tag()),
					})
				}
			} else {
				_ = db.AddError(err)
			}
			break
		}
	}
}

func New() *Validate {
	return &Validate{}
}
