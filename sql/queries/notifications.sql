-- name: CreateNotification :exec
INSERT INTO notifications (user_id, kind, actor_id, ref_id, body, created_at)
VALUES (sqlc.arg(user_id), sqlc.arg(kind), sqlc.arg(actor_id), sqlc.arg(ref_id), sqlc.arg(body), sqlc.arg(created_at));

-- name: ListNotifications :many
SELECT * FROM notifications WHERE user_id = sqlc.arg(user_id) ORDER BY read_at IS NULL DESC, id DESC LIMIT sqlc.arg(limit);

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM notifications WHERE user_id = sqlc.arg(user_id) AND read_at IS NULL;

-- name: MarkNotificationsRead :exec
UPDATE notifications SET read_at = sqlc.arg(read_at) WHERE user_id = sqlc.arg(user_id) AND read_at IS NULL;
