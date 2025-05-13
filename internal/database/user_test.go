package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetUser(t *testing.T) {
	store := NewStore(testDB)

	userTx, _ := createRandomUserOrAdminTx(t, RoleTypeUser)

	result, err := store.GetUser(context.Background(), userTx.Auth.ID)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	require.Equal(t, userTx.User.ID, result.ID)
	require.Equal(t, userTx.User.Username, result.Username)
	require.Equal(t, userTx.User.FullName, result.FullName)
	require.Equal(t, userTx.User.CreatedAt, result.CreatedAt)
	require.Equal(t, userTx.User.UpdatedAt, result.UpdatedAt)
}
