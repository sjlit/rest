package schema

import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
)

type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	db      *gorm.DB
}

type cacheEntry struct {
	schemas       []Schema
	lastUpdatedAt int64
	cachedAt      time.Time
}

type CacheOption func(*Cache)

func WithTTL(d time.Duration) CacheOption {
	return func(c *Cache) {
		c.ttl = d
	}
}

func NewCache(db *gorm.DB, opts ...CacheOption) *Cache {
	c := &Cache{
		entries: make(map[string]*cacheEntry),
		db:      db,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Cache) key(module, table string) string {
	return module + ":" + table
}

func (c *Cache) Invalidate(module, table string) {
	c.mu.Lock()
	delete(c.entries, c.key(module, table))
	c.mu.Unlock()
}

func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	c.entries = make(map[string]*cacheEntry)
	c.mu.Unlock()
}

func (c *Cache) GetSchemas(ctx context.Context, moduleName, tableName string) ([]Schema, error) {
	key := c.key(moduleName, tableName)

	// Step 1: read-lock check
	c.mu.RLock()
	ent, ok := c.entries[key]
	c.mu.RUnlock()

	if ok {
		// TTL check
		if c.ttl > 0 && time.Since(ent.cachedAt) > c.ttl {
			ok = false
		}
	}

	if ok {
		// Step 2: timestamp validation (lock-free)
		var lastUpdated int64
		err := c.db.WithContext(ctx).Model(&Schema{}).
			Select("COALESCE(MAX(updated_at), 0)").
			Where("module_name = ? AND table_name = ?", moduleName, tableName).
			Scan(&lastUpdated).Error
		if err != nil {
			// Timestamp query failed: degrade to direct query
			ok = false
		} else if lastUpdated == ent.lastUpdatedAt {
			return ent.schemas, nil
		} else {
			ok = false
		}
	}

	if !ok {
		// Step 3: write-lock with double-check
		c.mu.Lock()
		ent, ok = c.entries[key]
		if ok {
			// If TTL is enabled and expired, treat as miss
			if c.ttl > 0 && time.Since(ent.cachedAt) > c.ttl {
				ok = false
			} else {
				// Double-check: another goroutine may have refreshed while we waited
				var lastUpdated int64
				err := c.db.Model(&Schema{}).
					Select("COALESCE(MAX(updated_at), 0)").
					Where("module_name = ? AND table_name = ?", moduleName, tableName).
					Scan(&lastUpdated).Error
				if err == nil && lastUpdated == ent.lastUpdatedAt {
					c.mu.Unlock()
					return ent.schemas, nil
				}
				ok = false
			}
		}

		// Full query from db
		var values []Schema
		var lastUpdated int64

		values, err := gorm.G[Schema](c.db).
			Where("module_name = ? AND table_name = ?", moduleName, tableName).
			Order("position ASC").
			Find(ctx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = nil
		}
		if err != nil {
			c.mu.Unlock()
			return nil, err
		}

		// Get max updated_at
		err = c.db.Model(&Schema{}).
			Select("COALESCE(MAX(updated_at), 0)").
			Where("module_name = ? AND table_name = ?", moduleName, tableName).
			Scan(&lastUpdated).Error
		if err != nil {
			c.mu.Unlock()
			return nil, err
		}

		c.entries[key] = &cacheEntry{
			schemas:       values,
			lastUpdatedAt: lastUpdated,
			cachedAt:      time.Now(),
		}
		c.mu.Unlock()
		return values, nil
	}

	return ent.schemas, nil
}

func (c *Cache) GetVisibleSchemas(ctx context.Context, moduleName, tableName, scenario string) ([]Schema, error) {
	schemas, err := c.GetSchemas(ctx, moduleName, tableName)
	if err != nil {
		return nil, err
	}
	result := make([]Schema, 0, len(schemas))
	for _, row := range schemas {
		if row.Scenarios.Has(scenario) {
			result = append(result, row)
		}
	}
	return result, nil
}
