package rest

import (
	"context"
	"testing"

	"git.nobla.cn/golang/rest/schema"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type IntegOrder struct {
	ID     uint `json:"id" gorm:"primarykey"`
	UserID uint `json:"user_id"`
	Total  int  `json:"total"`
}

type IntegUser struct {
	ID     uint         `json:"id" gorm:"primarykey"`
	Name   string       `json:"name" gorm:"size:100"`
	Orders []IntegOrder `json:"orders" gorm:"foreignKey:UserID" relation:"has_many:Orders::integ_orders"`
}

func TestIntegrationPreloadDetail(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&schema.Schema{}, &IntegUser{}, &IntegOrder{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	userModel, err := NewModel[IntegUser](WithDB(db), WithModuleName("integration"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}

	// Verify orders schema was auto-discovered
	allSchemas, err := schema.GetSchemas(context.Background(), db, "integration", "integ_users")
	if err != nil {
		t.Fatalf("GetSchemas failed: %v", err)
	}
	var ordersSchema *schema.Schema
	for i := range allSchemas {
		if allSchemas[i].Column == "Orders" {
			ordersSchema = &allSchemas[i]
			break
		}
	}
	if ordersSchema == nil {
		t.Fatal("Orders schema not found after AutoMigrate")
	}
	if ordersSchema.Relations.Type != "has_many" {
		t.Fatalf("expected orders relation type has_many, got %s", ordersSchema.Relations.Type)
	}

	// Create test data
	user := IntegUser{Name: "alice", Orders: []IntegOrder{{Total: 100}, {Total: 200}}}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	// Detail query should load Orders
	result, err := userModel.Detail(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("Detail failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Name != "alice" {
		t.Errorf("name mismatch: want alice, got %s", result.Name)
	}
	if len(result.Orders) != 2 {
		t.Errorf("orders count: want 2, got %d", len(result.Orders))
	}
	if result.Orders[0].Total != 100 {
		t.Errorf("first order total: want 100, got %d", result.Orders[0].Total)
	}
}

func TestIntegrationPreloadList(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&schema.Schema{}, &IntegUser{}, &IntegOrder{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	userModel, err := NewModel[IntegUser](WithDB(db), WithModuleName("integration"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}

	// Create test data
	db.Create(&IntegUser{Name: "bob", Orders: []IntegOrder{{Total: 300}}})
	db.Create(&IntegUser{Name: "carol", Orders: []IntegOrder{{Total: 400}, {Total: 500}}})

	// List query should load Orders (limit 10, but auto-migrate creates a schema row too)
	list, err := userModel.List(context.Background(), 0, 10, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) < 2 {
		t.Fatalf("expected at least 2 users, got %d", len(list))
	}

	foundBob := false
	for _, u := range list {
		if u.Name == "bob" {
			foundBob = true
			if len(u.Orders) != 1 {
				t.Errorf("bob orders: want 1, got %d", len(u.Orders))
			}
		}
	}
	if !foundBob {
		t.Error("bob not found in list")
	}
}
