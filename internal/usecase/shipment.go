package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"shipment_ms/internal/domain"
	"shipment_ms/internal/repository"
)

type ShipmentUseCase struct {
	repo     repository.ShipmentRepository
	temporal TemporalClient
	log      *slog.Logger
}

func NewShipmentUseCase(repo repository.ShipmentRepository, t TemporalClient, log *slog.Logger) *ShipmentUseCase {
	return &ShipmentUseCase{repo: repo, temporal: t, log: log}
}

// ── 1. CreateShipment ────────────────────────────────────────────────────────

type CreateShipmentInput struct {
	OrderID        string
	OrderCreatedAt time.Time
	Address        domain.Address
	WorkflowID     string
}

func (uc *ShipmentUseCase) CreateShipment(ctx context.Context, in CreateShipmentInput) (*domain.Shipment, error) {
	s := &domain.Shipment{
		ID:             uuid.NewString(),
		OrderID:        in.OrderID,
		Address:        in.Address,
		TrackingNumber: generateTrackingNumber(),
		DeliveryDate:   in.OrderCreatedAt.UTC().AddDate(0, 0, 7),
		Status:         domain.Created,
		WorkflowID:     in.WorkflowID,
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

// ConfirmDelivery is idempotent: already-confirmed shipments are returned
// immediately without a DB write or Temporal signal.
// The Temporal signal is sent only after the DB commit succeeds.
// A signal failure is non-fatal — the shipment stays confirmed in the DB and
// the error is logged and surfaced to the caller as a warning.
func (uc *ShipmentUseCase) ConfirmDelivery(ctx context.Context, shipmentID string) (*domain.Shipment, error) {
	s, err := uc.repo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	// Idempotency: already confirmed — nothing to do.
	if s.Confirmed {
		uc.log.InfoContext(ctx, "confirm delivery: already confirmed, skipping",
			"shipment_id", shipmentID,
		)
		return s, nil
	}

	s.MarkDeliveredAndConfirmed() // sets Status=DELIVERED, Confirmed=true, UpdatedAt=now

	persisted, err := uc.repo.Update(ctx, s)
	if err != nil {
		return nil, err
	}

	uc.log.InfoContext(ctx, "shipment confirmed",
		"shipment_id", persisted.ID,
		"order_id", persisted.OrderID,
	)

	// Signal after DB commit. Failure is non-fatal.
	if err := uc.temporal.SignalDeliveryConfirmed(ctx, s.WorkflowID, s.OrderID, persisted.ID); err != nil {
		uc.log.WarnContext(ctx, "delivery confirmed in DB but temporal signal failed",
			"shipment_id", persisted.ID,
			"order_id", s.OrderID,
			"workflow_id", s.WorkflowID,
			"error", err,
		)
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
