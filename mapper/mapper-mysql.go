package mapper

import (
	"database/sql"
	// "fmt"
)

type MySQLMapper struct {
	Schema map[string]map[string]string
}

// NewMySQLMapper initializes a Mapper that communicates with MySQL Databases.
func NewMySQLMapper() Mapper {
	return &MySQLMapper{}
}

func (mapper *MySQLMapper) getTables(db *sql.DB, dbName string) ([]string, error) {
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

func (mapper *MySQLMapper) getColumns(db *sql.DB, dbName string, tableName string) (map[string]string, error) {
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

func (mapper *MySQLMapper) MapSchema(db *sql.DB, dbName string) (map[string]map[string]string, error) {
	tables, err := mapper.getTables(db, dbName)
	if err != nil {
		return nil, err
	}

	mapper.Schema = make(map[string]map[string]string)

	for _, table := range tables {
		columns, err := mapper.getColumns(db, dbName, table)
		if err != nil {
			return nil, err
		}
		mapper.Schema[table] = columns
	}
	// fmt.Println("Schema: ",mapper.Schema)
	return mapper.Schema, nil
}
