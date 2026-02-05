package docker

import (
	"testing"

	"github.com/moby/moby/api/types/container"
)

func TestGetStatusPriority(t *testing.T) {
	tests := []struct {
		status   string
		expected int
	}{
		{"RUNNING", 1},
		{"PAUSED", 2},
		{"ERROR", 3},
		{"STOPPED", 4},
		{"UNKNOWN", 5},
		{"", 5},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := getStatusPriority(tt.status)
			if result != tt.expected {
				t.Errorf("getStatusPriority(%q) = %d, want %d", tt.status, result, tt.expected)
			}
		})
	}
}

func TestParseContainerStatus(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		status   string
		expected string
	}{
		{"running container", "running", "", "RUNNING"},
		{"paused container", "paused", "", "PAUSED"},
		{"stopped container", "exited", "", "STOPPED"},
		{"error container", "exited", "Error: exit code 1", "ERROR"},
		{"restarting container", "restarting", "", "STOPPED"},
		{"dead container", "dead", "", "STOPPED"},
		{"unknown state", "unknown", "", "STOPPED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseContainerStatus(tt.state, tt.status)
			if result != tt.expected {
				t.Errorf("parseContainerStatus(%q, %q) = %q, want %q",
					tt.state, tt.status, result, tt.expected)
			}
		})
	}
}

func TestFormatImageName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"short name", "nginx", "nginx"},
		{"name with tag", "nginx:latest", "nginx"},
		{"long name without tag", "very-long-repository-name-that-exceeds-17-chars", "very-long-repo..."},
		{"long name with tag", "very-long-repository-name:latest", "very-long-repo..."},
		{"exactly 17 chars", "12345678901234567", "12345678901234567"},
		{"18 chars triggers truncation", "123456789012345678", "12345678901..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatImageName(tt.input)
			if result != tt.expected {
				t.Errorf("formatImageName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatPorts(t *testing.T) {
	tests := []struct {
		name     string
		ports    []container.PortSummary
		expected string
	}{
		{
			name:     "no ports",
			ports:    []container.PortSummary{},
			expected: "",
		},
		{
			name: "single public port",
			ports: []container.PortSummary{
				{PublicPort: 8080, PrivatePort: 80},
			},
			expected: "8080",
		},
		{
			name: "multiple public ports",
			ports: []container.PortSummary{
				{PublicPort: 8080, PrivatePort: 80},
				{PublicPort: 8443, PrivatePort: 443},
			},
			expected: "8080,8443",
		},
		{
			name: "more than 3 ports",
			ports: []container.PortSummary{
				{PublicPort: 8080, PrivatePort: 80},
				{PublicPort: 8443, PrivatePort: 443},
				{PublicPort: 3000, PrivatePort: 3000},
				{PublicPort: 3001, PrivatePort: 3001},
			},
			expected: "8080,8443,3000", // Only first 3
		},
		{
			name: "private port only",
			ports: []container.PortSummary{
				{PublicPort: 0, PrivatePort: 80},
			},
			expected: "80",
		},
		{
			name: "duplicate ports filtered",
			ports: []container.PortSummary{
				{PublicPort: 8080, PrivatePort: 80},
				{PublicPort: 8080, PrivatePort: 80}, // Duplicate
			},
			expected: "8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPorts(tt.ports)
			if result != tt.expected {
				t.Errorf("formatPorts() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestFormatContainerInspect is skipped due to complex type structure
// TODO: Add integration test or simplify the test data structure
// func TestFormatContainerInspect(t *testing.T) {
// 	...
// }

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || contains(s[1:], substr)))
}
