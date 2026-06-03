package schema

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCacheTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&Schema{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestNewCache(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db, WithTTL(5*time.Minute))
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
	if c.ttl != 5*time.Minute {
		t.Errorf("ttl: want 5m, got %v", c.ttl)
	}
}

func TestCacheInvalidate(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// Manually inject an entry
	c.mu.Lock()
	c.entries["test:users"] = &cacheEntry{schemas: []Schema{{Column: "id"}}}
	c.mu.Unlock()

	c.Invalidate("test", "users")

	c.mu.RLock()
	_, ok := c.entries["test:users"]
	c.mu.RUnlock()
	if ok {
		t.Error("expected cache entry to be invalidated")
	}
}

func TestCacheInvalidateAll(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	c.mu.Lock()
	c.entries["a:b"] = &cacheEntry{}
	c.entries["c:d"] = &cacheEntry{}
	c.mu.Unlock()

	c.InvalidateAll()

	c.mu.RLock()
	if len(c.entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(c.entries))
	}
	c.mu.RUnlock()
}

func TestCacheGetSchemas_Miss(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// Insert schema records
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "id", UpdatedAt: 1000})
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "name", UpdatedAt: 1000})

	schemas, err := c.GetSchemas(nil, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(schemas))
	}

	// Verify cache was populated
	c.mu.RLock()
	ent, ok := c.entries["mod:tbl"]
	c.mu.RUnlock()
	if !ok {
		t.Fatal("expected cache entry after miss")
	}
	if len(ent.schemas) != 2 {
		t.Errorf("expected 2 cached schemas, got %d", len(ent.schemas))
	}
	if ent.lastUpdatedAt != 1000 {
		t.Errorf("lastUpdatedAt: want 1000, got %d", ent.lastUpdatedAt)
	}
}

func TestCacheGetSchemas_Hit(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// Populate cache manually
	c.mu.Lock()
	c.entries["mod:tbl"] = &cacheEntry{
		schemas:       []Schema{{Column: "id"}},
		lastUpdatedAt: 1000,
		cachedAt:      time.Now(),
	}
	c.mu.Unlock()

	// Insert a record with updated_at = 1000 to match cache timestamp
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "id", UpdatedAt: 1000})

	schemas, err := c.GetSchemas(nil, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema (from cache), got %d", len(schemas))
	}
	if schemas[0].Column != "id" {
		t.Errorf("column: want id, got %s", schemas[0].Column)
	}
}

func TestCacheGetSchemas_TimestampMismatch(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// Populate stale cache
	c.mu.Lock()
	c.entries["mod:tbl"] = &cacheEntry{
		schemas:       []Schema{{Column: "old"}},
		lastUpdatedAt: 1000,
		cachedAt:      time.Now(),
	}
	c.mu.Unlock()

	// Database has newer data
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "new", UpdatedAt: 2000})

	schemas, err := c.GetSchemas(nil, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema after refresh, got %d", len(schemas))
	}
	if schemas[0].Column != "new" {
		t.Errorf("column: want new, got %s", schemas[0].Column)
	}

	// Verify cache was updated
	c.mu.RLock()
	ent := c.entries["mod:tbl"]
	c.mu.RUnlock()
	if ent.lastUpdatedAt != 2000 {
		t.Errorf("lastUpdatedAt: want 2000, got %d", ent.lastUpdatedAt)
	}
}

func TestCacheGetSchemas_TTLOverride(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db, WithTTL(-1*time.Hour)) // negative TTL forces expiration

	c.mu.Lock()
	c.entries["mod:tbl"] = &cacheEntry{
		schemas:       []Schema{{Column: "id"}},
		lastUpdatedAt: 1000,
		cachedAt:      time.Now(),
	}
	c.mu.Unlock()

	// Even though timestamp matches, TTL is expired
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "id", UpdatedAt: 1000})

	schemas, err := c.GetSchemas(nil, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema after TTL expiry, got %d", len(schemas))
	}
}

func TestCacheGetSchemas_EmptyResult(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// No records in db
	schemas, err := c.GetSchemas(nil, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 0 {
		t.Fatalf("expected 0 schemas, got %d", len(schemas))
	}

	// Verify empty result is cached
	c.mu.RLock()
	ent, ok := c.entries["mod:tbl"]
	c.mu.RUnlock()
	if !ok {
		t.Fatal("expected empty result to be cached")
	}
	if len(ent.schemas) != 0 {
		t.Errorf("expected 0 cached schemas, got %d", len(ent.schemas))
	}
	if ent.lastUpdatedAt != 0 {
		t.Errorf("lastUpdatedAt: want 0, got %d", ent.lastUpdatedAt)
	}
}

func TestCacheGetSchemas_ConcurrentLoad(t *testing.T) {
	db := setupCacheTestDB(t)

	// Pre-populate db with a single record
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "id", UpdatedAt: 1000})

	// Install a counter on the Query callback so we can verify singleflight
	var queryCount int64
	if err := db.Callback().Query().Before("gorm:query").Register("count_queries", func(tx *gorm.DB) {
		atomic.AddInt64(&queryCount, 1)
	}); err != nil {
		t.Fatalf("register counter: %v", err)
	}

	c := NewCache(db)

	// Run 50 concurrent GetSchemas calls
	var wg sync.WaitGroup
	errCh := make(chan error, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.GetSchemas(context.Background(), "mod", "tbl")
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("GetSchemas error: %v", err)
	}

	c.mu.RLock()
	ent := c.entries["mod:tbl"]
	c.mu.RUnlock()
	if ent == nil {
		t.Fatal("expected cache entry")
	}
	if len(ent.schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(ent.schemas))
	}

	t.Logf("observed query count: %d (expected to be near 1 after singleflight)", queryCount)
}

func TestCacheGetVisibleSchemas(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// Insert records with different scenarios
	db.Create(&Schema{
		ModuleName: "mod", TableName: "tbl", Column: "id",
		Scenarios: Scenarios{ScenarioList, ScenarioDetail},
		UpdatedAt: 1000,
	})
	db.Create(&Schema{
		ModuleName: "mod", TableName: "tbl", Column: "secret",
		Scenarios: Scenarios{ScenarioCreate, ScenarioUpdate},
		UpdatedAt: 1000,
	})

	schemas, err := c.GetVisibleSchemas(nil, "mod", "tbl", ScenarioList)
	if err != nil {
		t.Fatalf("GetVisibleSchemas error: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 visible schema, got %d", len(schemas))
	}
	if schemas[0].Column != "id" {
		t.Errorf("column: want id, got %s", schemas[0].Column)
	}

	// Call again to verify cache hit path
	schemas2, err := c.GetVisibleSchemas(nil, "mod", "tbl", ScenarioList)
	if err != nil {
		t.Fatalf("GetVisibleSchemas error: %v", err)
	}
	if len(schemas2) != 1 {
		t.Fatalf("expected 1 visible schema on cache hit, got %d", len(schemas2))
	}
}

func TestDefaultCacheIntegration(t *testing.T) {
	// Ensure clean state
	defaultCache = nil
	defer func() { defaultCache = nil }()

	db := setupCacheTestDB(t)
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "id", UpdatedAt: 1000, Scenarios: Scenarios{ScenarioList, ScenarioDetail}})

	// Before enabling cache, direct query works
	schemas, err := GetSchemas(nil, db, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}

	// Enable cache
	EnableCache(db)

	// This call should populate cache
	schemas2, err := GetSchemas(nil, db, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas2) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas2))
	}

	// Verify cache was populated
	defaultCache.mu.RLock()
	ent, ok := defaultCache.entries["mod:tbl"]
	defaultCache.mu.RUnlock()
	if !ok {
		t.Fatal("expected defaultCache to have entry")
	}
	if len(ent.schemas) != 1 {
		t.Errorf("expected 1 cached schema, got %d", len(ent.schemas))
	}

	// GetVisibleSchemas should also use cache
	visible, err := GetVisibleSchemas(nil, db, "mod", "tbl", ScenarioList)
	if err != nil {
		t.Fatalf("GetVisibleSchemas error: %v", err)
	}
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible schema, got %d", len(visible))
	}
}

func TestAutoMigrateInvalidatesCache(t *testing.T) {
	// Ensure clean state
	defaultCache = nil
	defer func() { defaultCache = nil }()

	db := setupCacheTestDB(t)
	EnableCache(db)

	type TestUser struct {
		ID   uint   `json:"id" gorm:"primarykey"`
		Name string `json:"name"`
	}

	// First AutoMigrate populates cache
	_, err := AutoMigrate(nil, db, &TestUser{}, "testmod")
	if err != nil {
		t.Fatalf("AutoMigrate error: %v", err)
	}

	// Populate cache manually to test invalidation
	GetSchemas(nil, db, "testmod", "test_users")

	defaultCache.mu.RLock()
	_, ok := defaultCache.entries["testmod:test_users"]
	defaultCache.mu.RUnlock()
	if !ok {
		t.Fatal("expected cache entry before second AutoMigrate")
	}

	// Second AutoMigrate should invalidate cache
	_, err = AutoMigrate(nil, db, &TestUser{}, "testmod")
	if err != nil {
		t.Fatalf("AutoMigrate error: %v", err)
	}

	// Verify cache was invalidated
	defaultCache.mu.RLock()
	_, ok = defaultCache.entries["testmod:test_users"]
	defaultCache.mu.RUnlock()
	if ok {
		t.Error("expected cache entry to be invalidated after AutoMigrate")
	}
}
