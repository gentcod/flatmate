-- name: ListBlocksByBlocker :many
SELECT * FROM blocks WHERE blocker = sqlc.arg(blocker);

-- name: ListBlocksByBlocked :many
SELECT * FROM blocks WHERE blocked = sqlc.arg(blocked);

-- name: CountBlockDirection :one
SELECT COUNT(*) FROM blocks WHERE blocker = sqlc.arg(blocker) AND blocked = sqlc.arg(blocked);

-- name: AddBlock :exec
INSERT INTO blocks (blocker, blocked, created_at)
VALUES (sqlc.arg(blocker), sqlc.arg(blocked), sqlc.arg(created_at))
ON CONFLICT(blocker, blocked) DO NOTHING;
