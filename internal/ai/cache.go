package ai

import (
	"sync"
	"time"
)

type cacheEntry struct {
	response  *DecisionResponse
	createdAt time.Time
}

// Cache is an LRU cache for AI decisions.
type Cache struct {
	mu       sync.RWMutex
	entries  map[string]*cacheEntry
	order    []string // LRU order: newest at end
	maxSize  int
	ttl      time.Duration
}

// NewCache creates a new decision cache.
func NewCache(maxSize int, ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]*cacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves a cached decision by process signature.
func (c *Cache) Get(signature string) (*DecisionResponse, bool) {
	c.mu.RLock()
	entry, ok := c.entries[signature]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Check TTL
	if time.Since(entry.createdAt) > c.ttl {
		c.mu.Lock()
		delete(c.entries, signature)
		c.removeFromOrder(signature)
		c.mu.Unlock()
		return nil, false
	}

	// Move to end (most recently used)
	c.mu.Lock()
	c.removeFromOrder(signature)
	c.order = append(c.order, signature)
	c.mu.Unlock()

	resp := *entry.response
	resp.FromCache = true
	return &resp, true
}

// Put stores a decision in the cache.
func (c *Cache) Put(signature string, response *DecisionResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	for len(c.entries) >= c.maxSize && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}

	c.entries[signature] = &cacheEntry{
		response:  response,
		createdAt: time.Now(),
	}
	c.removeFromOrder(signature)
	c.order = append(c.order, signature)
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Clear empties the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
	c.order = nil
}

func (c *Cache) removeFromOrder(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			return
		}
	}
}
