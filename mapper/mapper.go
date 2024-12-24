package mapper

import (
	"database/sql"
)

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

type dBType string

const (
	dBTypeMySQL    dBType = "mysql"
	dBTypePostgres dBType = "postgres"
	dBTypeSQLite   dBType = "sqlite"
)

// AlloweddBTypes contains all valid database types
var AllowedDBTypes = map[dBType]bool{
	dBTypeMySQL:    true,
	dBTypePostgres: true,
	dBTypeSQLite:   true,
}

// IsValid checks if the DBType is valid
func isValidDBType(s string) bool {
	dt := dBType(s)
	_, ok := AllowedDBTypes[dt]
	return ok
}