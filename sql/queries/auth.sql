-- name: CreateAuth :one
INSERT INTO auth (id, email, harshed_password)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateAdminAuth :one
INSERT INTO auth (id, email, harshed_password, role)
VALUES ($1, $2, $3, 'admin')
RETURNING *;

-- name: ValidateAuth :one
SELECT * FROM auth
WHERE email = $1 LIMIT 1;

-- name: GetAuth :one
SELECT id, email, role, restricted, deleted, created_at, updated_at FROM auth
WHERE id = $1 LIMIT 1;

-- name: UpdateAuth :one
UPDATE auth 
SET 
   email = COALESCE(sqlc.narg(email), email),
   harshed_password = COALESCE(sqlc.narg(harshed_password), harshed_password), 
   password_changed_at = COALESCE(sqlc.narg(password_changed_at), password_changed_at),
   updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: RestrictAuth :exec
UPDATE auth
SET restricted = TRUE, updated_at = $2
WHERE id = $1;

-- name: DeleteAuth :exec
UPDATE auth
SET deleted = TRUE, updated_at = $2
WHERE id = $1;

-- name: GetDeletedUsers :one
SELECT COUNT(*) 
   FROM auth 
WHERE role = 'user' 
   and deleted = TRUE
   AND updated_at < NOW() - INTERVAL '30 days'
;

-- name: DeleteUserAuthCron :many
DELETE FROM auth 
WHERE id IN (
   SELECT id FROM
   auth WHERE role = 'user'
   and deleted = TRUE
   and updated_at < NOW() - INTERVAL '30 days'
   LIMIT $1
)
RETURNING *;