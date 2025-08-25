-- name: GetOutBoxEvent :one
SELECT * FROM outbox
WHERE event_id = $1;

-- name: ListEvents :many
SELECT * FROM outbox;

-- name: ListUnprocessedEvents :many
SELECT * FROM outbox
WHERE COALESCE(status, '') NOT IN ('pending', 'processed')
AND type = ANY($1::varchar[]);

-- name: ListFailedEvents :many
SELECT * FROM outbox
WHERE status = 'failed';

-- name: InsertOutboxEvent :one
INSERT INTO outbox (event_id, type, payload, provider) VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: UpdateOutboxEvent :exec
UPDATE outbox 
SET 
  status = $2,
  last_error = $3,
  last_attempt_at = $4
WHERE id = $1;
