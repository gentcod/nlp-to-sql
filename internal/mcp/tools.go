package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	db "github.com/gentcod/nlp-to-sql/internal/database"
	"github.com/gentcod/nlp-to-sql/internal/rag"
	"github.com/gentcod/nlp-to-sql/internal/security"
	mp "github.com/gentcod/nlp-to-sql/mapper"
	"github.com/gentcod/nlp-to-sql/util"
)

// ToolExecutionContext holds optional runtime dependencies for executing tools against a live database and LLM
type ToolExecutionContext struct {
	Conn     *sql.DB
	DBDriver string
	DBName   string
	LLMOpts  rag.LLMOpts
	LLMType  string
}

// ToolRegistry manages MCP tools
type ToolRegistry struct {
	store     db.Store
	execCtx   ToolExecutionContext
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

// ConfigureExecutionContext wires live database and LLM options into the built-in tool executors
func (tr *ToolRegistry) ConfigureExecutionContext(execCtx ToolExecutionContext) {
	tr.toolMutex.Lock()
	tr.execCtx = execCtx
	tr.toolMutex.Unlock()

	tr.registerBuiltInTools()
}

// registerBuiltInTools registers standard tools
func (tr *ToolRegistry) registerBuiltInTools() {
	tr.toolMutex.RLock()
	execCtx := tr.execCtx
	tr.toolMutex.RUnlock()

	// 1. SQL Generator Tool
	tr.RegisterTool("generate_query", &Tool{
		Name:        "generate_query",
		Description: "Generate a validated, read-only SQL SELECT query from a natural language question",
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
				"llm_type": map[string]interface{}{
					"type":        "string",
					"description": "Optional LLM provider type (muse, gemini, openai)",
					"enum":        []string{"muse", "gemini", "openai"},
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum result rows",
					"default":     1000,
				},
			},
			Required: []string{"question", "database_id"},
		},
	}, &QueryGeneratorExecutor{store: tr.store, execCtx: execCtx, validator: tr.validator})

	// 2. Schema Inspector Tool
	tr.RegisterTool("inspect_schema", &Tool{
		Name:        "inspect_schema",
		Description: "Inspect database schema tables and column types",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"database_id": map[string]interface{}{
					"type":        "string",
					"description": "Database to inspect",
				},
				"table_pattern": map[string]interface{}{
					"type":        "string",
					"description": "Optional table name substring filter",
				},
			},
			Required: []string{"database_id"},
		},
	}, &SchemaInspectorExecutor{store: tr.store, execCtx: execCtx})

	// 3. Query Validator Tool (AX: allows agents to pre-flight validate SQL before execution)
	tr.RegisterTool("validate_query", &Tool{
		Name:        "validate_query",
		Description: "Validate a SQL query for syntax correctness, read-only SELECT safety, and restricted column policies",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "SQL query to validate",
				},
				"database_id": map[string]interface{}{
					"type":        "string",
					"description": "Target database identifier",
				},
			},
			Required: []string{"query"},
		},
	}, &QueryValidatorExecutor{validator: tr.validator})

	// 4. Query Executor Tool
	tr.RegisterTool("execute_query", &Tool{
		Name:        "execute_query",
		Description: "Execute a validated read-only SQL SELECT query against the database",
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
	}, &QueryExecutorExecutor{store: tr.store, execCtx: execCtx, validator: tr.validator})
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
		val, ok := arguments[required]
		if !ok || val == nil {
			return fmt.Errorf("missing required argument: %s", required)
		}
		if s, isStr := val.(string); isStr && strings.TrimSpace(s) == "" {
			return fmt.Errorf("missing required argument: %s cannot be empty", required)
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
	store     db.Store
	execCtx   ToolExecutionContext
	validator *security.QueryValidator
}

func (e *QueryGeneratorExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	question, _ := args["question"].(string)
	databaseID, _ := args["database_id"].(string)
	reqLLMType, _ := args["llm_type"].(string)

	llmType := strings.TrimSpace(reqLLMType)
	if llmType == "" {
		llmType = e.execCtx.LLMType
	}
	llmType = rag.NormalizeLLMType(llmType)

	// If LLM and live DB are configured, generate using schema context
	if e.execCtx.Conn != nil && (e.execCtx.LLMOpts.ApiKey != "" || e.execCtx.LLMOpts.BaseUrl != "") {
		opts := e.execCtx.LLMOpts
		if mapper := mp.InitMapper(e.execCtx.DBDriver); mapper != nil {
			dbName := databaseID
			if dbName == "" || dbName == "default" || dbName == "primary" {
				dbName = e.execCtx.DBName
			}
			if schema, err := mapper.MapSchema(e.execCtx.Conn, dbName); err == nil {
				opts.Context = schema
			}
		}

		llm := rag.InitLLM(llmType, opts)
		if llm == nil {
			return nil, fmt.Errorf("unsupported llm_type %q (supported: muse, gemini, openai)", llmType)
		}

		rawSQL, err := llm.GenerateQuery(question)
		if err != nil {
			return nil, fmt.Errorf("LLM query generation failed (%s): %w", llmType, err)
		}
		cleanSQL := util.SanitizeSQL(rawSQL)
		valRes := e.validator.InspectQuery(cleanSQL)
		if !valRes.IsValid {
			return map[string]interface{}{
				"query":                  cleanSQL,
				"llm_type":               llmType,
				"status":                 "validation_failed",
				"validation_stage":       valRes.Stage,
				"error_reason":           valRes.ErrorReason,
				"agent_remediation_hint": valRes.AgentRemediationHint,
			}, fmt.Errorf("generated query failed policy validation: %s", valRes.ErrorReason)
		}
		return map[string]interface{}{
			"query":       valRes.NormalizedSQL,
			"database_id": databaseID,
			"llm_type":    llmType,
			"status":      "generated",
		}, nil
	}

	return map[string]interface{}{
		"query":       "SELECT * FROM example",
		"database_id": databaseID,
		"llm_type":    llmType,
		"status":      "generated",
	}, nil
}

// SchemaInspectorExecutor inspects database schema
type SchemaInspectorExecutor struct {
	store   db.Store
	execCtx ToolExecutionContext
}

func (e *SchemaInspectorExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	databaseID, _ := args["database_id"].(string)
	pattern, _ := args["table_pattern"].(string)

	if e.execCtx.Conn != nil && e.execCtx.DBDriver != "" {
		if mapper := mp.InitMapper(e.execCtx.DBDriver); mapper != nil {
			dbName := databaseID
			if dbName == "" || dbName == "default" || dbName == "primary" {
				dbName = e.execCtx.DBName
			}
			schemaMap, err := mapper.MapSchema(e.execCtx.Conn, dbName)
			if err == nil {
				filteredSchema := make(map[string]map[string]string)
				tables := make([]string, 0, len(schemaMap))
				for tbl, cols := range schemaMap {
					if pattern == "" || strings.Contains(strings.ToLower(tbl), strings.ToLower(pattern)) {
						tables = append(tables, tbl)
						filteredSchema[tbl] = cols
					}
				}
				return map[string]interface{}{
					"database_id": databaseID,
					"tables":      tables,
					"schema":      filteredSchema,
					"status":      "inspected",
				}, nil
			}
		}
	}

	return map[string]interface{}{
		"database_id": databaseID,
		"tables":      []string{},
		"status":      "inspected",
	}, nil
}

// QueryValidatorExecutor validates SQL queries and returns detailed AST & policy diagnostics
type QueryValidatorExecutor struct {
	validator *security.QueryValidator
}

func (e *QueryValidatorExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	res := e.validator.InspectQuery(query)
	return res, nil
}

// QueryExecutorExecutor executes SQL queries
type QueryExecutorExecutor struct {
	store     db.Store
	execCtx   ToolExecutionContext
	validator *security.QueryValidator
}

func (e *QueryExecutorExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	start := time.Now()
	query, _ := args["query"].(string)
	databaseID, _ := args["database_id"].(string)
	limit := 1000
	switch l := args["limit"].(type) {
	case float64:
		limit = int(l)
	case int:
		limit = l
	}

	// Validate query via explicit validation state machine
	valRes := e.validator.InspectQuery(query)
	if !valRes.IsValid {
		return nil, fmt.Errorf("query validation failed at stage %s: %s (hint: %s)",
			valRes.Stage, valRes.ErrorReason, valRes.AgentRemediationHint)
	}

	if e.execCtx.Conn != nil {
		rows, err := db.GetData(e.execCtx.Conn, valRes.NormalizedSQL)
		if err != nil {
			return nil, fmt.Errorf("query execution failed: %w", err)
		}
		if limit > 0 && len(rows) > limit {
			rows = rows[:limit]
		}
		return map[string]interface{}{
			"database_id":   databaseID,
			"query":         valRes.NormalizedSQL,
			"rows":          rows,
			"row_count":     len(rows),
			"query_time_ms": time.Since(start).Milliseconds(),
			"status":        "executed",
		}, nil
	}

	return map[string]interface{}{
		"database_id":   databaseID,
		"query":         valRes.NormalizedSQL,
		"rows":          []map[string]interface{}{},
		"row_count":     0,
		"query_time_ms": time.Since(start).Milliseconds(),
		"status":        "executed",
	}, nil
}
