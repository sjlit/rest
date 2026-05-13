package rest

import (
	"context"
	"testing"

	"git.nobla.cn/golang/rest/query"
	"git.nobla.cn/golang/rest/schema"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type pagingUser struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func setupPagingModel(t *testing.T) *Model[pagingUser] {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&schema.Schema{}); err != nil {
		t.Fatalf("failed to migrate schemas table: %v", err)
	}
	ctx := context.Background()
	model, err := NewModel[pagingUser](ctx, WithDB(db), WithModuleName("paging_test"))
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	// Seed
	db.Create(&pagingUser{Name: "Alice", Age: 30})
	db.Create(&pagingUser{Name: "Bob", Age: 25})
	db.Create(&pagingUser{Name: "Charlie", Age: 35})
	db.Create(&pagingUser{Name: "Diana", Age: 28})
	return model
}

func setupPagingModelWithCleanup(t *testing.T) (*Model[pagingUser], func()) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&schema.Schema{}); err != nil {
		t.Fatalf("failed to migrate schemas table: %v", err)
	}
	ctx := context.Background()
	model, err := NewModel[pagingUser](ctx, WithDB(db), WithModuleName("paging_test"))
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	// Seed
	db.Create(&pagingUser{Name: "Alice", Age: 30})
	db.Create(&pagingUser{Name: "Bob", Age: 25})
	db.Create(&pagingUser{Name: "Charlie", Age: 35})
	db.Create(&pagingUser{Name: "Diana", Age: 28})
	return model, func() {
		db.Exec("DELETE FROM paging_users")
	}
}

func TestList(t *testing.T) {
	model, cleanup := setupPagingModelWithCleanup(t)
	defer cleanup()
	ctx := context.Background()

	qb := query.NewBuilder().OrderBy("id", "ASC")
	users, err := model.List(ctx, 0, 2, qb)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
	if users[0].Name != "Alice" {
		t.Errorf("expected Alice, got %s", users[0].Name)
	}
}

func TestListNoSideEffect(t *testing.T) {
	model, cleanup := setupPagingModelWithCleanup(t)
	defer cleanup()
	ctx := context.Background()

	qb := query.NewBuilder().Where("age", query.OpGte, 25).OrderBy("id", "ASC")
	originalLimit := qb.Spec().Limit
	originalOffset := qb.Spec().Offset

	_, err := model.List(ctx, 1, 2, qb)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if qb.Spec().Limit != originalLimit {
		t.Errorf("builder limit mutated: %d -> %d", originalLimit, qb.Spec().Limit)
	}
	if qb.Spec().Offset != originalOffset {
		t.Errorf("builder offset mutated: %d -> %d", originalOffset, qb.Spec().Offset)
	}
}

func TestCount(t *testing.T) {
	model, cleanup := setupPagingModelWithCleanup(t)
	defer cleanup()
	ctx := context.Background()

	qb := query.NewBuilder().Where("age", query.OpGte, 28)
	total, err := model.Count(ctx, qb)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected count 3, got %d", total)
	}
}

func TestCountNoSideEffect(t *testing.T) {
	model, cleanup := setupPagingModelWithCleanup(t)
	defer cleanup()
	ctx := context.Background()

	qb := query.NewBuilder().Where("age", query.OpGte, 25).Limit(2).Offset(1)
	originalLimit := qb.Spec().Limit
	originalOffset := qb.Spec().Offset

	_, err := model.Count(ctx, qb)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if qb.Spec().Limit != originalLimit {
		t.Errorf("builder limit mutated by Count: %d -> %d", originalLimit, qb.Spec().Limit)
	}
	if qb.Spec().Offset != originalOffset {
		t.Errorf("builder offset mutated by Count: %d -> %d", originalOffset, qb.Spec().Offset)
	}
}

func TestPaginate(t *testing.T) {
	model, cleanup := setupPagingModelWithCleanup(t)
	defer cleanup()
	ctx := context.Background()

	qb := query.NewBuilder().OrderBy("id", "ASC")
	result, err := model.Paginate(ctx, 1, 2, qb)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if result.TotalCount != 4 {
		t.Errorf("expected total 4, got %d", result.TotalCount)
	}
	if result.TotalPages != 2 {
		t.Errorf("expected 2 pages, got %d", result.TotalPages)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 users, got %d", len(result.Data))
	}
	if result.Data[0].Name != "Alice" {
		t.Errorf("expected Alice, got %s", result.Data[0].Name)
	}

	// Page 2
	result, err = model.Paginate(ctx, 2, 2, qb)
	if err != nil {
		t.Fatalf("Paginate page 2 failed: %v", err)
	}
	if result.Data[0].Name != "Charlie" {
		t.Errorf("expected Charlie on page 2, got %s", result.Data[0].Name)
	}
}

func TestPaginateNormalization(t *testing.T) {
	model, cleanup := setupPagingModelWithCleanup(t)
	defer cleanup()
	ctx := context.Background()

	qb := query.NewBuilder()
	result, err := model.Paginate(ctx, 0, 0, qb)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if result.Page != 1 {
		t.Errorf("expected page normalized to 1, got %d", result.Page)
	}
	if result.PageSize != 20 {
		t.Errorf("expected size normalized to 20, got %d", result.PageSize)
	}
}

func TestCursor(t *testing.T) {
	model, cleanup := setupPagingModelWithCleanup(t)
	defer cleanup()
	ctx := context.Background()

	qb := query.NewBuilder().OrderBy("id", "ASC")
	result, err := model.Cursor(ctx, "", 2, qb)
	if err != nil {
		t.Fatalf("Cursor failed: %v", err)
	}
	if !result.HasMore {
		t.Error("expected HasMore=true")
	}
	if result.NextCursor == "" {
		t.Error("expected NextCursor non-empty")
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 users, got %d", len(result.Data))
	}

	// Second page using cursor
	result, err = model.Cursor(ctx, result.NextCursor, 2, qb)
	if err != nil {
		t.Fatalf("Cursor page 2 failed: %v", err)
	}
	if result.HasMore {
		t.Error("expected HasMore=false for last page")
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 users on last page, got %d", len(result.Data))
	}

	// When HasMore=false, NextCursor is empty; calling Cursor again with empty cursor restarts from beginning.
	// Cursor pagination stops when HasMore=false.
}

func TestCursorInvalid(t *testing.T) {
	model, cleanup := setupPagingModelWithCleanup(t)
	defer cleanup()
	ctx := context.Background()

	qb := query.NewBuilder()
	_, err := model.Cursor(ctx, "not-valid-base64!!!", 2, qb)
	if err == nil {
		t.Error("expected error for invalid cursor")
	}
}
