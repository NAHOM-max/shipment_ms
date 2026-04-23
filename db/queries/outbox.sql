-- name: InsertOutboxEvent :exec
INSERT INTO outbox_events (
    id, aggregate_type, aggregate_id, event_type, payload, status, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, 'PENDING', NOW(), NOW()
);

-- name: FetchPendingOutboxEvents :many
SELECT * FROM outbox_events
WHERE  status IN ('PENDING', 'FAILED')
  AND  retry_count < $1
ORDER  BY created_at
LIMIT  $2;

-- name: MarkOutboxEventSent :exec
UPDATE outbox_events
SET    status = 'SENT', updated_at = NOW()
WHERE  id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET    status      = 'FAILED',
       retry_count = retry_count + 1,
       updated_at  = NOW()
WHERE  id = $1;
