package formats

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/sjlit/rest/v3/schema"
)

func stringFormat(ctx context.Context, value any, model any, scm schema.Schema) any {
	return fmt.Sprint(value)
}

func integerFormat(ctx context.Context, value any, model any, scm schema.Schema) any {
	var n int
	switch v := value.(type) {
	case float32, float64:
		n = int(reflect.ValueOf(v).Float())
	case int, int8, int16, int32, int64:
		n = int(reflect.ValueOf(v).Int())
	case uint, uint8, uint16, uint32, uint64:
		n = int(reflect.ValueOf(v).Uint())
	case string:
		n, _ = strconv.Atoi(v)
	case []byte:
		n, _ = strconv.Atoi(string(v))
	}
	return n
}

func decimalFormat(ctx context.Context, value any, model any, scm schema.Schema) any {
	var n float64
	switch v := value.(type) {
	case float32, float64:
		n = reflect.ValueOf(v).Float()
	case int, int8, int16, int32, int64:
		n = float64(reflect.ValueOf(v).Int())
	case uint, uint8, uint16, uint32, uint64:
		n = float64(reflect.ValueOf(v).Uint())
	case string:
		n, _ = strconv.ParseFloat(v, 64)
	case []byte:
		n, _ = strconv.ParseFloat(string(v), 64)
	}
	return n
}

func dateFormat(ctx context.Context, value any, model any, scm schema.Schema) any {
	if t, ok := value.(time.Time); ok {
		return t.Format("2006-01-02")
	}
	if t, ok := value.(*sql.NullTime); ok && t != nil && t.Valid {
		return t.Time.Format("2006-01-02")
	}
	if t, ok := value.(int64); ok {
		return time.Unix(t, 0).Format("2006-01-02")
	}
	return ""
}

func timeFormat(ctx context.Context, value any, model any, scm schema.Schema) any {
	if t, ok := value.(time.Time); ok {
		return t.Format("15:04:05")
	}
	if t, ok := value.(*sql.NullTime); ok && t != nil && t.Valid {
		return t.Time.Format("15:04:05")
	}
	if t, ok := value.(int64); ok {
		return time.Unix(t, 0).Format("15:04:05")
	}
	return ""
}

func datetimeFormat(ctx context.Context, value any, model any, scm schema.Schema) any {
	if t, ok := value.(time.Time); ok {
		return t.Format("2006-01-02 15:04:05")
	}
	if t, ok := value.(*sql.NullTime); ok && t != nil && t.Valid {
		return t.Time.Format("2006-01-02 15:04:05")
	}
	if t, ok := value.(int64); ok && t > 0 {
		return time.Unix(t, 0).Format("2006-01-02 15:04:05")
	}
	return ""
}

func percentageFormat(ctx context.Context, value any, model any, scm schema.Schema) any {
	n, ok := decimalFormat(ctx, value, model, scm).(float64)
	if !ok {
		return "0.00%"
	}
	if n <= 1 {
		return fmt.Sprintf("%.2f%%", n*100)
	}
	return fmt.Sprintf("%.2f%%", n)
}

func durationFormat(ctx context.Context, value any, model any, scm schema.Schema) any {
	n, ok := integerFormat(ctx, value, model, scm).(int)
	if !ok {
		return "00:00:00"
	}
	hour := n / 3600
	minVal := (n - hour*3600) / 60
	sec := n - hour*3600 - minVal*60
	return fmt.Sprintf("%02d:%02d:%02d", hour, minVal, sec)
}

func dropdownFormat(ctx context.Context, value any, model any, scm schema.Schema) any {
	if scm.Attributes.Values != nil {
		for _, v := range scm.Attributes.Values {
			if v.Value == fmt.Sprint(value) {
				return v.Label
			}
		}
	}
	return value
}
