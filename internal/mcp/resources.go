package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	db "github.com/gentcod/nlp-to-sql/internal/database"
	mp "github.com/gentcod/nlp-to-sql/mapper"
)

type cachedResourceEntry struct {
	data      interface{}
	expiresAt time.Time
}

// DatabaseConfig holds optional runtime DB connection metadata for dynamic resource resolution
type DatabaseConfig struct {
	Conn   *sql.DB
	Driver string
	Name   string
}

// ResourceManager manages MCP resources with explicit TTL caching and schema mapping
type ResourceManager struct {
	store         db.Store
	dbConfig      DatabaseConfig
	resourceCache map[string]cachedResourceEntry
	cacheTTL      time.Duration
	cacheMutex    sync.RWMutex
	subscriptions map[string][]chan Resource
	subMutex      sync.RWMutex
}

// NewResourceManager creates a new resource manager
func NewResourceManager(store db.Store) *ResourceManager {
	return &ResourceManager{
		store:         store,
		resourceCache: make(map[string]cachedResourceEntry),
		cacheTTL:      15 * time.Minute,
		subscriptions: make(map[string][]chan Resource),
	}
}

// ConfigureDatabase attaches a live *sql.DB connection and dialect metadata for real schema inspection
func (rm *ResourceManager) ConfigureDatabase(cfg DatabaseConfig) {
	rm.cacheMutex.Lock()
	rm.dbConfig = cfg
	rm.resourceCache = make(map[string]cachedResourceEntry)
	rm.cacheMutex.Unlock()
}

// GetCacheState inspects the explicit state of a cached URI
func (rm *ResourceManager) GetCacheState(uri string) CacheState {
	rm.cacheMutex.RLock()
	entry, ok := rm.resourceCache[uri]
	rm.cacheMutex.RUnlock()

	if !ok {
		return CacheStateMissing
	}
	if time.Now().After(entry.expiresAt) {
		return CacheStateExpired
	}
	return CacheStateFresh
}

// InvalidateResource evicts a resource from cache and notifies subscribers
func (rm *ResourceManager) InvalidateResource(uri string) {
	rm.cacheMutex.Lock()
	delete(rm.resourceCache, uri)
	rm.cacheMutex.Unlock()

	rm.PublishResourceUpdate(uri, Resource{
		URI:     uri,
		Name:    uri,
		Updated: time.Now(),
	})
}

// ListResources lists available resources
func (rm *ResourceManager) ListResources(ctx context.Context, cursor string) ([]Resource, string, error) {
	resources := []Resource{
		{
			URI:         "databases://",
			Name:        "Databases",
			Description: "Available database connections",
			MimeType:    "application/json",
		},
		{
			URI:         "schemas://",
			Name:        "Database Schemas",
			Description: "Database schema definitions",
			MimeType:    "application/json",
		},
	}

	return resources, "", nil
}

// ReadResource reads a specific resource with explicit cache state transitions (MISSING -> FRESH -> EXPIRED)
func (rm *ResourceManager) ReadResource(ctx context.Context, uri string) (interface{}, error) {
	rm.cacheMutex.RLock()
	if entry, ok := rm.resourceCache[uri]; ok {
		if time.Now().Before(entry.expiresAt) {
			rm.cacheMutex.RUnlock()
			return entry.data, nil
		}
	}
	rm.cacheMutex.RUnlock()

	// Parse URI: {scheme}://{path}
	parts := strings.Split(uri, "://")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid resource URI: %s", uri)
	}

	scheme := parts[0]
	path := parts[1]

	var resource interface{}
	var err error

	switch scheme {
	case "databases":
		resource, err = rm.readDatabaseResource(ctx, path)
	case "schemas":
		resource, err = rm.readSchemaResource(ctx, path)
	case "tables":
		resource, err = rm.readTableResource(ctx, path)
	default:
		return nil, fmt.Errorf("unknown resource scheme: %s", scheme)
	}

	if err != nil {
		return nil, err
	}

	// Cache result with TTL
	rm.cacheMutex.Lock()
	rm.resourceCache[uri] = cachedResourceEntry{
		data:      resource,
		expiresAt: time.Now().Add(rm.cacheTTL),
	}
	rm.cacheMutex.Unlock()

	return resource, nil
}

// readDatabaseResource reads database information
func (rm *ResourceManager) readDatabaseResource(ctx context.Context, path string) (interface{}, error) {
	rm.cacheMutex.RLock()
	cfg := rm.dbConfig
	rm.cacheMutex.RUnlock()

	dbName := path
	if dbName == "" && cfg.Name != "" {
		dbName = cfg.Name
	}

	driver := cfg.Driver
	if driver == "" {
		driver = "postgres"
	}

	return map[string]interface{}{
		"type":   "database",
		"name":   dbName,
		"driver": driver,
		"ready":  true,
	}, nil
}

// readSchemaResource reads schema information using the project's Mapper if a live DB is configured
func (rm *ResourceManager) readSchemaResource(ctx context.Context, path string) (interface{}, error) {
	rm.cacheMutex.RLock()
	cfg := rm.dbConfig
	rm.cacheMutex.RUnlock()

	if cfg.Conn != nil && cfg.Driver != "" {
		mapper := mp.InitMapper(cfg.Driver)
		if mapper != nil {
			dbName := path
			if dbName == "" {
				dbName = cfg.Name
			}
			schemaMap, err := mapper.MapSchema(cfg.Conn, dbName)
			if err == nil {
				tables := make([]string, 0, len(schemaMap))
				for tbl := range schemaMap {
					tables = append(tables, tbl)
				}
				return map[string]interface{}{
					"type":   "schema",
					"tables": tables,
					"schema": schemaMap,
				}, nil
			}
		}
	}

	return map[string]interface{}{
		"type":   "schema",
		"tables": []string{},
	}, nil
}

// readTableResource reads table information
func (rm *ResourceManager) readTableResource(ctx context.Context, path string) (interface{}, error) {
	rm.cacheMutex.RLock()
	cfg := rm.dbConfig
	rm.cacheMutex.RUnlock()

	if cfg.Conn != nil && cfg.Driver != "" {
		mapper := mp.InitMapper(cfg.Driver)
		if mapper != nil {
			schemaMap, err := mapper.MapSchema(cfg.Conn, cfg.Name)
			if err == nil {
				if cols, ok := schemaMap[path]; ok {
					return map[string]interface{}{
						"type":    "table",
						"table":   path,
						"columns": cols,
					}, nil
				}
			}
		}
	}

	return map[string]interface{}{
		"type":    "table",
		"table":   path,
		"columns": []string{},
	}, nil
}

// Subscribe subscribes to resource changes
func (rm *ResourceManager) Subscribe(ctx context.Context, uri string) <-chan Resource {
	ch := make(chan Resource, 1)

	rm.subMutex.Lock()
	rm.subscriptions[uri] = append(rm.subscriptions[uri], ch)
	rm.subMutex.Unlock()

	return ch
}

// PublishResourceUpdate publishes a resource update
func (rm *ResourceManager) PublishResourceUpdate(uri string, resource Resource) {
	rm.subMutex.RLock()
	subscribers := rm.subscriptions[uri]
	rm.subMutex.RUnlock()

	for _, ch := range subscribers {
		select {
		case ch <- resource:
		default:
			// Channel full, skip
		}
	}
}
