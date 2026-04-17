package domain

import "time"

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusInTransit Status = "IN_TRANSIT"
	StatusDelivered Status = "DELIVERED"
	StatusCancelled Status = "CANCELLED"
)

type Shipment struct {
	ID          string
	Origin      string
	Destination string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
