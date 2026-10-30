-- name: GetUserByEmailHash :one
SELECT * FROM users WHERE email_hash = sqlc.arg(email_hash);

-- name: CountUsersByEmailHash :one
SELECT COUNT(*) FROM users WHERE email_hash = sqlc.arg(email_hash);

-- name: CreateUser :execresult
INSERT INTO users (email_hash, pass_hash, created_at)
VALUES (sqlc.arg(email_hash), sqlc.arg(pass_hash), sqlc.arg(created_at));

-- name: GetUserDeactivation :one
SELECT deactivated_at FROM users WHERE id = sqlc.arg(id);

-- name: DeactivateUser :exec
UPDATE users SET deactivated_at = sqlc.arg(deactivated_at) WHERE id = sqlc.arg(id);

-- name: ActiveUserIDs :many
SELECT id FROM users WHERE deactivated_at IS NULL AND id != sqlc.arg(id);
