package temporal

import "go.temporal.io/sdk/client"

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
