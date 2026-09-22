package mcp

import (
	"context"
	"testing"

	db "github.com/gentcod/nlp-to-sql/internal/database"
	"github.com/stretchr/testify/assert"
)

// MockStore is a dummy store for testing purposes
type MockStore struct {
	db.Store // embed the interface to satisfy the signature
}

func TestMCPServer_Initialization(t *testing.T) {
	mockStore := &MockStore{}
	credManager, err := NewCredentialManager(make([]byte, 32))
	assert.NoError(t, err)

	auditLogger := NewAuditLogger()
	server := NewMCPServer(mockStore, credManager, auditLogger)

	// Test capabilities
	caps := server.Capabilities()
	assert.True(t, caps.Resources.Subscribe)
	assert.True(t, caps.Tools.ListChanged)

	// Test initialization
	clientInfo := ClientInfo{Name: "TestClient", Version: "1.0.0"}
	err = server.Initialize(context.Background(), clientInfo)
	assert.NoError(t, err)

	// Test Invalid Client info
	invalidClientInfo := ClientInfo{Name: "   ", Version: "1.0.0"}
	err = server.Initialize(context.Background(), invalidClientInfo)
	assert.Error(t, err)
}

func TestMCPServer_ListResources(t *testing.T) {
	mockStore := &MockStore{}
	credManager, _ := NewCredentialManager(make([]byte, 32))
	server := NewMCPServer(mockStore, credManager, NewAuditLogger())

	resources, cursor, err := server.ListResources(context.Background(), "")
	assert.NoError(t, err)
	assert.Empty(t, cursor)
	assert.Len(t, resources, 2)
	assert.Equal(t, "databases://", resources[0].URI)
}

func TestMCPServer_ListTools(t *testing.T) {
	mockStore := &MockStore{}
	credManager, _ := NewCredentialManager(make([]byte, 32))
	server := NewMCPServer(mockStore, credManager, NewAuditLogger())

	tools, err := server.ListTools(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tools), 3) // execute_query, inspect_schema, generate_query
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
