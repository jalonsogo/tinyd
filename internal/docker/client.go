// Package docker provides Docker API operations with timeout management and error handling.
package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/moby/moby/client"
)

// Client wraps the Docker client with tinyd-specific operations and timeout management
type Client struct {
	cli            *client.Client
	defaultTimeout time.Duration
}

// NewClient creates a new Docker client wrapper with sensible defaults
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Client{
		cli:            cli,
		defaultTimeout: 10 * time.Second,
	}, nil
}

// Close closes the underlying Docker client
func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}

// WithTimeout creates a context with the default timeout
func (c *Client) WithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.defaultTimeout)
}

// WithCustomTimeout creates a context with a custom timeout
func (c *Client) WithCustomTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// SetDefaultTimeout updates the default timeout for operations
func (c *Client) SetDefaultTimeout(timeout time.Duration) {
	c.defaultTimeout = timeout
}

// Underlying returns the raw Docker client for advanced operations
func (c *Client) Underlying() *client.Client {
	return c.cli
}

// Operation timeout constants
const (
	TimeoutQuick  = 5 * time.Second   // List, inspect operations
	TimeoutMedium = 15 * time.Second  // Start, stop, delete operations
	TimeoutLong   = 60 * time.Second  // Pull, create operations
)
