package temporal

import (
	"context"
	"errors"
	"fmt"
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
	tc client.Client
}

func NewClient(hostPort string) (*Client, error) {
	c, err := client.Dial(client.Options{HostPort: hostPort})
	if err != nil {
		return nil, fmt.Errorf("temporal dial %q: %w", hostPort, err)
	}
	return &Client{tc: c}, nil
}

func (c *Client) Close() {
	c.tc.Close()
}

// SignalDeliveryConfirmed signals the workflow identified by orderID that the
// shipment has been delivered and confirmed. It must only be called after the
// DB commit has succeeded.
//
// Transient errors are retried with exponential backoff. Non-retryable
// Temporal application errors are returned immediately.
func (c *Client) SignalDeliveryConfirmed(ctx context.Context, orderID, shipmentID string) error {
	payload := map[string]string{"shipment_id": shipmentID}

	var lastErr error
	backoff := retryInitialBackoff

	for attempt := 1; attempt <= retryMaxAttempts; attempt++ {
		err := c.tc.SignalWorkflow(ctx, orderID, "", signalDeliveryConfirmed, payload)
		if err == nil {
			return nil
		}

		if isNonRetryable(err) {
			return fmt.Errorf("signal DeliveryConfirmed (order %s, shipment %s): %w", orderID, shipmentID, err)
		}

		lastErr = err

		// Respect context cancellation between retries.
		select {
		case <-ctx.Done():
			return fmt.Errorf("signal DeliveryConfirmed cancelled after %d attempt(s): %w", attempt, ctx.Err())
		case <-time.After(backoff):
		}

		backoff = min(time.Duration(float64(backoff)*retryBackoffFactor), retryMaxBackoff)
	}

	return fmt.Errorf("signal DeliveryConfirmed (order %s, shipment %s) failed after %d attempts: %w",
		orderID, shipmentID, retryMaxAttempts, lastErr)
}

// isNonRetryable returns true for errors that should not be retried:
// Temporal application errors marked non-retryable, and context errors.
func isNonRetryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return appErr.NonRetryable()
	}
	return false
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
