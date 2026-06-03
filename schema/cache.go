package schema

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	db      *gorm.DB
	sf      singleflight.Group
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

	// Step 1: read-lock fast path with TTL + timestamp validation
	c.mu.RLock()
	ent, ok := c.entries[key]
	c.mu.RUnlock()

	if ok {
		if c.ttl > 0 && time.Since(ent.cachedAt) > c.ttl {
			ok = false
		}
	}

	if ok {
		var lastUpdated int64
		err := c.db.WithContext(ctx).Model(&Schema{}).
			Select("COALESCE(MAX(updated_at), 0)").
			Where("module_name = ? AND table_name = ?", moduleName, tableName).
			Scan(&lastUpdated).Error
		if err != nil {
			ok = false
		} else if lastUpdated == ent.lastUpdatedAt {
			return ent.schemas, nil
		} else {
			ok = false
		}
	}

	if !ok {
		// singleflight coalesces concurrent loads for the same key
		v, err, _ := c.sf.Do(key, func() (any, error) {
			// Re-check under write lock (another goroutine may have populated)
			c.mu.Lock()
			defer c.mu.Unlock()
			ent, ok := c.entries[key]
			if ok {
				if !(c.ttl > 0 && time.Since(ent.cachedAt) > c.ttl) {
					var lastUpdated int64
					if e := c.db.Model(&Schema{}).
						Select("COALESCE(MAX(updated_at), 0)").
						Where("module_name = ? AND table_name = ?", moduleName, tableName).
						Scan(&lastUpdated).Error; e == nil && lastUpdated == ent.lastUpdatedAt {
						return &cacheEntry{schemas: ent.schemas, lastUpdatedAt: ent.lastUpdatedAt, cachedAt: ent.cachedAt}, nil
					}
				}
			}
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
				return nil, err
			}
			if err := c.db.Model(&Schema{}).
				Select("COALESCE(MAX(updated_at), 0)").
				Where("module_name = ? AND table_name = ?", moduleName, tableName).
				Scan(&lastUpdated).Error; err != nil {
				return nil, err
			}
			entry := &cacheEntry{
				schemas:       values,
				lastUpdatedAt: lastUpdated,
				cachedAt:      time.Now(),
			}
			c.entries[key] = entry
			return entry, nil
		})
		if err != nil {
			return nil, err
		}
		return v.(*cacheEntry).schemas, nil
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
