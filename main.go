package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gentcod/nlp-to-sql/api"
	db "github.com/gentcod/nlp-to-sql/internal/database"
	mp "github.com/gentcod/nlp-to-sql/mapper"

	"github.com/gentcod/nlp-to-sql/rag"
	"github.com/gentcod/nlp-to-sql/util"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"
)

func main() {
	config, err := util.LoadConfig(".env")
	if err != nil {
		log.Fatal("cannot load config", err)
	}

	// conn, err := sql.Open("mysql", config.DBUrl)
	conn, err := sql.Open("postgres", config.DBUrl)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer conn.Close()

	// store := db.NewStore(conn)

	mapper := mp.InitMapper("postgres")
	schema, err := mapper.MapSchema(conn, config.DBName)
	if err != nil {
		log.Fatal("Error mapping schema:", err)
	}

	llm := rag.NewGeminiLLM(rag.LLMOpts{
		DBType:    "postgres",
		Query:     "I want to check the details of the last registered customer",
		Context:   schema,
		ApiKey:    config.ApiKey,
		OrgId:     config.OrgId,
		ProjectId: config.ProjectId,
		Model:     config.Model,
		Temp:      config.Temp,
	})
	query, err := llm.GenerateQuery()
	if err != nil {
		log.Fatal("Error connecting to LLM: ", err)
	}

	fmt.Println("Query: ", query)

	data, err := db.GetData(conn, query)
	if err != nil {
		log.Fatal("Error getting queried data: ", err)
	}

	fmt.Println(data)

	// runGinServer(config, store)
}

func runGinServer(config util.Config, store db.Store) {
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal("Couldn't initialize the server:", err)
	}

	err = server.Start(config.Port)
	if err != nil {
		log.Fatal("Couldn't start up server:", err)
	}
}
