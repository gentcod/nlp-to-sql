package mcp

import (
	"context"
	"testing"
	"time"

	db "github.com/gentcod/nlp-to-sql/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStore is a dummy store for testing purposes
type MockStore struct {
	db.Store // embed the interface to satisfy the signature
}

func TestMCPServer_LifecycleStateMachine(t *testing.T) {
	mockStore := &MockStore{}
	credManager, err := NewCredentialManager(make([]byte, 32))
	require.NoError(t, err)

	auditLogger := NewAuditLogger()
	server := NewMCPServer(mockStore, credManager, auditLogger)

	// Initial state should be UNINITIALIZED
	assert.Equal(t, ServerStateUninitialized, server.State())

	// Test capabilities
	caps := server.Capabilities()
	assert.True(t, caps.Resources.Subscribe)
	assert.True(t, caps.Tools.ListChanged)

	// Invalid Client info keeps state UNINITIALIZED
	invalidClientInfo := ClientInfo{Name: "   ", Version: "1.0.0"}
	err = server.Initialize(context.Background(), invalidClientInfo)
	assert.Error(t, err)
	assert.Equal(t, ServerStateUninitialized, server.State())

	// Valid initialization transitions to READY
	clientInfo := ClientInfo{Name: "TestClient", Version: "1.0.0"}
	err = server.Initialize(context.Background(), clientInfo)
	assert.NoError(t, err)
	assert.Equal(t, ServerStateReady, server.State())
	assert.Equal(t, ServerStateReady, server.ServerInfo().State)

	// Shutdown transitions to SHUTTING_DOWN and blocks subsequent calls
	err = server.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, ServerStateShuttingDown, server.State())

	_, err = server.ListTools(context.Background())
	assert.Error(t, err)
}

func TestMCPServer_ListResourcesAndCacheState(t *testing.T) {
	mockStore := &MockStore{}
	credManager, _ := NewCredentialManager(make([]byte, 32))
	server := NewMCPServer(mockStore, credManager, NewAuditLogger())

	resources, cursor, err := server.ListResources(context.Background(), "")
	assert.NoError(t, err)
	assert.Empty(t, cursor)
	assert.Len(t, resources, 2)
	assert.Equal(t, "databases://", resources[0].URI)

	// Cache starts MISSING, transitions to FRESH after ReadResource, and MISSING after Invalidate
	assert.Equal(t, CacheStateMissing, server.resourceManager.GetCacheState("databases://primary"))
	res, err := server.ReadResource(context.Background(), "databases://primary")
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, CacheStateFresh, server.resourceManager.GetCacheState("databases://primary"))

	server.resourceManager.InvalidateResource("databases://primary")
	assert.Equal(t, CacheStateMissing, server.resourceManager.GetCacheState("databases://primary"))
}

func TestMCPServer_ListToolsAndValidateQueryTool(t *testing.T) {
	mockStore := &MockStore{}
	credManager, _ := NewCredentialManager(make([]byte, 32))
	server := NewMCPServer(mockStore, credManager, NewAuditLogger())

	tools, err := server.ListTools(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tools), 4) // execute_query, inspect_schema, generate_query, validate_query

	// Call validate_query with a safe SELECT statement wrapped in markdown fences
	res, err := server.CallTool(context.Background(), "validate_query", map[string]interface{}{
		"query": "```sql\nSELECT id, username FROM users\n```",
	})
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// Call execute_query with a forbidden DROP statement -> rejected by validator
	_, err = server.CallTool(context.Background(), "execute_query", map[string]interface{}{
		"query":       "DROP TABLE users",
		"database_id": "primary",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query validation failed")
}

func TestMCPServer_CallTool_MissingArgs(t *testing.T) {
	mockStore := &MockStore{}
	credManager, _ := NewCredentialManager(make([]byte, 32))
	server := NewMCPServer(mockStore, credManager, NewAuditLogger())

	// Call generate_query without required arguments
	_, err := server.CallTool(context.Background(), "generate_query", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument")
}

func TestCredentialManager_TTLStateTransitions(t *testing.T) {
	cm, err := NewCredentialManager(make([]byte, 32))
	require.NoError(t, err)

	// Initially MISSING
	assert.Equal(t, CacheStateMissing, cm.GetCredentialState("tenant-1"))

	// Store transitions to FRESH and round-trips special characters in JSON cleanly
	orig := &DatabaseCredential{
		Host:     "localhost",
		Port:     5432,
		User:     "admin",
		Password: `p@ss"word\with"quotes`,
		Database: "prod_db",
		Driver:   "postgres",
	}
	err = cm.StoreCredential("tenant-1", orig)
	require.NoError(t, err)
	assert.Equal(t, CacheStateFresh, cm.GetCredentialState("tenant-1"))

	decrypted, state, err := cm.GetCredential("tenant-1")
	require.NoError(t, err)
	assert.Equal(t, CacheStateFresh, state)
	assert.Equal(t, orig.Password, decrypted.Password)
	assert.Equal(t, orig.Database, decrypted.Database)

	// Expire TTL and verify transition to EXPIRED + auto-eviction
	cm.SetCacheTTL(-1 * time.Second)
	err = cm.StoreCredential("tenant-expired", orig)
	require.NoError(t, err)
	assert.Equal(t, CacheStateExpired, cm.GetCredentialState("tenant-expired"))

	_, state, err = cm.GetCredential("tenant-expired")
	assert.Error(t, err)
	assert.Equal(t, CacheStateExpired, state)
	assert.Equal(t, CacheStateMissing, cm.GetCredentialState("tenant-expired"))
}
