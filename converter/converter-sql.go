package converter

import (
	"database/sql"
	"fmt"
	"time"

	db "github.com/gentcod/nlp-to-sql/internal/database"
	"github.com/gentcod/nlp-to-sql/internal/rag"
	"github.com/gentcod/nlp-to-sql/internal/security"
	"github.com/gentcod/nlp-to-sql/util"
)

type SQLConverter struct {
	Response  string
	Opts      rag.LLMOpts
	validator *security.QueryValidator
}

// NewSQLConverter initializes a Converter that can be used to handle SQL queries
func NewSQLConverter(ragOpts rag.LLMOpts) Converter {
	return &SQLConverter{
		Opts:      ragOpts,
		validator: security.NewQueryValidator(),
	}
}

// ConvertDetailed executes the full NLP-to-SQL state machine without shared struct data races
func (converter *SQLConverter) ConvertDetailed(conn *sql.DB, llmType, que string, schema map[string]map[string]string) (ConversionDetails, error) {
	start := time.Now()
	details := ConversionDetails{
		Question: que,
		Stage:    PipelineStageGenerating,
	}

	// Copy options locally so concurrent requests never race on converter.Opts.Context
	localOpts := converter.Opts
	localOpts.Context = schema

	llm := rag.InitLLM(llmType, localOpts)
	if llm == nil {
		details.Stage = PipelineStageFailed
		details.LatencyMs = time.Since(start).Milliseconds()
		return details, fmt.Errorf("unsupported LLM provider: %s", llmType)
	}

	// Stage 1: Generate SQL (attempt CachedContent first; gracefully fall back to direct GenerateQuery if schema < 32k token cache threshold)
	var rawQuery string
	var err error
	if mdl, ctxErr := llm.GetModelContext(); ctxErr == nil && mdl != nil {
		if model, ok := mdl.(*rag.Model); ok {
			rawQuery, err = model.GenerateCachedResponse(que)
		}
	}
	if rawQuery == "" || err != nil {
		rawQuery, err = llm.GenerateQuery(que)
	}
	if err != nil {
		details.Stage = PipelineStageFailed
		details.LatencyMs = time.Since(start).Milliseconds()
		return details, fmt.Errorf("error evaluating chat with LLM: %v", err)
	}

	// Stage 2: Sanitize & Validate SQL AST and Policy
	details.Stage = PipelineStageValidating
	cleanQuery := util.SanitizeSQL(rawQuery)
	details.SQL = cleanQuery

	validator := converter.validator
	if validator == nil {
		validator = security.NewQueryValidator()
	}
	valRes := validator.InspectQuery(cleanQuery)
	if !valRes.IsValid || !util.ValidQuery(cleanQuery) {
		details.Stage = PipelineStageFailed
		details.RemediationHint = valRes.AgentRemediationHint
		details.LatencyMs = time.Since(start).Milliseconds()
		return details, fmt.Errorf("the generated query violates the rule of the policy of omitting sensitive data: %s", valRes.ErrorReason)
	}

	// Stage 3: Execute Query against Database
	details.Stage = PipelineStageExecuting
	data, err := db.GetData(conn, valRes.NormalizedSQL)
	if err != nil {
		details.Stage = PipelineStageFailed
		details.LatencyMs = time.Since(start).Milliseconds()
		return details, fmt.Errorf("error getting queried data: %v", err)
	}
	details.Data = data

	// Stage 4: Summarize Data into Natural Language Response
	details.Stage = PipelineStageSummarizing
	summary, err := llm.GenerateResponse(data, que)
	if err != nil {
		details.Stage = PipelineStageFailed
		details.LatencyMs = time.Since(start).Milliseconds()
		return details, fmt.Errorf("error converting data to textual response: %v", err)
	}

	details.Summary = summary
	details.Stage = PipelineStageCompleted
	details.LatencyMs = time.Since(start).Milliseconds()
	return details, nil
}

func (converter *SQLConverter) Convert(conn *sql.DB, llmType, que string, schema map[string]map[string]string) (string, error) {
	details, err := converter.ConvertDetailed(conn, llmType, que, schema)
	if err != nil {
		return "", err
	}
	return details.Summary, nil
}
