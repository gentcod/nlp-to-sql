package converter

import (
	"database/sql"
)

type Converter interface {
	// Convert converts a textual request to database query which is used to get data.
	// The data returned from the database is then converted to textual response containing information based on the request context.
	Convert(conn *sql.DB, llmType, que string, schema map[string]map[string]string) (string, error)
}
