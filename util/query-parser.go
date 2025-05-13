package util

import (
	"strings"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/xwb1989/sqlparser"
)

const (
	keywordAPI             = "api"
	keywordPassword        = "password"
	keywordHarshedPassword = "harshedpassword"
)

// AllowedKeywords contains all valid database types
var allowedKeywords = map[string]bool{
	keywordAPI:             true,
	keywordPassword:        true,
	keywordHarshedPassword: true,
}

// containsRestrictedWords checks if the any sensitive keyword is present in a query.
// It helps to add checks in a case where prompts could be engineered to disregard safety checks.
func containsRestrictedWords(input string) bool {
	input = strings.ToLower(input)

	for word := range allowedKeywords {
		if strings.Contains(input, strings.ToLower(word)) {
			return true
		}
	}
	return false
}

// ValidQuery checks if parsed SQL Query is a valid query
// a valid query in this case is a correct SQL which is also a SELECT statement.
// It adds extra security to ensure only SELECT queries are validated.
func ValidQuery(query string) bool {
	if containsRestrictedWords(query) {
		return false
	}

	mStmt, merr := sqlparser.Parse(query)
	if mStmt != nil {
		switch mStmt := mStmt.(type) {
		case *sqlparser.Select:
			_ = mStmt
			return true
		default:
			return false
		}
	}

	pStmt, perr := parser.ParseOne(query)
	if pStmt.AST != nil {
		switch pStmt.AST.StatementTag() {
		case "SELECT":
			return true
		default:
			return false
		}
	}

	return merr == nil || perr == nil
}
