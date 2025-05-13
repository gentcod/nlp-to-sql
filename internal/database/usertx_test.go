package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/gentcod/nlp-to-sql/util"
	"github.com/stretchr/testify/require"
)

func TestCreateUserTx(t *testing.T) {
	createRandomUserOrAdminTx(t, RoleTypeUser)
}

func TestUpdateUserTx(t *testing.T) {
	store := NewStore(testDB)

	userTx, _ := createRandomUserOrAdminTx(t, RoleTypeUser)

	email := util.RandomEmail(10)
	username := util.RandomUser()
	fullName := util.RandomUser()

	harshedPassword, err := util.HashPassword(util.RandomStr(8))
	require.NoError(t, err)

	arg := UpdateUserTxParams{
		UpdateAuthParams: UpdateAuthParams{
			ID: userTx.Auth.ID,
			Email: sql.NullString{
				String: email,
				Valid:  email != "",
			},
			HarshedPassword: sql.NullString{
				String: harshedPassword,
				Valid:  harshedPassword != "",
			},
		},
		UpdateUserParams: UpdateUserParams{
			ID: userTx.User.ID,
			Username: sql.NullString{
				String: username,
				Valid:  username != "",
			},
			FullName: sql.NullString{
				String: fullName,
				Valid:  fullName != "",
			},
		},
	}

	result, err := store.UpdateUserTx(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	require.Equal(t, userTx.Auth.ID, result.Auth.ID)
	require.Equal(t, arg.UpdateAuthParams.Email.String, result.Auth.Email)
	require.Equal(t, arg.UpdateAuthParams.HarshedPassword.String, result.Auth.HarshedPassword)
	require.Equal(t, userTx.Auth.CreatedAt, result.Auth.CreatedAt)
	require.NotZero(t, result.Auth.UpdatedAt)

	require.Equal(t, userTx.User.ID, result.User.ID)
	require.Equal(t, userTx.Auth.ID, result.User.AuthID)
	require.Equal(t, arg.UpdateUserParams.Username.String, result.User.Username)
	require.Equal(t, arg.UpdateUserParams.FullName.String, result.User.FullName)
	require.Equal(t, userTx.User.CreatedAt, result.User.CreatedAt)
	require.NotZero(t, result.User.UpdatedAt)
}

func TestDeleteUserTx(t *testing.T) {
	store := NewStore(testDB)

	userTx, _ := createRandomUserOrAdminTx(t, RoleTypeUser)

	err := store.DeleteUserTx(context.Background(), userTx.Auth.ID, userTx.User.ID)
	require.NoError(t, err)

	user, err := store.GetUser(context.Background(), userTx.User.AuthID)
	require.Error(t, err)
	require.Empty(t, user)

	auth, err := store.ValidateAuth(context.Background(), userTx.Auth.Email)
	require.NoError(t, err)
	require.True(t, auth.Deleted)
}
