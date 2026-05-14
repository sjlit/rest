package schema

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestParseFieldRelations(t *testing.T) {
	type Order struct {
		ID     uint
		UserID uint
	}
	type User struct {
		ID     uint
		Name   string
		Orders []Order `gorm:"foreignKey:UserID" relation:"has_many:Orders:order:orders"`
	}

	s, err := schema.Parse(&User{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var ordersField *schema.Field
	for _, f := range s.Fields {
		if f.Name == "Orders" {
			ordersField = f
			break
		}
	}
	if ordersField == nil {
		t.Fatal("Orders field not found")
	}

	rel := parseFieldRelations(ordersField)
	if rel.Type != "has_many" {
		t.Errorf("type: want has_many, got %s", rel.Type)
	}
	if rel.Name != "Orders" {
		t.Errorf("name: want Orders, got %s", rel.Name)
	}
	if rel.Module != "order" {
		t.Errorf("module: want order, got %s", rel.Module)
	}
	if rel.Table != "orders" {
		t.Errorf("table: want orders, got %s", rel.Table)
	}
}

func TestParseFieldRelationsNoTag(t *testing.T) {
	type User struct {
		ID   uint
		Name string
	}

	s, err := schema.Parse(&User{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var nameField *schema.Field
	for _, f := range s.Fields {
		if f.Name == "Name" {
			nameField = f
			break
		}
	}
	if nameField == nil {
		t.Fatal("Name field not found")
	}

	rel := parseFieldRelations(nameField)
	if rel.Type != "" {
		t.Errorf("expected empty relation for field without tag, got %+v", rel)
	}
}
