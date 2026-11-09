package rag

import (
	"fmt"
	"strings"

	"github.com/gentcod/nlp-to-sql/util"
)

const (
	LLMTypeGemini = "gemini"
	LLMTypeOpenAI = "openai"
	LLMTypeMuse   = "muse"
)

// LLM represents the LLM choice for text generation
type LLM interface {
	// GenerateQuery initializes connection to the LLM API parsing some specified LLMOpts
	// these options are used to generate SQL Query
	GenerateQuery(que string) (string, error)

	// GenerateResponse take data gotten after database has been queried
	// to return response in a textual or conversational manner
	// using the question asked for a furher context-aware response
	GenerateResponse(data any, que string) (string, error)

	GetModelContext() (interface{}, error)
}

// LLMOpts contains fields needed to connect to an LLM
type LLMOpts struct {
	Context   any
	ApiKey    string
	BaseUrl   string
	OrgId     string
	ProjectId string
	Model     string
	Temp      string
}

// SanitizeContext ensures any schema map in LLMOpts.Context has sensitive columns stripped before reaching any LLM.
func SanitizeContext(ctx any) any {
	if schemaMap, ok := ctx.(map[string]map[string]string); ok {
		return util.SanitizeSchema(schemaMap)
	}
	return ctx
}

// queryInstructions builds the unified zero-trust SQL generation instructions shared by all LLM providers (Muse, Gemini, OpenAI).
func queryInstructions(schema any) string {
	safeSchema := SanitizeContext(schema)
	return strings.Join([]string{
		"Generate a SQL query from the user's question.",
		"STRICT SECURITY & PROJECTION RULES:",
		"1. Only read-only SELECT queries or aggregate read queries (COUNT, SUM, AVG, MIN, MAX) are allowed.",
		fmt.Sprintf("2. The sanitized database schema (table -> column -> type) is: %v", safeSchema),
		"3. Only reference tables and columns explicitly present in the provided schema.",
		"4. When selecting rows from a table, prefer listing explicit safe column names from the schema (e.g., SELECT id, email, role FROM ...) rather than SELECT * so hidden or sensitive columns are never queried.",
		"5. Never reference or filter on sensitive fields such as password, hashed_password, harshed_password, secret, token, or api keys under any circumstances.",
		"6. Return ONLY the raw SQL query as plain text without markdown code blocks or conversational text.",
	}, "\n")
}

// NormalizeLLMType normalizes an LLM provider identifier, defaulting to "gemini" when empty.
func NormalizeLLMType(llmType string) string {
	normalized := strings.ToLower(strings.TrimSpace(llmType))
	if normalized == "" {
		return LLMTypeGemini
	}
	return normalized
}

// InitLLM initializes LLM based on type and specification required to communicate with selected LLM API
func InitLLM(llmType string, opts LLMOpts) LLM {
	opts.Context = SanitizeContext(opts.Context)
	switch NormalizeLLMType(llmType) {
	case LLMTypeGemini:
		return NewGeminiLLM(opts)
	case LLMTypeOpenAI:
		return NewOpenAiLLM(opts)
	case LLMTypeMuse:
		return NewMuseLLM(opts)
	default:
		return nil
	}
}
