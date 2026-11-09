package mcp

import "time"

// ServerState represents the explicit lifecycle state of the MCPServer
type ServerState string

const (
	ServerStateUninitialized ServerState = "UNINITIALIZED"
	ServerStateReady         ServerState = "READY"
	ServerStateShuttingDown  ServerState = "SHUTTING_DOWN"
)

// CacheState represents the explicit TTL state of a cached resource or credential
type CacheState string

const (
	CacheStateMissing CacheState = "MISSING"
	CacheStateFresh   CacheState = "FRESH"
	CacheStateExpired CacheState = "EXPIRED"
)

// ClientInfo represents a connecting client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Capabilities represents server capabilities
type Capabilities struct {
	Resources ResourceCapabilities `json:"resources"`
	Tools     ToolCapabilities     `json:"tools"`
	Prompts   PromptCapabilities   `json:"prompts"`
}

// ResourceCapabilities describes resource handling
type ResourceCapabilities struct {
	Subscribe bool `json:"subscribe"`
}

// ToolCapabilities describes tool handling
type ToolCapabilities struct {
	ListChanged bool `json:"listChanged"`
}

// PromptCapabilities describes prompt handling
type PromptCapabilities struct {
	ListChanged bool `json:"listChanged"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string    `json:"uri"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	MimeType    string    `json:"mimeType,omitempty"`
	Updated     time.Time `json:"updated,omitempty"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema defines tool input
type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required"`
}

// ToolCall represents a tool call request
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents a tool execution result with rich agent/developer metadata
type ToolResult struct {
	Content         interface{} `json:"content"`
	IsError         bool        `json:"isError,omitempty"`
	Stage           string      `json:"stage,omitempty"`
	RemediationHint string      `json:"remediationHint,omitempty"`
	LatencyMs       int64       `json:"latencyMs,omitempty"`
	Timestamp       time.Time   `json:"timestamp"`
}

// TextContent represents text result
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
