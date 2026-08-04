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
	model, err := NewTypedModel[hookTestModel](WithDB(db), WithModuleName("test"))
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

type updateTestModel struct {
	ID     uint   `gorm:"primaryKey"`
	Name   string `gorm:"size:100"`
	Secret string `gorm:"size:100;<-:create"` // create-only, protected by Disable=[update]
}

func TestUpdateSkipsDisabledFields(t *testing.T) {
	db := setupModelTestDB(t)
	m, err := NewTypedModel[updateTestModel](WithDB(db), WithModuleName("upd_test"))
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	// Seed a record directly via gorm
	seed := updateTestModel{Name: "alice", Secret: "old-secret"}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed create: %v", err)
	}

	// Update via Model: even though caller asks for these columns, they must be ignored
	update := updateTestModel{Name: "bob", Secret: "new-secret"}
	_, err = m.Update(context.Background(), seed.ID, &update, "name", "secret")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	var got updateTestModel
	if err := db.First(&got, seed.ID).Error; err != nil {
		t.Fatalf("First: %v", err)
	}
	if got.Name != "bob" {
		t.Errorf("Name should be updated: want 'bob', got %q", got.Name)
	}
	if got.Secret != "old-secret" {
		t.Errorf("Secret (create-only) must NOT be updated: want 'old-secret', got %q", got.Secret)
	}
}
