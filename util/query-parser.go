package util

import (
	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/xwb1989/sqlparser"
)

// ValidQuery checks if parsed SQL Query is a valid query
// a valid query in this case is a correct SQL which is also a SELECT statement.
// It adds extra security to ensure only SELECT queries are validated.
func ValidQuery(query string) bool {
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
