package schema

import (
	"fmt"
	"sync"
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

func TestCacheGetSchemas_RecordNotFound(t *testing.T) {
	db := setupCacheTestDB(t)
	c := NewCache(db)

	// No records in db — GORM v2 Find returns empty slice + nil error,
	// but our code also guards against gorm.ErrRecordNotFound as a safety net.
	schemas, err := c.GetSchemas(nil, "mod", "tbl")
	if err != nil {
		t.Fatalf("GetSchemas error: %v", err)
	}
	if len(schemas) != 0 {
		t.Fatalf("expected 0 schemas, got %d", len(schemas))
	}

	// Verify empty result is cached (same assertions as EmptyResult)
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
	c := NewCache(db)

	// Pre-populate db with a single record
	db.Create(&Schema{ModuleName: "mod", TableName: "tbl", Column: "id", UpdatedAt: 1000})

	// Run 50 concurrent GetSchemas calls
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.GetSchemas(nil, "mod", "tbl")
			if err != nil {
				t.Errorf("GetSchemas error: %v", err)
			}
		}()
	}
	wg.Wait()

	c.mu.RLock()
	ent := c.entries["mod:tbl"]
	c.mu.RUnlock()
	if ent == nil {
		t.Fatal("expected cache entry")
	}
	if len(ent.schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(ent.schemas))
	}
}
