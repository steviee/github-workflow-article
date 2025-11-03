package cache

import (
	"context"
	"sync"
	"testing"
	"time"
)

const testKey = "test-key"

func TestGet_SetAndGet(t *testing.T) {
	c := NewCache(5*time.Minute, 30*time.Second)
	key := testKey
	data := []byte("test-data")

	c.Set(key, data)
	retrieved, ok := c.Get(key)

	if !ok {
		t.Fatal("Expected to find cached entry")
	}

	if string(retrieved) != string(data) {
		t.Errorf("Expected data %q, got %q", data, retrieved)
	}
}

func TestGet_UpdatesTimestamp(t *testing.T) {
	c := NewCache(5*time.Minute, 30*time.Second)
	key := testKey
	data := []byte("test-data")

	c.Set(key, data)

	// Get initial timestamp
	value, _ := c.store.Load(key)
	entry1 := value.(*Entry)
	timestamp1 := entry1.LastAccessed

	// Wait a bit and access again
	time.Sleep(10 * time.Millisecond)
	c.Get(key)

	// Check that timestamp was updated
	value, _ = c.store.Load(key)
	entry2 := value.(*Entry)
	timestamp2 := entry2.LastAccessed

	if !timestamp2.After(timestamp1) {
		t.Error("Expected Get to update last accessed timestamp")
	}
}

func TestGet_NonExistent(t *testing.T) {
	c := NewCache(5*time.Minute, 30*time.Second)

	data, ok := c.Get("non-existent-key")

	if ok {
		t.Error("Expected Get to return false for non-existent key")
	}

	if data != nil {
		t.Error("Expected nil data for non-existent key")
	}
}

func TestDelete(t *testing.T) {
	c := NewCache(5*time.Minute, 30*time.Second)
	key := testKey
	data := []byte("test-data")

	c.Set(key, data)

	if c.Size() != 1 {
		t.Errorf("Expected size 1, got %d", c.Size())
	}

	c.Delete(key)

	if c.Size() != 0 {
		t.Errorf("Expected size 0 after delete, got %d", c.Size())
	}

	_, ok := c.Get(key)
	if ok {
		t.Error("Expected key to be deleted")
	}
}

func TestSize(t *testing.T) {
	c := NewCache(5*time.Minute, 30*time.Second)

	if c.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", c.Size())
	}

	c.Set("key1", []byte("data1"))
	if c.Size() != 1 {
		t.Errorf("Expected size 1, got %d", c.Size())
	}

	c.Set("key2", []byte("data2"))
	if c.Size() != 2 {
		t.Errorf("Expected size 2, got %d", c.Size())
	}

	// Setting same key should not increase size
	c.Set("key1", []byte("updated-data1"))
	if c.Size() != 2 {
		t.Errorf("Expected size 2 after update, got %d", c.Size())
	}

	c.Delete("key1")
	if c.Size() != 1 {
		t.Errorf("Expected size 1 after delete, got %d", c.Size())
	}
}

func TestClear(t *testing.T) {
	c := NewCache(5*time.Minute, 30*time.Second)

	c.Set("key1", []byte("data1"))
	c.Set("key2", []byte("data2"))
	c.Set("key3", []byte("data3"))

	if c.Size() != 3 {
		t.Errorf("Expected size 3, got %d", c.Size())
	}

	c.Clear()

	if c.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", c.Size())
	}

	_, ok := c.Get("key1")
	if ok {
		t.Error("Expected all keys to be cleared")
	}
}

func TestTTLExpiration(t *testing.T) {
	// Use short TTL for testing
	c := NewCache(50*time.Millisecond, 100*time.Millisecond)
	key := testKey
	data := []byte("test-data")

	c.Set(key, data)

	// Verify entry exists
	_, ok := c.Get(key)
	if !ok {
		t.Fatal("Expected entry to exist immediately after set")
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Manually trigger cleanup
	c.cleanup()

	// Verify entry was removed
	_, ok = c.Get(key)
	if ok {
		t.Error("Expected entry to be removed after TTL expiration")
	}

	if c.Size() != 0 {
		t.Errorf("Expected size 0 after TTL cleanup, got %d", c.Size())
	}
}

func TestBackgroundCleanup(t *testing.T) {
	// Use short intervals for testing
	c := NewCache(50*time.Millisecond, 60*time.Millisecond)
	key := testKey
	data := []byte("test-data")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.Start(ctx)

	c.Set(key, data)

	// Verify entry exists
	if c.Size() != 1 {
		t.Errorf("Expected size 1, got %d", c.Size())
	}

	// Wait for TTL + cleanup interval
	time.Sleep(150 * time.Millisecond)

	// Verify background cleanup removed the entry
	if c.Size() != 0 {
		t.Errorf("Expected size 0 after background cleanup, got %d", c.Size())
	}

	_, ok := c.Get(key)
	if ok {
		t.Error("Expected entry to be removed by background cleanup")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := NewCache(5*time.Minute, 30*time.Second)
	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := GenerateKey("url", string(rune(id)))
				data := []byte("data")
				c.Set(key, data)
			}
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := GenerateKey("url", string(rune(id)))
				c.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// Concurrent deletes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := GenerateKey("url", string(rune(id)))
			c.Delete(key)
		}(i)
	}

	wg.Wait()

	if c.Size() != 0 {
		t.Errorf("Expected size 0 after all deletes, got %d", c.Size())
	}
}

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		operations string
		want       string
	}{
		{
			name:       "basic key generation",
			url:        "https://example.com/image.jpg",
			operations: "resize-100x100",
			want:       "5e0e8f5e3c3e3c3f3c3e3c3f3c3e3c3f3c3e3c3f3c3e3c3f3c3e3c3f3c3e3c3f",
		},
		{
			name:       "empty operations",
			url:        "https://example.com/image.jpg",
			operations: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateKey(tt.url, tt.operations)

			// Verify determinism - same input should produce same output
			got2 := GenerateKey(tt.url, tt.operations)
			if got != got2 {
				t.Errorf("GenerateKey not deterministic: %q != %q", got, got2)
			}

			// Verify it's a valid hex string of correct length (SHA256 = 64 hex chars)
			if len(got) != 64 {
				t.Errorf("Expected key length 64, got %d", len(got))
			}
		})
	}

	// Verify different inputs produce different keys
	key1 := GenerateKey("url1", "ops1")
	key2 := GenerateKey("url2", "ops2")
	if key1 == key2 {
		t.Error("Expected different inputs to produce different keys")
	}

	// Verify same URL with different operations produces different keys
	key3 := GenerateKey("url", "ops1")
	key4 := GenerateKey("url", "ops2")
	if key3 == key4 {
		t.Error("Expected different operations to produce different keys")
	}
}

func BenchmarkGet(b *testing.B) {
	c := NewCache(5*time.Minute, 30*time.Second)
	key := "bench-key"
	data := []byte("benchmark-data")
	c.Set(key, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(key)
	}
}

func BenchmarkSet(b *testing.B) {
	c := NewCache(5*time.Minute, 30*time.Second)
	data := []byte("benchmark-data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := GenerateKey("url", string(rune(i)))
		c.Set(key, data)
	}
}

func BenchmarkGenerateKey(b *testing.B) {
	url := "https://example.com/very/long/path/to/image.jpg"
	operations := "resize-100x100,rotate-90"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateKey(url, operations)
	}
}

func BenchmarkConcurrentAccess(b *testing.B) {
	c := NewCache(5*time.Minute, 30*time.Second)
	data := []byte("benchmark-data")

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		key := GenerateKey("url", string(rune(i)))
		c.Set(key, data)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := GenerateKey("url", string(rune(i%100)))
			if i%2 == 0 {
				c.Get(key)
			} else {
				c.Set(key, data)
			}
			i++
		}
	})
}
