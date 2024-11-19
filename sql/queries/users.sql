-- name: CreateUser :one
INSERT INTO users (id, auth_id, username, full_name)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateUser :one
UPDATE users 
SET 
   username = COALESCE(sqlc.narg(username), username), 
   full_name = COALESCE(sqlc.narg(full_name), full_name),
   updated_at = COALESCE(sqlc.narg(updated_at), updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetUser :one
SELECT id, username, full_name, created_at, updated_at FROM users
WHERE username = $1 LIMIT 1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;