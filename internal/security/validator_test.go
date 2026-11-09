package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryValidator_StateTransitions(t *testing.T) {
	qv := NewQueryValidator()

	t.Run("Valid plain SELECT query transitions to VALIDATED", func(t *testing.T) {
		res := qv.InspectQuery("SELECT id, username, full_name FROM users WHERE id = 1")
		assert.True(t, res.IsValid)
		assert.Equal(t, StageValidated, res.Stage)
		assert.Equal(t, "SELECT", res.StatementType)
		assert.NoError(t, qv.ValidateQuery("SELECT id, username FROM users"))
	})

	t.Run("Markdown fenced SQL is sanitized and validated", func(t *testing.T) {
		fenced := "```sql\nSELECT count(*) FROM users;\n```"
		res := qv.InspectQuery(fenced)
		assert.True(t, res.IsValid)
		assert.Equal(t, StageValidated, res.Stage)
		assert.Equal(t, "SELECT count(*) FROM users;", res.NormalizedSQL)
	})

	t.Run("Restricted sensitive column transitions to REJECTED with remediation hint", func(t *testing.T) {
		res := qv.InspectQuery("SELECT id, email, harshed_password FROM auth")
		assert.False(t, res.IsValid)
		assert.Equal(t, StageRejected, res.Stage)
		assert.Contains(t, res.ErrorReason, "password")
		assert.NotEmpty(t, res.AgentRemediationHint)
	})

	t.Run("Destructive DROP/DELETE query transitions to REJECTED", func(t *testing.T) {
		res := qv.InspectQuery("DROP TABLE users")
		assert.False(t, res.IsValid)
		assert.Equal(t, StageRejected, res.Stage)
		assert.NotEmpty(t, res.AgentRemediationHint)
	})

	t.Run("Empty query transitions to REJECTED", func(t *testing.T) {
		res := qv.InspectQuery("   ")
		assert.False(t, res.IsValid)
		assert.Equal(t, StageRejected, res.Stage)
	})
}
