package http

import (
	"errors"
	"time"

	"shipment_ms/internal/domain"
)

type createShipmentRequest struct {
	OrderID        string         `json:"order_id"`
	OrderCreatedAt time.Time      `json:"order_created_at"`
	Address        addressRequest `json:"address"`
}

type addressRequest struct {
	Name    string `json:"name"`
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
}

func (r createShipmentRequest) validate() error {
	if r.OrderID == "" {
		return errors.New("order_id is required")
	}
	if r.OrderCreatedAt.IsZero() {
		return errors.New("order_created_at is required")
	}
	if r.Address.Name == "" {
		return errors.New("address.name is required")
	}
	if r.Address.Street == "" {
		return errors.New("address.street is required")
	}
	if r.Address.City == "" {
		return errors.New("address.city is required")
	}
	if r.Address.Country == "" {
		return errors.New("address.country is required")
	}
	return nil
}

func (r addressRequest) toDomain() domain.Address {
	return domain.Address{
		Name:    r.Name,
		Street:  r.Street,
		City:    r.City,
		Country: r.Country,
	}
}

type updateShipmentStatusRequest struct {
	ShipmentID string                `json:"shipment_id"`
	Status     domain.DeliveryStatus `json:"status"`
}

func (r updateShipmentStatusRequest) validate() error {
	if r.ShipmentID == "" {
		return errors.New("shipment_id is required")
	}
	if r.Status == "" {
		return errors.New("status is required")
	}
	switch r.Status {
	case domain.Created, domain.Processing, domain.Processed, domain.DeliveryStarted, domain.Delivered:
	default:
		return errors.New("status is not a valid DeliveryStatus")
	}
	return nil
}

type confirmDeliveryRequest struct {
	ShipmentID string `json:"shipment_id"`
}

func (r confirmDeliveryRequest) validate() error {
	if r.ShipmentID == "" {
		return errors.New("shipment_id is required")
	}
	return nil
}
