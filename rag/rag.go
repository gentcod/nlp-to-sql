package rag

import "github.com/xwb1989/sqlparser"

// LLM represents the LLM choice for text generation
type LLM interface {
	// GenerateQuery initializes connection to the LLM API parsing some specified LLMOpts 
	// these options are used to generate SQL Query
	GenerateQuery() (string, error)

	// GenerateResponse take data gotten after database has been queried
	// to return response in a textual or conversational manner
	GenerateResponse(data any) (string, error)
}

// LLMOpts contains fields needed to connect to an LLM
type LLMOpts struct {
	DBType    string
	Query     string
	Context   any
	ApiKey    string
	OrgId     string
	ProjectId string
	Model     string
	Temp      string
}

// InitLLM initializes LLM based on type and specification required to communicate with selected LLM API
func InitLLM(llmType string, opts LLMOpts) LLM {
	if llmType == "gemini" {
		return NewGeminiLLM(opts)
	}

	if llmType == "openai" {
		return NewOpenAiLLM(opts)
	}

	return nil
}

// validQuery checks if parsed SQL Query is a valid query
func validQuery(query string) bool {
	_, err := sqlparser.Parse(query)
	return err == nil
}
