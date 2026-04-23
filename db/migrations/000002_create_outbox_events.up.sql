CREATE TABLE outbox_events (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type TEXT        NOT NULL,
    aggregate_id   TEXT        NOT NULL,
    event_type     TEXT        NOT NULL,
    payload        JSONB       NOT NULL,
    status         TEXT        NOT NULL DEFAULT 'PENDING',
    retry_count    INT         NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Worker polls only PENDING/FAILED rows ordered by creation time.
CREATE INDEX idx_outbox_pending
    ON outbox_events (created_at)
    WHERE status IN ('PENDING', 'FAILED');
