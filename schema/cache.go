package schema

import (
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
