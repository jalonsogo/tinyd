package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"tinyd/internal/types"
)

// FetchNetworks retrieves all networks with usage information
func (c *Client) FetchNetworks(ctx context.Context) ([]types.Network, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutQuick)
		defer cancel()
	}

	// Get all containers to determine which networks are in use
	containersResult, err := c.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("operation timed out after %s", TimeoutQuick)
		}
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Build a set of network IDs that are in use
	networksInUse := make(map[string]bool)
	for _, container := range containersResult.Items {
		if container.NetworkSettings != nil {
			for networkID := range container.NetworkSettings.Networks {
				networksInUse[networkID] = true
			}
		}
	}

	result, err := c.cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("operation timed out after %s", TimeoutQuick)
		}
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	networks := make([]types.Network, 0, len(result.Items))

	for _, net := range result.Items {
		parsedNetwork := parseNetwork(net)
		// Check if network is in use
		if networksInUse[net.Name] || networksInUse[net.ID] {
			parsedNetwork.InUse = true
		}
		networks = append(networks, parsedNetwork)
	}

	// Sort networks by priority: In Use > Unused
	sort.SliceStable(networks, func(i, j int) bool {
		return getNetworkPriority(networks[i]) < getNetworkPriority(networks[j])
	})

	return networks, nil
}

// DeleteNetwork removes a network
func (c *Client) DeleteNetwork(ctx context.Context, networkID string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutMedium)
		defer cancel()
	}

	_, err := c.cli.NetworkRemove(ctx, networkID, client.NetworkRemoveOptions{})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("delete operation timed out after %s", TimeoutMedium)
		}
		return fmt.Errorf("failed to delete network: %w", err)
	}
	return nil
}

// InspectNetwork retrieves detailed network information
func (c *Client) InspectNetwork(ctx context.Context, networkID string) (string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutQuick)
		defer cancel()
	}

	inspectResult, err := c.cli.NetworkInspect(ctx, networkID, client.NetworkInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to inspect network: %w", err)
	}

	return formatNetworkInspect(inspectResult), nil
}

// Helper functions

func parseNetwork(net network.Summary) types.Network {
	// Extract IPv4 and IPv6 subnets
	ipv4 := "--"
	ipv6 := "--"

	if len(net.IPAM.Config) > 0 {
		for _, config := range net.IPAM.Config {
			subnet := config.Subnet.String()
			if subnet != "" && subnet != "invalid Prefix" {
				// Simple check: if contains ":", it's likely IPv6
				if strings.Contains(subnet, ":") {
					if ipv6 == "--" {
						ipv6 = subnet
					}
				} else {
					if ipv4 == "--" {
						ipv4 = subnet
					}
				}
			}
		}
	}

	// Determine if network is in use
	// Default to false, will be checked against networksInUse map in caller if needed
	inUse := false

	// Format network ID
	networkID := net.ID
	if len(networkID) > 12 {
		networkID = networkID[:12]
	}

	return types.Network{
		ID:     networkID,
		Name:   net.Name,
		Driver: net.Driver,
		Scope:  net.Scope,
		IPv4:   ipv4,
		IPv6:   ipv6,
		InUse:  inUse,
	}
}

func getNetworkPriority(net types.Network) int {
	if net.InUse {
		return 1
	}
	return 2
}

func formatNetworkInspect(net client.NetworkInspectResult) string {
	// Format inspect data as JSON (pretty-printed)
	jsonData, err := json.MarshalIndent(net, "", "  ")
	if err != nil {
		return fmt.Sprintf("Failed to format network data: %v", err)
	}

	var b strings.Builder
	b.WriteString("=== NETWORK DETAILS ===\n\n")
	b.WriteString(string(jsonData))

	return b.String()
}
