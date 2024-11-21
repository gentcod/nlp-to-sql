package converter

import (
	"github.com/gentcod/nlp-to-sql/rag"
)

type Converter interface {
	// Convert converts a textual request to database query which is used to get data.
	// The data returned from the database is then converted to textual response containing information based on request context.
	Convert(llmType, dbUrl, dbName string, ragOpts rag.LLMOpts) (string, error)
}