package formats

import (
	"context"
	"reflect"
	"testing"

	"git.nobla.cn/golang/rest/v3/schema"
	"gorm.io/gorm"
)

type fmtOrder struct {
	ID    uint   `json:"id"`
	Total string `json:"total"`
}

type fmtUser struct {
	ID     uint       `json:"id"`
	Name   string     `json:"name"`
	Orders []fmtOrder `json:"orders"`
}

func TestFormatModelWithEmptyHasMany(t *testing.T) {
	f := DefaultFormatter()

	user := fmtUser{
		ID:     1,
		Name:   "alice",
		Orders: []fmtOrder{},
	}

	schemas := []schema.Schema{
		{Column: "id", Label: "ID", Format: "integer"},
		{Column: "name", Label: "Name", Format: "string"},
		{
			Column:    "orders",
			Label:     "Orders",
			Format:    "relation",
			Relations: schema.Relation{Type: "has_many", Name: "Orders"},
		},
	}

	result := f.FormatModel(context.Background(), reflect.ValueOf(user), schemas, &gorm.Statement{}, FormatRaw)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	orders, ok := m["orders"].([]any)
	if !ok {
		t.Fatalf("expected []any for empty has_many, got %T", m["orders"])
	}
	if len(orders) != 0 {
		t.Errorf("expected empty orders slice, got %v", orders)
	}
}

func TestFormatModelWithHasManyData(t *testing.T) {
	f := DefaultFormatter()

	user := fmtUser{
		ID:   1,
		Name: "alice",
		Orders: []fmtOrder{
			{ID: 10, Total: "100.00"},
			{ID: 11, Total: "200.00"},
		},
	}

	schemas := []schema.Schema{
		{Column: "id", Label: "ID", Format: "integer"},
		{Column: "name", Label: "Name", Format: "string"},
	}

	result := f.FormatModel(context.Background(), reflect.ValueOf(user), schemas, &gorm.Statement{}, FormatRaw)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["id"] != uint(1) {
		t.Errorf("id mismatch: got %v", m["id"])
	}
	if m["name"] != "alice" {
		t.Errorf("name mismatch: got %v", m["name"])
	}
}
