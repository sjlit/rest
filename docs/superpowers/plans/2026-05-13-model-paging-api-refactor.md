# Model[T] Paging API Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `Model[T].Find` with `List/Paginate/Cursor/Count` methods and eliminate `Builder` side effects via `Clone`.

**Architecture:** Keep `Builder` mutable but add `Clone()` for isolation. `Query.Count` clones internally. `Model` methods clone when they need to modify builder state. Three paging modes: `List` (offset/limit, no count), `Paginate` (page/size with auto-count), `Cursor` (base64-encoded offset, load-more style).

**Tech Stack:** Go 1.25, GORM v2, SQLite (tests)

---

## File Structure

| File | Responsibility |
|------|----------------|
| `rest/types.go` | Add `PageResult[T]` and `CursorResult[T]` types |
| `query/ast.go` | Add `QuerySpec.Clone()` method |
| `query/builder.go` | Add `Builder.Clone()` method |
| `query/query.go` | Refactor `Count` to use cloned spec; extract `compileWithSpec` |
| `rest/model.go` | Delete `Find`; add `List`, `Count`, `Paginate`, `Cursor` |
| `rest/resource.go` | Update `Search` to call `Paginate` instead of `Find` |
| `rest/model_test.go` | **Create** — tests for all new `Model` paging methods |
| `query/query_test.go` | Update `TestPage` references if needed (likely no change needed) |

---

## Task 1: Add Result Types, QuerySpec.Clone, and Builder.Clone

**Files:**
- Modify: `rest/types.go`
- Modify: `query/ast.go`
- Modify: `query/builder.go`
- Test: `query/builder_test.go` (add Clone test)

- [ ] **Step 1: Add PageResult and CursorResult to types.go**

Append to `rest/types.go` (after `SearchResult`):

```go
type PageResult[T any] struct {
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalCount int64  `json:"total_count"`
	TotalPages int    `json:"total_pages"`
	Data       []*T   `json:"data"`
}

type CursorResult[T any] struct {
	Data       []*T   `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}
```

- [ ] **Step 2: Add QuerySpec.Clone() to ast.go**

Append to `query/ast.go` (after `QuerySpec` struct):

```go
func (s QuerySpec) Clone() QuerySpec {
	cloned := QuerySpec{
		Source: s.Source,
		Limit:  s.Limit,
		Offset: s.Offset,
	}
	if len(s.Selects) > 0 {
		cloned.Selects = make([]Expr, len(s.Selects))
		copy(cloned.Selects, s.Selects)
	}
	if len(s.Joins) > 0 {
		cloned.Joins = make([]Join, len(s.Joins))
		copy(cloned.Joins, s.Joins)
	}
	if len(s.Where) > 0 {
		cloned.Where = make([]Clause, len(s.Where))
		copy(cloned.Where, s.Where)
	}
	if len(s.GroupBy) > 0 {
		cloned.GroupBy = make([]string, len(s.GroupBy))
		copy(cloned.GroupBy, s.GroupBy)
	}
	if len(s.Having) > 0 {
		cloned.Having = make([]Clause, len(s.Having))
		copy(cloned.Having, s.Having)
	}
	if len(s.OrderBy) > 0 {
		cloned.OrderBy = make([]Order, len(s.OrderBy))
		copy(cloned.OrderBy, s.OrderBy)
	}
	return cloned
}
```

- [ ] **Step 3: Add Builder.Clone() to builder.go**

Append to `query/builder.go`:

```go
func (b *Builder) Clone() *Builder {
	return &Builder{spec: b.spec.Clone()}
}
```

- [ ] **Step 4: Write test for Builder.Clone()**

Add to `query/builder_test.go`:

```go
func TestBuilderClone(t *testing.T) {
	b := NewBuilder().Where("age", OpGt, 18).Limit(10).Offset(5)
	cloned := b.Clone()

	// Modify clone should not affect original
	cloned.Where("name", OpEq, "alice").Limit(20)

	if len(b.Spec().Where) != 1 {
		t.Errorf("original where mutated: expected 1, got %d", len(b.Spec().Where))
	}
	if b.Spec().Limit != 10 {
		t.Errorf("original limit mutated: expected 10, got %d", b.Spec().Limit)
	}
	if cloned.Spec().Limit != 20 {
		t.Errorf("clone limit wrong: expected 20, got %d", cloned.Spec().Limit)
	}
	if len(cloned.Spec().Where) != 2 {
		t.Errorf("clone where wrong: expected 2, got %d", len(cloned.Spec().Where))
	}
}
```

- [ ] **Step 5: Run builder tests**

```bash
cd /mobe/workspace/rest
go test ./query/ -run TestBuilderClone -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add rest/types.go query/ast.go query/builder.go query/builder_test.go
git commit -m "feat: add PageResult, CursorResult, QuerySpec.Clone, Builder.Clone"
```

---

## Task 2: Refactor Query.Count to Eliminate Side Effects

**Files:**
- Modify: `query/query.go`
- Test: `query/query_test.go`

- [ ] **Step 1: Modify Count and extract compileWithSpec**

Replace the `compile` method and `Count` method in `query/query.go`:

```go
func (q *Query) compile(ctx context.Context) (*gorm.DB, error) {
	return q.compileWithSpec(ctx, q.builder.Spec())
}

func (q *Query) compileWithSpec(ctx context.Context, spec QuerySpec) (*gorm.DB, error) {
	if err := Validate(spec); err != nil {
		return nil, err
	}
	db := q.db.WithContext(ctx)
	if q.model != nil {
		db = db.Model(q.model)
	}
	compiler := NewCompiler()
	return compiler.Compile(spec, db)
}
```

Then replace the `Count` method:

```go
func (q *Query) Count(ctx context.Context) (int64, error) {
	spec := q.builder.Spec().Clone()
	spec.Offset = 0
	spec.Limit = 0

	db, err := q.compileWithSpec(ctx, spec)
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.Count(&count).Error
	return count, err
}
```

- [ ] **Step 2: Verify Count no longer mutates builder — add test**

Add to `query/query_test.go`:

```go
func TestCountDoesNotMutateBuilder(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	b := NewBuilder().Where("age", OpGte, 25).Limit(2).Offset(1)
	q := New(db, &testUser{}, b)

	_, err := q.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	// Builder should remain untouched
	if b.Spec().Limit != 2 {
		t.Errorf("builder limit mutated by Count: expected 2, got %d", b.Spec().Limit)
	}
	if b.Spec().Offset != 1 {
		t.Errorf("builder offset mutated by Count: expected 1, got %d", b.Spec().Offset)
	}
}
```

- [ ] **Step 3: Run query tests**

```bash
cd /mobe/workspace/rest
go test ./query/ -v
```

Expected: All PASS (including existing tests)

- [ ] **Step 4: Commit**

```bash
git add query/query.go query/query_test.go
git commit -m "refactor: Query.Count clones spec to eliminate side effects"
```

---

## Task 3: Add Model.List and Model.Count

**Files:**
- Modify: `rest/model.go`
- Create: `rest/model_test.go`

- [ ] **Step 1: Add cursor helper functions to model.go**

Add at the top of `rest/model.go` (after imports):

```go
import (
	"encoding/base64"
	"strconv"
)

func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeCursor(cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(b))
}
```

Wait — model.go already imports `context`, `reflect`, `slices`. We need to add `encoding/base64` and `strconv` to the imports.

Replace the import block in `rest/model.go`:

```go
import (
	"context"
	"encoding/base64"
	"reflect"
	"slices"
	"strconv"

	"git.nobla.cn/golang/rest/internal/inflector"
	"git.nobla.cn/golang/rest/query"
	"git.nobla.cn/golang/rest/schema"
	"gorm.io/gorm"
	gormSchema "gorm.io/gorm/schema"
)
```

- [ ] **Step 2: Add List and Count methods to model.go**

Replace the `Find` method in `rest/model.go` with `List` and `Count` (delete `Find` first, then add the new methods below `Detail`):

```go
func (m *Model[T]) List(ctx context.Context, offset, limit int, queryBuilder *query.Builder) ([]*T, error) {
	if !m.HasScenario(schema.ScenarioSearch) {
		return nil, ErrPermissionDenied
	}
	var (
		model   T
		schemas []schema.Schema
		err     error
	)
	if schemas, err = schema.GetVisibleSchemas(ctx, m.GetDB(), m.naming.ModuleName, m.naming.TableName, schema.ScenarioList); err != nil {
		return nil, err
	}
	childCtx := WithRuntimeScope(ctx, &RuntimeScope{
		ModuleName: m.naming.ModuleName,
		TableName:  m.naming.TableName,
		Scenario:   schema.ScenarioList,
		Schemas:    schemas,
		Context:    m.ctx,
	})

	listBuilder := queryBuilder.Clone()
	if offset >= 0 {
		listBuilder.Offset(offset)
	}
	if limit > 0 {
		listBuilder.Limit(limit)
	}

	search := query.New(m.GetDB(), model, listBuilder)
	values := make([]*T, 0)
	if err = search.All(childCtx, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func (m *Model[T]) Count(ctx context.Context, queryBuilder *query.Builder) (int64, error) {
	if !m.HasScenario(schema.ScenarioSearch) {
		return 0, ErrPermissionDenied
	}
	var model T
	childCtx := WithRuntimeScope(ctx, &RuntimeScope{
		ModuleName: m.naming.ModuleName,
		TableName:  m.naming.TableName,
		Scenario:   schema.ScenarioList,
		Context:    m.ctx,
	})
	search := query.New(m.GetDB(), model, queryBuilder)
	return search.Count(childCtx)
}
```

- [ ] **Step 3: Create model_test.go with List and Count tests**

Create `rest/model_test.go`:

```go
package rest

import (
	"context"
	"testing"

	"git.nobla.cn/golang/rest/query"
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

func TestList(t *testing.T) {
	model := setupPagingModel(t)
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
	model := setupPagingModel(t)
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
	model := setupPagingModel(t)
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
	model := setupPagingModel(t)
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
```

- [ ] **Step 4: Run model tests**

```bash
cd /mobe/workspace/rest
go test ./rest/ -run TestList -v
go test ./rest/ -run TestCount -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add rest/model.go rest/model_test.go
git commit -m "feat: add Model.List and Model.Count with no side effects"
```

---

## Task 4: Add Model.Paginate and Model.Cursor

**Files:**
- Modify: `rest/model.go`
- Modify: `rest/model_test.go`

- [ ] **Step 1: Add Paginate and Cursor methods to model.go**

Append to `rest/model.go` (after `Count`):

```go
func (m *Model[T]) Paginate(ctx context.Context, page, size int, queryBuilder *query.Builder) (*PageResult[T], error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}

	totalCount, err := m.Count(ctx, queryBuilder)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * size
	data, err := m.List(ctx, offset, size, queryBuilder)
	if err != nil {
		return nil, err
	}

	totalPages := int((totalCount + int64(size) - 1) / int64(size))

	return &PageResult[T]{
		Page:       page,
		PageSize:   size,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Data:       data,
	}, nil
}

func (m *Model[T]) Cursor(ctx context.Context, cursor string, limit int, queryBuilder *query.Builder) (*CursorResult[T], error) {
	offset, err := decodeCursor(cursor)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}

	data, err := m.List(ctx, offset, limit+1, queryBuilder)
	if err != nil {
		return nil, err
	}

	hasMore := len(data) > limit
	if hasMore {
		data = data[:limit]
	}

	nextCursor := ""
	if hasMore {
		nextCursor = encodeCursor(offset + limit)
	}

	return &CursorResult[T]{
		Data:       data,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}
```

- [ ] **Step 2: Add Paginate and Cursor tests to model_test.go**

Append to `rest/model_test.go`:

```go
func TestPaginate(t *testing.T) {
	model := setupPagingModel(t)
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
	model := setupPagingModel(t)
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
	model := setupPagingModel(t)
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
}

func TestCursorInvalid(t *testing.T) {
	model := setupPagingModel(t)
	ctx := context.Background()

	qb := query.NewBuilder()
	_, err := model.Cursor(ctx, "not-valid-base64!!!", 2, qb)
	if err == nil {
		t.Error("expected error for invalid cursor")
	}
}
```

- [ ] **Step 3: Run all model tests**

```bash
cd /mobe/workspace/rest
go test ./rest/ -v
```

Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add rest/model.go rest/model_test.go
git commit -m "feat: add Model.Paginate and Model.Cursor"
```

---

## Task 5: Update Resource.Search to Use Paginate

**Files:**
- Modify: `rest/resource.go`
- Modify: `rest/resource.go` (delete commented Find references)

- [ ] **Step 1: Replace Resource.Search implementation**

Replace the `Search` method in `rest/resource.go` with:

```go
func (r *Resource[T]) Search(res http.ResponseWriter, req *http.Request) {
	var (
		err        error
		pageIndex  int
		pageSize   int
		schemas    []schema.Schema
		result     *PageResult[T]
	)
	pageIndex, _ = strconv.Atoi(req.URL.Query().Get("page"))
	pageSize, _ = strconv.Atoi(req.URL.Query().Get("page_size"))
	if pageSize <= 0 {
		pageSize = 15
	}
	if pageIndex > 0 {
		pageIndex--
	}
	if pageIndex < 0 {
		pageIndex = 0
	}
	if schemas, err = schema.GetVisibleSchemas(req.Context(), r.model.GetDB(), r.model.GetNaming().ModuleName, r.model.GetNaming().TableName, schema.ScenarioSearch); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	queryBuilder := r.buildQuery(req, schemas)
	if result, err = r.model.Paginate(req.Context(), pageIndex+1, pageSize, queryBuilder); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	if r.formatter != nil {
		result.Data = r.formatter.FormatModels(req.Context(), result.Data, schemas, r.model.GetDB().Statement, "")
	}
	r.Respond(res, req, result)
}
```

- [ ] **Step 2: Verify resource.go compiles**

```bash
cd /mobe/workspace/rest
go build ./rest/
```

Expected: Success (no errors)

- [ ] **Step 3: Commit**

```bash
git add rest/resource.go
git commit -m "refactor: Resource.Search uses Model.Paginate instead of Find"
```

---

## Task 6: Delete Model.Find and Final Verification

**Files:**
- Modify: `rest/model.go`

- [ ] **Step 1: Delete the old Find method from model.go**

Remove the entire `Find` method (lines 210-241 approximately) from `rest/model.go`. The method signature is:

```go
func (m *Model[T]) Find(ctx context.Context, offset, limit int, queryBuilder *query.Builder) (totalCount int64, values []*T, err error) {
```

Delete from `if !m.HasScenario(schema.ScenarioSearch)` through the closing `}`.

- [ ] **Step 2: Verify build and all tests pass**

```bash
cd /mobe/workspace/rest
go build ./...
go test ./... -v
```

Expected: All packages build successfully, all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add rest/model.go
git commit -m "BREAKING CHANGE: remove Model.Find (replaced by List/Paginate/Cursor/Count)"
```

---

## Self-Review Checklist

**1. Spec coverage:**
- ✅ `List` — Task 3
- ✅ `Paginate` — Task 4
- ✅ `Cursor` — Task 4
- ✅ `Count` — Task 3
- ✅ `QuerySpec.Clone()` — Task 1
- ✅ `Builder.Clone()` — Task 1
- ✅ `Query.Count` side-effect-free — Task 2
- ✅ `Resource.Search` uses `Paginate` — Task 5
- ✅ `Find` deleted — Task 6

**2. Placeholder scan:**
- No "TBD", "TODO", "implement later" found
- No vague "add error handling" steps
- All steps contain concrete code or exact commands

**3. Type consistency:**
- `PageResult[T]` and `CursorResult[T]` defined in Task 1, used in Task 4
- `QuerySpec.Clone()` defined in Task 1, used in Task 2
- `Builder.Clone()` defined in Task 1, used in Task 3
- Method signatures match across all tasks
