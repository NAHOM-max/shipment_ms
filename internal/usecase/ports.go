package usecase

import "context"

// TemporalClient is the port the use case layer needs from Temporal.
// The concrete implementation lives in internal/infrastructure/temporal.
type TemporalClient interface {
	SignalDeliveryConfirmed(ctx context.Context, shipmentID string) error
}
