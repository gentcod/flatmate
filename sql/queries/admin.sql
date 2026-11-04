-- name: GetUserByID :one
SELECT * FROM users WHERE id = sqlc.arg(id);

-- name: GetUserAdmin :one
SELECT is_admin FROM users WHERE id = sqlc.arg(id);

-- name: SetUserAdmin :exec
UPDATE users SET is_admin = sqlc.arg(is_admin) WHERE id = sqlc.arg(id);

-- name: ListReports :many
SELECT * FROM reports ORDER BY resolved_at IS NOT NULL, created_at DESC LIMIT sqlc.arg(limit);

-- name: GetReport :one
SELECT * FROM reports WHERE id = sqlc.arg(id);

-- name: ResolveReport :exec
UPDATE reports SET resolved_at = sqlc.arg(resolved_at), resolution = sqlc.arg(resolution) WHERE id = sqlc.arg(id);

-- name: CountAllUsers :one
SELECT COUNT(*) FROM users;

-- name: CountActiveUsers :one
SELECT COUNT(*) FROM users WHERE deactivated_at IS NULL;

-- name: CountOpenReports :one
SELECT COUNT(*) FROM reports WHERE resolved_at IS NULL;

-- name: CountListings :one
SELECT COUNT(*) FROM listings;

-- name: CountReviews :one
SELECT COUNT(*) FROM reviews;

-- name: CountExpenses :one
SELECT COUNT(*) FROM expenses;
