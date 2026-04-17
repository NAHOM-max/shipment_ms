package temporal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

const signalDeliveryConfirmed = "DeliveryConfirmed"

const (
	retryMaxAttempts    = 4
	retryInitialBackoff = 200 * time.Millisecond
	retryMaxBackoff     = 5 * time.Second
	retryBackoffFactor  = 2.0
)

type Client struct {
	tc  client.Client
	log *slog.Logger
}

func NewClient(hostPort string, log *slog.Logger) (*Client, error) {
	c, err := client.Dial(client.Options{HostPort: hostPort})
	if err != nil {
		return nil, fmt.Errorf("temporal dial %q: %w", hostPort, err)
	}
	return &Client{tc: c, log: log}, nil
}

func (c *Client) Close() {
	c.tc.Close()
}

// SignalDeliveryConfirmed signals the workflow identified by orderID.
// Must only be called after the DB commit has succeeded.
// Transient errors are retried with exponential backoff.
// Non-retryable Temporal application errors and context errors return immediately.
func (c *Client) SignalDeliveryConfirmed(ctx context.Context, orderID, shipmentID string) error {
	payload := map[string]string{"shipment_id": shipmentID}
	backoff := retryInitialBackoff

	for attempt := 1; attempt <= retryMaxAttempts; attempt++ {
		err := c.tc.SignalWorkflow(ctx, orderID, "", signalDeliveryConfirmed, payload)
		if err == nil {
			c.log.InfoContext(ctx, "temporal signal sent",
				"signal", signalDeliveryConfirmed,
				"order_id", orderID,
				"shipment_id", shipmentID,
				"attempt", attempt,
			)
			return nil
		}

		if isNonRetryable(err) {
			return fmt.Errorf("signal DeliveryConfirmed (order %s, shipment %s): %w", orderID, shipmentID, err)
		}

		c.log.WarnContext(ctx, "temporal signal failed, retrying",
			"signal", signalDeliveryConfirmed,
			"order_id", orderID,
			"shipment_id", shipmentID,
			"attempt", attempt,
			"backoff_ms", backoff.Milliseconds(),
			"error", err,
		)

		// Use NewTimer so the timer can be stopped and GC'd if the context
		// fires first — time.After leaks the timer until it fires.
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("signal DeliveryConfirmed cancelled after %d attempt(s): %w", attempt, ctx.Err())
		case <-timer.C:
		}

		backoff = min(time.Duration(float64(backoff)*retryBackoffFactor), retryMaxBackoff)
	}

	return fmt.Errorf("signal DeliveryConfirmed (order %s, shipment %s) failed after %d attempts",
		orderID, shipmentID, retryMaxAttempts)
}

func isNonRetryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var appErr *temporal.ApplicationError
	return errors.As(err, &appErr) && appErr.NonRetryable()
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
