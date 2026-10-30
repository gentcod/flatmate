-- name: GetAnswers :many
SELECT * FROM answers WHERE user_id = sqlc.arg(user_id);

-- name: DeleteAnswers :exec
DELETE FROM answers WHERE user_id = sqlc.arg(user_id);

-- name: InsertAnswer :exec
INSERT INTO answers (user_id, dimension, value, hard, weight)
VALUES (sqlc.arg(user_id), sqlc.arg(dimension), sqlc.arg(value), sqlc.arg(hard), sqlc.arg(weight));
