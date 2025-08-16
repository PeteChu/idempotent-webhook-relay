-- name: GetOutBoxEvent :one
SELECT * FROM outbox
WHERE event_id = $1;

-- name: ListPendingEvents :many
SELECT * FROM outbox
WHERE status = 'pending';

-- name: InsertOutboxEvent :one
INSERT INTO outbox (event_id, payload) VALUES ($1, $2)
RETURNING id;

