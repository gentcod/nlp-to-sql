package main

import (
	"database/sql"
	"log"

	// "github.com/gentcod/nlp-to-sql/api"
	"github.com/gentcod/nlp-to-sql/chat"
	conv "github.com/gentcod/nlp-to-sql/converter"

	"github.com/gentcod/nlp-to-sql/rag"
	"github.com/gentcod/nlp-to-sql/util"
	_ "github.com/go-sql-driver/mysql"
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

	converter := conv.NewSQLConverter(rag.LLMOpts{
		ApiKey:    config.ApiKey,
		OrgId:     config.OrgId,
		ProjectId: config.ProjectId,
		Model:     config.Model,
		Temp:      config.Temp,
	})
	// runGinServer(config, store)
	runChatServer(config, converter)
}

// func runGinServer(config util.Config, store db.Store) {
// 	server, err := api.NewServer(config, store)
// 	if err != nil {
// 		log.Fatal("couldn't initialize the server:", err)
// 	}

// 	err = server.Start(config.Port)
// 	if err != nil {
// 		log.Fatal("couldn't start up server:", err)
// 	}
// }

func runChatServer(config util.Config, converter conv.Converter) {
	server, err := chat.NewWebSocketServer(config, converter)
	if err != nil {
		log.Fatal("couldn't initialize the chat-server:", err)
	}

	err = server.StartChatServer(config)
	if err != nil {
		log.Fatal("couldn't start up server:", err)
	}
}
