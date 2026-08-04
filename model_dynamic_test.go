package rest

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"git.nobla.cn/golang/rest/schema"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type dynUser struct {
	ID   uint   `json:"id" gorm:"primarykey"`
	Name string `json:"name" gorm:"size:100"`
}

type dynOrder struct {
	ID    uint `json:"id" gorm:"primarykey"`
	Total int  `json:"total"`
}

var dynTestDBCounter int

// setupDynamicTestDB 每个测试使用独立的命名内存库,
// 避免 file::memory:?cache=shared 跨测试共享数据导致断言相互干扰
func setupDynamicTestDB(t *testing.T) *gorm.DB {
	dynTestDBCounter++
	dsn := fmt.Sprintf("file:rest_dyn_%d?mode=memory&cache=shared", dynTestDBCounter)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		if strings.Contains(err.Error(), "cgo") || strings.Contains(err.Error(), "stub") {
			t.Skip("sqlite requires cgo")
		}
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&schema.Schema{}); err != nil {
		t.Fatalf("failed to migrate schema table: %v", err)
	}
	return db
}

func TestNewModelWithInstanceCRUDRoundTrip(t *testing.T) {
	db := setupDynamicTestDB(t)
	m, err := NewModel(&dynUser{}, WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}
	if got := m.ModelType(); got != reflect.TypeOf(dynUser{}) {
		t.Fatalf("ModelType: want %v, got %v", reflect.TypeOf(dynUser{}), got)
	}
	ctx := context.Background()

	// Create
	user := &dynUser{Name: "alice"}
	if _, err := m.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("Create did not assign primary key")
	}

	// Detail with type assertion
	v, err := m.Detail(ctx, user.ID)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	got, ok := v.(*dynUser)
	if !ok {
		t.Fatalf("Detail: want *dynUser, got %T", v)
	}
	if got.Name != "alice" {
		t.Errorf("Detail name: want alice, got %q", got.Name)
	}

	// List with type assertion
	lv, err := m.List(ctx, 0, 10, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	list, ok := lv.([]*dynUser)
	if !ok {
		t.Fatalf("List: want []*dynUser, got %T", lv)
	}
	if len(list) != 1 {
		t.Fatalf("List len: want 1, got %d", len(list))
	}

	// Update
	if _, err := m.Update(ctx, user.ID, &dynUser{Name: "bob"}, "name"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	v, err = m.Detail(ctx, user.ID)
	if err != nil {
		t.Fatalf("Detail after Update: %v", err)
	}
	got, _ = v.(*dynUser)
	if got.Name != "bob" {
		t.Errorf("Update name: want bob, got %q", got.Name)
	}

	// Delete
	if _, err := m.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := m.Detail(ctx, user.ID); err == nil {
		t.Fatal("Detail after Delete: want error, got nil")
	}
}

func TestNewModelAcceptsValueInstanceAndReflectType(t *testing.T) {
	db := setupDynamicTestDB(t)
	ctx := context.Background()

	// value instance
	m1, err := NewModel(dynUser{}, WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewModel(value): %v", err)
	}
	if _, err := m1.Create(ctx, &dynUser{Name: "value"}); err != nil {
		t.Fatalf("Create via value-instance model: %v", err)
	}

	// reflect.Type
	m2, err := NewModel(reflect.TypeOf(dynUser{}), WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewModel(reflect.Type): %v", err)
	}
	if got := m2.ModelType(); got != reflect.TypeOf(dynUser{}) {
		t.Fatalf("ModelType: want %v, got %v", reflect.TypeOf(dynUser{}), got)
	}
	if _, err := m2.Create(ctx, &dynUser{Name: "type"}); err != nil {
		t.Fatalf("Create via reflect.Type model: %v", err)
	}
}

func TestNewModelRejectsInvalidInput(t *testing.T) {
	db := setupDynamicTestDB(t)
	if _, err := NewModel(nil, WithDB(db)); err == nil {
		t.Error("nil: want error, got nil")
	}
	if _, err := NewModel(42, WithDB(db)); err == nil {
		t.Error("non-struct int: want error, got nil")
	}
	if _, err := NewModel("dyn", WithDB(db)); err == nil {
		t.Error("non-struct string: want error, got nil")
	}
}

func TestModelCreateTypeMismatch(t *testing.T) {
	db := setupDynamicTestDB(t)
	m, err := NewModel(&dynUser{}, WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	_, err = m.Create(context.Background(), &dynOrder{Total: 1})
	if err == nil {
		t.Fatal("Create with wrong type: want error, got nil")
	}
	if !strings.Contains(err.Error(), "type mismatch") {
		t.Fatalf("Create with wrong type: want 'type mismatch' error, got: %v", err)
	}
}

func TestModelDynamicHooks(t *testing.T) {
	db := setupDynamicTestDB(t)
	m, err := NewModel(&dynUser{}, WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}

	var (
		beforeCalled bool
		gotModel     any
		afterDiff    []*DiffAttr
	)
	m.RegisterBeforeCreate(func(ctx context.Context, db *gorm.DB, model any) error {
		beforeCalled = true
		gotModel = model
		if u, ok := model.(*dynUser); ok && u.Name == "" {
			u.Name = "hooked"
		}
		return nil
	})
	m.RegisterAfterCreate(func(ctx context.Context, db *gorm.DB, model any, diffAttrs []*DiffAttr) {
		afterDiff = diffAttrs
	})

	created := &dynUser{}
	if _, err := m.Create(context.Background(), created); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !beforeCalled {
		t.Error("before create hook not called")
	}
	if _, ok := gotModel.(*dynUser); !ok {
		t.Errorf("hook model: want *dynUser, got %T", gotModel)
	}
	if afterDiff == nil {
		t.Error("after create hook not called")
	}

	// hook mutation must have been persisted
	v, err := m.Detail(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if got, ok := v.(*dynUser); !ok || got.Name != "hooked" {
		t.Errorf("hook mutation: want name hooked, got %+v", v)
	}
}

func TestModelBatchInstantiation(t *testing.T) {
	db := setupDynamicTestDB(t)
	ctx := context.Background()
	models := []any{&dynUser{}, &dynOrder{}}
	for _, model := range models {
		m, err := NewModel(model, WithDB(db), WithModuleName("dyn"))
		if err != nil {
			t.Fatalf("NewModel(%T): %v", model, err)
		}
		want := reflect.TypeOf(model).Elem()
		if got := m.ModelType(); got != want {
			t.Errorf("ModelType: want %v, got %v", want, got)
		}
		// CRUD via a reflect-created instance proves the loop works end to end
		inst := reflect.New(want).Interface()
		if _, err := m.Create(ctx, inst); err != nil {
			t.Fatalf("Create(%T): %v", model, err)
		}
		id := reflect.ValueOf(inst).Elem().FieldByName("ID").Interface()
		v, err := m.Detail(ctx, id)
		if err != nil {
			t.Fatalf("Detail(%T): %v", model, err)
		}
		if reflect.TypeOf(v) != reflect.PointerTo(want) {
			t.Errorf("Detail(%T): want %v, got %T", model, reflect.PointerTo(want), v)
		}
	}
}

func TestModelDynamicPaginationAndCursor(t *testing.T) {
	db := setupDynamicTestDB(t)
	m, err := NewModel(&dynUser{}, WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if _, err := m.Create(ctx, &dynUser{Name: fmt.Sprintf("u%d", i)}); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	total, pageData, err := m.Paginate(ctx, 0, 2, nil)
	if err != nil {
		t.Fatalf("Paginate: %v", err)
	}
	if total != 5 {
		t.Errorf("Paginate total: want 5, got %d", total)
	}
	page, ok := pageData.([]*dynUser)
	if !ok {
		t.Fatalf("Paginate: want []*dynUser, got %T", pageData)
	}
	if len(page) != 2 {
		t.Errorf("Paginate page len: want 2, got %d", len(page))
	}

	next, hasMore, cursorData, err := m.Cursor(ctx, "", 2, nil)
	if err != nil {
		t.Fatalf("Cursor: %v", err)
	}
	if !hasMore || next == "" {
		t.Errorf("Cursor: want hasMore=true and non-empty next, got hasMore=%v next=%q", hasMore, next)
	}
	cursorList, ok := cursorData.([]*dynUser)
	if !ok {
		t.Fatalf("Cursor: want []*dynUser, got %T", cursorData)
	}
	if len(cursorList) != 2 {
		t.Errorf("Cursor len: want 2, got %d", len(cursorList))
	}
}
