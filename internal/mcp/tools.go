package mcp

import (
	"context"
	"fmt"
	"sync"

	db "github.com/gentcod/nlp-to-sql/internal/database"
	"github.com/gentcod/nlp-to-sql/internal/security"
)

// ToolRegistry manages MCP tools
type ToolRegistry struct {
	store     db.Store
	validator *security.QueryValidator
	tools     map[string]*Tool
	executors map[string]ToolExecutor
	toolMutex sync.RWMutex
}

// ToolExecutor is the interface for tool execution
type ToolExecutor interface {
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(store db.Store) *ToolRegistry {
	tr := &ToolRegistry{
		store:     store,
		validator: security.NewQueryValidator(),
		tools:     make(map[string]*Tool),
		executors: make(map[string]ToolExecutor),
	}

	tr.registerBuiltInTools()
	return tr
}

// registerBuiltInTools registers standard tools
func (tr *ToolRegistry) registerBuiltInTools() {
	// SQL Generator Tool
	tr.RegisterTool("generate_query", &Tool{
		Name:        "generate_query",
		Description: "Generate a SQL query from natural language",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"question": map[string]interface{}{
					"type":        "string",
					"description": "Natural language question about the database",
				},
				"database_id": map[string]interface{}{
					"type":        "string",
					"description": "Target database identifier",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum result rows",
					"default":     1000,
				},
			},
			Required: []string{"question", "database_id"},
		},
	}, &QueryGeneratorExecutor{store: tr.store})

	// Schema Inspector Tool
	tr.RegisterTool("inspect_schema", &Tool{
		Name:        "inspect_schema",
		Description: "Inspect database schema and structure",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"database_id": map[string]interface{}{
					"type":        "string",
					"description": "Database to inspect",
				},
				"table_pattern": map[string]interface{}{
					"type":        "string",
					"description": "Optional table name pattern",
				},
			},
			Required: []string{"database_id"},
		},
	}, &SchemaInspectorExecutor{store: tr.store})

	// Query Executor Tool
	tr.RegisterTool("execute_query", &Tool{
		Name:        "execute_query",
		Description: "Execute a validated SQL query",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "SQL query to execute",
				},
				"database_id": map[string]interface{}{
					"type":        "string",
					"description": "Target database",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum rows to return",
					"default":     1000,
				},
			},
			Required: []string{"query", "database_id"},
		},
	}, &QueryExecutorExecutor{store: tr.store, validator: tr.validator})
}

// RegisterTool registers a custom tool
func (tr *ToolRegistry) RegisterTool(name string, tool *Tool, executor ToolExecutor) {
	tr.toolMutex.Lock()
	defer tr.toolMutex.Unlock()

	tr.tools[name] = tool
	tr.executors[name] = executor
}

// ListTools returns all available tools
func (tr *ToolRegistry) ListTools(ctx context.Context) ([]Tool, error) {
	tr.toolMutex.RLock()
	defer tr.toolMutex.RUnlock()

	tools := make([]Tool, 0, len(tr.tools))
	for _, tool := range tr.tools {
		tools = append(tools, *tool)
	}
	return tools, nil
}

// ValidateTool validates a tool call
func (tr *ToolRegistry) ValidateTool(name string, arguments map[string]interface{}) error {
	tr.toolMutex.RLock()
	tool, ok := tr.tools[name]
	tr.toolMutex.RUnlock()

	if !ok {
		return fmt.Errorf("tool not found: %s", name)
	}

	// Validate required arguments
	for _, required := range tool.InputSchema.Required {
		if _, ok := arguments[required]; !ok {
			return fmt.Errorf("missing required argument: %s", required)
		}
	}

	return nil
}

// CallTool executes a tool
func (tr *ToolRegistry) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
	tr.toolMutex.RLock()
	executor, ok := tr.executors[name]
	tr.toolMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no executor for tool: %s", name)
	}

	return executor.Execute(ctx, arguments)
}

// ===== Tool Executors =====

// QueryGeneratorExecutor executes the query generation tool
type QueryGeneratorExecutor struct {
	store db.Store
}

func (e *QueryGeneratorExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	question := args["question"].(string)
	databaseID := args["database_id"].(string)

	// TODO: Call LLM to generate query
	// This will be implemented in the LLM adapter layer
	_ = question
	_ = databaseID

	return map[string]interface{}{
		"query":  "SELECT * FROM example",
		"status": "generated",
	}, nil
}

// SchemaInspectorExecutor inspects database schema
type SchemaInspectorExecutor struct {
	store db.Store
}

func (e *SchemaInspectorExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	databaseID := args["database_id"].(string)
	pattern := ""
	if p, ok := args["table_pattern"]; ok {
		pattern = p.(string)
	}

	// TODO: Retrieve schema from database
	_ = databaseID
	_ = pattern

	return map[string]interface{}{
		"tables": []string{},
		"status": "inspected",
	}, nil
}

// QueryExecutorExecutor executes SQL queries
type QueryExecutorExecutor struct {
	store     db.Store
	validator *security.QueryValidator
}

func (e *QueryExecutorExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query := args["query"].(string)
	databaseID := args["database_id"].(string)
	limit := 1000
	if l, ok := args["limit"]; ok {
		limit = int(l.(float64))
	}

	// Validate query
	if err := e.validator.ValidateQuery(query); err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	// TODO: Execute query against database
	_ = databaseID
	_ = limit

	return map[string]interface{}{
		"rows":   []map[string]interface{}{},
		"status": "executed",
	}, nil
}
