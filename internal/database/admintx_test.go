package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/gentcod/nlp-to-sql/util"
	"github.com/stretchr/testify/require"
)

func TestCreateAdminUserTx(t *testing.T) {
	createRandomUserOrAdminTx(t, RoleTypeAdmin)
}

func TestUpdateAdminTx(t *testing.T) {
	store := NewStore(testDB)

	_, adminTx := createRandomUserOrAdminTx(t, RoleTypeAdmin)

	email := util.RandomEmail(10)
	username := util.RandomUser()
	fullName := util.RandomUser()

	harshedPassword, err := util.HashPassword(util.RandomStr(8))
	require.NoError(t, err)

	arg := UpdateAdminTxParams{
		UpdateAuthParams: UpdateAuthParams{
			ID: adminTx.Auth.ID,
			Email: sql.NullString{
				String: email,
				Valid:  email != "",
			},
			HarshedPassword: sql.NullString{
				String: harshedPassword,
				Valid:  harshedPassword != "",
			},
		},
		UpdateAdminParams: UpdateAdminParams{
			ID: adminTx.Admin.ID,
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

	result, err := store.UpdateAdminTx(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	require.Equal(t, adminTx.Auth.ID, result.Auth.ID)
	require.Equal(t, arg.UpdateAuthParams.Email.String, result.Auth.Email)
	require.Equal(t, arg.UpdateAuthParams.HarshedPassword.String, result.Auth.HarshedPassword)
	require.Equal(t, adminTx.Auth.CreatedAt, result.Auth.CreatedAt)
	require.NotZero(t, result.Auth.UpdatedAt)

	require.Equal(t, adminTx.Admin.ID, result.Admin.ID)
	require.Equal(t, adminTx.Auth.ID, result.Admin.AuthID)
	require.Equal(t, arg.UpdateAdminParams.Username.String, result.Admin.Username)
	require.Equal(t, arg.UpdateAdminParams.FullName.String, result.Admin.FullName)
	require.Equal(t, adminTx.Admin.CreatedAt, result.Admin.CreatedAt)
	require.NotZero(t, result.Admin.UpdatedAt)
}

func TestDeleteAdminTx(t *testing.T) {
	store := NewStore(testDB)

	_, adminTx := createRandomUserOrAdminTx(t, RoleTypeAdmin)

	err := store.DeleteAdminTx(context.Background(), adminTx.Auth.ID, adminTx.Admin.ID)
	require.NoError(t, err)

	admin, err := store.GetAdmin(context.Background(), adminTx.Admin.AuthID)
	require.Error(t, err)
	require.Empty(t, admin)

	auth, err := store.ValidateAuth(context.Background(), adminTx.Auth.Email)
	require.NoError(t, err)
	require.True(t, auth.Deleted)
}
