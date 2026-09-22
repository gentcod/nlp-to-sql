package util

import (
	"fmt"
	"sync"
	"time"
)

type SchemaCache struct {
	Data       map[string]map[string]string
	LastUpdate time.Time
	mu         sync.RWMutex
}

func NewSchemaCache() *SchemaCache {
	return &SchemaCache{}
}

// Load the schema from the database and store it in the cache.
func (cache *SchemaCache) CacheSchema(schema map[string]map[string]string) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.Data = schema
	cache.LastUpdate = time.Now()
	return nil
}

// Provide thread-safe access to the cached data.
func (cache *SchemaCache) GetSchema() map[string]map[string]string {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.Data
}

// Set up a background goroutine to periodically refresh the schema.
func (cache *SchemaCache) AutoRefresh(schema map[string]map[string]string, interval time.Duration) error {
	go func() {
		for {
			time.Sleep(interval)

			err := cache.CacheSchema(schema)
			if err != nil {
				fmt.Println(fmt.Errorf("error refreshing schema cache: %v", err))
			}
		}
	}()

	return nil
}

// If the schema changes infrequently, add logic to validate or expire the cache based on the LastUpdate timestamp.
func (cache *SchemaCache) GetSchemaWithExpiry(maxAge time.Duration) (map[string]map[string]string, bool) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if time.Since(cache.LastUpdate) > maxAge {
		return nil, false // Cache is expired
	}

	return cache.Data, true // Cache is valid
}

// Manual Refresh API: Allow manual cache refresh via an admin endpoint or CLI.
func (cache *SchemaCache) RefreshSchema(schema map[string]map[string]string) error {
	return cache.CacheSchema(schema)
}
