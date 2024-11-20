package rag

import "github.com/xwb1989/sqlparser"

type LLM interface {
	// initializes connection to the LLM API parsing some specified LLMOpts to return generated SQLQuer
	GenerateQuery() (string, error)

	// GenerateResponse takens data gotten after Query has been fired
	// to return data in a textual or conversational manner
	GenerateResponse(data any)
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

func InitLLM(llmType string, opts LLMOpts) LLM {
	if llmType == "gemini" {
		return NewGeminiLLM(opts)
	}

	if llmType == "openai" {
		return NewOpenAiLLM(opts)
	}

	return nil
}

func validQuery(query string) bool {
	_, err := sqlparser.Parse(query)
	return err == nil
}
