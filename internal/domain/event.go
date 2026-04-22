package domain

import "time"

type DeliveryConfirmedEvent struct {
	ShipmentID     string
	OrderID        string
	TrackingNumber string
	DeliveredAt    time.Time
}
