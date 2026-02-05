package docker

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	if client.cli == nil {
		t.Error("Client wrapper has nil underlying client")
	}

	if client.defaultTimeout != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %v", client.defaultTimeout)
	}
}

func TestClientWithTimeout(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	ctx, cancel := client.WithTimeout()
	defer cancel()

	// Check that context has a deadline
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}

	// Check that deadline is approximately 10 seconds in the future
	expectedDeadline := time.Now().Add(client.defaultTimeout)
	diff := expectedDeadline.Sub(deadline)
	if diff < 0 {
		diff = -diff
	}

	// Allow 100ms tolerance
	if diff > 100*time.Millisecond {
		t.Errorf("Deadline mismatch: expected ~%v, got %v (diff: %v)",
			expectedDeadline, deadline, diff)
	}
}

func TestClientWithCustomTimeout(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	customTimeout := 30 * time.Second
	ctx, cancel := client.WithCustomTimeout(customTimeout)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}

	expectedDeadline := time.Now().Add(customTimeout)
	diff := expectedDeadline.Sub(deadline)
	if diff < 0 {
		diff = -diff
	}

	if diff > 100*time.Millisecond {
		t.Errorf("Deadline mismatch: expected ~%v, got %v", expectedDeadline, deadline)
	}
}

func TestSetDefaultTimeout(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	newTimeout := 20 * time.Second
	client.SetDefaultTimeout(newTimeout)

	if client.defaultTimeout != newTimeout {
		t.Errorf("Expected timeout %v, got %v", newTimeout, client.defaultTimeout)
	}

	// Verify WithTimeout() uses the new default
	ctx, cancel := client.WithTimeout()
	defer cancel()

	deadline, _ := ctx.Deadline()
	expectedDeadline := time.Now().Add(newTimeout)
	diff := expectedDeadline.Sub(deadline)
	if diff < 0 {
		diff = -diff
	}

	if diff > 100*time.Millisecond {
		t.Errorf("New default timeout not applied correctly")
	}
}

func TestContextCancellation(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	ctx, cancel := client.WithTimeout()

	// Cancel immediately
	cancel()

	// Check that context is cancelled
	select {
	case <-ctx.Done():
		// Expected - context was cancelled
		if ctx.Err() != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", ctx.Err())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Context was not cancelled")
	}
}

func TestTimeoutConstants(t *testing.T) {
	tests := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
	}{
		{"TimeoutQuick", TimeoutQuick, 5 * time.Second},
		{"TimeoutMedium", TimeoutMedium, 15 * time.Second},
		{"TimeoutLong", TimeoutLong, 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.timeout != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.timeout, tt.expected)
			}
		})
	}
}

func TestClientClose(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Close should not error
	if err := client.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Closing nil client should not panic
	nilClient := &Client{}
	if err := nilClient.Close(); err != nil {
		t.Errorf("Close() on nil client returned error: %v", err)
	}
}

func TestUnderlying(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	underlying := client.Underlying()
	if underlying == nil {
		t.Error("Underlying() returned nil")
	}

	if underlying != client.cli {
		t.Error("Underlying() returned different client than internal cli")
	}
}
