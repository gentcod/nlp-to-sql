package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestrictUserAuth(t *testing.T) {
	store := NewStore(testDB)

	userTx := createRandomUserTx(t, RoleTypeUser)

	err := store.RestrictAuth(context.Background(), userTx.Auth.ID)
	require.NoError(t, err)

	result, err := store.GetAuth(context.Background(), userTx.Auth.ID)
	require.Equal(t, userTx.Auth.ID, result.ID)
	require.Equal(t, userTx.Auth.Email, result.Email)
	require.NotEqual(t, userTx.Auth.Restricted, result.Restricted)
	require.Equal(t, userTx.Auth.Deleted, result.Deleted)
	require.False(t, userTx.Auth.Restricted)
	require.True(t, result.Restricted)
}