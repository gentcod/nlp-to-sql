package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/gentcod/nlp-to-sql/api"
	"github.com/gentcod/nlp-to-sql/chat"
	conv "github.com/gentcod/nlp-to-sql/converter"
	"github.com/gentcod/nlp-to-sql/cron"
	db "github.com/gentcod/nlp-to-sql/internal/database"
	"github.com/gentcod/nlp-to-sql/internal/mcp"

	"github.com/gentcod/nlp-to-sql/internal/rag"
	"github.com/gentcod/nlp-to-sql/util"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func main() {
	config, err := util.LoadConfig(".env")
	if err != nil {
		log.Fatal("cannot load config", err)
	}

	conn, err := sql.Open(config.DBDriver, config.DBUrl)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer conn.Close()

	store := db.Store(db.NewStore(conn))

	llmOpts := rag.LLMOpts{
		ApiKey:    config.ApiKey,
		BaseUrl:   config.LLMBaseUrl,
		OrgId:     config.OrgId,
		ProjectId: config.ProjectId,
		Model:     config.Model,
		Temp:      config.Temp,
	}
	defaultLLMType := rag.NormalizeLLMType(config.LLMType)

	// Initialize MCP Server with explicit lifecycle state and runtime DB/LLM context
	credKey := make([]byte, 32)
	if len(config.TokenSymmetricKey) >= 32 {
		copy(credKey, []byte(config.TokenSymmetricKey)[:32])
	}
	credManager, err := mcp.NewCredentialManager(credKey)
	if err != nil {
		log.Printf("Warning: failed to initialize MCP credential manager: %v\n", err)
	}
	auditLogger := mcp.NewAuditLogger()
	mcpServer := mcp.NewMCPServer(store, credManager, auditLogger)
	mcpServer.ConfigureRuntime(
		mcp.DatabaseConfig{
			Conn:   conn,
			Driver: config.DBDriver,
			Name:   config.DBName,
		},
		mcp.ToolExecutionContext{
			Conn:     conn,
			DBDriver: config.DBDriver,
			DBName:   config.DBName,
			LLMOpts:  llmOpts,
			LLMType:  defaultLLMType,
		},
	)

	err = mcpServer.Initialize(context.Background(), mcp.ClientInfo{Name: "nlp-to-sql-host", Version: "2.0.0"})
	if err != nil {
		log.Printf("Warning: failed to initialize MCP server: %v\n", err)
	} else {
		log.Printf("MCP Server Core initialized successfully (state=%s, default_llm=%s).\n", mcpServer.State(), defaultLLMType)
	}

	converter := conv.NewSQLConverter(llmOpts)

	dbcron := cron.NewDBCron(store, cron.CronConfig{
		BatchSize: config.CronBatchSize,
		LogPath:   config.LogPath,
	})

	err = dbcron.InitCron()
	if err != nil {
		log.Fatal("error initializing database cron", err)
	}

	runGinServer(config, store, converter, mcpServer)
}

func runGinServer(config util.Config, store db.Store, converter conv.Converter, mcpServer *mcp.MCPServer) {
	websocketSrv, err := chat.NewWebSocketServer(config, converter)
	if err != nil {
		log.Fatal("couldn't initialize the chat-server:", err)
	}

	server, err := api.NewServer(config, store, websocketSrv)
	if err != nil {
		log.Fatal("couldn't initialize the server:", err)
	}
	server.SetMCPServer(mcpServer)

	err = server.Start(config.Port)
	if err != nil {
		log.Fatal("couldn't start up server:", err)
	}
}
