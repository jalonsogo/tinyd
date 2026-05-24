// Package dmr is a small client for Docker Model Runner's HTTP API.
//
// Mirrors the shape of internal/docker: a Client struct with context-bound
// timeouts (TimeoutQuick / TimeoutMedium / TimeoutLong) and per-operation
// methods that return typed values from tinyd/internal/types.
//
// The DMR JSON shapes are not strictly documented; structs here use
// `omitempty` + tolerant field types so unexpected payloads don't crash.
package dmr

import (
	"context"
	"net/http"
	"os"
	"time"
)

const defaultBaseURL = "http://localhost:12434"

// Operation timeout constants — separate from internal/docker so model pulls
// (which can take minutes for multi-GB downloads) get appropriate headroom.
const (
	TimeoutQuick  = 5 * time.Second  // list, inspect, status probes
	TimeoutMedium = 30 * time.Second // delete, unload
	TimeoutLong   = 30 * time.Minute // pull — models routinely exceed 5 GB
)

// Client wraps the DMR HTTP API.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a DMR client. Honors DMR_BASE_URL if set, falls back to
// http://localhost:12434 (Docker Desktop's default).
func NewClient() *Client {
	base := os.Getenv("DMR_BASE_URL")
	if base == "" {
		base = defaultBaseURL
	}
	return &Client{
		baseURL: base,
		http:    &http.Client{}, // per-request timeouts come from ctx
	}
}

// BaseURL returns the configured DMR endpoint (useful for error messages).
func (c *Client) BaseURL() string { return c.baseURL }

// Available probes whether DMR is reachable. Returns true only on a clean
// HTTP response with a sane status code — a connection refused (DMR not
// enabled in Docker Desktop) returns false.
func (c *Client) Available(ctx context.Context) bool {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(2 * time.Second)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

// WithTimeout creates a context with the default quick timeout.
func (c *Client) WithTimeout() (context.Context, context.CancelFunc) {
	return c.WithCustomTimeout(TimeoutQuick)
}

// WithCustomTimeout creates a context with the given timeout.
func (c *Client) WithCustomTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
