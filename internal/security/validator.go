package security

import (
	"fmt"
	"strings"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/gentcod/nlp-to-sql/util"
	"github.com/xwb1989/sqlparser"
)

// ValidationStage represents the explicit state of the SQL validation pipeline
type ValidationStage string

const (
	StageSanitize        ValidationStage = "SANITIZE_NORMALIZE"
	StageRestrictedCheck ValidationStage = "RESTRICTED_IDENTIFIER_CHECK"
	StageASTVerify       ValidationStage = "AST_READONLY_VERIFY"
	StageValidated       ValidationStage = "VALIDATED"
	StageRejected        ValidationStage = "REJECTED"
)

// ValidationResult provides rich diagnostics for Users (UX), Developers (DX), and AI Agents (AX)
type ValidationResult struct {
	IsValid              bool            `json:"is_valid"`
	Stage                ValidationStage `json:"stage"`
	NormalizedSQL        string          `json:"normalized_sql,omitempty"`
	StatementType        string          `json:"statement_type,omitempty"`
	ErrorReason          string          `json:"error_reason,omitempty"`
	AgentRemediationHint string          `json:"agent_remediation_hint,omitempty"`
}

// QueryValidator validates SQL queries using a deterministic 4-stage state machine
type QueryValidator struct {
	restrictedWords []string
}

// NewQueryValidator creates a new query validator
func NewQueryValidator() *QueryValidator {
	return &QueryValidator{
		restrictedWords: []string{"password", "harshedpassword", "hashed_password", "api", "secret", "token"},
	}
}

// InspectQuery runs the explicit validation state machine and returns a structured ValidationResult
func (qv *QueryValidator) InspectQuery(rawQuery string) ValidationResult {
	// Stage 1: Sanitize & Normalize
	normalized := util.SanitizeSQL(rawQuery)
	if len(normalized) == 0 {
		return ValidationResult{
			IsValid:              false,
			Stage:                StageRejected,
			ErrorReason:          "query cannot be empty",
			AgentRemediationHint: "Provide a non-empty SQL SELECT statement.",
		}
	}

	// Stage 2: Restricted Identifier Check
	lower := strings.ToLower(normalized)
	for _, word := range qv.restrictedWords {
		if strings.Contains(lower, word) {
			return ValidationResult{
				IsValid:              false,
				Stage:                StageRejected,
				NormalizedSQL:        normalized,
				ErrorReason:          fmt.Sprintf("query references restricted or sensitive identifier: %q", word),
				AgentRemediationHint: fmt.Sprintf("Rewrite the SELECT query to explicitly list safe columns and omit sensitive fields like %q.", word),
			}
		}
	}

	// Stage 3: AST Read-Only Verification (MySQL & PostgreSQL dialects)
	if mStmt, merr := sqlparser.Parse(normalized); merr == nil && mStmt != nil {
		switch mStmt.(type) {
		case *sqlparser.Select:
			return ValidationResult{
				IsValid:       true,
				Stage:         StageValidated,
				NormalizedSQL: normalized,
				StatementType: "SELECT",
			}
		default:
			return ValidationResult{
				IsValid:              false,
				Stage:                StageRejected,
				NormalizedSQL:        normalized,
				StatementType:        fmt.Sprintf("%T", mStmt),
				ErrorReason:          "only read-only SELECT statements are permitted",
				AgentRemediationHint: "Convert the statement into a read-only SELECT query. Mutating statements (INSERT, UPDATE, DELETE, DROP, ALTER) are strictly forbidden.",
			}
		}
	}

	if pStmt, perr := parser.ParseOne(normalized); perr == nil && pStmt.AST != nil {
		tag := pStmt.AST.StatementTag()
		if tag == "SELECT" {
			return ValidationResult{
				IsValid:       true,
				Stage:         StageValidated,
				NormalizedSQL: normalized,
				StatementType: tag,
			}
		}
		return ValidationResult{
			IsValid:              false,
			Stage:                StageRejected,
			NormalizedSQL:        normalized,
			StatementType:        tag,
			ErrorReason:          fmt.Sprintf("statement type %q is not permitted; only SELECT is allowed", tag),
			AgentRemediationHint: "Use a read-only SELECT statement instead of " + tag + ".",
		}
	}

	return ValidationResult{
		IsValid:              false,
		Stage:                StageRejected,
		NormalizedSQL:        normalized,
		ErrorReason:          "SQL syntax validation failed for both PostgreSQL and MySQL dialects",
		AgentRemediationHint: "Check the SQL syntax, balanced quotes/parentheses, and ensure only a single valid SELECT query is returned without conversational text.",
	}
}

// ValidateQuery checks if the query passes all stages of the validation state machine
func (qv *QueryValidator) ValidateQuery(query string) error {
	res := qv.InspectQuery(query)
	if !res.IsValid {
		return fmt.Errorf("%s (hint: %s)", res.ErrorReason, res.AgentRemediationHint)
	}
	return nil
}
