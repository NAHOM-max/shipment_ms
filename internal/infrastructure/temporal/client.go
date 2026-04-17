package temporal

import (
	"context"

	"go.temporal.io/sdk/client"
)

const (
	deliveryWorkflowID = "delivery-workflow"
	signalDelivered    = "delivery-confirmed"
)

type Client struct {
	temporal client.Client
}

func NewClient(hostPort string) (*Client, error) {
	c, err := client.Dial(client.Options{HostPort: hostPort})
	if err != nil {
		return nil, err
	}
	return &Client{temporal: c}, nil
}

func (c *Client) Close() {
	c.temporal.Close()
}

// SignalDeliveryConfirmed satisfies usecase.TemporalClient.
func (c *Client) SignalDeliveryConfirmed(ctx context.Context, shipmentID string) error {
	return c.temporal.SignalWorkflow(
		ctx,
		deliveryWorkflowID,
		"",
		signalDelivered,
		shipmentID,
	)
}
