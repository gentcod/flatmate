-- name: InsertMessage :exec
INSERT INTO messages (sender, recipient, body, created_at)
VALUES (sqlc.arg(sender), sqlc.arg(recipient), sqlc.arg(body), sqlc.arg(created_at));

-- name: ListConversation :many
SELECT * FROM messages WHERE (sender = sqlc.arg(s1) AND recipient = sqlc.arg(r1)) OR (sender = sqlc.arg(s2) AND recipient = sqlc.arg(r2)) ORDER BY id ASC LIMIT sqlc.arg(limit);

-- name: ListRecentForUser :many
SELECT * FROM messages WHERE sender = sqlc.arg(a) OR recipient = sqlc.arg(b) ORDER BY id DESC;

-- name: MarkThreadRead :exec
UPDATE messages SET read_at = sqlc.arg(read_at) WHERE sender = sqlc.arg(sender) AND recipient = sqlc.arg(recipient) AND read_at IS NULL;

-- name: UnreadBySender :many
SELECT sender, COUNT(*) AS count FROM messages WHERE recipient = sqlc.arg(recipient) AND read_at IS NULL GROUP BY sender;
