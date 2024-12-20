-- name: CreateAuth :one
INSERT INTO auth (id, email, harshed_password)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ValidateAuth :one
SELECT * FROM auth
WHERE email = $1 LIMIT 1;

-- name: GetAuth :one
SELECT id, email FROM auth
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
SET restricted = TRUE
WHERE id = $1;

-- name: DeleteAuth :exec
UPDATE auth
SET deleted = TRUE
WHERE id = $1;

-- name: GetRestricted :one
SELECT COUNT(*) 
   FROM auth 
WHERE deleted = TRUE;

-- name: DeleteAuthCron :many
DELETE FROM auth 
WHERE id IN (
   SELECT id FROM
   auth WHERE deleted = TRUE
   LIMIT $1
)
RETURNING *;