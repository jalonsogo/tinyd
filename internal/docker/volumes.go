package docker

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
	"tinyd/internal/types"
)

// FetchVolumes retrieves all volumes with usage information
func (c *Client) FetchVolumes(ctx context.Context) ([]types.Volume, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutQuick)
		defer cancel()
	}

	// First, get all containers to determine which volumes are in use
	containersResult, err := c.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("operation timed out after %s", TimeoutQuick)
		}
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Build a map of volume names to container names
	volumeToContainers := make(map[string][]string)
	for _, container := range containersResult.Items {
		// Skip containers without names
		if len(container.Names) == 0 {
			continue
		}

		containerName := container.Names[0]
		if len(containerName) > 0 && containerName[0] == '/' {
			containerName = containerName[1:] // Remove leading slash
		}

		// Check container mounts
		for _, mount := range container.Mounts {
			if mount.Type == "volume" {
				volumeToContainers[mount.Name] = append(volumeToContainers[mount.Name], containerName)
			}
		}
	}

	result, err := c.cli.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("operation timed out after %s", TimeoutQuick)
		}
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	volumes := make([]types.Volume, 0, len(result.Items))

	for _, vol := range result.Items {
		volume := parseVolume(vol, volumeToContainers)
		volumes = append(volumes, volume)
	}

	// Sort volumes by priority: In Use > Unused
	sort.SliceStable(volumes, func(i, j int) bool {
		return getVolumePriority(volumes[i]) < getVolumePriority(volumes[j])
	})

	return volumes, nil
}

// DeleteVolume removes a volume
func (c *Client) DeleteVolume(ctx context.Context, volumeName string, force bool) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutMedium)
		defer cancel()
	}

	_, err := c.cli.VolumeRemove(ctx, volumeName, client.VolumeRemoveOptions{Force: force})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("delete operation timed out after %s", TimeoutMedium)
		}
		return fmt.Errorf("failed to delete volume: %w", err)
	}
	return nil
}

// InspectVolume retrieves detailed volume information
func (c *Client) InspectVolume(ctx context.Context, volumeName string) (string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutQuick)
		defer cancel()
	}

	inspectResult, err := c.cli.VolumeInspect(ctx, volumeName, client.VolumeInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to inspect volume: %w", err)
	}

	// Get containers using this volume
	containersResult, err := c.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	var containerNames []string
	for _, container := range containersResult.Items {
		for _, mount := range container.Mounts {
			if mount.Type == "volume" && mount.Name == volumeName {
				if len(container.Names) == 0 {
					continue
				}

				containerName := container.Names[0]
				if len(containerName) > 0 && containerName[0] == '/' {
					containerName = containerName[1:] // Remove leading slash
				}
				containerNames = append(containerNames, containerName)
				break
			}
		}
	}

	return formatVolumeInspect(inspectResult.Volume, containerNames), nil
}

// Helper functions

func parseVolume(vol volume.Volume, volumeToContainers map[string][]string) types.Volume {
	name := vol.Name
	if len(name) > 25 {
		name = name[:22] + "..."
	}

	mountpoint := vol.Mountpoint
	if len(mountpoint) > 30 {
		mountpoint = "..." + mountpoint[len(mountpoint)-27:]
	}

	created := "unknown"
	if vol.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, vol.CreatedAt); err == nil {
			created = formatTimeAgo(t)
		}
	}

	// Determine if volume is in use and which containers use it
	inUse := false
	containers := "--"
	if containerList, ok := volumeToContainers[vol.Name]; ok && len(containerList) > 0 {
		inUse = true
		containers = strings.Join(containerList, ", ")
	} else if vol.UsageData != nil && vol.UsageData.RefCount > 0 {
		inUse = true
	}

	return types.Volume{
		Name:       name,
		Driver:     vol.Driver,
		Mountpoint: mountpoint,
		Scope:      vol.Scope,
		Created:    created,
		InUse:      inUse,
		Containers: containers,
	}
}

func getVolumePriority(vol types.Volume) int {
	if vol.InUse {
		return 1
	}
	return 2
}

func formatVolumeInspect(vol volume.Volume, containerNames []string) string {
	var b strings.Builder

	// Basic info section
	b.WriteString("=== VOLUME INFO ===\n")
	b.WriteString(fmt.Sprintf("Name: %s\n", vol.Name))
	b.WriteString(fmt.Sprintf("Driver: %s\n", vol.Driver))
	b.WriteString(fmt.Sprintf("Mountpoint: %s\n", vol.Mountpoint))
	b.WriteString(fmt.Sprintf("Scope: %s\n", vol.Scope))
	if vol.CreatedAt != "" {
		b.WriteString(fmt.Sprintf("Created: %s\n", vol.CreatedAt))
	}

	// Driver options
	if len(vol.Options) > 0 {
		b.WriteString("\n=== DRIVER OPTIONS ===\n")
		for key, value := range vol.Options {
			b.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}

	// Labels
	if len(vol.Labels) > 0 {
		b.WriteString("\n=== LABELS ===\n")
		for key, value := range vol.Labels {
			b.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}

	// Usage data
	if vol.UsageData != nil {
		b.WriteString("\n=== USAGE ===\n")
		b.WriteString(fmt.Sprintf("Size: %s\n", units.HumanSize(float64(vol.UsageData.Size))))
		b.WriteString(fmt.Sprintf("Reference Count: %d\n", vol.UsageData.RefCount))
	}

	// Containers using this volume
	if len(containerNames) > 0 {
		b.WriteString("\n=== CONTAINERS USING THIS VOLUME ===\n")
		for _, name := range containerNames {
			b.WriteString(fmt.Sprintf("  - %s\n", name))
		}
	} else {
		b.WriteString("\n=== CONTAINERS ===\n")
		b.WriteString("No containers are using this volume\n")
	}

	return b.String()
}
