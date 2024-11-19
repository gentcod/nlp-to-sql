package main

import (
	"database/sql"
	"log"

	"github.com/gentcod/nlp-to-sql/api"
	db "github.com/gentcod/nlp-to-sql/internal/database"
	mp "github.com/gentcod/nlp-to-sql/mapper"
	"github.com/gentcod/nlp-to-sql/rag"
	"github.com/gentcod/nlp-to-sql/util"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

func main() {
	config, err := util.LoadConfig(".env")
	if err != nil {
		log.Fatal("cannot load config", err)
	}

	conn, err := sql.Open("mysql", config.DBUrl)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer conn.Close()

	// store := db.NewStore(conn)

	mapper := mp.NewMapper()
	err = mapper.MapSchema(conn, config.DBName)
	if err != nil {
		log.Fatal("Error mapping schema:", err)
	}

	err = rag.Connllm(rag.LLMOpts{
		Query: "What can you say about Oyefule Oluwatayo",
		Context: mapper.Schema,
		ApiKey: config.ApiKey,
		OrgId: config.OrgId,
		ProjectId: config.ProjectId,
		Model: config.Model,
		Temp: config.Temp,
	})
	if err != nil {
		log.Fatal("Error connecting to LLM: ", err)
	}

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