-- name: ListActionsByActor :many
SELECT * FROM actions WHERE actor = sqlc.arg(actor);

-- name: ListInterestActors :many
SELECT actor FROM actions WHERE target = sqlc.arg(target) AND kind = 'interest';

-- name: CountInterestFrom :one
SELECT COUNT(*) FROM actions WHERE actor = sqlc.arg(actor) AND target = sqlc.arg(target) AND kind = 'interest';

-- name: AddAction :exec
INSERT INTO actions (actor, target, kind, created_at)
VALUES (sqlc.arg(actor), sqlc.arg(target), sqlc.arg(kind), sqlc.arg(created_at))
ON CONFLICT(actor, target, kind) DO NOTHING;

-- name: RemoveAction :exec
DELETE FROM actions WHERE actor = sqlc.arg(actor) AND target = sqlc.arg(target) AND kind = sqlc.arg(kind);

-- name: DeleteActionsBetween :exec
DELETE FROM actions WHERE actor = sqlc.arg(actor) AND target = sqlc.arg(target);
