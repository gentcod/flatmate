-- name: ListReportTargets :many
SELECT target FROM reports WHERE reporter = sqlc.arg(reporter);

-- name: CountReport :one
SELECT COUNT(*) FROM reports WHERE reporter = sqlc.arg(reporter) AND target = sqlc.arg(target);

-- name: AddReport :exec
INSERT INTO reports (reporter, target, reason, created_at)
VALUES (sqlc.arg(reporter), sqlc.arg(target), sqlc.arg(reason), sqlc.arg(created_at));
