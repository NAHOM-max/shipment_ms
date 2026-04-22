package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

// Create inserts a new shipment. If a row with the same order_id already
// exists (ON CONFLICT DO NOTHING returns no rows), it fetches and returns
// the existing record — making the operation fully idempotent.
func (r *shipmentRepo) Create(ctx context.Context, s *domain.Shipment) (*domain.Shipment, error) {
	uid, err := parseUUID(s.ID)
	if err != nil {
		return nil, err
	}

	row, err := r.q.CreateShipment(ctx, CreateShipmentParams{
		ID:             uid,
		OrderID:        s.OrderID,
		TrackingNumber: s.TrackingNumber,
		DeliveryDate:   toTimestamptz(s.DeliveryDate),
		Status:         string(s.Status),
		Confirmed:      s.Confirmed,
		Name:           s.Address.Name,
		Street:         s.Address.Street,
		City:           s.Address.City,
		Country:        s.Address.Country,
		WorkflowID:     s.WorkflowID,
		CreatedAt:      toTimestamptz(s.CreatedAt),
		UpdatedAt:      toTimestamptz(s.UpdatedAt),
	})
	if err != nil {
		// ON CONFLICT DO NOTHING → pgx returns ErrNoRows when the insert
		// was skipped; fall back to fetching the existing record.
		if errors.Is(err, pgx.ErrNoRows) {
			return r.GetByOrderID(ctx, s.OrderID)
		}
		return nil, fmt.Errorf("create shipment: %w", err)
	}

	return toDomain(row), nil
}

func (r *shipmentRepo) GetByID(ctx context.Context, id string) (*domain.Shipment, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	row, err := r.q.GetShipmentByID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("id %q: %w", id, repository.ErrNotFound)
		}
		return nil, fmt.Errorf("get shipment by id: %w", err)
	}

	return toDomain(row), nil
}

func (r *shipmentRepo) GetByOrderID(ctx context.Context, orderID string) (*domain.Shipment, error) {
	row, err := r.q.GetShipmentByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("order_id %q: %w", orderID, repository.ErrNotFound)
		}
		return nil, fmt.Errorf("get shipment by order_id: %w", err)
	}

	return toDomain(row), nil
}

func (r *shipmentRepo) Update(ctx context.Context, s *domain.Shipment) (*domain.Shipment, error) {
	uid, err := parseUUID(s.ID)
	if err != nil {
		return nil, err
	}

	row, err := r.q.UpdateShipment(ctx, UpdateShipmentParams{
		ID:             uid,
		TrackingNumber: s.TrackingNumber,
		DeliveryDate:   toTimestamptz(s.DeliveryDate),
		Status:         string(s.Status),
		Confirmed:      s.Confirmed,
		Name:           s.Address.Name,
		Street:         s.Address.Street,
		City:           s.Address.City,
		Country:        s.Address.Country,
		UpdatedAt:      toTimestamptz(time.Now().UTC()),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("id %q: %w", s.ID, repository.ErrNotFound)
		}
		return nil, fmt.Errorf("update shipment: %w", err)
	}

	return toDomain(row), nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func parseUUID(id string) (pgtype.UUID, error) {
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return uid, fmt.Errorf("invalid uuid %q: %w", id, err)
	}
	return uid, nil
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
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
		WorkflowID: row.WorkflowID,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}
