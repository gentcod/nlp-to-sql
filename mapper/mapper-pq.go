package mapper

import (
	"database/sql"
)

type PQMapper struct {
	Schema map[string]map[string]string
}

// NewPQMapper initializes a Mapper that communicates with Postgres Databases.
func NewPQMapper() Mapper {
	return &PQMapper{}
}

func (mapper *PQMapper) getTables(db *sql.DB, dbName string) ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public';
	`

	rows, err := db.Query(query)
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

func (mapper *PQMapper) getColumns(db *sql.DB, dbName string, tableName string) (map[string]string, error) {
	query := `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = 'public'
		AND table_name = $1;
	`

	rows, err := db.Query(query, tableName)
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

func (mapper *PQMapper) MapSchema(db *sql.DB, dbName string) (map[string]map[string]string, error) {
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
	return mapper.Schema, nil
}
