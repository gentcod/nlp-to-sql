package converter

import (
	"database/sql"
	"fmt"
	"log"

	mp "github.com/gentcod/nlp-to-sql/mapper"
)

type Converter interface {
	Convert(dbtype, dbUrl, dbName, message string) error
}

type SQLConverter struct {
	sqlQuery string
}

func NewSQLConverter() Converter {
	return &SQLConverter{}
}

func (converter *SQLConverter) Convert(dbtype, dbUrl, dbName, message string) error {

	// GET DB SCHEMA FOR RAG CONTEXT
	conn, err := sql.Open(dbtype, dbUrl)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer conn.Close()

	// store := db.NewStore(conn)

	mapper := mp.InitMapper(dbtype)
	schema, err := mapper.MapSchema(conn, dbName)
	if err != nil {
		log.Fatal("Error mapping schema:", err)
	}

	fmt.Println(schema)

	// QUERY OPENAI API TO GENERATE QUERY USING DBSCHEMA
	// llm := rag.NewGeminiLLM(rag.LLMOpts{
	// 	Query: "I want to check the games that were played by the last registered customer",
	// 	Context: mapper.Schema,
	// 	ApiKey: config.ApiKey,
	// 	OrgId: config.OrgId,
	// 	ProjectId: config.ProjectId,
	// 	Model: config.Model,
	// 	Temp: config.Temp,
	// })
	// query, err := llm.GenerateQuery()
	// if err != nil {
	// 	log.Fatal("Error connecting to LLM: ", err)
	// }

	// fmt.Println(llm)

	// // data, err := db.GetData(conn, query)
	// data, err := db.GetData(conn, query)
	// if err != nil {
	// 	log.Fatal("Error getting queried data: ", err)
	// }

	// fmt.Println(data)

	converter.sqlQuery = message
	return nil
}
