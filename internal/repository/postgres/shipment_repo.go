package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"shipment_ms/internal/domain"
	"shipment_ms/internal/repository"
)

type shipmentRepo struct {
	q *Queries
}

func NewShipmentRepository(pool *pgxpool.Pool) repository.ShipmentRepository {
	return &shipmentRepo{q: New(pool)}
}

func (r *shipmentRepo) Create(ctx context.Context, s *domain.Shipment) error {
	var uid pgtype.UUID
	if err := uid.Scan(s.ID); err != nil {
		return fmt.Errorf("invalid uuid %q: %w", s.ID, err)
	}
	_, err := r.q.CreateShipment(ctx, CreateShipmentParams{
		ID:             uid,
		OrderID:        s.OrderID,
		TrackingNumber: s.TrackingNumber,
		DeliveryDate:   pgtype.Timestamptz{Time: s.DeliveryDate, Valid: true},
		Status:         string(s.Status),
		Confirmed:      s.Confirmed,
		Name:           s.Address.Name,
		Street:         s.Address.Street,
		City:           s.Address.City,
		Country:        s.Address.Country,
		CreatedAt:      pgtype.Timestamptz{Time: s.CreatedAt, Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: s.UpdatedAt, Valid: true},
	})
	return err
}

func (r *shipmentRepo) GetByID(ctx context.Context, id string) (*domain.Shipment, error) {
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return nil, fmt.Errorf("invalid uuid %q: %w", id, err)
	}
	row, err := r.q.GetShipmentByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	return toDomain(row), nil
}

func (r *shipmentRepo) GetByOrderID(ctx context.Context, orderID string) (*domain.Shipment, error) {
	row, err := r.q.GetShipmentByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return toDomain(row), nil
}

func (r *shipmentRepo) UpdateStatus(ctx context.Context, id string, status domain.DeliveryStatus) error {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return fmt.Errorf("invalid uuid %q: %w", id, err)
	}
	_, err = r.q.UpdateShipment(ctx, UpdateShipmentParams{
		ID:             uid,
		TrackingNumber: existing.TrackingNumber,
		DeliveryDate:   pgtype.Timestamptz{Time: existing.DeliveryDate, Valid: true},
		Status:         string(status),
		Confirmed:      existing.Confirmed,
		Name:           existing.Address.Name,
		Street:         existing.Address.Street,
		City:           existing.Address.City,
		Country:        existing.Address.Country,
		UpdatedAt:      pgtype.Timestamptz{Time: existing.UpdatedAt, Valid: true},
	})
	return err
}

func (r *shipmentRepo) List(ctx context.Context) ([]*domain.Shipment, error) {
	panic("not implemented")
}

func toDomain(row Shipment) *domain.Shipment {
	return &domain.Shipment{
		ID:             row.ID.String(),
		OrderID:        row.OrderID,
		TrackingNumber: row.TrackingNumber,
		DeliveryDate:   row.DeliveryDate.Time,
		Status:         domain.DeliveryStatus(row.Status),
		Confirmed:      row.Confirmed,
		Address: domain.Address{
			Name:    row.Name,
			Street:  row.Street,
			City:    row.City,
			Country: row.Country,
		},
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
