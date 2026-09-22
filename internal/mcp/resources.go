package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	db "github.com/gentcod/nlp-to-sql/internal/database"
)

// ResourceManager manages MCP resources
type ResourceManager struct {
	store         db.Store
	resourceCache map[string]interface{}
	cacheMutex    sync.RWMutex
	subscriptions map[string][]chan Resource
	subMutex      sync.RWMutex
}

// NewResourceManager creates a new resource manager
func NewResourceManager(store db.Store) *ResourceManager {
	return &ResourceManager{
		store:         store,
		resourceCache: make(map[string]interface{}),
		subscriptions: make(map[string][]chan Resource),
	}
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

// ReadResource reads a specific resource
func (rm *ResourceManager) ReadResource(ctx context.Context, uri string) (interface{}, error) {
	rm.cacheMutex.RLock()
	if cached, ok := rm.resourceCache[uri]; ok {
		rm.cacheMutex.RUnlock()
		return cached, nil
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

	// Cache result
	rm.cacheMutex.Lock()
	rm.resourceCache[uri] = resource
	rm.cacheMutex.Unlock()

	return resource, nil
}

// readDatabaseResource reads database information
func (rm *ResourceManager) readDatabaseResource(ctx context.Context, path string) (interface{}, error) {
	// Implementation would connect to database and retrieve metadata
	return map[string]interface{}{
		"type":  "database",
		"name":  path,
		"ready": true,
	}, nil
}

// readSchemaResource reads schema information
func (rm *ResourceManager) readSchemaResource(ctx context.Context, path string) (interface{}, error) {
	// Implementation would retrieve schema from database
	return map[string]interface{}{
		"type":   "schema",
		"tables": []string{},
	}, nil
}

// readTableResource reads table information
func (rm *ResourceManager) readTableResource(ctx context.Context, path string) (interface{}, error) {
	// Implementation would retrieve table structure
	return map[string]interface{}{
		"type":    "table",
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
