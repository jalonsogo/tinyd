package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"tinyd/internal/types"
)

// FetchContainers retrieves all containers with their stats
func (c *Client) FetchContainers(ctx context.Context) ([]types.Container, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutQuick)
		defer cancel()
	}

	// List all containers (including stopped ones)
	result, err := c.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("operation timed out after %s", TimeoutQuick)
		}
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	containers := make([]types.Container, 0, len(result.Items))

	for _, dockerContainer := range result.Items {
		container := c.parseContainer(ctx, dockerContainer)
		containers = append(containers, container)
	}

	// Sort containers by status priority: RUNNING > PAUSED > ERROR > STOPPED
	sort.SliceStable(containers, func(i, j int) bool {
		return getStatusPriority(containers[i].Status) < getStatusPriority(containers[j].Status)
	})

	return containers, nil
}

// parseContainer converts a Docker API container to our display type
func (c *Client) parseContainer(ctx context.Context, dockerContainer container.Summary) types.Container {
	// Format container name (remove leading /)
	name := "unknown"
	if len(dockerContainer.Names) > 0 {
		name = strings.TrimPrefix(dockerContainer.Names[0], "/")
	}

	// Parse status
	status := parseContainerStatus(string(dockerContainer.State), dockerContainer.Status)

	// Format image (shorten if too long)
	img := formatImageName(dockerContainer.Image)

	// Format ports
	ports := formatPorts(dockerContainer.Ports)

	// Get stats for running containers
	cpu := "--"
	mem := "--"
	if string(dockerContainer.State) == "running" {
		cpu, mem, _ = c.fetchContainerStats(ctx, dockerContainer.ID)
	}

	// Format container ID
	containerID := dockerContainer.ID
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}

	return types.Container{
		ID:     containerID,
		Name:   name,
		Status: status,
		CPU:    cpu,
		Mem:    mem,
		Image:  img,
		Ports:  ports,
	}
}

// fetchContainerStats retrieves CPU and memory stats for a container
func (c *Client) fetchContainerStats(ctx context.Context, containerID string) (cpu string, mem string, err error) {
	statsResp, err := c.cli.ContainerStats(ctx, containerID, client.ContainerStatsOptions{Stream: false})
	if err != nil {
		return "--", "--", err
	}
	defer statsResp.Body.Close()

	var statsJSON struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage  uint64   `json:"total_usage"`
				PercpuUsage []uint64 `json:"percpu_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
		} `json:"memory_stats"`
	}

	if err := json.NewDecoder(statsResp.Body).Decode(&statsJSON); err != nil {
		return "--", "--", err
	}

	// Calculate CPU percentage
	cpu = "--"
	cpuDelta := float64(statsJSON.CPUStats.CPUUsage.TotalUsage) - float64(statsJSON.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(statsJSON.CPUStats.SystemUsage) - float64(statsJSON.PreCPUStats.SystemUsage)
	if systemDelta > 0.0 && cpuDelta > 0.0 && len(statsJSON.CPUStats.CPUUsage.PercpuUsage) > 0 {
		cpuPercent := (cpuDelta / systemDelta) * float64(len(statsJSON.CPUStats.CPUUsage.PercpuUsage)) * 100.0
		cpu = fmt.Sprintf("%.1f", cpuPercent)
	}

	// Format memory
	mem = "--"
	if statsJSON.MemoryStats.Usage > 0 {
		mem = units.BytesSize(float64(statsJSON.MemoryStats.Usage))
	}

	return cpu, mem, nil
}

// StartContainer starts a container
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutMedium)
		defer cancel()
	}

	_, err := c.cli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("start operation timed out after %s", TimeoutMedium)
		}
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// StopContainer stops a container
func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutMedium)
		defer cancel()
	}

	// Allow 10 seconds for graceful shutdown
	timeout := 10
	_, err := c.cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{Timeout: &timeout})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("stop operation timed out after %s", TimeoutMedium)
		}
		return fmt.Errorf("failed to stop container: %w", err)
	}
	return nil
}

// RestartContainer restarts a container
func (c *Client) RestartContainer(ctx context.Context, containerID string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutMedium)
		defer cancel()
	}

	timeout := 10
	_, err := c.cli.ContainerRestart(ctx, containerID, client.ContainerRestartOptions{Timeout: &timeout})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("restart operation timed out after %s", TimeoutMedium)
		}
		return fmt.Errorf("failed to restart container: %w", err)
	}
	return nil
}

// DeleteContainer removes a container
func (c *Client) DeleteContainer(ctx context.Context, containerID string, force bool) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutMedium)
		defer cancel()
	}

	_, err := c.cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("delete operation timed out after %s", TimeoutMedium)
		}
		return fmt.Errorf("failed to delete container: %w", err)
	}
	return nil
}

// GetContainerLogs retrieves container logs
func (c *Client) GetContainerLogs(ctx context.Context, containerID string, tailLines string) (string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutQuick)
		defer cancel()
	}

	options := client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tailLines,
	}

	logs, err := c.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer logs.Close()

	logBytes, err := io.ReadAll(logs)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return string(logBytes), nil
}

// InspectContainer retrieves detailed container information as JSON
func (c *Client) InspectContainer(ctx context.Context, containerID string) (string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutQuick)
		defer cancel()
	}

	inspectResult, err := c.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to inspect: %w", err)
	}

	// Return full container inspect data as JSON
	jsonBytes, err := json.MarshalIndent(inspectResult.Container, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal inspect result: %w", err)
	}

	return string(jsonBytes), nil
}

// Helper functions

func parseContainerStatus(state, status string) string {
	s := "STOPPED"
	if string(state) == "running" {
		s = "RUNNING"
	} else if string(state) == "paused" {
		s = "PAUSED"
	} else if string(state) == "restarting" {
		s = "RESTARTING"
	} else if string(state) == "dead" || string(state) == "exited" {
		// Check exit code for errors
		if status != "" && strings.Contains(strings.ToLower(status), "error") {
			s = "ERROR"
		} else if status != "" && (strings.Contains(strings.ToLower(status), "exit") || strings.Contains(status, "(")) {
			// Check for non-zero exit codes in format "Exited (1)" or "Exited (42)"
			if strings.Contains(status, "Exited (") && !strings.Contains(status, "Exited (0)") {
				s = "ERROR"
			}
		}
	}
	return s
}

func formatImageName(img string) string {
	if len(img) > 17 {
		parts := strings.Split(img, ":")
		if len(parts) > 0 {
			img = parts[0]
			if len(img) > 17 {
				img = img[:14] + "..."
			}
		}
	}
	return img
}

func formatPorts(ports []container.PortSummary) string {
	if len(ports) == 0 {
		return ""
	}

	var portStrs []string
	seen := make(map[uint16]bool)

	for _, port := range ports {
		if port.PublicPort > 0 && !seen[port.PublicPort] {
			portStrs = append(portStrs, fmt.Sprintf("%d", port.PublicPort))
			seen[port.PublicPort] = true
		} else if port.PrivatePort > 0 && !seen[port.PrivatePort] && port.PublicPort == 0 {
			portStrs = append(portStrs, fmt.Sprintf("%d", port.PrivatePort))
			seen[port.PrivatePort] = true
		}
	}

	if len(portStrs) > 3 {
		portStrs = portStrs[:3]
	}

	return strings.Join(portStrs, ",")
}

func getStatusPriority(status string) int {
	switch status {
	case "RUNNING":
		return 1
	case "RESTARTING":
		return 2
	case "PAUSED":
		return 3
	case "ERROR":
		return 4
	case "STOPPED":
		return 5
	default:
		return 6
	}
}

func formatContainerInspect(inspect container.InspectResponse) string {
	var b strings.Builder

	b.WriteString("=== STATS ===\n")
	b.WriteString(fmt.Sprintf("ID: %s\n", inspect.ID[:12]))
	b.WriteString(fmt.Sprintf("Name: %s\n", inspect.Name))
	if inspect.State != nil {
		b.WriteString(fmt.Sprintf("Status: %s\n", inspect.State.Status))
		b.WriteString(fmt.Sprintf("Running: %t\n", inspect.State.Running))
		if inspect.State.Running {
			b.WriteString(fmt.Sprintf("Started: %s\n", inspect.State.StartedAt))
		}
	}
	b.WriteString(fmt.Sprintf("Created: %s\n", inspect.Created))

	b.WriteString("\n=== IMAGE ===\n")
	b.WriteString(fmt.Sprintf("Image: %s\n", inspect.Image))

	b.WriteString("\n=== BIND MOUNTS ===\n")
	if len(inspect.Mounts) == 0 {
		b.WriteString("No mounts\n")
	} else {
		for _, mount := range inspect.Mounts {
			b.WriteString(fmt.Sprintf("Type: %s\n", string(mount.Type)))
			b.WriteString(fmt.Sprintf("Source: %s\n", mount.Source))
			b.WriteString(fmt.Sprintf("Destination: %s\n", mount.Destination))
			b.WriteString(fmt.Sprintf("RW: %t\n\n", mount.RW))
		}
	}

	return b.String()
}
