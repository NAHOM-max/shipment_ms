package repository

import (
	"context"

	"shipment_ms/internal/domain"
)

type ShipmentRepository interface {
	Create(ctx context.Context, s *domain.Shipment) error
	GetByID(ctx context.Context, id string) (*domain.Shipment, error)
	UpdateStatus(ctx context.Context, id string, status domain.Status) error
	List(ctx context.Context) ([]*domain.Shipment, error)
}
