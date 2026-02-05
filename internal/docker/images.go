package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
	"tinyd/internal/types"
)

// FetchImages retrieves all images
func (c *Client) FetchImages(ctx context.Context) ([]types.Image, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutQuick)
		defer cancel()
	}

	result, err := c.cli.ImageList(ctx, client.ImageListOptions{All: true})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("operation timed out after %s", TimeoutQuick)
		}
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	images := make([]types.Image, 0, len(result.Items))

	for _, img := range result.Items {
		image := parseImage(img)
		images = append(images, image)
	}

	// Sort images by priority: In Use > Unused > Dangling
	sort.SliceStable(images, func(i, j int) bool {
		return getImagePriority(images[i]) < getImagePriority(images[j])
	})

	return images, nil
}

// DeleteImage removes an image
func (c *Client) DeleteImage(ctx context.Context, imageID string, force bool) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutMedium)
		defer cancel()
	}

	_, err := c.cli.ImageRemove(ctx, imageID, client.ImageRemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("delete operation timed out after %s", TimeoutMedium)
		}
		return fmt.Errorf("failed to delete image: %w", err)
	}
	return nil
}

// PullImage pulls an image from a registry
func (c *Client) PullImage(ctx context.Context, imageName string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutLong)
		defer cancel()
	}

	reader, err := c.cli.ImagePull(ctx, imageName, client.ImagePullOptions{})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("pull operation timed out after %s", TimeoutLong)
		}
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Read the pull response to completion (required for the pull to actually happen)
	_, err = io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read pull response: %w", err)
	}

	return nil
}

// InspectImage retrieves detailed image information as JSON
func (c *Client) InspectImage(ctx context.Context, imageID string) (string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutQuick)
		defer cancel()
	}

	inspectResult, err := c.cli.ImageInspect(ctx, imageID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect image: %w", err)
	}

	// Return full image inspect data as JSON
	jsonBytes, err := json.Marshal(inspectResult)
	if err != nil {
		return "", fmt.Errorf("failed to marshal inspect result: %w", err)
	}

	return string(jsonBytes), nil
}

// RunContainer creates and starts a container from an image
func (c *Client) RunContainer(ctx context.Context, img *types.Image, containerName string, ports []types.PortMapping, volumes []types.VolumeMapping, envVars []types.EnvVar) (string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutMedium)
		defer cancel()
	}

	// Build image reference
	imageRef := img.Repository + ":" + img.Tag

	// Build container config
	config := &container.Config{
		Image: imageRef,
	}

	// Add environment variables
	if len(envVars) > 0 {
		env := make([]string, len(envVars))
		for i, e := range envVars {
			env[i] = e.Key + "=" + e.Value
		}
		config.Env = env
	}

	// Build host config for ports and volumes
	hostConfig := &container.HostConfig{}

	// Add volume mounts
	if len(volumes) > 0 {
		mounts := make([]string, len(volumes))
		for i, v := range volumes {
			if v.IsNamed {
				mounts[i] = v.VolumeName + ":" + v.Container
			} else {
				mounts[i] = v.Host + ":" + v.Container
			}
		}
		hostConfig.Binds = mounts
	}

	// Create container
	resp, err := c.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:     config,
		HostConfig: hostConfig,
		Name:       containerName,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	_, err = c.cli.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return resp.ID[:12], nil
}

// Helper functions

func parseImage(img image.Summary) types.Image {
	// Get repository and tag
	repo := "<none>"
	tag := "<none>"
	if len(img.RepoTags) > 0 {
		parts := strings.Split(img.RepoTags[0], ":")
		if len(parts) == 2 {
			repo = parts[0]
			tag = parts[1]
		} else {
			repo = img.RepoTags[0]
		}
	} else if len(img.RepoDigests) > 0 {
		parts := strings.Split(img.RepoDigests[0], "@")
		if len(parts) > 0 {
			repo = parts[0]
		}
	}

	// Shorten repository if too long
	if len(repo) > 30 {
		repo = repo[:27] + "..."
	}

	// Format size
	size := units.HumanSize(float64(img.Size))

	// Format created time
	created := time.Unix(img.Created, 0)
	createdStr := formatTimeAgo(created)

	// Format ID
	imageID := img.ID
	if len(imageID) > 12 {
		imageID = imageID[7:19] // Skip "sha256:" prefix and take 12 chars
	}

	// Determine if image is in use or dangling
	inUse := img.Containers > 0
	dangling := (repo == "<none>" || tag == "<none>")

	return types.Image{
		ID:         imageID,
		Repository: repo,
		Tag:        tag,
		Size:       size,
		Created:    createdStr,
		InUse:      inUse,
		Dangling:   dangling,
	}
}

func getImagePriority(img types.Image) int {
	if img.InUse {
		return 1
	} else if img.Dangling {
		return 3
	}
	return 2 // Unused but not dangling
}

func formatImageInspect(inspect client.ImageInspectResult) string {
	var b strings.Builder

	// Basic info section
	b.WriteString("=== IMAGE INFO ===\n")
	b.WriteString(fmt.Sprintf("ID: %s\n", inspect.ID[:19]))
	if len(inspect.RepoTags) > 0 {
		b.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(inspect.RepoTags, ", ")))
	}
	if len(inspect.RepoDigests) > 0 {
		b.WriteString(fmt.Sprintf("Digests: %s\n", strings.Join(inspect.RepoDigests, ", ")))
	}
	b.WriteString(fmt.Sprintf("Created: %s\n", inspect.Created))
	b.WriteString(fmt.Sprintf("Size: %s\n", units.HumanSize(float64(inspect.Size))))

	// Architecture section
	b.WriteString("\n=== ARCHITECTURE ===\n")
	b.WriteString(fmt.Sprintf("OS: %s\n", inspect.Os))
	b.WriteString(fmt.Sprintf("Architecture: %s\n", inspect.Architecture))
	if inspect.Variant != "" {
		b.WriteString(fmt.Sprintf("Variant: %s\n", inspect.Variant))
	}

	// Layers section
	b.WriteString("\n=== LAYERS ===\n")
	if len(inspect.RootFS.Layers) == 0 {
		b.WriteString("No layers found\n")
	} else {
		b.WriteString(fmt.Sprintf("Total layers: %d\n\n", len(inspect.RootFS.Layers)))
		for i, layer := range inspect.RootFS.Layers {
			b.WriteString(fmt.Sprintf("Layer %d:\n", i+1))
			b.WriteString(fmt.Sprintf("  %s\n\n", layer))
		}
	}

	// Config section (entrypoint, cmd, env)
	if inspect.Config != nil {
		b.WriteString("=== CONFIG ===\n")
		if len(inspect.Config.Entrypoint) > 0 {
			b.WriteString(fmt.Sprintf("Entrypoint: %s\n", strings.Join(inspect.Config.Entrypoint, " ")))
		}
		if len(inspect.Config.Cmd) > 0 {
			b.WriteString(fmt.Sprintf("Cmd: %s\n", strings.Join(inspect.Config.Cmd, " ")))
		}
		if len(inspect.Config.Env) > 0 {
			b.WriteString("\nEnvironment Variables:\n")
			for _, env := range inspect.Config.Env {
				b.WriteString(fmt.Sprintf("  %s\n", env))
			}
		}
		if len(inspect.Config.ExposedPorts) > 0 {
			b.WriteString("\nExposed Ports:\n")
			for port := range inspect.Config.ExposedPorts {
				b.WriteString(fmt.Sprintf("  %s\n", port))
			}
		}
	}

	return b.String()
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case duration < 30*24*time.Hour:
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case duration < 365*24*time.Hour:
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(duration.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}
