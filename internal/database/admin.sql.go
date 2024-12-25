// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: admin.sql

package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const createAdmin = `-- name: CreateAdmin :one
INSERT INTO admins (id, auth_id, username, full_name)
VALUES ($1, $2, $3, $4)
RETURNING id, auth_id, username, full_name, created_at, updated_at
`

type CreateAdminParams struct {
	ID       uuid.UUID `json:"id"`
	AuthID   uuid.UUID `json:"auth_id"`
	Username string    `json:"username"`
	FullName string    `json:"full_name"`
}

func (q *Queries) CreateAdmin(ctx context.Context, arg CreateAdminParams) (Admin, error) {
	row := q.db.QueryRowContext(ctx, createAdmin,
		arg.ID,
		arg.AuthID,
		arg.Username,
		arg.FullName,
	)
	var i Admin
	err := row.Scan(
		&i.ID,
		&i.AuthID,
		&i.Username,
		&i.FullName,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteAdmin = `-- name: DeleteAdmin :exec
DELETE FROM admins WHERE id = $1
`

func (q *Queries) DeleteAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteAdmin, id)
	return err
}

const getAdmin = `-- name: GetAdmin :one
SELECT id, username, full_name, created_at, updated_at FROM admins
WHERE auth_id = $1 LIMIT 1
`

type GetAdminRow struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (q *Queries) GetAdmin(ctx context.Context, authID uuid.UUID) (GetAdminRow, error) {
	row := q.db.QueryRowContext(ctx, getAdmin, authID)
	var i GetAdminRow
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.FullName,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateAdmin = `-- name: UpdateAdmin :one
UPDATE admins
SET 
   username = COALESCE($1, username), 
   full_name = COALESCE($2, full_name),
   updated_at = $3
WHERE id = $4
RETURNING id, auth_id, username, full_name, created_at, updated_at
`

type UpdateAdminParams struct {
	Username  sql.NullString `json:"username"`
	FullName  sql.NullString `json:"full_name"`
	UpdatedAt time.Time      `json:"updated_at"`
	ID        uuid.UUID      `json:"id"`
}

func (q *Queries) UpdateAdmin(ctx context.Context, arg UpdateAdminParams) (Admin, error) {
	row := q.db.QueryRowContext(ctx, updateAdmin,
		arg.Username,
		arg.FullName,
		arg.UpdatedAt,
		arg.ID,
	)
	var i Admin
	err := row.Scan(
		&i.ID,
		&i.AuthID,
		&i.Username,
		&i.FullName,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}