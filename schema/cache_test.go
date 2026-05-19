package schema

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCacheTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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
