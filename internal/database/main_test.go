package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/gentcod/nlp-to-sql/util"
	_ "github.com/lib/pq"
)

var testQueries *Queries
var testDB *sql.DB

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../../test.env")
	if err != nil {
		log.Fatal("cannot load config", err)
	}

	testDB, err = sql.Open(config.DBDriver, config.DBUrl)
	if err != nil {
		log.Fatal("Couldn't connect to db:", err)
	}

	testQueries = New(testDB)

	//Initialize connection test, terminate test if error occurs
	os.Exit(m.Run())
}
