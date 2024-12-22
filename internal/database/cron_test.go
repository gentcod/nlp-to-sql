package db

import (
	"context"
	"testing"
	"time"

	"github.com/gentcod/nlp-to-sql/util"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var config util.Config

func TestDBCron(t *testing.T) {
	store := NewStore(testDB)

	for i := 0; i < 4; i++ {
		userTx, _ := createRandomUserOrAdminTx(t, RoleTypeUser)

		err := store.DeleteUserTx(context.Background(), userTx.Auth.ID, userTx.User.ID)
		require.NoError(t, err)

		_, err = testDB.Exec(`
		UPDATE auth
		SET updated_at = NOW() - INTERVAL '30 days'
		WHERE id = $1`, userTx.Auth.ID)
		require.NoError(t, err)
	}
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	<-ticker.C

	totalDeleted, err := store.DeleteExpDeletedUserRecords(context.Background(), 2)
	require.NoError(t, err)
	require.GreaterOrEqual(t, totalDeleted, 4)

	newTotalDeleted, err := store.DeleteExpDeletedUserRecords(context.Background(), 2)
	require.NoError(t, err)
	require.Equal(t, newTotalDeleted, 0)
}
