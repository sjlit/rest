package rest

import (
	"context"
	"fmt"
	"reflect"

	"gorm.io/gorm"
)

func recursiveTier[T comparable](parent T, values []*TierValue[T]) []*TierValue[T] {
	items := make([]*TierValue[T], 0, len(values)/2)
	for idx, row := range values {
		if row.Used {
			continue
		}
		if row.Parent == parent {
			values[idx].Used = true
			row.Children = recursiveTier(row.Value, values)
			items = append(items, row)
		}
	}
	return items
}

func IsEmpty(i any) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Pointer, reflect.Map, reflect.Slice,
		reflect.Chan, reflect.Func, reflect.Interface:
		return v.IsNil()
	}
	return false
}

func ModelTypes[T any](ctx context.Context, db *gorm.DB, model any, tenant, labelColumn, valueColumn string) (values []*TypeValue[T], err error) {
	var tx *gorm.DB
	if ctx == nil {
		tx = db
	} else {
		tx = db.WithContext(ctx)
	}
	result := make([]map[string]any, 0, 10)
	if tenant == "" {
		err = tx.Model(model).Select(labelColumn, valueColumn).Scan(&result).Error
	} else {
		err = tx.Model(model).Select(labelColumn, valueColumn).Where(TenantId+"=?", tenant).Scan(&result).Error
	}
	if err != nil {
		return
	}
	values = make([]*TypeValue[T], 0, len(result))
	for _, pairs := range result {
		feed := &TypeValue[T]{}
		for k, v := range pairs {
			if k == labelColumn {
				if s, ok := v.(string); ok {
					feed.Label = s
				} else {
					feed.Label = fmt.Sprint(v)
				}
				//这里不直接返回, 有可能key和value是同一个字段
			}
			if k == valueColumn {
				if p, ok := v.(T); ok {
					feed.Value = p
				}
			}
		}
		values = append(values, feed)
	}
	return values, nil
}

// ModelTiers 查询指定模型的层级数据
func ModelTiers[T comparable](ctx context.Context, db *gorm.DB, model any, tenant, parentColumn, labelColumn, valueColumn string) (values []*TierValue[T], err error) {
	var tx *gorm.DB
	if ctx == nil {
		tx = db
	} else {
		tx = db.WithContext(ctx)
	}
	result := make([]map[string]any, 0, 10)
	if tenant == "" {
		err = tx.Model(model).Select(parentColumn, labelColumn, valueColumn).Scan(&result).Error
	} else {
		err = tx.Model(model).Select(parentColumn, labelColumn, valueColumn).Where(TenantId+"=?", tenant).Scan(&result).Error
	}
	if err != nil {
		return
	}
	values = make([]*TierValue[T], 0, len(result))
	for _, pairs := range result {
		feed := &TierValue[T]{}
		for k, v := range pairs {
			if k == parentColumn {
				if p, ok := v.(T); ok {
					feed.Parent = p
				}
				continue
			}
			if k == labelColumn {
				if s, ok := v.(string); ok {
					feed.Label = s
				} else {
					feed.Label = fmt.Sprint(v)
				}
			}
			if k == valueColumn {
				if p, ok := v.(T); ok {
					feed.Value = p
				}
			}
		}
		values = append(values, feed)
	}
	var none T
	return recursiveTier(none, values), nil
}


