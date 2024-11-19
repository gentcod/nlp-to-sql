-- name: CreateAuth :one
INSERT INTO auth (id, email, harshed_password)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAuth :one
SELECT * FROM auth
WHERE email = $1 LIMIT 1;

-- name: UpdateAuth :one
UPDATE auth 
SET 
   email = COALESCE(sqlc.narg(email), email),
   harshed_password = COALESCE(sqlc.narg(harshed_password), harshed_password), 
   password_changed_at = COALESCE(sqlc.narg(password_changed_at), password_changed_at),
   updated_at = COALESCE(sqlc.narg(updated_at), updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;