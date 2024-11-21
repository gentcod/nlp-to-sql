package converter

import (
	"database/sql"
	"fmt"

	db "github.com/gentcod/nlp-to-sql/internal/database"
	mp "github.com/gentcod/nlp-to-sql/mapper"
	"github.com/gentcod/nlp-to-sql/rag"
)

type SQLConverter struct {
	Response string
}

// NewSQLConverter initializes a Converter that can be used to handle SQL queries
func NewSQLConverter() Converter {
	return &SQLConverter{}
}

func (converter *SQLConverter) Convert(llmType, dbUrl, dbName string, ragOpts rag.LLMOpts) (string, error) {

	conn, err := sql.Open(ragOpts.DBType, dbUrl)
	if err != nil {
		return converter.Response, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer conn.Close()

	mapper := mp.InitMapper(ragOpts.DBType)
	schema, err := mapper.MapSchema(conn, dbName)
	if err != nil {
		return converter.Response, fmt.Errorf("error mapping schema: %v", err)
	}

	fmt.Printf("Database Schema: %+v\n", schema)

	ragOpts.Context = schema
	llm := rag.InitLLM(
		llmType,
		ragOpts,
	)
	query, err := llm.GenerateQuery()
	if err != nil {
		return converter.Response, fmt.Errorf("error evaluating chat with LLM: %v", err)
	}

	fmt.Printf("Query: %+v\n", query)

	data, err := db.GetData(conn, query)
	if err != nil {
		return converter.Response, fmt.Errorf("error getting queried data: %v", err)
	}

	fmt.Printf("Queried data: %+v\n", data)

	converter.Response, err = llm.GenerateResponse(data)
	if err != nil {
		return converter.Response, fmt.Errorf("error converting data to textual response: %v", err)
	}

	return converter.Response, nil
}
