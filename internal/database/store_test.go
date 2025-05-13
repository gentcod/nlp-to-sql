package db

import (
	"context"
	"testing"

	"github.com/gentcod/nlp-to-sql/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createRandomUserOrAdminTx(t *testing.T, role RoleType) (userTx UserTxResult, adminTx AdminTxResult) {
	store := NewStore(testDB)

	harshedPassword, err := util.HashPassword(util.RandomStr(8))
	require.NoError(t, err)

	if role == RoleTypeUser {
		arg := CreateUserTxParams{
			CreateAuthParams: CreateAuthParams{
				ID:              uuid.New(),
				Email:           util.RandomEmail(10),
				HarshedPassword: harshedPassword,
			},
			CreateUserParams: CreateUserParams{
				ID:       uuid.New(),
				Username: util.RandomUser(),
				FullName: util.RandomUser(),
			},
		}

		userTx, err = store.CreateUserTx(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, userTx)

		require.NotZero(t, userTx.Auth.ID)
		require.Equal(t, arg.CreateAuthParams.Email, userTx.Auth.Email)
		require.Equal(t, arg.CreateAuthParams.HarshedPassword, userTx.Auth.HarshedPassword)
		require.Equal(t, RoleTypeUser, userTx.Auth.Role.RoleType)
		require.True(t, userTx.Auth.PasswordChangedAt.IsZero())
		require.NotZero(t, userTx.Auth.CreatedAt)

		require.NotZero(t, userTx.User.ID)
		require.Equal(t, arg.CreateAuthParams.ID, userTx.User.AuthID)
		require.Equal(t, arg.CreateUserParams.Username, userTx.User.Username)
		require.Equal(t, arg.CreateUserParams.FullName, userTx.User.FullName)
		require.True(t, userTx.User.UpdatedAt.IsZero())
		require.NotZero(t, userTx.User.CreatedAt)
		return

	} else {
		arg := CreateAdminTxParams{
			CreateAdminAuthParams: CreateAdminAuthParams{
				ID:              uuid.New(),
				Email:           util.RandomEmail(10),
				HarshedPassword: harshedPassword,
			},
			CreateAdminParams: CreateAdminParams{
				ID:       uuid.New(),
				Username: util.RandomUser(),
				FullName: util.RandomUser(),
			},
		}

		adminTx, err = store.CreateAdminTx(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, adminTx)

		require.NotZero(t, adminTx.Auth.ID)
		require.Equal(t, arg.CreateAdminAuthParams.Email, adminTx.Auth.Email)
		require.Equal(t, arg.CreateAdminAuthParams.HarshedPassword, adminTx.Auth.HarshedPassword)
		require.Equal(t, RoleTypeAdmin, adminTx.Auth.Role.RoleType)
		require.True(t, adminTx.Auth.PasswordChangedAt.IsZero())
		require.NotZero(t, adminTx.Auth.CreatedAt)

		require.NotZero(t, adminTx.Admin.ID)
		require.Equal(t, arg.CreateAdminAuthParams.ID, adminTx.Admin.AuthID)
		require.Equal(t, arg.CreateAdminParams.Username, adminTx.Admin.Username)
		require.Equal(t, arg.CreateAdminParams.FullName, adminTx.Admin.FullName)
		require.True(t, adminTx.Admin.UpdatedAt.IsZero())
		require.NotZero(t, adminTx.Admin.CreatedAt)
	}

	return
}

func TestGetData(t *testing.T) {
	query := `SELECT * FROM users LIMIT 10;`
	valid := util.ValidQuery(query)
	require.True(t, valid)

	result, err := GetData(testDB, query)
	require.NoError(t, err)
	require.NotEmpty(t, result)
}
