package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CreateAdminTxParams struct {
	CreateAdminAuthParams CreateAdminAuthParams
	CreateAdminParams     CreateAdminParams
}

type AdminTxResult struct {
	Auth  Auth
	Admin Admin
}

// CreateAdminTx is used to create Admin record and auth record in the same database transaction
func (store *SQLStore) CreateAdminTx(ctx context.Context, arg CreateAdminTxParams) (AdminTxResult, error) {
	var result AdminTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Auth, err = q.CreateAdminAuth(ctx, arg.CreateAdminAuthParams)
		if err != nil {
			return err
		}

		arg.CreateAdminParams.AuthID = result.Auth.ID
		result.Admin, err = q.CreateAdmin(ctx, arg.CreateAdminParams)
		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}

type UpdateAdminTxParams struct {
	UpdateAuthParams  UpdateAuthParams
	UpdateAdminParams UpdateAdminParams
}

// UpdateAdminTx is used to update either the Admin record or auth record or both in the same database transaction
func (store *SQLStore) UpdateAdminTx(ctx context.Context, arg UpdateAdminTxParams) (AdminTxResult, error) {
	var result AdminTxResult

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

		if arg.UpdateAdminParams.Username.Valid || arg.UpdateAdminParams.FullName.Valid {
			arg.UpdateAdminParams.UpdatedAt = time.Now()
			result.Admin, err = q.UpdateAdmin(ctx, arg.UpdateAdminParams)
			if err != nil {
				return fmt.Errorf("failed to update Admin: %w", err)
			}
		}

		return nil
	})

	return result, err
}

// DeleteAdminTx is used to delete an admin record and it's associated auth in the same database transaction
func (store *SQLStore) DeleteAdminTx(ctx context.Context, authID uuid.UUID, adminID uuid.UUID) error {
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

		err = q.DeleteAdmin(ctx, adminID)
		if err != nil {
			return fmt.Errorf("failed to delete Admin: %w", err)
		}

		return nil
	})

	return err
}
