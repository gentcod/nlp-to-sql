// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: auth.sql

package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const createAuth = `-- name: CreateAuth :one
INSERT INTO auth (id, email, harshed_password)
VALUES ($1, $2, $3)
RETURNING id, email, harshed_password, password_changed_at, created_at, updated_at, restricted, deleted
`

type CreateAuthParams struct {
	ID              uuid.UUID `json:"id"`
	Email           string    `json:"email"`
	HarshedPassword string    `json:"harshed_password"`
}

func (q *Queries) CreateAuth(ctx context.Context, arg CreateAuthParams) (Auth, error) {
	row := q.db.QueryRowContext(ctx, createAuth, arg.ID, arg.Email, arg.HarshedPassword)
	var i Auth
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.HarshedPassword,
		&i.PasswordChangedAt,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Restricted,
		&i.Deleted,
	)
	return i, err
}

const deleteAuth = `-- name: DeleteAuth :exec
UPDATE auth
SET deleted = TRUE
WHERE id = $1
`

func (q *Queries) DeleteAuth(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteAuth, id)
	return err
}

const deleteAuthCron = `-- name: DeleteAuthCron :many
DELETE FROM auth 
WHERE id IN (
   SELECT id FROM
   auth WHERE deleted = TRUE
   LIMIT $1
)
RETURNING id, email, harshed_password, password_changed_at, created_at, updated_at, restricted, deleted
`

func (q *Queries) DeleteAuthCron(ctx context.Context, limit int32) ([]Auth, error) {
	rows, err := q.db.QueryContext(ctx, deleteAuthCron, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Auth{}
	for rows.Next() {
		var i Auth
		if err := rows.Scan(
			&i.ID,
			&i.Email,
			&i.HarshedPassword,
			&i.PasswordChangedAt,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Restricted,
			&i.Deleted,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getAuth = `-- name: GetAuth :one
SELECT id, email FROM auth
WHERE id = $1 LIMIT 1
`

type GetAuthRow struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

func (q *Queries) GetAuth(ctx context.Context, id uuid.UUID) (GetAuthRow, error) {
	row := q.db.QueryRowContext(ctx, getAuth, id)
	var i GetAuthRow
	err := row.Scan(&i.ID, &i.Email)
	return i, err
}

const getRestricted = `-- name: GetRestricted :one
SELECT COUNT(*) 
   FROM auth 
WHERE deleted = TRUE
`

func (q *Queries) GetRestricted(ctx context.Context) (int64, error) {
	row := q.db.QueryRowContext(ctx, getRestricted)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const restrictAuth = `-- name: RestrictAuth :exec
UPDATE auth
SET restricted = TRUE
WHERE id = $1
`

func (q *Queries) RestrictAuth(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, restrictAuth, id)
	return err
}

const updateAuth = `-- name: UpdateAuth :one
UPDATE auth 
SET 
   email = COALESCE($1, email),
   harshed_password = COALESCE($2, harshed_password), 
   password_changed_at = COALESCE($3, password_changed_at),
   updated_at = $4
WHERE id = $5
RETURNING id, email, harshed_password, password_changed_at, created_at, updated_at, restricted, deleted
`

type UpdateAuthParams struct {
	Email             sql.NullString `json:"email"`
	HarshedPassword   sql.NullString `json:"harshed_password"`
	PasswordChangedAt sql.NullTime   `json:"password_changed_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	ID                uuid.UUID      `json:"id"`
}

func (q *Queries) UpdateAuth(ctx context.Context, arg UpdateAuthParams) (Auth, error) {
	row := q.db.QueryRowContext(ctx, updateAuth,
		arg.Email,
		arg.HarshedPassword,
		arg.PasswordChangedAt,
		arg.UpdatedAt,
		arg.ID,
	)
	var i Auth
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.HarshedPassword,
		&i.PasswordChangedAt,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Restricted,
		&i.Deleted,
	)
	return i, err
}

const validateAuth = `-- name: ValidateAuth :one
SELECT id, email, harshed_password, password_changed_at, created_at, updated_at, restricted, deleted FROM auth
WHERE email = $1 LIMIT 1
`

func (q *Queries) ValidateAuth(ctx context.Context, email string) (Auth, error) {
	row := q.db.QueryRowContext(ctx, validateAuth, email)
	var i Auth
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.HarshedPassword,
		&i.PasswordChangedAt,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Restricted,
		&i.Deleted,
	)
	return i, err
}
