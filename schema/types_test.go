package schema

import (
	"testing"
)

func TestRelationScanValue(t *testing.T) {
	rel := Relation{
		Type:   "has_many",
		Name:   "Orders",
		Module: "order",
		Table:  "orders",
	}

	val, err := rel.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}

	scanned := Relation{}
	if err := scanned.Scan(val); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if scanned.Type != "has_many" {
		t.Errorf("type: want has_many, got %s", scanned.Type)
	}
	if scanned.Name != "Orders" {
		t.Errorf("name: want Orders, got %s", scanned.Name)
	}
	if scanned.Module != "order" {
		t.Errorf("module: want order, got %s", scanned.Module)
	}
	if scanned.Table != "orders" {
		t.Errorf("table: want orders, got %s", scanned.Table)
	}
}

func TestRelationScanNil(t *testing.T) {
	var rel Relation
	if err := rel.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) error: %v", err)
	}
	if rel.Type != "" || rel.Name != "" {
		t.Errorf("expected empty relation after Scan(nil), got %+v", rel)
	}
}
