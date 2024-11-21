package db

import (
	"database/sql"
	"fmt"
	// "log"
)

// Store provides all functions to execute db SQL queries and transactions
type Store interface {
	Querier
}

// SQLStore provides all functions to execute db SQL queries
type SQLStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

// GetData queries the database to return related data
func GetData(db *sql.DB, query string) ([]map[string]interface{}, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch column names: %w", err)
	}

	values := make([]interface{}, len(columns))
	valPointers := make([]interface{}, len(columns))
	for i := range values {
		valPointers[i] = &values[i]
	}

	results := []map[string]interface{}{}
	for rows.Next() {
		if err := rows.Scan(valPointers...); err != nil {
			return nil, fmt.Errorf("row scanning failed: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return results, nil
}

// func DebugQuery(db *sql.DB) {
// 	// Sample query
// 	// query := `SELECT g.name AS name, u.userId AS userId
// 	// 	FROM Game g
// 	// 	JOIN Ticket t ON g.gameId = t.gameId
// 	// 	JOIN User u ON t.userId = u.userId
// 	// 	ORDER BY u.createdAt DESC
// 	// 	LIMIT 1`
// 	query := `select * from users u where u.username like 'ify%'`

// 	// Execute query
// 	rows, err := db.Query(query)
// 	if err != nil {
// 		log.Fatalf("Query execution failed: %v", err)
// 	}
// 	defer rows.Close()

// 	// Get column names
// 	columns, err := rows.Columns()
// 	if err != nil {
// 		log.Fatalf("Failed to fetch columns: %v", err)
// 	}

// 	// Iterate over rows
// 	for rows.Next() {
// 		values := make([]interface{}, len(columns))
// 		for i := range values {
// 			values[i] = new(interface{})
// 		}

// 		if err := rows.Scan(values...); err != nil {
// 			log.Fatalf("Failed to scan row: %v", err)
// 		}

// 		// Print row data
// 		rowData := make(map[string]interface{})
// 		for i, colName := range columns {
// 			val := *(values[i].(*interface{}))
// 			rowData[colName] = val
// 		}
// 		fmt.Printf("Row: %v\n", rowData)
// 	}

// 	if err := rows.Err(); err != nil {
// 		log.Fatalf("Error iterating over rows: %v", err)
// 	}
// }
