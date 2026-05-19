package schema

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

var (
	timeKind    = reflect.TypeFor[time.Time]().Kind()
	timePtrKind = reflect.TypeOf(&time.Time{}).Kind()
)

func parseFieldName(name string) string {
	tokens := strings.Split(name, "_")
	for i, s := range tokens {
		tokens[i] = strings.Title(s)
	}
	return strings.Join(tokens, " ")
}

func parseFieldNative(field *schema.Field) uint8 {
	if _, ok := field.Tag.Lookup("virtual"); ok {
		return 0
	}
	return 1
}

func parseFieldType(field *schema.Field) string {
	var dataType string
	reflectType := field.FieldType
	for reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}
	if dataType = field.Tag.Get("type"); dataType != "" {
		return dataType
	}
	dataValue := reflect.Indirect(reflect.New(reflectType))
	switch dataValue.Kind() {
	case reflect.Bool:
		dataType = TypeBoolean
	case reflect.Int8, reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint8, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		dataType = TypeInteger
	case reflect.Float32, reflect.Float64:
		dataType = TypeFloat
	default:
		dataType = TypeString
	}
	return dataType
}

func parseFieldFormat(field *schema.Field) string {
	var format string
	format = field.Tag.Get("format")
	if format != "" {
		return format
	}
	//如果有枚举值，直接设置为下拉类型
	enum := field.Tag.Get("enum")
	if enum != "" {
		return FormatDropdown
	}
	reflectType := field.FieldType
	for reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}
	//时间处理
	dataValue := reflect.Indirect(reflect.New(reflectType))
	if field.Name == "CreatedAt" || field.Name == "UpdatedAt" || field.Name == "DeletedAt" {
		switch dataValue.Kind() {
		case reflect.Int8, reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint8, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return FormatTimestamp
		default:
			return FormatDatetime
		}
	}
	if strings.Contains(strings.ToLower(field.Name), "pass") {
		return FormatPassword
	}
	switch dataValue.Kind() {
	case timeKind, timePtrKind:
		format = FormatDatetime
	case reflect.Bool:
		format = FormatBoolean
	case reflect.Int8, reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint8, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		format = FormatInteger
	case reflect.Float32, reflect.Float64:
		format = FormatFloat
	case reflect.Struct:
		if _, ok := dataValue.Interface().(time.Time); ok {
			format = FormatDatetime
		}
	default:
		if field.Size >= 1024 {
			format = FormatText
		} else {
			format = FormatString
		}
	}
	return format
}

func parseFieldRules(field *schema.Field) Rule {
	r := Rule{
		Required: []string{},
	}
	if field.GORMDataType == schema.String {
		r.Max = field.Size
	}
	if field.GORMDataType == schema.Int || field.GORMDataType == schema.Float || field.GORMDataType == schema.Uint {
		r.Max = field.Scale
	}

	rs := field.Tag.Get("rule")
	if rs != "" {
		ss := strings.Split(rs, ";")
		for _, s := range ss {
			vs := strings.SplitN(s, ":", 2)
			ls := len(vs)
			if ls == 0 {
				continue
			}
			switch vs[0] {
			case "required", "require":
				if ls > 1 {
					bs := strings.Split(vs[1], ",")
					for _, i := range bs {
						if slices.Contains(requiredScenario, i) {
							r.Required = append(r.Required, i)
						}
					}
				} else {
					r.Required = requiredScenario
				}
			case "unique":
				r.Unique = true
			case "safe":
				r.Safe = true
			case "regexp":
				if ls > 1 {
					r.Regular = vs[1]
				}
			}
		}
	}
	if field.PrimaryKey {
		r.Unique = true
	}
	return r
}

func parseFieldScenario(index int, field *schema.Field) Scenarios {
	var ss Scenarios
	if v, ok := field.Tag.Lookup("scenarios"); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			ss = strings.Split(v, ";")
		}
	} else {
		if field.PrimaryKey {
			ss = []string{ScenarioList, ScenarioDetail, ScenarioExport}
		} else if field.Name == "CreatedAt" || field.Name == "UpdatedAt" {
			ss = []string{ScenarioList}
		} else if field.Name == "DeletedAt" || field.Name == "Namespace" {
			//不添加任何显示场景
			ss = []string{}
		} else {
			if index < 10 {
				//高级字段只配置一些简单的场景
				ss = []string{ScenarioSearch, ScenarioList, ScenarioCreate, ScenarioUpdate, ScenarioDetail, ScenarioExport}
			} else {
				//高级字段只配置一些简单的场景
				ss = []string{ScenarioCreate, ScenarioUpdate, ScenarioDetail, ScenarioExport}
			}
		}
	}
	return ss
}

func parseFieldAttributes(field *schema.Field) Attribute {
	attr := Attribute{
		Match:        MatchFuzzy,
		DefaultValue: field.DefaultValue,
		Readonly:     []string{},
		Disable:      []string{},
		Visible:      make([]VisibleCondition, 0),
		Values:       make([]EnumValue, 0),
		Live:         LiveValue{},
	}
	if field.Name == "CreatedAt" || field.Name == "UpdatedAt" {
		attr.EndOfNow = true
	}
	//赋值属性
	props := field.Tag.Get("props")
	if props != "" {
		vs := strings.Split(props, ";")
		for _, str := range vs {
			kv := strings.SplitN(str, ":", 2)
			if len(kv) != 2 {
				continue
			}
			sv := strings.TrimSpace(kv[1])
			switch strings.ToLower(strings.TrimSpace(kv[0])) {
			case "icon":
				attr.Icon = sv
			case "match":
				if slices.Contains(matchEnums, sv) {
					attr.Match = sv
				}
			case "endofnow", "end_of_now":
				if ok, _ := strconv.ParseBool(sv); ok {
					attr.EndOfNow = true
				}
			case "time_search_range", "timeSearchRange", "timesearchrange":
				if slices.Contains(timeSearchRangeEnums, sv) {
					attr.TimeSearchRange = sv
				}
			case "invisible":
				if ok, _ := strconv.ParseBool(sv); ok {
					attr.Invisible = true
				}
			case "suffix":
				attr.Suffix = sv
			case "tag":
				attr.Tag = sv
			case "tooltip":
				attr.Tooltip = sv
			case "uploadurl", "uploaduri", "upload_url", "upload_uri":
				attr.UploadUrl = sv
			case "description":
				attr.Description = sv
			case "readonly":
				bs := strings.Split(sv, ",")
				for _, i := range bs {
					if slices.Contains(readonlyScenario, i) {
						attr.Readonly = append(attr.Readonly, i)
					}
				}
			}
		}
	}
	//live的赋值
	live := field.Tag.Get("live")
	if live != "" {
		attr.Live.Enable = true
		vs := strings.Split(live, ";")
		for _, str := range vs {
			kv := strings.SplitN(str, ":", 2)
			if len(kv) != 2 {
				continue
			}
			switch kv[0] {
			case "method":
				attr.Live.Method = kv[1]
			case "type":
				if kv[1] == LiveTypeDropdown || kv[1] == LiveTypeCascader {
					attr.Live.Type = kv[1]
				} else {
					attr.Live.Type = LiveTypeDropdown
				}
			case "url", "uri":
				attr.Live.Url = kv[1]
			case "columns":
				attr.Live.Columns = strings.Split(kv[1], ",")
			}
		}
		if attr.Live.Enable {
			if attr.Live.Method == "" {
				attr.Live.Method = "GET"
			}
		}
		attr.Match = MatchExactly
	}

	dropdown := field.Tag.Get("dropdown")
	if dropdown != "" {
		attr.DropdownOptions = &DropdownOptions{}
		vs := strings.Split(dropdown, ";")
		for _, str := range vs {
			kv := strings.SplitN(str, ":", 2)
			if len(kv) == 0 {
				continue
			}
			switch kv[0] {
			case "created":
				attr.DropdownOptions.Created = true
			case "filterable":
				attr.DropdownOptions.Filterable = true
			case "autocomplete":
				attr.DropdownOptions.Autocomplete = true
			case "default_first":
				attr.DropdownOptions.DefaultFirst = true
			}
		}
	}

	//显示条件
	conditions := field.Tag.Get("condition")
	if conditions != "" {
		vs := strings.Split(conditions, ";")
		for _, str := range vs {
			kv := strings.SplitN(str, ":", 2)
			if len(kv) != 2 {
				continue
			}
			cond := VisibleCondition{
				Column: kv[0],
				Values: make([]any, 0),
			}
			vv := strings.Split(kv[1], ",")
			for _, x := range vv {
				x = strings.TrimSpace(x)
				if x == "" {
					continue
				}
				cond.Values = append(cond.Values, x)
			}
			attr.Visible = append(attr.Visible, cond)
		}
	}
	//赋值枚举值
	enumns := field.Tag.Get("enum")
	if enumns != "" {
		vs := strings.Split(enumns, ";")
		for _, str := range vs {
			kv := strings.SplitN(str, ":", 2)
			if len(kv) != 2 {
				continue
			}
			fv := EnumValue{Value: kv[0]}
			//颜色分隔符
			if pos := strings.IndexByte(kv[1], '#'); pos > -1 {
				fv.Label = kv[1][:pos]
				fv.Color = kv[1][pos:]
			} else {
				fv.Label = kv[1]
			}
			attr.Values = append(attr.Values, fv)
		}
		attr.Match = MatchExactly
	}
	if !field.Creatable {
		attr.Disable = append(attr.Disable, ScenarioCreate)
	}
	if !field.Updatable {
		attr.Disable = append(attr.Disable, ScenarioUpdate)
	}
	attr.Tooltip = field.Comment
	return attr
}

func parseFieldPosition(field *schema.Field, i int) int {
	s := field.Tag.Get("position")
	n, _ := strconv.Atoi(s)
	if n > 0 {
		return n
	}
	return i + 100
}

func parseFieldRelations(field *schema.Field) Relation {
	var rel Relation
	tag := field.Tag.Get("relation")
	if tag == "" {
		return rel
	}
	parts := strings.Split(tag, ":")
	if len(parts) >= 1 {
		rel.Type = parts[0]
	}
	if len(parts) >= 2 {
		rel.Name = parts[1]
	} else {
		rel.Name = field.Name
	}
	if len(parts) >= 3 {
		rel.Module = parts[2]
	}
	if len(parts) >= 4 {
		rel.Table = parts[3]
	}
	return rel
}

// GetSchemas 获取表字段
func GetSchemas(ctx context.Context, db *gorm.DB, moduleName, tableName string) ([]Schema, error) {
	if defaultCache != nil {
		return defaultCache.GetSchemas(ctx, moduleName, tableName)
	}

	var (
		err    error
		values []Schema
	)
	values = make([]Schema, 0)
	if moduleName == "" || tableName == "" {
		return nil, ErrMissingModuleName
	}
	values, err = gorm.G[Schema](db).Where("module_name=? AND table_name=?", moduleName, tableName).Order("position ASC").Find(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return values, err
}

func GetVisibleSchemas(ctx context.Context, db *gorm.DB, moduleName, tableName, scenario string) ([]Schema, error) {
	if defaultCache != nil {
		return defaultCache.GetVisibleSchemas(ctx, moduleName, tableName, scenario)
	}

	schemas, err := GetSchemas(ctx, db, moduleName, tableName)
	if err != nil {
		return nil, err
	}
	result := make([]Schema, 0, len(schemas))
	for _, row := range schemas {
		if row.Scenarios.Has(scenario) {
			result = append(result, row)
		}
	}
	return result, nil
}

func AutoMigrate(ctx context.Context, db *gorm.DB, model any, moduleName string) (tableName string, err error) {
	var (
		pos            int
		columnName     string
		columnIsExists bool
		columnLabel    string
		schemas        []Schema
		models         []Schema
		stmt           *gorm.Statement
	)
	stmt = &gorm.Statement{
		DB:      db,
		Context: ctx,
		Clauses: map[string]clause.Clause{},
	}
	if err = stmt.Parse(model); err != nil {
		return
	}
	tableName = stmt.Table
	if schemas, err = GetSchemas(ctx, db, moduleName, stmt.Table); err != nil {
		return
	}
	if len(schemas) > 0 {
		pos = len(schemas)
	}
	models = make([]Schema, 0)
	for index, field := range stmt.Schema.Fields {
		columnName = field.DBName
		if columnName == "-" {
			continue
		}
		if columnName == "" {
			columnName = field.Name
		}
		columnIsExists = false
		for _, sm := range schemas {
			if sm.Column == columnName {
				columnIsExists = true
				break
			}
		}
		if columnIsExists {
			continue
		}
		columnLabel = field.Tag.Get("comment")
		if columnLabel == "" {
			columnLabel = parseFieldName(field.DBName)
		}
		isPrimaryKey := uint8(0)
		if field.PrimaryKey {
			isPrimaryKey = 1
		}
		schemaModel := Schema{
			ModuleName: moduleName,
			TableName:  stmt.Table,
			Enable:     1,
			Column:     columnName,
			Label:      columnLabel,
			Type:       strings.ToLower(parseFieldType(field)),
			Format:     strings.ToLower(parseFieldFormat(field)),
			Native:     parseFieldNative(field),
			PrimaryKey: isPrimaryKey,
			Rules:      parseFieldRules(field),
			Scenarios:  parseFieldScenario(index, field),
			Attributes: parseFieldAttributes(field),
			Position:   parseFieldPosition(field, pos),
			Relations:  parseFieldRelations(field),
		}
		//如果启用了在线调取接口功能，那么设置一下字段的format格式
		if schemaModel.Attributes.Live.Enable {
			if schemaModel.Attributes.Live.Type != "" {
				schemaModel.Format = schemaModel.Attributes.Live.Type
			}
		}
		models = append(models, schemaModel)
		pos++
	}
	if len(models) > 0 {
		err = gorm.G[Schema](db).CreateInBatches(ctx, &models, 50)
	}
	InvalidateCache(moduleName, tableName)
	return
}
