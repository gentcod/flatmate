-- name: GetStay :one
SELECT * FROM stays WHERE id = sqlc.arg(id);

-- name: GetStayBetween :one
SELECT * FROM stays WHERE user_a = sqlc.arg(a) AND user_b = sqlc.arg(b);

-- name: ListStaysForUser :many
SELECT * FROM stays WHERE user_a = sqlc.arg(user_id) OR user_b = sqlc.arg(user_id) ORDER BY updated_at DESC;

-- name: CreateStay :execresult
INSERT INTO stays (user_a, user_b, proposed_by, listing_id, status, created_at, updated_at)
VALUES (sqlc.arg(user_a), sqlc.arg(user_b), sqlc.arg(proposed_by), sqlc.arg(listing_id), sqlc.arg(status), sqlc.arg(created_at), sqlc.arg(updated_at));

-- name: UpdateStayStatus :exec
UPDATE stays SET status = sqlc.arg(status), updated_at = sqlc.arg(updated_at) WHERE id = sqlc.arg(id);

-- name: UpdateStayProposal :exec
UPDATE stays SET proposed_by = sqlc.arg(proposed_by), listing_id = sqlc.arg(listing_id), status = sqlc.arg(status), updated_at = sqlc.arg(updated_at) WHERE id = sqlc.arg(id);

-- name: GetStayForListing :one
SELECT * FROM stays WHERE listing_id = sqlc.arg(listing_id) AND (user_a = sqlc.arg(user_id) OR user_b = sqlc.arg(user_id));
