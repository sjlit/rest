package rest

import (
	"context"
	"errors"
	"testing"

	"github.com/sjlit/rest/v3/schema"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type hookTestModel struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"size:100"`
	Value int
}

func setupHookTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	// AutoMigrate the schema table so that schema.AutoMigrate doesn't fail
	if err := db.AutoMigrate(&schema.Schema{}); err != nil {
		t.Fatalf("failed to migrate schema table: %v", err)
	}
	return db
}

func TestGlobalBeforeCreateBlocks(t *testing.T) {
	globalBeforeCreate = nil // Clean up
	RegisterBeforeCreate(func(ctx context.Context, db *gorm.DB, model any) error {
		return errors.New("blocked by before create")
	})
	defer func() { globalBeforeCreate = nil }()

	db := setupHookTestDB(t)
	model, err := NewTypedModel[hookTestModel](WithDB(db), WithModuleName("test"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}

	model.initLocalHooks()

	m := hookTestModel{Name: "test", Value: 1}
	_, err = model.Create(context.Background(), &m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "blocked by before create" {
		t.Errorf("expected 'blocked by before create', got %q", err.Error())
	}
}

func TestGlobalAfterCreateExecutes(t *testing.T) {
	globalAfterCreate = nil
	var capturedName string
	RegisterAfterCreate(func(ctx context.Context, db *gorm.DB, model any, diffAttrs []*DiffAttr) {
		if m, ok := model.(*hookTestModel); ok {
			capturedName = m.Name
		}
	})
	defer func() { globalAfterCreate = nil }()

	db := setupHookTestDB(t)
	model, err := NewTypedModel[hookTestModel](WithDB(db), WithModuleName("test"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}
	model.initLocalHooks()

	m := hookTestModel{Name: "alice", Value: 1}
	_, err = model.Create(context.Background(), &m)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if capturedName != "alice" {
		t.Errorf("expected capturedName 'alice', got %q", capturedName)
	}
}

func TestLocalAfterCreateExecutes(t *testing.T) {
	globalAfterCreate = nil

	db := setupHookTestDB(t)
	model, err := NewTypedModel[hookTestModel](WithDB(db), WithModuleName("test"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}
	model.initLocalHooks()

	var capturedName string
	model.RegisterAfterCreate(func(ctx context.Context, db *gorm.DB, m *hookTestModel, diffAttrs []*DiffAttr) {
		capturedName = m.Name
	})

	m := hookTestModel{Name: "bob", Value: 2}
	_, err = model.Create(context.Background(), &m)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if capturedName != "bob" {
		t.Errorf("expected capturedName 'bob', got %q", capturedName)
	}
}

func TestCreateHookExecutionOrder(t *testing.T) {
	globalAfterCreate = nil
	var order []string
	RegisterAfterCreate(func(ctx context.Context, db *gorm.DB, model any, diffAttrs []*DiffAttr) {
		order = append(order, "global")
	})
	defer func() { globalAfterCreate = nil }()

	db := setupHookTestDB(t)
	model, err := NewTypedModel[hookTestModel](WithDB(db), WithModuleName("test"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}
	model.initLocalHooks()

	model.RegisterAfterCreate(func(ctx context.Context, db *gorm.DB, m *hookTestModel, diffAttrs []*DiffAttr) {
		order = append(order, "local")
	})

	m := hookTestModel{Name: "charlie", Value: 3}
	_, err = model.Create(context.Background(), &m)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if len(order) != 2 || order[0] != "global" || order[1] != "local" {
		t.Errorf("expected order [global, local], got %v", order)
	}
}

func TestAfterCreatePanicRecover(t *testing.T) {
	globalAfterCreate = nil
	var afterPanic bool
	RegisterAfterCreate(func(ctx context.Context, db *gorm.DB, model any, diffAttrs []*DiffAttr) {
		panic("intentional panic")
	})
	RegisterAfterCreate(func(ctx context.Context, db *gorm.DB, model any, diffAttrs []*DiffAttr) {
		afterPanic = true
	})
	defer func() { globalAfterCreate = nil }()

	db := setupHookTestDB(t)
	model, err := NewTypedModel[hookTestModel](WithDB(db), WithModuleName("test"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}
	model.initLocalHooks()

	m := hookTestModel{Name: "dave", Value: 4}
	_, err = model.Create(context.Background(), &m)
	if err != nil {
		t.Fatalf("Create should not error after panic recovery, got: %v", err)
	}
	if !afterPanic {
		t.Error("expected second AfterCreate hook to still execute after panic")
	}
}

func TestDeleteHookReceivesFullModel(t *testing.T) {
	globalAfterDelete = nil
	var capturedName string
	RegisterAfterDelete(func(ctx context.Context, db *gorm.DB, model any) {
		if m, ok := model.(*hookTestModel); ok {
			capturedName = m.Name
		}
	})
	defer func() { globalAfterDelete = nil }()

	db := setupHookTestDB(t)
	model, err := NewTypedModel[hookTestModel](WithDB(db), WithModuleName("test"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}
	model.initLocalHooks()

	// Create a record first
	m := hookTestModel{Name: "eve", Value: 5}
	_, err = model.Create(context.Background(), &m)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if m.ID == 0 {
		t.Fatal("expected ID to be set after create")
	}

	// Delete it
	_, err = model.Delete(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if capturedName != "eve" {
		t.Errorf("expected capturedName 'eve', got %q", capturedName)
	}
}
