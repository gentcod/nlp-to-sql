package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gentcod/nlp-to-sql/internal/mcp"
	"github.com/gin-gonic/gin"
)

// SetMCPServer attaches an initialized MCPServer to the HTTP API Server
func (server *Server) SetMCPServer(mcpSrv *mcp.MCPServer) {
	server.mcpServer = mcpSrv
}

func (server *Server) getMCPServer() *mcp.MCPServer {
	if server.mcpServer != nil {
		return server.mcpServer
	}
	// Fallback lazy initialization so v2 routes always work even if SetMCPServer wasn't explicitly called
	credManager, _ := mcp.NewCredentialManager(make([]byte, 32))
	srv := mcp.NewMCPServer(server.store, credManager, mcp.NewAuditLogger())
	_ = srv.Initialize(context.TODO(), mcp.ClientInfo{Name: "api-v2-gateway", Version: "2.0.0"})
	server.mcpServer = srv
	return srv
}

func (server *Server) handleMCPInfo(ctx *gin.Context) {
	mcpSrv := server.getMCPServer()
	ctx.JSON(http.StatusOK, apiServerResponse("MCP server info retrieved", gin.H{
		"server":       mcpSrv.ServerInfo(),
		"capabilities": mcpSrv.Capabilities(),
	}))
}

func (server *Server) handleMCPListResources(ctx *gin.Context) {
	mcpSrv := server.getMCPServer()
	cursor := ctx.Query("cursor")
	resources, nextCursor, err := mcpSrv.ListResources(ctx, cursor)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, apiServerResponse("MCP resources listed", gin.H{
		"resources":   resources,
		"next_cursor": nextCursor,
	}))
}

func (server *Server) handleMCPReadResource(ctx *gin.Context) {
	mcpSrv := server.getMCPServer()
	uri := ctx.Query("uri")
	if uri == "" {
		var req struct {
			URI string `json:"uri"`
		}
		_ = ctx.ShouldBindJSON(&req)
		uri = req.URI
	}
	res, err := mcpSrv.ReadResource(ctx, uri)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, apiErrorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, apiServerResponse("MCP resource read", res))
}

func (server *Server) handleMCPListTools(ctx *gin.Context) {
	mcpSrv := server.getMCPServer()
	tools, err := mcpSrv.ListTools(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, apiServerResponse("MCP tools listed", gin.H{
		"tools": tools,
	}))
}

func (server *Server) handleMCPCallTool(ctx *gin.Context) {
	mcpSrv := server.getMCPServer()
	var req mcp.ToolCall
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, apiErrorResponse(err))
		return
	}

	start := time.Now()
	result, err := mcpSrv.CallTool(ctx, req.Name, req.Arguments)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
			"data": mcp.ToolResult{
				Content:   result,
				IsError:   true,
				LatencyMs: latency,
				Timestamp: time.Now(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, apiServerResponse("MCP tool executed", mcp.ToolResult{
		Content:   result,
		IsError:   false,
		LatencyMs: latency,
		Timestamp: time.Now(),
	}))
}
