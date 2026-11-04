-- name: UpsertReview :exec
INSERT INTO reviews (author, target, rating, body, created_at)
VALUES (sqlc.arg(author), sqlc.arg(target), sqlc.arg(rating), sqlc.arg(body), sqlc.arg(created_at))
ON CONFLICT(author, target) DO UPDATE SET rating = excluded.rating, body = excluded.body, created_at = excluded.created_at;

-- name: ListReviewsForUser :many
SELECT * FROM reviews WHERE target = sqlc.arg(target) ORDER BY id DESC;

-- name: GetReview :one
SELECT * FROM reviews WHERE author = sqlc.arg(author) AND target = sqlc.arg(target);

-- name: UpsertListingReview :exec
INSERT INTO listing_reviews (listing_id, author, rating, body, kind, created_at)
VALUES (sqlc.arg(listing_id), sqlc.arg(author), sqlc.arg(rating), sqlc.arg(body), sqlc.arg(kind), sqlc.arg(created_at))
ON CONFLICT(listing_id, author) DO UPDATE SET rating = excluded.rating, body = excluded.body, kind = excluded.kind, created_at = excluded.created_at;

-- name: ListListingReviews :many
SELECT * FROM listing_reviews WHERE listing_id = sqlc.arg(listing_id) ORDER BY id DESC;

-- name: GetListingReview :one
SELECT * FROM listing_reviews WHERE listing_id = sqlc.arg(listing_id) AND author = sqlc.arg(author);
