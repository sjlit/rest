package formats

import (
	"context"
	"testing"
	"time"

	"git.nobla.cn/golang/rest/schema"
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
		{"int64", tm.Unix(), time.Unix(tm.Unix(), 0).Format("2006-01-02")},
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
		{"int64", tm.Unix(), time.Unix(tm.Unix(), 0).Format("15:04:05")},
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
		{"int64", tm.Unix(), time.Unix(tm.Unix(), 0).Format("2006-01-02 15:04:05")},
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
