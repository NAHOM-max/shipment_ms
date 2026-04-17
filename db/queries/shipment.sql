-- name: CreateShipment :one
INSERT INTO shipments (
    id,
    order_id,
    tracking_number,
    delivery_date,
    status,
    confirmed,
    name,
    street,
    city,
    country,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (order_id) DO NOTHING
RETURNING *;

-- name: GetShipmentByID :one
SELECT * FROM shipments
WHERE id = $1;

-- name: GetShipmentByOrderID :one
SELECT * FROM shipments
WHERE order_id = $1;

-- name: UpdateShipment :one
UPDATE shipments
SET
    tracking_number = $2,
    delivery_date   = $3,
    status          = $4,
    confirmed       = $5,
    name            = $6,
    street          = $7,
    city            = $8,
    country         = $9,
    updated_at      = $10
WHERE id = $1
RETURNING *;
