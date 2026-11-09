package converter

import (
	"database/sql"
)

// PipelineStage represents the explicit execution state of an NLP-to-SQL conversion
type PipelineStage string

const (
	PipelineStageIdle        PipelineStage = "IDLE"
	PipelineStageGenerating  PipelineStage = "GENERATING_SQL"
	PipelineStageValidating  PipelineStage = "VALIDATING_SQL"
	PipelineStageExecuting   PipelineStage = "EXECUTING_QUERY"
	PipelineStageSummarizing PipelineStage = "SUMMARIZING_RESPONSE"
	PipelineStageCompleted   PipelineStage = "COMPLETED"
	PipelineStageFailed      PipelineStage = "FAILED"
)

// ConversionDetails captures structured telemetry across each stage of the NLP-to-SQL pipeline
type ConversionDetails struct {
	Question        string                   `json:"question"`
	SQL             string                   `json:"sql"`
	Data            []map[string]interface{} `json:"data"`
	Summary         string                   `json:"response"`
	Stage           PipelineStage            `json:"stage"`
	LatencyMs       int64                    `json:"latency_ms"`
	RemediationHint string                   `json:"remediation_hint,omitempty"`
}

type Converter interface {
	// Convert converts a textual request to database query which is used to get data.
	// The data returned from the database is then converted to textual response containing information based on the request context.
	Convert(conn *sql.DB, llmType, que string, schema map[string]map[string]string) (string, error)
}

// DetailedConverter is an optional extension interface that returns structured pipeline output for UI and Agent consumers
type DetailedConverter interface {
	Converter
	ConvertDetailed(conn *sql.DB, llmType, que string, schema map[string]map[string]string) (ConversionDetails, error)
}
