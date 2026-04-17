package postgres

import (
	"context"
	"database/sql"

	"shipment_ms/internal/domain"
	"shipment_ms/internal/repository"
)

type shipmentRepo struct {
	db *sql.DB
}

func NewShipmentRepository(db *sql.DB) repository.ShipmentRepository {
	return &shipmentRepo{db: db}
}

func (r *shipmentRepo) Create(ctx context.Context, s *domain.Shipment) error {
	panic("not implemented")
}

func (r *shipmentRepo) GetByID(ctx context.Context, id string) (*domain.Shipment, error) {
	panic("not implemented")
}

func (r *shipmentRepo) UpdateStatus(ctx context.Context, id string, status domain.Status) error {
	panic("not implemented")
}

func (r *shipmentRepo) List(ctx context.Context) ([]*domain.Shipment, error) {
	panic("not implemented")
}
