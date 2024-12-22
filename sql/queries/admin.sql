-- name: CreateAdmin :one
INSERT INTO admins (id, auth_id, username, full_name)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateAdmin :one
UPDATE admins
SET 
   username = COALESCE(sqlc.narg(username), username), 
   full_name = COALESCE(sqlc.narg(full_name), full_name),
   updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetAdmin :one
SELECT id, username, full_name, created_at, updated_at FROM admins
WHERE auth_id = $1 LIMIT 1;

-- name: DeleteAdmin :exec
DELETE FROM admins WHERE id = $1;