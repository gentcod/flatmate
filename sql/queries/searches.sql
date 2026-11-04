-- name: CreateSearch :execresult
INSERT INTO saved_searches (user_id, name, area, max_rent, created_at)
VALUES (sqlc.arg(user_id), sqlc.arg(name), sqlc.arg(area), sqlc.arg(max_rent), sqlc.arg(created_at));

-- name: ListSearches :many
SELECT * FROM saved_searches WHERE user_id = sqlc.arg(user_id) ORDER BY id DESC;

-- name: DeleteSearch :exec
DELETE FROM saved_searches WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);
