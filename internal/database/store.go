package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	// "log"
)

// Store provides all functions to execute db SQL queries and transactions
type Store interface {
	Querier
	CreateUserTx(ctx context.Context, arg CreateUserTxParams) (UserTxResult, error)
	UpdateUserTx(ctx context.Context, arg UpdateUserTxParams) (UserTxResult, error)
}

// SQLStore provides all functions to execute db SQL queries
type SQLStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

// execTx executes a function within a database transaction
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

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

// GetData queries the database to return related data
func GetData(db *sql.DB, query string) ([]map[string]interface{}, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch column names: %w", err)
	}

	values := make([]interface{}, len(columns))
	valPointers := make([]interface{}, len(columns))
	for i := range values {
		valPointers[i] = &values[i]
	}

	results := []map[string]interface{}{}
	for rows.Next() {
		if err := rows.Scan(valPointers...); err != nil {
			return nil, fmt.Errorf("row scanning failed: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return results, nil
}
