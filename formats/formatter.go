package formats

import (
	"context"
	"reflect"
	"sync"

	"git.nobla.cn/golang/rest/schema"
	"gorm.io/gorm"
)

const (
	FormatRaw  = "raw"
	FormatBoth = "both"
)

type FormatFunc func(ctx context.Context, value any, model any, scm schema.Schema) any

type multiValue struct {
	Text  any `json:"label"`
	Value any `json:"value"`
}

type Formatter struct {
	callbacks sync.Map
}

func NewFormatter() *Formatter {
	return &Formatter{}
}

func DefaultFormatter() *Formatter {
	f := NewFormatter()
	f.Register("string", stringFormat)
	f.Register("integer", integerFormat)
	f.Register("decimal", decimalFormat)
	f.Register("date", dateFormat)
	f.Register("time", timeFormat)
	f.Register("datetime", datetimeFormat)
	f.Register("duration", durationFormat)
	f.Register("dropdown", dropdownFormat)
	f.Register("timestamp", datetimeFormat)
	f.Register("percentage", percentageFormat)
	return f
}

func (f *Formatter) Register(name string, fn FormatFunc) {
	f.callbacks.Store(name, fn)
}

func (f *Formatter) Format(ctx context.Context, format string, value any, model any, scm schema.Schema) any {
	v, ok := f.callbacks.Load(format)
	if ok {
		return v.(FormatFunc)(ctx, value, model, scm)
	}
	return value
}

func (f *Formatter) getModelValue(refValue reflect.Value, scm schema.Schema, stmt *gorm.Statement) any {
	if stmt.Schema == nil {
		return nil
	}
	field := stmt.Schema.LookUpField(scm.Column)
	if field == nil {
		return nil
	}
	fieldVal := refValue.FieldByName(field.Name)
	if !fieldVal.IsValid() || !fieldVal.CanInterface() {
		return nil
	}
	return fieldVal.Interface()
}

func (f *Formatter) FormatModel(ctx context.Context, refValue reflect.Value, schemas []schema.Schema, stmt *gorm.Statement, format string) any {
	values := make(map[string]any)
	multiValues := make(map[string]multiValue)
	modelValue := refValue.Interface()
	indirectValue := reflect.Indirect(refValue)
	for _, scm := range schemas {
		switch format {
		case FormatRaw:
			values[scm.Column] = f.getModelValue(indirectValue, scm, stmt)
		case FormatBoth:
			v := multiValue{Value: f.getModelValue(indirectValue, scm, stmt)}
			v.Text = f.Format(ctx, scm.Format, v.Value, modelValue, scm)
			multiValues[scm.Column] = v
		default:
			values[scm.Column] = f.Format(ctx, scm.Format, f.getModelValue(indirectValue, scm, stmt), modelValue, scm)
		}
	}
	if format == FormatBoth {
		return multiValues
	}
	return values
}

func (f *Formatter) FormatModels(ctx context.Context, models any, schemas []schema.Schema, stmt *gorm.Statement, format string) any {
	refValue := reflect.Indirect(reflect.ValueOf(models))
	if refValue.Kind() != reflect.Slice {
		return []any{}
	}
	length := refValue.Len()
	values := make([]any, length)
	for i := range length {
		rowValue := refValue.Index(i)
		values[i] = f.FormatModel(ctx, rowValue, schemas, stmt, format)
	}
	return values
}
