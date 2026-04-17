package http

import (
	"time"

	"shipment_ms/internal/domain"
)

type shipmentResponse struct {
	ID             string    `json:"id"`
	OrderID        string    `json:"order_id"`
	TrackingNumber string    `json:"tracking_number"`
	DeliveryDate   time.Time `json:"delivery_date"`
	Status         string    `json:"status"`
	Confirmed      bool      `json:"confirmed"`
	Address        addressResponse `json:"address"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type addressResponse struct {
	Name    string `json:"name"`
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func toShipmentResponse(s *domain.Shipment) shipmentResponse {
	return shipmentResponse{
		ID:             s.ID,
		OrderID:        s.OrderID,
		TrackingNumber: s.TrackingNumber,
		DeliveryDate:   s.DeliveryDate,
		Status:         string(s.Status),
		Confirmed:      s.Confirmed,
		Address: addressResponse{
			Name:    s.Address.Name,
			Street:  s.Address.Street,
			City:    s.Address.City,
			Country: s.Address.Country,
		},
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
