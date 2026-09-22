package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	db "github.com/gentcod/nlp-to-sql/internal/database"
)

// MCPServer implements the Model Context Protocol server
type MCPServer struct {
	name              string
	version           string
	resourceManager   *ResourceManager
	toolRegistry      *ToolRegistry
	credentialManager *CredentialManager
	auditLogger       *AuditLogger

	mu sync.RWMutex
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(
	store db.Store,
	credentialManager *CredentialManager,
	auditLogger *AuditLogger,
) *MCPServer {
	return &MCPServer{
		name:              "nlp-to-sql",
		version:           "2.0.0",
		resourceManager:   NewResourceManager(store),
		toolRegistry:      NewToolRegistry(store),
		credentialManager: credentialManager,
		auditLogger:       auditLogger,
	}
}

// Capabilities returns the server's capabilities
func (s *MCPServer) Capabilities() Capabilities {
	return Capabilities{
		Resources: ResourceCapabilities{
			Subscribe: true,
		},
		Tools: ToolCapabilities{
			ListChanged: true,
		},
		Prompts: PromptCapabilities{
			ListChanged: true,
		},
	}
}

func validateClientInfo(clientInfo ClientInfo) error {
	if strings.TrimSpace(clientInfo.Name) == "" {
		return fmt.Errorf("client name is required")
	}
	return nil
}

// Initialize performs server initialization
func (s *MCPServer) Initialize(ctx context.Context, clientInfo ClientInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate client
	if err := validateClientInfo(clientInfo); err != nil {
		return fmt.Errorf("invalid client info: %w", err)
	}

	// Log initialization
	s.auditLogger.LogServerInit(clientInfo)

	return nil
}

// ListResources returns available resources
func (s *MCPServer) ListResources(ctx context.Context, cursor string) ([]Resource, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.resourceManager.ListResources(ctx, cursor)
}

// ReadResource reads a specific resource
func (s *MCPServer) ReadResource(ctx context.Context, uri string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.resourceManager.ReadResource(ctx, uri)
}

// ListTools returns available tools
func (s *MCPServer) ListTools(ctx context.Context) ([]Tool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.toolRegistry.ListTools(ctx)
}

// CallTool executes a tool
func (s *MCPServer) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate tool call
	if err := s.toolRegistry.ValidateTool(toolName, arguments); err != nil {
		s.auditLogger.LogToolCallFailure(toolName, err)
		return nil, err
	}

	// Log tool call
	s.auditLogger.LogToolCall(toolName, arguments)

	// Execute tool
	result, err := s.toolRegistry.CallTool(ctx, toolName, arguments)
	if err != nil {
		s.auditLogger.LogToolCallFailure(toolName, err)
		return nil, err
	}

	s.auditLogger.LogToolCallSuccess(toolName)
	return result, nil
}

// ServerInfo returns server information
type ServerInfo struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Protocol string `json:"protocol"`
}

func (s *MCPServer) ServerInfo() ServerInfo {
	return ServerInfo{
		Name:     s.name,
		Version:  s.version,
		Protocol: "2024-11",
	}
}
