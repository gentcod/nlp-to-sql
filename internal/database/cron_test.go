package db

import (
	"context"
	"testing"

	"github.com/gentcod/nlp-to-sql/util"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var config util.Config

func TestDBCron(t *testing.T) {
	store := NewStore(testDB)

	for i := 0; i < 4; i++ {
		userTx := createRandomUserTx(t, RoleTypeUser)

		err := store.DeleteUserTx(context.Background(), userTx.Auth.ID, userTx.User.ID)
		require.NoError(t, err)
	}

	totalDeleted, err := store.DeleteExpRestrictedRecords(context.Background(), 2)
	require.NoError(t, err)
	require.Equal(t, totalDeleted, 4)

	newTotalDeleted, err := store.DeleteExpRestrictedRecords(context.Background(), 2)
	require.NoError(t, err)
	require.Equal(t, newTotalDeleted, 0)
}
