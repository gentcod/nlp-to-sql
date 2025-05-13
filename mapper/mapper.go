package mapper

import (
	"database/sql"
)

const (
	dBTypeMySQL    = "mysql"
	dBTypePostgres = "postgres"
	dBTypeSQLite   = "sqlite"
)

// AlloweddBTypes contains all valid database types
var allowedDBTypes = map[string]bool{
	dBTypeMySQL:    true,
	dBTypePostgres: true,
	dBTypeSQLite:   true,
}

// IsValid checks if the DBType is valid
func isValidDBType(s string) bool {
	_, ok := allowedDBTypes[s]
	return ok
}

// Mapper connets to the database to get database information.
type Mapper interface {
	// getTables is used to get database Entities or Table names.
	getTables(db *sql.DB, dbName string) ([]string, error)

	// getColumns is used to get column names, field names or properties from the Entities or tables in the database.
	getColumns(db *sql.DB, dbName string, tableName string) (map[string]string, error)

	// MapSchema returns a schema declaration of the database.
	MapSchema(db *sql.DB, dbName string) (map[string]map[string]string, error)
}

// InitMapper returns mapper based on database type.
func InitMapper(dbType string) Mapper {
	if isValidDBType(dbType) {
		if dbType == "mysql" {
			return NewMySQLMapper()
		}

		if dbType == "postgres" {
			return NewPQMapper()
		}
	}

	return nil
}
