package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"shipment_ms/internal/domain"
	"shipment_ms/internal/repository"
)

type ShipmentUseCase struct {
	repo     repository.ShipmentRepository
	temporal TemporalClient
}

func NewShipmentUseCase(repo repository.ShipmentRepository, t TemporalClient) *ShipmentUseCase {
	return &ShipmentUseCase{repo: repo, temporal: t}
}

// ── 1. CreateShipment ────────────────────────────────────────────────────────

type CreateShipmentInput struct {
	OrderID          string
	OrderCreatedAt   time.Time
	Address          domain.Address
}

// CreateShipment is idempotent by OrderID: if a shipment for the order already
// exists the existing record is returned without error.
func (uc *ShipmentUseCase) CreateShipment(ctx context.Context, in CreateShipmentInput) (*domain.Shipment, error) {
	s := &domain.Shipment{
		ID:             uuid.NewString(),
		OrderID:        in.OrderID,
		Address:        in.Address,
		TrackingNumber: generateTrackingNumber(),
		DeliveryDate:   in.OrderCreatedAt.UTC().AddDate(0, 0, 7),
		Status:         domain.Created,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	return uc.repo.Create(ctx, s)
}

// ── 2. UpdateShipmentStatus ──────────────────────────────────────────────────

type UpdateShipmentStatusInput struct {
	ShipmentID string
	NewStatus  domain.DeliveryStatus
}

// UpdateShipmentStatus validates the transition against the domain state
// machine before persisting.
func (uc *ShipmentUseCase) UpdateShipmentStatus(ctx context.Context, in UpdateShipmentStatusInput) (*domain.Shipment, error) {
	s, err := uc.repo.GetByID(ctx, in.ShipmentID)
	if err != nil {
		return nil, err
	}

	if !domain.IsValidTransition(s.Status, in.NewStatus) {
		return nil, fmt.Errorf("transition %s → %s: %w", s.Status, in.NewStatus, domain.ErrInvalidTransition)
	}

	s.Status = in.NewStatus
	s.UpdatedAt = time.Now().UTC()

	return uc.repo.Update(ctx, s)
}

// ── 3. ConfirmDelivery ───────────────────────────────────────────────────────

// ConfirmDelivery is idempotent: if the shipment is already confirmed it
// returns the current record immediately. Temporal is signalled only after
// the state is successfully persisted.
func (uc *ShipmentUseCase) ConfirmDelivery(ctx context.Context, shipmentID string) (*domain.Shipment, error) {
	s, err := uc.repo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	if s.Confirmed {
		return s, nil
	}

	s.MarkDeliveredAndConfirmed()

	persisted, err := uc.repo.Update(ctx, s)
	if err != nil {
		return nil, err
	}

	if err := uc.temporal.SignalDeliveryConfirmed(ctx, s.OrderID, persisted.ID); err != nil {
		// Signal failure is non-fatal: the shipment is already confirmed in the
		// DB. Log-worthy but should not roll back the user-visible state.
		return persisted, fmt.Errorf("shipment confirmed but temporal signal failed: %w", err)
	}

	return persisted, nil
}

// ── 4. GetShipment ───────────────────────────────────────────────────────────

func (uc *ShipmentUseCase) GetShipment(ctx context.Context, shipmentID string) (*domain.Shipment, error) {
	return uc.repo.GetByID(ctx, shipmentID)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func generateTrackingNumber() string {
	return "TRK-" + uuid.NewString()[:8]
}
