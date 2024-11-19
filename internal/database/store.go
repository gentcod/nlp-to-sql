package db

import (
	"database/sql"
)

//Store provides all functions to execute db SQL queries and transactions
type Store interface {
	Querier
}

//SQLStore provides all functions to execute db SQL queries
type SQLStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db: db,
		Queries: New(db),
	}
}