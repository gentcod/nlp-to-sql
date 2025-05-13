package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRestrictUserAuth(t *testing.T) {
	store := NewStore(testDB)

	userTx, _ := createRandomUserOrAdminTx(t, RoleTypeUser)

	err := store.RestrictAuth(context.Background(), RestrictAuthParams{
		ID:        userTx.Auth.ID,
		UpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	result, err := store.GetAuth(context.Background(), userTx.Auth.ID)
	require.Equal(t, userTx.Auth.ID, result.ID)
	require.Equal(t, userTx.Auth.Email, result.Email)
	require.NotEqual(t, userTx.Auth.Restricted, result.Restricted)
	require.NotEqual(t, userTx.Auth.UpdatedAt, result.UpdatedAt)
	require.Equal(t, userTx.Auth.Deleted, result.Deleted)
	require.False(t, userTx.Auth.Restricted)
	require.True(t, result.Restricted)
}
