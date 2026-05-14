package rest

import (
	"context"
	"testing"

	"git.nobla.cn/golang/rest/schema"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupModelTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&schema.Schema{}); err != nil {
		t.Fatalf("failed to migrate schema table: %v", err)
	}
	return db
}

func TestApplyPreloadsBasic(t *testing.T) {
	db := setupModelTestDB(t)
	model, err := NewModel[hookTestModel](WithDB(db), WithModuleName("test"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}

	schemas := []schema.Schema{
		{Column: "id", Relations: schema.Relation{}},
		{Column: "orders", Relations: schema.Relation{Type: "has_many", Name: "Orders"}},
	}

	resultDB := model.applyPreloads(context.Background(), db, schemas, schema.ScenarioDetail, make(map[string]bool), "")
	if resultDB == nil {
		t.Fatal("expected non-nil db")
	}
}
