-- name: InsertExpense :exec
INSERT INTO expenses (payer, peer, amount, currency, note, split, payer_share, peer_share, created_at)
VALUES (sqlc.arg(payer), sqlc.arg(peer), sqlc.arg(amount), sqlc.arg(currency), sqlc.arg(note), sqlc.arg(split), sqlc.arg(payer_share), sqlc.arg(peer_share), sqlc.arg(created_at));

-- name: GetExpense :one
SELECT * FROM expenses WHERE id = sqlc.arg(id);

-- name: ListExpenses :many
SELECT * FROM expenses WHERE (payer = sqlc.arg(s1) AND peer = sqlc.arg(r1)) OR (payer = sqlc.arg(s2) AND peer = sqlc.arg(r2)) ORDER BY id ASC;

-- name: DeleteExpense :exec
DELETE FROM expenses WHERE id = sqlc.arg(id);
