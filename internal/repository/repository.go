package repository

import (
	"context"
	"errors"

	"shipment_ms/internal/domain"
)

// ErrNotFound is returned when a shipment cannot be located by the given key.
var ErrNotFound = errors.New("shipment not found")

type ShipmentRepository interface {
	// Create persists a new shipment. If a shipment with the same OrderID
	// already exists it returns the existing record without error (idempotent).
	Create(ctx context.Context, s *domain.Shipment) (*domain.Shipment, error)

	GetByID(ctx context.Context, id string) (*domain.Shipment, error)
	GetByOrderID(ctx context.Context, orderID string) (*domain.Shipment, error)

	// Update overwrites all mutable fields of an existing shipment.
	Update(ctx context.Context, s *domain.Shipment) (*domain.Shipment, error)
}
