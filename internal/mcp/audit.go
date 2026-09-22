package mcp

import (
	"log"
)

// AuditLogger represents the logger used for MCP server actions
type AuditLogger struct {
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{}
}

// LogServerInit logs when the server is initialized by a client
func (a *AuditLogger) LogServerInit(clientInfo ClientInfo) {
	log.Printf("MCP Server initialized by client: %s %s", clientInfo.Name, clientInfo.Version)
}

// LogToolCallFailure logs a failed tool execution
func (a *AuditLogger) LogToolCallFailure(toolName string, err error) {
	log.Printf("MCP Tool Call Failed [%s]: %v", toolName, err)
}

// LogToolCall logs the initiation of a tool call
func (a *AuditLogger) LogToolCall(toolName string, arguments map[string]interface{}) {
	log.Printf("MCP Tool Call Started [%s]", toolName)
}

// LogToolCallSuccess logs a successful tool execution
func (a *AuditLogger) LogToolCallSuccess(toolName string) {
	log.Printf("MCP Tool Call Succeeded [%s]", toolName)
}
