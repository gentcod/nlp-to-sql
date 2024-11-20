package mapper

import "database/sql"

type Mapper interface {
	getTables(db *sql.DB, dbName string) ([]string, error)
	getColumns(db *sql.DB, dbName string, tableName string) (map[string]string, error)
	MapSchema(db *sql.DB, dbName string) (map[string]map[string]string, error)
}

func InitMapper(dbType string) Mapper {

	if dbType == "mysql" {
		return NewMySQLMapper()
	}

	if dbType == "postgres" {
		return NewPQMapper()
	}

	return nil
}