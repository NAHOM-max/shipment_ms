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

	// WithinTx executes fn inside a database transaction. The TxRepository
	// passed to fn shares the same transaction for all operations.
	WithinTx(ctx context.Context, fn func(TxRepository) error) error
}

// OutboxRepository manages outbox events for reliable async publishing.
type OutboxRepository interface {
	CreateOutboxEvent(ctx context.Context, event *domain.OutboxEvent) error
	FetchPending(ctx context.Context, maxRetries, limit int) ([]*domain.OutboxEvent, error)
	MarkSent(ctx context.Context, eventID string) error
	MarkFailed(ctx context.Context, eventID string) error
}

// TxRepository combines shipment and outbox operations within a transaction.
// Passed to the callback in ShipmentRepository.WithinTx.
type TxRepository interface {
	UpdateShipment(ctx context.Context, s *domain.Shipment) (*domain.Shipment, error)
	CreateOutboxEvent(ctx context.Context, event *domain.OutboxEvent) error
}
