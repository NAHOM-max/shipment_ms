CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE shipments (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id         TEXT        NOT NULL UNIQUE,
    tracking_number  TEXT        NOT NULL UNIQUE,
    delivery_date    TIMESTAMPTZ NOT NULL,
    status           TEXT        NOT NULL,
    confirmed        BOOLEAN     NOT NULL DEFAULT FALSE,
    name             TEXT        NOT NULL,
    street           TEXT        NOT NULL,
    city             TEXT        NOT NULL,
    country          TEXT        NOT NULL,
    workflow_id      TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
