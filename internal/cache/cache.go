// Package cache provides thread-safe in-memory caching for image data
// with automatic expiration and cleanup.
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// Entry represents a cached image with metadata
type Entry struct {
	Data         []byte
	LastAccessed time.Time
}

// Cache provides thread-safe in-memory caching for image data
type Cache struct {
	store           sync.Map
	ttl             time.Duration
	cleanupInterval time.Duration
	mu              sync.RWMutex
	size            int
}

// NewCache creates a new Cache instance with specified TTL and cleanup interval
func NewCache(ttl time.Duration, cleanupInterval time.Duration) *Cache {
	return &Cache{
		ttl:             ttl,
		cleanupInterval: cleanupInterval,
	}
}

// Get retrieves image data from cache and updates the last accessed timestamp.
// Returns the data and true if found, nil and false otherwise.
func (c *Cache) Get(key string) ([]byte, bool) {
	value, ok := c.store.Load(key)
	if !ok {
		return nil, false
	}

	entry := value.(*Entry)

	// Update last accessed timestamp
	entry.LastAccessed = time.Now()
	c.store.Store(key, entry)

	return entry.Data, true
}

// Set stores image data in the cache with current timestamp
func (c *Cache) Set(key string, data []byte) {
	entry := &Entry{
		Data:         data,
		LastAccessed: time.Now(),
	}

	// Check if this is a new entry
	_, existed := c.store.Load(key)
	c.store.Store(key, entry)

	if !existed {
		c.mu.Lock()
		c.size++
		c.mu.Unlock()
	}
}

// Delete removes an entry from the cache
func (c *Cache) Delete(key string) {
	_, existed := c.store.LoadAndDelete(key)
	if existed {
		c.mu.Lock()
		c.size--
		c.mu.Unlock()
	}
}

// Size returns the number of entries currently in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.store.Range(func(key, _ interface{}) bool {
		c.store.Delete(key)
		return true
	})

	c.mu.Lock()
	c.size = 0
	c.mu.Unlock()
}

// Start begins the background cleanup goroutine that removes expired entries.
// The goroutine runs until the provided context is cancelled.
func (c *Cache) Start(ctx context.Context) {
	ticker := time.NewTicker(c.cleanupInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.cleanup()
			}
		}
	}()
}

// cleanup removes entries that have exceeded the TTL
func (c *Cache) cleanup() {
	now := time.Now()
	keysToDelete := []interface{}{}

	c.store.Range(func(key, value interface{}) bool {
		entry := value.(*Entry)
		if now.Sub(entry.LastAccessed) > c.ttl {
			keysToDelete = append(keysToDelete, key)
		}
		return true
	})

	for _, key := range keysToDelete {
		c.store.LoadAndDelete(key)
		c.mu.Lock()
		c.size--
		c.mu.Unlock()
	}
}

// GenerateKey creates a deterministic SHA256 hash key from URL and operations
func GenerateKey(url string, operations string) string {
	input := url + operations
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
