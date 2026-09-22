package converter

import (
	"database/sql"
	"fmt"

	db "github.com/gentcod/nlp-to-sql/internal/database"
	"github.com/gentcod/nlp-to-sql/internal/rag"
	"github.com/gentcod/nlp-to-sql/util"
)

type SQLConverter struct {
	Response string
	Opts     rag.LLMOpts
}

// NewSQLConverter initializes a Converter that can be used to handle SQL queries
func NewSQLConverter(ragOpts rag.LLMOpts) Converter {
	return &SQLConverter{
		Opts: ragOpts,
	}
}

func (converter *SQLConverter) Convert(conn *sql.DB, llmType, que string, schema map[string]map[string]string) (string, error) {
	converter.Opts.Context = schema
	llm := rag.InitLLM(
		llmType,
		converter.Opts,
	)

	mdl, err := llm.GetModelContext()
	if err != nil {
		return converter.Response, fmt.Errorf("error getting model context: %v", err)
	}

	model, ok := mdl.(rag.Model)
	if !ok {
		return converter.Response, fmt.Errorf("error casting Gemini RAG model: %v", err)
	}

	query, err := model.GenerateCachedResponse(que)
	// query, err := llm.GenerateQuery(que)
	fmt.Println(query)
	if err != nil {
		return converter.Response, fmt.Errorf("error evaluating chat with LLM: %v", err)
	}

	if !util.ValidQuery(query) {
		return converter.Response, fmt.Errorf("the generated query violates the rule of the policy of omitting sensitive data.")
	}

	data, err := db.GetData(conn, query)
	if err != nil {
		return converter.Response, fmt.Errorf("error getting queried data: %v", err)
	}

	converter.Response, err = llm.GenerateResponse(data, que)
	if err != nil {
		return converter.Response, fmt.Errorf("error converting data to textual response: %v", err)
	}

	return converter.Response, nil
}
