package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidQuery(t *testing.T) {
	postgresQuery := `SELECT COUNT(*), currency FROM accounts WHERE created_at = NOW() - INTERVAL '1 year' GROUP BY currency`
	mysqlQuery := `SELECT COUNT(*), currency 
		FROM accounts 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 YEAR) 
		GROUP BY currency`
	tSQuery := `SELECT * FROM users;`
	tUQuery := `DELETE FROM users WHERE id = $1;`
	tDQuery := `INSERT INTO users (id, auth_id, username, full_name)
		VALUES ($1, $2, $3, $4)
		RETURNING *;`
	stmt := `Hello there`

	require.Equal(t, ValidQuery(postgresQuery), true)
	require.Equal(t, ValidQuery(mysqlQuery), true)
	require.Equal(t, ValidQuery(tSQuery), true)
	require.Equal(t, ValidQuery(tUQuery), false)
	require.Equal(t, ValidQuery(tDQuery), false)
	require.Equal(t, ValidQuery(stmt), false)
}

func TestPasswordHash(t *testing.T) {
	password := RandomStr(8)
	hashedPassword, err := HashPassword(password)
	require.NoError(t, err)

	err = CheckPassword(password, hashedPassword)
	require.NoError(t, err)
}
