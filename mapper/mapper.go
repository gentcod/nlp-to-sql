package mapper

import (
	"database/sql"
	// "fmt"
)

type Mapper struct {
	Schema map[string]map[string]string
}

func NewMapper() *Mapper {
	return &Mapper{}
}

func getTables(db *sql.DB, dbName string) ([]string, error) {
	query := `SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ?;`

	rows, err := db.Query(query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}
	return tables, nil
}

func getColumns(db *sql.DB, dbName string, tableName string) (map[string]string, error) {
	query := `
		SELECT COLUMN_NAME, DATA_TYPE 
		FROM INFORMATION_SCHEMA.COLUMNS 
		WHERE TABLE_SCHEMA = ?
		  AND TABLE_NAME = ?;
	`

	rows, err := db.Query(query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]string)
	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			return nil, err
		}
		columns[columnName] = dataType
	}
	return columns, nil
}

func (mapper *Mapper) MapSchema(db *sql.DB, dbName string) error {
	tables, err := getTables(db, dbName)
	if err != nil {
		return err
	}

	mapper.Schema = make(map[string]map[string]string)

	for _, table := range tables {
		columns, err := getColumns(db, dbName, table)
		if err != nil {
			return err
		}
		mapper.Schema[table] = columns
	}
	// fmt.Println("Schema: ",mapper.Schema)
	return nil
}

// func addSynonyms(schema map[string]map[string]string, synonyms map[string]string) map[string]map[string]string {
// 	for table, columns := range schema {
// 		fmt.Printf("Table: %+v, Columns: %+v", table, columns)
// 		for synonym, actualColumn := range synonyms {
// 			if _, exists := columns[actualColumn]; exists {
// 				columns[synonym] = actualColumn
// 			}
// 		}
// 	}
// 	return schema
// }