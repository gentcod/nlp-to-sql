package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	db "github.com/gentcod/nlp-to-sql/internal/database"
)

// MCPServer implements the Model Context Protocol server with an explicit lifecycle state machine
type MCPServer struct {
	name              string
	version           string
	state             ServerState
	resourceManager   *ResourceManager
	toolRegistry      *ToolRegistry
	credentialManager *CredentialManager
	auditLogger       *AuditLogger

	mu sync.RWMutex
}

// NewMCPServer creates a new MCP server instance in the UNINITIALIZED state
func NewMCPServer(
	store db.Store,
	credentialManager *CredentialManager,
	auditLogger *AuditLogger,
) *MCPServer {
	if auditLogger == nil {
		auditLogger = NewAuditLogger()
	}
	return &MCPServer{
		name:              "nlp-to-sql",
		version:           "2.0.0",
		state:             ServerStateUninitialized,
		resourceManager:   NewResourceManager(store),
		toolRegistry:      NewToolRegistry(store),
		credentialManager: credentialManager,
		auditLogger:       auditLogger,
	}
}

// State returns the current lifecycle state of the MCP server
func (s *MCPServer) State() ServerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// ConfigureRuntime attaches live DB and LLM execution context to the server's ResourceManager and ToolRegistry
func (s *MCPServer) ConfigureRuntime(dbCfg DatabaseConfig, toolCtx ToolExecutionContext) {
	s.mu.RLock()
	rm := s.resourceManager
	tr := s.toolRegistry
	s.mu.RUnlock()

	if rm != nil {
		rm.ConfigureDatabase(dbCfg)
	}
	if tr != nil {
		tr.ConfigureExecutionContext(toolCtx)
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

// Initialize performs server initialization and transitions state -> READY
func (s *MCPServer) Initialize(ctx context.Context, clientInfo ClientInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == ServerStateShuttingDown {
		return fmt.Errorf("cannot initialize MCP server: server is shutting down")
	}

	// Validate client
	if err := validateClientInfo(clientInfo); err != nil {
		return fmt.Errorf("invalid client info: %w", err)
	}

	s.state = ServerStateReady

	// Log initialization
	s.auditLogger.LogServerInit(clientInfo)

	return nil
}

// Shutdown gracefully transitions the server to SHUTTING_DOWN
func (s *MCPServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = ServerStateShuttingDown
	return nil
}

// ListResources returns available resources
func (s *MCPServer) ListResources(ctx context.Context, cursor string) ([]Resource, string, error) {
	s.mu.RLock()
	state := s.state
	rm := s.resourceManager
	s.mu.RUnlock()

	if state == ServerStateShuttingDown {
		return nil, "", fmt.Errorf("MCP server is shutting down")
	}

	return rm.ListResources(ctx, cursor)
}

// ReadResource reads a specific resource
func (s *MCPServer) ReadResource(ctx context.Context, uri string) (interface{}, error) {
	s.mu.RLock()
	state := s.state
	rm := s.resourceManager
	s.mu.RUnlock()

	if state == ServerStateShuttingDown {
		return nil, fmt.Errorf("MCP server is shutting down")
	}

	return rm.ReadResource(ctx, uri)
}

// ListTools returns available tools
func (s *MCPServer) ListTools(ctx context.Context) ([]Tool, error) {
	s.mu.RLock()
	state := s.state
	tr := s.toolRegistry
	s.mu.RUnlock()

	if state == ServerStateShuttingDown {
		return nil, fmt.Errorf("MCP server is shutting down")
	}

	return tr.ListTools(ctx)
}

// CallTool executes a tool concurrently without holding a write lock across tool execution
func (s *MCPServer) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	s.mu.RLock()
	state := s.state
	tr := s.toolRegistry
	logger := s.auditLogger
	s.mu.RUnlock()

	if state == ServerStateShuttingDown {
		return nil, fmt.Errorf("cannot execute tool %q: MCP server is shutting down", toolName)
	}

	// Validate tool call
	if err := tr.ValidateTool(toolName, arguments); err != nil {
		logger.LogToolCallFailure(toolName, err)
		return nil, err
	}

	// Log tool call
	logger.LogToolCall(toolName, arguments)

	// Execute tool
	result, err := tr.CallTool(ctx, toolName, arguments)
	if err != nil {
		logger.LogToolCallFailure(toolName, err)
		return nil, err
	}

	logger.LogToolCallSuccess(toolName)
	return result, nil
}

// ServerInfo returns server information including its current lifecycle state
type ServerInfo struct {
	Name     string      `json:"name"`
	Version  string      `json:"version"`
	Protocol string      `json:"protocol"`
	State    ServerState `json:"state"`
}

func (s *MCPServer) ServerInfo() ServerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return ServerInfo{
		Name:     s.name,
		Version:  s.version,
		Protocol: "2024-11",
		State:    s.state,
	}
}
