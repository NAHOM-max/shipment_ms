package domain

import "time"

type Address struct {
	Street  string
	City    string
	State   string
	Country string
	ZipCode string
}

type Shipment struct {
	ID             string
	OrderID        string
	Address        Address
	TrackingNumber string
	DeliveryDate   time.Time
	Status         DeliveryStatus
	Confirmed      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *Shipment) MarkDeliveredAndConfirmed() {
	s.Status = Delivered
	s.Confirmed = true
	s.UpdatedAt = time.Now().UTC()
}
