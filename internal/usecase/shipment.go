package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"shipment_ms/internal/domain"
	"shipment_ms/internal/repository"
)

const (
	eventTypeDeliveryConfirmed = "delivery.confirmed"
	aggregateTypeShipment      = "shipment"
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
// immediately. The shipment update and outbox event insertion are committed
// in a single transaction. The Temporal signal fires only after that commit.
func (uc *ShipmentUseCase) ConfirmDelivery(ctx context.Context, shipmentID string) (*domain.Shipment, error) {
	s, err := uc.repo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	if s.Confirmed {
		uc.log.InfoContext(ctx, "confirm delivery: already confirmed, skipping",
			"shipment_id", shipmentID,
		)
		return s, nil
	}

	s.MarkDeliveredAndConfirmed()

	payload, err := buildOutboxPayload(s)
	if err != nil {
		return nil, err
	}

	outboxEvent := &domain.OutboxEvent{
		ID:            uuid.NewString(),
		AggregateType: aggregateTypeShipment,
		AggregateID:   s.ID,
		EventType:     eventTypeDeliveryConfirmed,
		Payload:       payload,
	}

	// ── single transaction: update shipment + insert outbox event ────────────
	var persisted *domain.Shipment
	if err := uc.repo.WithinTx(ctx, func(tx repository.TxRepository) error {
		updated, err := tx.UpdateShipment(ctx, s)
		if err != nil {
			return err
		}
		persisted = updated
		return tx.CreateOutboxEvent(ctx, outboxEvent)
	}); err != nil {
		return nil, fmt.Errorf("confirm delivery transaction: %w", err)
	}
	// ── commit done ──────────────────────────────────────────────────────────

	uc.log.InfoContext(ctx, "shipment confirmed and outbox event created",
		"shipment_id", persisted.ID,
		"order_id", persisted.OrderID,
		"event_id", outboxEvent.ID,
	)

	// Signal Temporal after commit. Failure is non-fatal — the outbox worker
	// guarantees Kafka delivery regardless.
	if err := uc.temporal.SignalDeliveryConfirmed(ctx, s.WorkflowID, s.OrderID, persisted.ID); err != nil {
		uc.log.WarnContext(ctx, "delivery confirmed in DB but temporal signal failed",
			"shipment_id", persisted.ID,
			"order_id", s.OrderID,
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

func buildOutboxPayload(s *domain.Shipment) ([]byte, error) {
	v := struct {
		ShipmentID     string    `json:"shipment_id"`
		OrderID        string    `json:"order_id"`
		TrackingNumber string    `json:"tracking_number"`
		DeliveredAt    time.Time `json:"delivered_at"`
	}{
		ShipmentID:     s.ID,
		OrderID:        s.OrderID,
		TrackingNumber: s.TrackingNumber,
		DeliveredAt:    s.UpdatedAt,
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("build outbox payload: %w", err)
	}
	return b, nil
}
