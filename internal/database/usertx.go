package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CreateUserTxParams struct {
	CreateAuthParams CreateAuthParams
	CreateUserParams CreateUserParams
}

type UserTxResult struct {
	Auth Auth
	User User
}

// CreateUserTx is used to create user record and auth record in the same database transaction
func (store *SQLStore) CreateUserTx(ctx context.Context, arg CreateUserTxParams) (UserTxResult, error) {
	var result UserTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Auth, err = q.CreateAuth(ctx, arg.CreateAuthParams)
		if err != nil {
			return err
		}

		arg.CreateUserParams.AuthID = result.Auth.ID
		result.User, err = q.CreateUser(ctx, arg.CreateUserParams)
		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}

type UpdateUserTxParams struct {
	UpdateAuthParams UpdateAuthParams
	UpdateUserParams UpdateUserParams
}

// UpdateUserTx is used to update either the user record or auth record or both in the same database transaction
func (store *SQLStore) UpdateUserTx(ctx context.Context, arg UpdateUserTxParams) (UserTxResult, error) {
	var result UserTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		if arg.UpdateAuthParams.Email.Valid ||
			arg.UpdateAuthParams.HarshedPassword.Valid ||
			arg.UpdateAuthParams.PasswordChangedAt.Valid {
			arg.UpdateAuthParams.UpdatedAt = time.Now()
			result.Auth, err = q.UpdateAuth(ctx, arg.UpdateAuthParams)
			if err != nil {
				return fmt.Errorf("failed to update auth: %w", err)
			}
		}

		if arg.UpdateUserParams.Username.Valid || arg.UpdateUserParams.FullName.Valid {
			arg.UpdateUserParams.UpdatedAt = time.Now()
			result.User, err = q.UpdateUser(ctx, arg.UpdateUserParams)
			if err != nil {
				return fmt.Errorf("failed to update user: %w", err)
			}
		}

		return nil
	})

	return result, err
}

// DeleteUserTx is used to delete a user record and it's associated auth in the same database transaction
func (store *SQLStore) DeleteUserTx(ctx context.Context, authID uuid.UUID, userID uuid.UUID) error {
	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		arg := DeleteAuthParams{
			ID:        authID,
			UpdatedAt: time.Now(),
		}
		err = q.DeleteAuth(ctx, arg)
		if err != nil {
			return fmt.Errorf("failed to update auth to deleted: %w", err)
		}

		err = q.DeleteUser(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		return nil
	})

	return err
}
