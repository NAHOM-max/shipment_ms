package domain

import "time"

type Address struct {
	Name    string
	Street  string
	City    string
	Country string
}

type Shipment struct {
	ID             string
	OrderID        string
	Address        Address
	TrackingNumber string
	DeliveryDate   time.Time
	Status         DeliveryStatus
	Confirmed      bool
	WorkflowID     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *Shipment) MarkDeliveredAndConfirmed() {
	s.Status = Delivered
	s.Confirmed = true
	s.UpdatedAt = time.Now().UTC()
}
