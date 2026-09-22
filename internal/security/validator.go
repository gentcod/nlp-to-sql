package security

import "fmt"

// QueryValidator validates SQL queries to ensure they are safe
type QueryValidator struct {
}

// NewQueryValidator creates a new query validator
func NewQueryValidator() *QueryValidator {
	return &QueryValidator{}
}

// ValidateQuery checks if the query is a simple SELECT and doesn't contain destructive commands
func (qv *QueryValidator) ValidateQuery(query string) error {
	// A basic placeholder for query validation.
	// In production, this would employ an SQL parser to ensure it's a READ-ONLY query.
	if len(query) == 0 {
		return fmt.Errorf("query cannot be empty")
	}
	return nil
}
