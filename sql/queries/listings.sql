-- name: GetListing :one
SELECT * FROM listings WHERE id = sqlc.arg(id);

-- name: ListListings :many
SELECT * FROM listings ORDER BY id DESC LIMIT sqlc.arg(limit);

-- name: ListListingsByUser :many
SELECT * FROM listings WHERE user_id = sqlc.arg(user_id) ORDER BY id DESC;

-- name: CreateListing :execresult
INSERT INTO listings (user_id, title, description, area_text, monthly_rent, currency, photo_url, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(title), sqlc.arg(description), sqlc.arg(area_text), sqlc.arg(monthly_rent), sqlc.arg(currency), sqlc.arg(photo_url), sqlc.arg(created_at), sqlc.arg(updated_at));

-- name: UpdateListing :exec
UPDATE listings SET title = sqlc.arg(title), description = sqlc.arg(description), area_text = sqlc.arg(area_text), monthly_rent = sqlc.arg(monthly_rent), currency = sqlc.arg(currency), photo_url = sqlc.arg(photo_url), updated_at = sqlc.arg(updated_at) WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteListing :exec
DELETE FROM listings WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: UpdateListingStatus :exec
UPDATE listings SET status = sqlc.arg(status), updated_at = sqlc.arg(updated_at) WHERE id = sqlc.arg(id);
