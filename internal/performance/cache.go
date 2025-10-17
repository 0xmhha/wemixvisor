package performance

import (
	"container/list"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
)

// Cache implements an LRU cache with TTL support
type Cache struct {
	maxSize    int
	ttl        time.Duration
	items      map[string]*CacheItem
	lru        *list.List
	mu         sync.RWMutex
	logger     *logger.Logger
	stats      *CacheStats
	stopCleaner chan struct{}
}

// CacheItem represents a single cache item
type CacheItem struct {
	Key       string
	Value     interface{}
	ExpiresAt time.Time
	Size      int
	element   *list.Element
}

// CacheStats holds cache statistics
type CacheStats struct {
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	Evictions int64   `json:"evictions"`
	Size      int     `json:"size"`
	HitRate   float64 `json:"hit_rate"`
	mu        sync.RWMutex
}

// NewCache creates a new cache
func NewCache(maxSize int, ttl time.Duration, logger *logger.Logger) *Cache {
	return &Cache{
		maxSize:     maxSize,
		ttl:         ttl,
		items:       make(map[string]*CacheItem),
		lru:         list.New(),
		logger:      logger,
		stats:       &CacheStats{},
		stopCleaner: make(chan struct{}),
	}
}

// Start starts the cache cleaner
func (c *Cache) Start() error {
	go c.cleanupLoop()
	c.logger.Info("Cache started")
	return nil
}

// Stop stops the cache cleaner
func (c *Cache) Stop() {
	close(c.stopCleaner)
	c.Clear()
	c.logger.Info("Cache stopped")
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		c.recordMiss()
		return nil, false
	}

	// Check if item has expired
	if time.Now().After(item.ExpiresAt) {
		c.Delete(key)
		c.recordMiss()
		return nil, false
	}

	// Move to front (most recently used)
	c.mu.Lock()
	c.lru.MoveToFront(item.element)
	c.mu.Unlock()

	c.recordHit()
	return item.Value, true
}

// Set adds or updates an item in the cache
func (c *Cache) Set(key string, value interface{}, size int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if item already exists
	if item, exists := c.items[key]; exists {
		// Update existing item
		item.Value = value
		item.Size = size
		item.ExpiresAt = time.Now().Add(c.ttl)
		c.lru.MoveToFront(item.element)
		return
	}

	// Create new item
	item := &CacheItem{
		Key:       key,
		Value:     value,
		Size:      size,
		ExpiresAt: time.Now().Add(c.ttl),
	}

	// Add to LRU list
	element := c.lru.PushFront(key)
	item.element = element
	c.items[key] = item

	// Check if cache size exceeded
	if c.lru.Len() > c.maxSize {
		c.evictOldest()
	}
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		c.lru.Remove(item.element)
		delete(c.items, key)
	}
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheItem)
	c.lru.Init()
}

// evictOldest removes the least recently used item
func (c *Cache) evictOldest() {
	element := c.lru.Back()
	if element != nil {
		key := element.Value.(string)
		c.lru.Remove(element)
		delete(c.items, key)
		c.recordEviction()
	}
}

// cleanupLoop periodically removes expired items
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCleaner:
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

// cleanup removes expired items
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var keysToDelete []string

	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		if item, exists := c.items[key]; exists {
			c.lru.Remove(item.element)
			delete(c.items, key)
		}
	}

	if len(keysToDelete) > 0 {
		c.logger.Debug("Cleaned up expired cache items")
	}
}

// Optimize performs cache optimization
func (c *Cache) Optimize() {
	c.cleanup()

	// Log cache efficiency
	stats := c.GetStats()
	if stats.HitRate < 0.5 && stats.Hits+stats.Misses > 100 {
		c.logger.Warn("Low cache hit rate detected")
	}
}

// GetStats returns cache statistics
func (c *Cache) GetStats() *CacheStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	c.mu.RLock()
	c.stats.Size = len(c.items)
	c.mu.RUnlock()

	total := float64(c.stats.Hits + c.stats.Misses)
	if total > 0 {
		c.stats.HitRate = float64(c.stats.Hits) / total
	}

	return &CacheStats{
		Hits:      c.stats.Hits,
		Misses:    c.stats.Misses,
		Evictions: c.stats.Evictions,
		Size:      c.stats.Size,
		HitRate:   c.stats.HitRate,
	}
}

// recordHit increments the hit counter
func (c *Cache) recordHit() {
	c.stats.mu.Lock()
	c.stats.Hits++
	c.stats.mu.Unlock()
}

// recordMiss increments the miss counter
func (c *Cache) recordMiss() {
	c.stats.mu.Lock()
	c.stats.Misses++
	c.stats.mu.Unlock()
}

// recordEviction increments the eviction counter
func (c *Cache) recordEviction() {
	c.stats.mu.Lock()
	c.stats.Evictions++
	c.stats.mu.Unlock()
}