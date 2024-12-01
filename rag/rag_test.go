package rag

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidQuery(t *testing.T) {
	vSQuery := `SELECT * FROM users;`
	vUQuery := `DELETE FROM users WHERE id = $1;`
	vDQuery := `INSERT INTO users (id, auth_id, username, full_name)
	VALUES ($1, $2, $3, $4)
	RETURNING *;`
	stmt := `Hello there`

	require.Equal(t, ValidQuery(vSQuery), true)
	require.Equal(t, ValidQuery(vUQuery), false)
	require.Equal(t, ValidQuery(vDQuery), false)
	require.Equal(t, ValidQuery(stmt), false)
}
//jimoh neymar adekunle was here fuckers