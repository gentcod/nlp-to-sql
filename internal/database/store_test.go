package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/gentcod/nlp-to-sql/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createRandomUserTx(t *testing.T) UserTxResult {
	store := NewStore(testDB)

	harshedPassword, err := util.HashPassword(util.RandomStr(8))
	require.NoError(t, err)

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

	result, err := store.CreateUserTx(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	require.NotZero(t, result.Auth.ID)
	require.Equal(t, arg.CreateAuthParams.Email, result.Auth.Email)
	require.Equal(t, arg.CreateAuthParams.HarshedPassword, result.Auth.HarshedPassword)
	require.True(t, result.Auth.PasswordChangedAt.IsZero())
	require.NotZero(t, result.Auth.CreatedAt)

	require.NotZero(t, result.User.ID)
	require.Equal(t, arg.CreateAuthParams.ID, result.User.AuthID)
	require.Equal(t, arg.CreateUserParams.Username, result.User.Username)
	require.Equal(t, arg.CreateUserParams.FullName, result.User.FullName)
	require.True(t, result.User.UpdatedAt.IsZero())
	require.NotZero(t, result.User.CreatedAt)

	return result
}

func TestCreateUserTx(t *testing.T) {
	createRandomUserTx(t)
}

func TestUpdateUserTx(t *testing.T) {
	store := NewStore(testDB)

	userTx := createRandomUserTx(t)

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

	userTx := createRandomUserTx(t)

	err := store.DeleteUserTx(context.Background(), userTx.Auth.ID, userTx.User.ID)
	require.NoError(t, err)

	user, err := store.GetUser(context.Background(), userTx.User.AuthID)
	require.Error(t, err)
	require.Empty(t, user)

	auth, err := store.ValidateAuth(context.Background(), userTx.Auth.Email)
	require.NoError(t, err)
	require.True(t, auth.Deleted)
}

func TestGetData(t *testing.T) {
	query := `SELECT * FROM users LIMIT 10;`
	valid := util.ValidQuery(query)
	require.True(t, valid)

	result, err := GetData(testDB, query)
	require.NoError(t, err)
	require.NotEmpty(t, result)
}
