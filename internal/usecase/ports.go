package usecase

import (
	"context"

	"shipment_ms/internal/domain"
)

// TemporalClient is the port the use case layer needs from Temporal.
// The concrete implementation lives in internal/infrastructure/temporal.
type TemporalClient interface {
	SignalDeliveryConfirmed(ctx context.Context, workflowID, orderID, shipmentID string) error
}

// EventPublisher is the port for publishing domain events.
// The concrete implementation lives in internal/infrastructure/kafka.
type EventPublisher interface {
	PublishDeliveryConfirmed(ctx context.Context, event domain.DeliveryConfirmedEvent) error
}
