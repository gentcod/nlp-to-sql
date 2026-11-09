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
	keywordHashedPassword  = "hashed_password"
	keywordSecret          = "secret"
	keywordToken           = "token"
)

// AllowedKeywords contains restricted sensitive column/identifier keywords
var allowedKeywords = map[string]bool{
	keywordAPI:             true,
	keywordPassword:        true,
	keywordHarshedPassword: true,
	keywordHashedPassword:  true,
	keywordSecret:          true,
	keywordToken:           true,
}

// IsSensitiveIdentifier checks whether a column or identifier name contains restricted sensitive keywords.
func IsSensitiveIdentifier(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return false
	}
	for word := range allowedKeywords {
		if strings.Contains(lower, strings.ToLower(word)) {
			return true
		}
	}
	return false
}

// SanitizeSchema returns a deep copy of a database schema map with all sensitive columns removed.
// This ensures LLMs (Muse, Gemini, OpenAI) never see sensitive columns in their prompt context.
func SanitizeSchema(schema map[string]map[string]string) map[string]map[string]string {
	if schema == nil {
		return nil
	}
	sanitized := make(map[string]map[string]string, len(schema))
	for table, cols := range schema {
		safeCols := make(map[string]string, len(cols))
		for colName, colType := range cols {
			if !IsSensitiveIdentifier(colName) {
				safeCols[colName] = colType
			}
		}
		sanitized[table] = safeCols
	}
	return sanitized
}

// SanitizeSQL strips markdown code fences and trims excess whitespace that LLMs may emit.
func SanitizeSQL(query string) string {
	trimmed := strings.TrimSpace(query)
	if strings.HasPrefix(trimmed, "```") {
		// Strip opening fence line (e.g. ```sql or ```postgresql)
		if idx := strings.IndexByte(trimmed, '\n'); idx != -1 {
			trimmed = trimmed[idx+1:]
		} else {
			trimmed = strings.TrimPrefix(trimmed, "```sql")
			trimmed = strings.TrimPrefix(trimmed, "```postgresql")
			trimmed = strings.TrimPrefix(trimmed, "```mysql")
			trimmed = strings.TrimPrefix(trimmed, "```")
		}
	}
	trimmed = strings.TrimSpace(trimmed)
	if strings.HasSuffix(trimmed, "```") {
		trimmed = strings.TrimSuffix(trimmed, "```")
	}
	return strings.TrimSpace(trimmed)
}

// containsRestrictedWords checks if the any sensitive keyword is present in a query.
// It helps to add checks in a case where prompts could be engineered to disregard safety checks.
func containsRestrictedWords(input string) bool {
	return IsSensitiveIdentifier(input)
}

// ValidQuery checks if parsed SQL Query is a valid query
// a valid query in this case is a correct SQL which is also a SELECT statement.
// It adds extra security to ensure only SELECT queries are validated.
func ValidQuery(query string) bool {
	clean := SanitizeSQL(query)
	if clean == "" {
		return false
	}

	if containsRestrictedWords(clean) {
		return false
	}

	mStmt, merr := sqlparser.Parse(clean)
	if merr == nil && mStmt != nil {
		switch mStmt.(type) {
		case *sqlparser.Select:
			return true
		default:
			return false
		}
	}

	pStmt, perr := parser.ParseOne(clean)
	if perr == nil && pStmt.AST != nil {
		switch pStmt.AST.StatementTag() {
		case "SELECT":
			return true
		default:
			return false
		}
	}

	return false
}
