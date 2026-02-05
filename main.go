package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// Container represents a Docker container with display data
type Container struct {
	ID     string
	Name   string
	Status string
	CPU    string
	Mem    string
	Image  string
	Ports  string
}

// Image represents a Docker image
type Image struct {
	ID      string
	Repository string
	Tag     string
	Size    string
	Created string
	InUse   bool // Whether the image is used by any container
	Dangling bool // Whether the image has <none> tag/repo
}

// Volume represents a Docker volume
type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Scope      string
	Created    string
	InUse      bool   // Whether the volume is mounted to any container
	Containers string // Comma-separated list of container names using this volume
}

// Network represents a Docker network
type Network struct {
	ID      string
	Name    string
	Driver  string
	Scope   string
	IPv4    string
	IPv6    string
	InUse   bool // Whether the network has any connected containers
}

// Message types for Bubble Tea
type containerListMsg []Container
type imageListMsg []Image
type volumeListMsg []Volume
type networkListMsg []Network
type errMsg error
type tickMsg time.Time
type actionSuccessMsg string
type actionErrorMsg string
type logsMsg string
type inspectMsg string

// View modes
type viewMode int

const (
	viewModeList viewMode = iota
	viewModeLogs
	viewModeInspect
	viewModePortSelector
	viewModeStopConfirm
	viewModeFilter
	viewModeRunImage
	viewModeDeleteConfirm
	viewModePullImage
)

// Filter types for each tab
const (
	// Container filters
	containerFilterAll = iota
	containerFilterRunning
)

const (
	// Image filters
	imageFilterAll = iota
	imageFilterInUse
	imageFilterUnused
	imageFilterDangling
)

const (
	// Volume filters
	volumeFilterAll = iota
	volumeFilterInUse
	volumeFilterUnused
)

const (
	// Network filters
	networkFilterAll = iota
	networkFilterInUse
	networkFilterUnused
)

// Run modal field indices
const (
	runFieldContainerName = iota
	runFieldPortHost
	runFieldPortContainer
	runFieldVolumeHost
	runFieldVolumeContainer
	runFieldEnvKey
	runFieldEnvValue
)

// Model represents the application state
type model struct {
	activeTab        int
	selectedRow      int
	scrollOffset     int // For scrolling through long lists
	viewportHeight   int // Number of rows visible at once
	containers       []Container
	images           []Image
	volumes          []Volume
	networks         []Network
	width            int
	height           int
	showHelp         bool
	dockerClient     *client.Client
	err              error
	loading          bool
	statusMessage    string
	actionInProgress bool

	// Detail views
	currentView       viewMode
	logsContent       string
	logsScrollOffset  int
	logsSearchMode    bool
	logsSearchQuery   string
	inspectContent    string
	inspectMode       int // 0=stats, 1=image, 2=mounts
	selectedContainer *Container

	// Port selector
	availablePorts   []string
	selectedPortIdx  int

	// Filters
	containerFilter int
	imageFilter     int
	volumeFilter    int
	networkFilter   int
	filterOptions   []string
	selectedFilter  int

	// Run image modal
	selectedImage     *Image
	runContainerName  string
	runPortHost       string
	runPortContainer  string
	runPorts          []PortMapping
	runVolumes        []VolumeMapping
	runEnvVars        []EnvVar
	runSelectedVolume string
	runVolumeHost     string
	runVolumeContainer string
	runEnvKey         string
	runEnvValue       string
	runModalField     int // Track which field is being edited

	// Pull image modal
	pullImageName string

	// Delete confirmation
	selectedVolume *Volume
	selectedNetwork *Network

	// Components
	header     HeaderComponent
	tabs       TabsComponent
	actionBar  ActionBarComponent
	detailView DetailViewComponent
}

// PortMapping for run modal
type PortMapping struct {
	Host      string
	Container string
}

// VolumeMapping for run modal
type VolumeMapping struct {
	Host      string
	Container string
	IsNamed   bool
	VolumeName string
}

// EnvVar for run modal
type EnvVar struct {
	Key   string
	Value string
}

// Color palette matching the Pencil design
var (
	// Minimalistic color palette
	bgColor     = lipgloss.Color("#0a0a0a")
	borderColor = lipgloss.Color("#303030")
	lineColor   = lipgloss.Color("#1a1a1a")

	// Status colors (for dots)
	green  = lipgloss.Color("#00FF00")
	yellow = lipgloss.Color("#FFFF00")
	red    = lipgloss.Color("#FF0000")
	cyan   = lipgloss.Color("#00FFFF")

	// Text colors
	white      = lipgloss.Color("#FFFFFF")
	grayText   = lipgloss.Color("#666666")
	darkGray   = lipgloss.Color("#444444")
	lightGray  = lipgloss.Color("#999999")
)

// Styles - Minimalistic theme
var (
	normalStyle = lipgloss.NewStyle().
			Foreground(grayText).
			Background(bgColor)

	brightStyle = lipgloss.NewStyle().
			Foreground(white).
			Background(bgColor)

	selectedStyle = lipgloss.NewStyle().
			Foreground(white).
			Background(bgColor).
			Bold(true)

	// Status dot styles
	greenStyle = lipgloss.NewStyle().
			Foreground(green).
			Background(bgColor)

	yellowStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Background(bgColor).
			Bold(true)

	redStyle = lipgloss.NewStyle().
			Foreground(red).
			Background(bgColor)

	cyanStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Background(bgColor)

	grayStyle = lipgloss.NewStyle().
			Foreground(darkGray).
			Background(bgColor)

	// Border styles
	borderStyle = lipgloss.NewStyle().
			Foreground(borderColor).
			Background(bgColor)

	lineStyle = lipgloss.NewStyle().
			Foreground(lineColor).
			Background(bgColor)

	containerStyle = lipgloss.NewStyle().
			Background(bgColor)
)

func initialModel() model {
	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	// Initialize components
	tabs := []TabItem{
		{Name: "Containers", Shortcut: "^D"},
		{Name: "Images", Shortcut: "^I"},
		{Name: "Volumes", Shortcut: "^V"},
		{Name: "Networks", Shortcut: "^N"},
	}

	return model{
		activeTab:      0,
		selectedRow:    0,
		scrollOffset:   0,
		viewportHeight: 10, // Show 10 rows at a time (adjustable)
		containers:     []Container{},
		images:         []Image{},
		volumes:        []Volume{},
		networks:       []Network{},
		width:          90,
		height:         35,
		dockerClient:   cli,
		err:            err,
		loading:        true,

		// Initialize components
		header:     NewHeaderComponent("Docker TUI v2.0.1", "[F1] Help [Q]uit"),
		tabs:       NewTabsComponent(tabs, 0),
		actionBar:  NewActionBarComponent(),
		detailView: NewDetailViewComponent("", 15),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchContainers(m.dockerClient),
		fetchImages(m.dockerClient),
		fetchVolumes(m.dockerClient),
		fetchNetworks(m.dockerClient),
		tickCmd(),
	)
}

// Fetch containers from Docker API
func fetchContainers(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()

		// List all containers (including stopped ones)
		result, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
		if err != nil {
			return errMsg(err)
		}

		var displayContainers []Container

		for _, c := range result.Items {
			// Format container name (remove leading /)
			name := "unknown"
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}

			// Format status
			status := "STOPPED"
			if string(c.State) == "running" {
				status = "RUNNING"
			} else if string(c.State) == "paused" {
				status = "PAUSED"
			} else if string(c.State) == "restarting" || string(c.State) == "dead" || string(c.State) == "exited" {
				// Check exit code for errors
				if c.Status != "" && strings.Contains(strings.ToLower(c.Status), "error") {
					status = "ERROR"
				}
			}

			// Format image (shorten if too long)
			img := c.Image
			if len(img) > 17 {
				parts := strings.Split(img, ":")
				if len(parts) > 0 {
					img = parts[0]
					if len(img) > 17 {
						img = img[:14] + "..."
					}
				}
			}

			// Format ports
			ports := formatPorts(c.Ports)

			// Get stats for running containers
			cpu := "--"
			mem := "--"

			if string(c.State) == "running" {
				statsResp, err := cli.ContainerStats(ctx, c.ID, client.ContainerStatsOptions{Stream: false})
				if err == nil && statsResp.Body != nil {
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

					if decodeErr := json.NewDecoder(statsResp.Body).Decode(&statsJSON); decodeErr == nil {
						// Calculate CPU percentage
						cpuDelta := float64(statsJSON.CPUStats.CPUUsage.TotalUsage) - float64(statsJSON.PreCPUStats.CPUUsage.TotalUsage)
						systemDelta := float64(statsJSON.CPUStats.SystemUsage) - float64(statsJSON.PreCPUStats.SystemUsage)
						if systemDelta > 0.0 && cpuDelta > 0.0 && len(statsJSON.CPUStats.CPUUsage.PercpuUsage) > 0 {
							cpuPercent := (cpuDelta / systemDelta) * float64(len(statsJSON.CPUStats.CPUUsage.PercpuUsage)) * 100.0
							cpu = fmt.Sprintf("%.1f", cpuPercent)
						}

						// Format memory
						if statsJSON.MemoryStats.Usage > 0 {
							mem = units.BytesSize(float64(statsJSON.MemoryStats.Usage))
						}
					}
				}
			}

			// Format container ID
			containerID := c.ID
			if len(containerID) > 12 {
				containerID = containerID[:12]
			}

			displayContainers = append(displayContainers, Container{
				ID:     containerID,
				Name:   name,
				Status: status,
				CPU:    cpu,
				Mem:    mem,
				Image:  img,
				Ports:  ports,
			})
		}

		// Sort containers by status priority: RUNNING > PAUSED > ERROR > STOPPED
		sort.SliceStable(displayContainers, func(i, j int) bool {
			return getStatusPriority(displayContainers[i].Status) < getStatusPriority(displayContainers[j].Status)
		})

		return containerListMsg(displayContainers)
	}
}

// Get status priority for sorting (lower number = higher priority)
func getStatusPriority(status string) int {
	switch status {
	case "RUNNING":
		return 1
	case "PAUSED":
		return 2
	case "ERROR":
		return 3
	case "STOPPED":
		return 4
	default:
		return 5
	}
}

// Get image priority for sorting (lower number = higher priority)
// In Use > Unused > Dangling
func getImagePriority(img Image) int {
	if img.InUse {
		return 1
	} else if img.Dangling {
		return 3
	} else {
		return 2 // Unused but not dangling
	}
}

// Get volume priority for sorting (lower number = higher priority)
// In Use > Unused
func getVolumePriority(vol Volume) int {
	if vol.InUse {
		return 1
	}
	return 2
}

// Get network priority for sorting (lower number = higher priority)
// In Use > Unused
func getNetworkPriority(net Network) int {
	if net.InUse {
		return 1
	}
	return 2
}

// Format ports for display
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

// Fetch images from Docker API
func fetchImages(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()
		result, err := cli.ImageList(ctx, client.ImageListOptions{All: true})
		if err != nil {
			return errMsg(err)
		}

		var displayImages []Image

		for _, img := range result.Items {
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

			displayImages = append(displayImages, Image{
				ID:         imageID,
				Repository: repo,
				Tag:        tag,
				Size:       size,
				Created:    createdStr,
				InUse:      inUse,
				Dangling:   dangling,
			})
		}

		// Sort images by priority: In Use > Unused > Dangling
		sort.SliceStable(displayImages, func(i, j int) bool {
			return getImagePriority(displayImages[i]) < getImagePriority(displayImages[j])
		})

		return imageListMsg(displayImages)
	}
}

// Fetch volumes from Docker API
func fetchVolumes(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()

		// First, get all containers to determine which volumes are in use
		containersResult, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
		if err != nil {
			return errMsg(err)
		}

		// Build a map of volume names to container names
		volumeToContainers := make(map[string][]string)
		for _, c := range containersResult.Items {
			// Skip containers without names
			if len(c.Names) == 0 {
				continue
			}

			containerName := c.Names[0]
			if len(containerName) > 0 && containerName[0] == '/' {
				containerName = containerName[1:] // Remove leading slash
			}

			// Check container mounts
			for _, mount := range c.Mounts {
				if mount.Type == "volume" {
					volumeToContainers[mount.Name] = append(volumeToContainers[mount.Name], containerName)
				}
			}
		}

		result, err := cli.VolumeList(ctx, client.VolumeListOptions{})
		if err != nil {
			return errMsg(err)
		}

		var displayVolumes []Volume

		for _, vol := range result.Items {
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

			displayVolumes = append(displayVolumes, Volume{
				Name:       name,
				Driver:     vol.Driver,
				Mountpoint: mountpoint,
				Scope:      vol.Scope,
				Created:    created,
				InUse:      inUse,
				Containers: containers,
			})
		}

		// Sort volumes by priority: In Use > Unused
		sort.SliceStable(displayVolumes, func(i, j int) bool {
			return getVolumePriority(displayVolumes[i]) < getVolumePriority(displayVolumes[j])
		})

		return volumeListMsg(displayVolumes)
	}
}

// Fetch networks from Docker API
func fetchNetworks(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()

		// First, get all containers to determine which networks are in use
		containersResult, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
		if err != nil {
			return errMsg(err)
		}

		// Build a set of network IDs that are in use
		networksInUse := make(map[string]bool)
		for _, c := range containersResult.Items {
			if c.NetworkSettings != nil && c.NetworkSettings.Networks != nil {
				for networkName := range c.NetworkSettings.Networks {
					networksInUse[networkName] = true
				}
			}
		}

		result, err := cli.NetworkList(ctx, client.NetworkListOptions{})
		if err != nil {
			return errMsg(err)
		}

		var displayNetworks []Network

		for _, net := range result.Items {
			name := net.Name
			if len(name) > 20 {
				name = name[:17] + "..."
			}

			// Get IPv4 subnet
			ipv4 := "--"
			ipv6 := "--"
			if len(net.IPAM.Config) > 0 {
				for _, config := range net.IPAM.Config {
					subnet := config.Subnet.String()
					if strings.Contains(subnet, ".") {
						ipv4 = subnet
						if len(ipv4) > 18 {
							ipv4 = ipv4[:15] + "..."
						}
					} else if strings.Contains(subnet, ":") {
						ipv6 = subnet
						if len(ipv6) > 18 {
							ipv6 = ipv6[:15] + "..."
						}
					}
				}
			}

			networkID := net.ID
			if len(networkID) > 12 {
				networkID = networkID[:12]
			}

			// Determine if network is in use (has connected containers)
			inUse := networksInUse[net.Name] || networksInUse[net.ID]

			displayNetworks = append(displayNetworks, Network{
				ID:     networkID,
				Name:   name,
				Driver: net.Driver,
				Scope:  net.Scope,
				IPv4:   ipv4,
				IPv6:   ipv6,
				InUse:  inUse,
			})
		}

		// Sort networks by priority: In Use > Unused
		sort.SliceStable(displayNetworks, func(i, j int) bool {
			return getNetworkPriority(displayNetworks[i]) < getNetworkPriority(displayNetworks[j])
		})

		return networkListMsg(displayNetworks)
	}
}

// Format time ago helper
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration.Hours() < 1 {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	} else if duration.Hours() < 24 {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	} else if duration.Hours() < 24*7 {
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	} else if duration.Hours() < 24*30 {
		return fmt.Sprintf("%dw ago", int(duration.Hours()/24/7))
	} else if duration.Hours() < 24*365 {
		return fmt.Sprintf("%dmo ago", int(duration.Hours()/24/30))
	}
	return fmt.Sprintf("%dy ago", int(duration.Hours()/24/365))
}

// Ticker for periodic updates
func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Stop a container
func stopContainer(cli *client.Client, containerID, containerName string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return actionErrorMsg("Docker client not initialized")
		}

		ctx := context.Background()
		timeout := 10 // seconds

		_, err := cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{Timeout: &timeout})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to stop %s: %v", containerName, err))
		}

		return actionSuccessMsg(fmt.Sprintf("Stopped %s", containerName))
	}
}

// Start a container
func startContainer(cli *client.Client, containerID, containerName string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return actionErrorMsg("Docker client not initialized")
		}

		ctx := context.Background()

		_, err := cli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to start %s: %v", containerName, err))
		}

		return actionSuccessMsg(fmt.Sprintf("Started %s", containerName))
	}
}

// Restart a container
func restartContainer(cli *client.Client, containerID, containerName string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return actionErrorMsg("Docker client not initialized")
		}

		ctx := context.Background()
		timeout := 10 // seconds

		_, err := cli.ContainerRestart(ctx, containerID, client.ContainerRestartOptions{Timeout: &timeout})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to restart %s: %v", containerName, err))
		}

		return actionSuccessMsg(fmt.Sprintf("Restarted %s", containerName))
	}
}

// Open browser to container port
func openBrowser(port string) tea.Cmd {
	return func() tea.Msg {
		if port == "" || port == "--" {
			return actionErrorMsg("No ports exposed")
		}

		// Extract first port if multiple
		ports := strings.Split(port, ",")
		firstPort := strings.TrimSpace(ports[0])

		url := fmt.Sprintf("http://localhost:%s", firstPort)

		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			// -g flag opens in background without stealing focus
			cmd = exec.Command("open", "-g", url)
		case "linux":
			// Use nohup to prevent terminal blocking and run in background
			cmd = exec.Command("sh", "-c", fmt.Sprintf("nohup xdg-open %s >/dev/null 2>&1 &", url))
		case "windows":
			// /B flag prevents creating new window and opening in background
			cmd = exec.Command("cmd", "/c", "start", "/B", url)
		default:
			return actionErrorMsg("Unsupported operating system")
		}

		if err := cmd.Start(); err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to open browser: %v", err))
		}

		return actionSuccessMsg(fmt.Sprintf("Opening %s in browser (background)", url))
	}
}

// Parse ports string into individual port numbers
func parsePorts(portsStr string) []string {
	if portsStr == "" || portsStr == "--" {
		return []string{}
	}

	// Split by comma and clean up
	ports := strings.Split(portsStr, ",")
	var cleaned []string
	for _, p := range ports {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return cleaned
}

// Open browser with specific port
func openBrowserPort(port string) tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf("http://localhost:%s", port)

		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			// -g flag opens in background without stealing focus
			cmd = exec.Command("open", "-g", url)
		case "linux":
			// Use nohup to prevent terminal blocking and run in background
			cmd = exec.Command("sh", "-c", fmt.Sprintf("nohup xdg-open %s >/dev/null 2>&1 &", url))
		case "windows":
			// /B flag prevents creating new window and opening in background
			cmd = exec.Command("cmd", "/c", "start", "/B", url)
		default:
			return actionErrorMsg("Unsupported operating system")
		}

		if err := cmd.Start(); err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to open browser: %v", err))
		}

		return actionSuccessMsg(fmt.Sprintf("Opening %s in browser (background)", url))
	}
}

// Open console in container
func openConsole(containerID, containerName string, useDebug bool) tea.Cmd {
	var cmd *exec.Cmd

	if useDebug {
		// Use docker debug directly
		cmd = exec.Command("docker", "debug", containerID)
	} else {
		// Try different shells with docker exec
		shells := []string{"/bin/bash", "/bin/sh", "/bin/ash"}
		var selectedShell string

		for _, shell := range shells {
			testCmd := exec.Command("docker", "exec", containerID, "test", "-f", shell)
			if testCmd.Run() == nil {
				selectedShell = shell
				break
			}
		}

		// Fallback to /bin/sh if no shell found
		if selectedShell == "" {
			selectedShell = "/bin/sh"
		}

		// Write init script to display toolbar
		mode := "docker exec"
		if useDebug {
			mode = "docker debug"
		}

		// Create script that shows toolbar and starts shell
		initScript := createToolbarScript(containerName, mode, containerID, selectedShell)

		cmd = exec.Command("docker", "exec", "-it", containerID, selectedShell, "-c", initScript)
	}

	// Use tea.ExecProcess for altscreen support
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Console error: %v", err))
		}
		return actionSuccessMsg(fmt.Sprintf("Exited console for %s", containerName))
	})
}

func createToolbarScript(containerName, mode, containerID, shell string) string {
	// Gradient toolbar from #1D85E1 to #0F4FA9 (light to dark blue)
	modeText := "Exec"
	if mode == "docker debug" {
		modeText = "Debug"
	}

	// Build the toolbar text
	containerInfo := fmt.Sprintf("%s (%s)", containerName, containerID)
	modeInfo := fmt.Sprintf("Mode: %s", modeText)
	exitInfo := "Exit: type 'exit' or Ctrl+D"

	// Create shell script that generates gradient background
	return fmt.Sprintf(`
# Get terminal width
WIDTH=$(tput cols)

# Toolbar text
TEXT="  %s                    %s              %s     "

# RGB gradient colors: #1D85E1 to #0F4FA9
# Start: rgb(29, 133, 225)
# End: rgb(15, 79, 169)

# Calculate text length
TEXT_LEN=${#TEXT}

# Generate gradient toolbar
for i in $(seq 0 $((WIDTH - 1))); do
    # Calculate gradient position (0.0 to 1.0)
    if [ $WIDTH -gt 1 ]; then
        # Linear interpolation
        R=$((29 + (15 - 29) * i / (WIDTH - 1)))
        G=$((133 + (79 - 133) * i / (WIDTH - 1)))
        B=$((225 + (169 - 225) * i / (WIDTH - 1)))
    else
        R=29; G=133; B=225
    fi

    # Get character at position (or space if beyond text)
    if [ $i -lt $TEXT_LEN ]; then
        CHAR=$(printf "%%s" "$TEXT" | cut -c$((i + 1)))
        [ -z "$CHAR" ] && CHAR=" "
    else
        CHAR=" "
    fi

    # Print character with gradient background and white foreground
    printf '\033[48;2;%%d;%%d;%%dm\033[97m%%s\033[0m' $R $G $B "$CHAR"
done
printf '\n'

export PS1='\[\033[1;36m\][%s]\[\033[0m\] \[\033[1;32m\]\w\[\033[0m\] $ '
exec %s
`, containerInfo, modeInfo, exitInfo, containerName, shell)
}

// Get container logs
func getContainerLogs(cli *client.Client, containerID string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()
		options := client.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       "100", // Last 100 lines
		}

		logs, err := cli.ContainerLogs(ctx, containerID, options)
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to get logs: %v", err))
		}
		defer logs.Close()

		logBytes, err := io.ReadAll(logs)
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to read logs: %v", err))
		}

		return logsMsg(string(logBytes))
	}
}

// Get container inspect info
func inspectContainer(cli *client.Client, containerID string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()
		inspectResult, err := cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to inspect: %v", err))
		}

		inspectData := inspectResult.Container

		// Format inspect data
		var b strings.Builder

		// Stats section
		b.WriteString("=== STATS ===\n")
		b.WriteString(fmt.Sprintf("ID: %s\n", inspectData.ID[:12]))
		b.WriteString(fmt.Sprintf("Name: %s\n", inspectData.Name))
		if inspectData.State != nil {
			b.WriteString(fmt.Sprintf("Status: %s\n", inspectData.State.Status))
			b.WriteString(fmt.Sprintf("Running: %t\n", inspectData.State.Running))
			if inspectData.State.Running {
				b.WriteString(fmt.Sprintf("Started: %s\n", inspectData.State.StartedAt))
			}
		}
		b.WriteString(fmt.Sprintf("Created: %s\n", inspectData.Created))

		// Image section
		b.WriteString("\n=== IMAGE ===\n")
		b.WriteString(fmt.Sprintf("Image: %s\n", inspectData.Image))

		// Mounts section
		b.WriteString("\n=== BIND MOUNTS ===\n")
		if len(inspectData.Mounts) == 0 {
			b.WriteString("No mounts\n")
		} else {
			for _, mount := range inspectData.Mounts {
				b.WriteString(fmt.Sprintf("Type: %s\n", string(mount.Type)))
				b.WriteString(fmt.Sprintf("Source: %s\n", mount.Source))
				b.WriteString(fmt.Sprintf("Destination: %s\n", mount.Destination))
				b.WriteString(fmt.Sprintf("RW: %t\n\n", mount.RW))
			}
		}

		return inspectMsg(b.String())
	}
}

// Inspect image and show layers
func inspectImage(cli *client.Client, imageID string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()
		inspectResult, err := cli.ImageInspect(ctx, imageID)
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to inspect image: %v", err))
		}

		// Format inspect data
		var b strings.Builder

		// Basic info section
		b.WriteString("=== IMAGE INFO ===\n")
		b.WriteString(fmt.Sprintf("ID: %s\n", inspectResult.ID[:19]))
		if len(inspectResult.RepoTags) > 0 {
			b.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(inspectResult.RepoTags, ", ")))
		}
		if len(inspectResult.RepoDigests) > 0 {
			b.WriteString(fmt.Sprintf("Digests: %s\n", strings.Join(inspectResult.RepoDigests, ", ")))
		}
		b.WriteString(fmt.Sprintf("Created: %s\n", inspectResult.Created))
		b.WriteString(fmt.Sprintf("Size: %s\n", units.HumanSize(float64(inspectResult.Size))))

		// Architecture section
		b.WriteString("\n=== ARCHITECTURE ===\n")
		b.WriteString(fmt.Sprintf("OS: %s\n", inspectResult.Os))
		b.WriteString(fmt.Sprintf("Architecture: %s\n", inspectResult.Architecture))
		if inspectResult.Variant != "" {
			b.WriteString(fmt.Sprintf("Variant: %s\n", inspectResult.Variant))
		}

		// Layers section
		b.WriteString("\n=== LAYERS ===\n")
		if len(inspectResult.RootFS.Layers) == 0 {
			b.WriteString("No layers found\n")
		} else {
			b.WriteString(fmt.Sprintf("Total layers: %d\n\n", len(inspectResult.RootFS.Layers)))
			for i, layer := range inspectResult.RootFS.Layers {
				b.WriteString(fmt.Sprintf("Layer %d:\n", i+1))
				b.WriteString(fmt.Sprintf("  %s\n\n", layer))
			}
		}

		// Config section (entrypoint, cmd, env)
		if inspectResult.Config != nil {
			b.WriteString("=== CONFIG ===\n")
			if len(inspectResult.Config.Entrypoint) > 0 {
				b.WriteString(fmt.Sprintf("Entrypoint: %s\n", strings.Join(inspectResult.Config.Entrypoint, " ")))
			}
			if len(inspectResult.Config.Cmd) > 0 {
				b.WriteString(fmt.Sprintf("Cmd: %s\n", strings.Join(inspectResult.Config.Cmd, " ")))
			}
			if len(inspectResult.Config.Env) > 0 {
				b.WriteString("\nEnvironment Variables:\n")
				for _, env := range inspectResult.Config.Env {
					b.WriteString(fmt.Sprintf("  %s\n", env))
				}
			}
			if len(inspectResult.Config.ExposedPorts) > 0 {
				b.WriteString("\nExposed Ports:\n")
				for port := range inspectResult.Config.ExposedPorts {
					b.WriteString(fmt.Sprintf("  %s\n", port))
				}
			}
		}

		return inspectMsg(b.String())
	}
}

// Inspect volume and show details
func inspectVolume(cli *client.Client, volumeName string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()
		inspectResult, err := cli.VolumeInspect(ctx, volumeName, client.VolumeInspectOptions{})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to inspect volume: %v", err))
		}

		// Format inspect data
		var b strings.Builder

		vol := inspectResult.Volume

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

		// Find containers using this volume
		containersResult, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
		if err == nil {
			var containerNames []string
			for _, c := range containersResult.Items {
				for _, mount := range c.Mounts {
					if mount.Type == "volume" && mount.Name == volumeName {
						if len(c.Names) == 0 {
							continue
						}

						containerName := c.Names[0]
						if len(containerName) > 0 && containerName[0] == '/' {
							containerName = containerName[1:] // Remove leading slash
						}
						containerNames = append(containerNames, containerName)
						break
					}
				}
			}

			if len(containerNames) > 0 {
				b.WriteString("\n=== CONTAINERS USING THIS VOLUME ===\n")
				for _, name := range containerNames {
					b.WriteString(fmt.Sprintf("  - %s\n", name))
				}
			} else {
				b.WriteString("\n=== CONTAINERS ===\n")
				b.WriteString("No containers are using this volume\n")
			}
		}

		return inspectMsg(b.String())
	}
}

// Delete container
func deleteContainer(cli *client.Client, containerID, containerName string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()
		_, err := cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{Force: true, RemoveVolumes: false})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to delete container: %v", err))
		}

		return actionSuccessMsg(fmt.Sprintf("Container %s deleted", containerName))
	}
}

// Delete image
func deleteImage(cli *client.Client, imageID string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()
		_, err := cli.ImageRemove(ctx, imageID, client.ImageRemoveOptions{Force: false, PruneChildren: true})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to delete image: %v", err))
		}

		return actionSuccessMsg("Image deleted successfully")
	}
}

// Pull image
func pullImage(cli *client.Client, imageName string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()
		reader, err := cli.ImagePull(ctx, imageName, client.ImagePullOptions{})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to pull image: %v", err))
		}
		defer reader.Close()

		// Read the pull response to completion (required for the pull to actually happen)
		// We don't display progress, just wait for it to finish
		_, err = io.ReadAll(reader)
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to read pull response: %v", err))
		}

		return actionSuccessMsg(fmt.Sprintf("Image %s pulled successfully", imageName))
	}
}

// Delete volume
func deleteVolume(cli *client.Client, volumeName string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()
		_, err := cli.VolumeRemove(ctx, volumeName, client.VolumeRemoveOptions{Force: true})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to delete volume: %v", err))
		}

		return actionSuccessMsg(fmt.Sprintf("Volume %s deleted", volumeName))
	}
}

// Delete network
func deleteNetwork(cli *client.Client, networkID string) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()
		_, err := cli.NetworkRemove(ctx, networkID, client.NetworkRemoveOptions{})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to delete network: %v", err))
		}

		return actionSuccessMsg("Network deleted successfully")
	}
}

// Run container from image
func runContainer(cli *client.Client, image *Image, containerName string, ports []PortMapping, volumes []VolumeMapping, envVars []EnvVar) tea.Cmd {
	return func() tea.Msg {
		if cli == nil {
			return errMsg(fmt.Errorf("docker client not initialized"))
		}

		ctx := context.Background()

		// Build image reference
		imageRef := image.Repository + ":" + image.Tag

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

		// TODO: Port bindings - need to handle proper port mapping
		// For now, basic container creation without port mappings

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
		resp, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
			Config:     config,
			HostConfig: hostConfig,
			Name:       containerName,
		})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to create container: %v", err))
		}

		// Start container
		_, err = cli.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})
		if err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to start container: %v", err))
		}

		return actionSuccessMsg(fmt.Sprintf("Container started: %s", resp.ID[:12]))
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't process keys if action is in progress
		if m.actionInProgress {
			return m, nil
		}

		// In modal views, handle keys differently
		if m.currentView == viewModeRunImage {
			// Run modal - allow all keys for text input and navigation
			return m, m.handleRunModalInput(msg)
		} else if m.currentView == viewModePullImage {
			// Pull image modal - allow text input
			return m, m.handlePullModalInput(msg)
		} else if m.currentView == viewModeLogs && m.logsSearchMode {
			// Logs search mode - handle text input
			key := msg.String()
			switch key {
			case "backspace":
				if len(m.logsSearchQuery) > 0 {
					m.logsSearchQuery = m.logsSearchQuery[:len(m.logsSearchQuery)-1]
					m.logsScrollOffset = 0 // Reset scroll when search changes
				}
			case "ctrl+c", "q", "esc", "s", "S":
				// Pass through to main switch for special keys
			case "up", "k", "down", "j":
				// Pass through to main switch for navigation
			default:
				// Add character to search query (only if it's a single character)
				if len(key) == 1 {
					m.logsSearchQuery += key
					m.logsScrollOffset = 0 // Reset scroll when search changes
				}
			}
		} else if m.currentView == viewModeStopConfirm || m.currentView == viewModePortSelector || m.currentView == viewModeFilter || m.currentView == viewModeDeleteConfirm {
			key := msg.String()
			// Allow quit, ESC, and Enter to pass through to main switch
			// Allow up/down for port selector and filter modal
			if key == "ctrl+c" || key == "q" || key == "esc" || key == "enter" {
				// Pass through to main switch
			} else if (key == "up" || key == "k" || key == "down" || key == "j") && (m.currentView == viewModePortSelector || m.currentView == viewModeFilter) {
				// Allow navigation in port selector and filter modal
			} else {
				// Block all other keys in modal views
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.dockerClient != nil {
				m.dockerClient.Close()
			}
			return m, tea.Quit
		case "up", "k":
			// Logs view scrolling
			if m.currentView == viewModeLogs {
				if m.logsScrollOffset > 0 {
					m.logsScrollOffset--
				}
				return m, nil
			} else if m.currentView == viewModePortSelector {
				// Port selector navigation
				if m.selectedPortIdx > 0 {
					m.selectedPortIdx--
				}
			} else if m.currentView == viewModeFilter {
				// Filter modal navigation
				if m.selectedFilter > 0 {
					m.selectedFilter--
				}
			} else {
				// Normal list navigation
				if m.selectedRow > 0 {
					m.selectedRow--
					m.statusMessage = "" // Clear status when navigating

					// Scroll up if needed
					if m.selectedRow < m.scrollOffset {
						m.scrollOffset = m.selectedRow
					}
				}
			}
		case "down", "j":
			// Logs view scrolling
			if m.currentView == viewModeLogs {
				// Calculate filtered lines
				logLines := strings.Split(m.logsContent, "\n")
				var filteredLines []string
				if m.logsSearchMode && m.logsSearchQuery != "" {
					query := strings.ToLower(m.logsSearchQuery)
					for _, line := range logLines {
						if strings.Contains(strings.ToLower(line), query) {
							filteredLines = append(filteredLines, line)
						}
					}
				} else {
					filteredLines = logLines
				}

				// Calculate available lines (same as in renderLogs)
				availableLines := m.height - 7
				if availableLines < 5 {
					availableLines = 5
				}

				// Calculate max scroll
				maxScroll := len(filteredLines) - availableLines
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.logsScrollOffset < maxScroll {
					m.logsScrollOffset++
				}
				return m, nil
			} else if m.currentView == viewModePortSelector {
				// Port selector navigation
				if m.selectedPortIdx < len(m.availablePorts)-1 {
					m.selectedPortIdx++
				}
			} else if m.currentView == viewModeFilter {
				// Filter modal navigation
				if m.selectedFilter < len(m.filterOptions)-1 {
					m.selectedFilter++
				}
			} else {
				// Normal list navigation
				maxRow := m.getMaxRow()
				if m.selectedRow < maxRow-1 {
					m.selectedRow++
					m.statusMessage = "" // Clear status when navigating

					// Calculate maximum scroll to keep viewport full
					maxScroll := maxRow - m.viewportHeight
					if maxScroll < 0 {
						maxScroll = 0
					}

					// Scroll down if selected row goes beyond visible area
					if m.selectedRow >= m.scrollOffset+m.viewportHeight {
						m.scrollOffset = m.selectedRow - m.viewportHeight + 1
						// Clamp scroll offset to prevent empty space
						if m.scrollOffset > maxScroll {
							m.scrollOffset = maxScroll
						}
					}
				}
			}
		case "1", "ctrl+d":
			m.activeTab = 0
			m.selectedRow = 0
			m.scrollOffset = 0
			m.statusMessage = ""
		case "2", "ctrl+i":
			m.activeTab = 1
			m.selectedRow = 0
			m.scrollOffset = 0
			m.statusMessage = ""
		case "3", "ctrl+v":
			m.activeTab = 2
			m.selectedRow = 0
			m.scrollOffset = 0
			m.statusMessage = ""
		case "4", "ctrl+n":
			m.activeTab = 3
			m.selectedRow = 0
			m.scrollOffset = 0
			m.statusMessage = ""
		case "left", "h":
			// Navigate to previous tab
			if m.activeTab > 0 {
				m.activeTab--
				m.selectedRow = 0
				m.scrollOffset = 0
				m.statusMessage = ""
			}
		case "right":
			// Navigate to next tab
			if m.activeTab < 3 {
				m.activeTab++
				m.selectedRow = 0
				m.scrollOffset = 0
				m.statusMessage = ""
			}
		case "f1":
			m.showHelp = !m.showHelp
		case "r", "R":
			// Restart on containers tab, Run on images tab
			if m.activeTab == 0 {
				// Restart container
				filteredContainers := filterContainers(m.containers, m.containerFilter)
				if len(filteredContainers) > 0 && m.selectedRow < len(filteredContainers) {
					selectedContainer := filteredContainers[m.selectedRow]
					m.actionInProgress = true
					m.statusMessage = fmt.Sprintf("Restarting %s...", selectedContainer.Name)
					return m, restartContainer(m.dockerClient, selectedContainer.ID, selectedContainer.Name)
				}
			} else if m.activeTab == 1 && m.currentView == viewModeList {
				// Run image
				filteredImages := filterImages(m.images, m.containers, m.imageFilter)
				if len(filteredImages) > 0 && m.selectedRow < len(filteredImages) {
					selectedImage := filteredImages[m.selectedRow]
					m.selectedImage = &selectedImage
					m.currentView = viewModeRunImage
					// Reset form fields
					m.runContainerName = ""
					m.runPortHost = ""
					m.runPortContainer = ""
					m.runPorts = []PortMapping{}
					m.runVolumes = []VolumeMapping{}
					m.runEnvVars = []EnvVar{}
					m.runVolumeHost = ""
					m.runVolumeContainer = ""
					m.runEnvKey = ""
					m.runEnvValue = ""
					m.runModalField = 0
				}
			}
		case "s", "S":
			// Toggle search mode in logs view
			if m.currentView == viewModeLogs {
				m.logsSearchMode = !m.logsSearchMode
				if !m.logsSearchMode {
					// Clear search query when exiting search mode
					m.logsSearchQuery = ""
					m.logsScrollOffset = 0
				}
			} else if m.activeTab == 0 {
				// Start/Stop only works on containers tab
				filteredContainers := filterContainers(m.containers, m.containerFilter)
				if len(filteredContainers) > 0 && m.selectedRow < len(filteredContainers) {
					selectedContainer := filteredContainers[m.selectedRow]
					m.actionInProgress = true

					if selectedContainer.Status == "RUNNING" {
						m.statusMessage = fmt.Sprintf("Stopping %s...", selectedContainer.Name)
						return m, stopContainer(m.dockerClient, selectedContainer.ID, selectedContainer.Name)
					} else {
						m.statusMessage = fmt.Sprintf("Starting %s...", selectedContainer.Name)
						return m, startContainer(m.dockerClient, selectedContainer.ID, selectedContainer.Name)
					}
				}
			}
		case "o", "O":
			// Open browser only works on containers tab
			if m.activeTab == 0 {
				filteredContainers := filterContainers(m.containers, m.containerFilter)
				if len(filteredContainers) > 0 && m.selectedRow < len(filteredContainers) {
					selectedContainer := filteredContainers[m.selectedRow]

					// Parse ports
					ports := parsePorts(selectedContainer.Ports)

					if len(ports) == 0 {
						m.statusMessage = "ERROR: No ports exposed"
					} else if len(ports) == 1 {
						// Single port - open directly
						return m, openBrowser(selectedContainer.Ports)
					} else {
						// Multiple ports - show selector
						m.availablePorts = ports
						m.selectedPortIdx = 0
						m.currentView = viewModePortSelector
						m.selectedContainer = &selectedContainer
					}
				}
			}
		case "c", "C":
			// Open console with toolbar (uses altscreen)
			if m.activeTab == 0 {
				filteredContainers := filterContainers(m.containers, m.containerFilter)
				if len(filteredContainers) > 0 && m.selectedRow < len(filteredContainers) {
					selectedContainer := filteredContainers[m.selectedRow]
					if selectedContainer.Status == "RUNNING" {
						return m, openConsole(selectedContainer.ID, selectedContainer.Name, false)
					} else {
						m.statusMessage = "ERROR: Container must be running"
					}
				}
			}
		case "l", "L":
			// View logs
			if m.activeTab == 0 {
				filteredContainers := filterContainers(m.containers, m.containerFilter)
				if len(filteredContainers) > 0 && m.selectedRow < len(filteredContainers) {
					selectedContainer := filteredContainers[m.selectedRow]
					m.selectedContainer = &selectedContainer
					m.currentView = viewModeLogs
					m.logsScrollOffset = 0
					return m, getContainerLogs(m.dockerClient, selectedContainer.ID)
				}
			}
		case "i", "I":
			// Inspect container, image, or volume
			if m.activeTab == 0 {
				// Containers tab
				filteredContainers := filterContainers(m.containers, m.containerFilter)
				if len(filteredContainers) > 0 && m.selectedRow < len(filteredContainers) {
					selectedContainer := filteredContainers[m.selectedRow]
					m.selectedContainer = &selectedContainer
					m.currentView = viewModeInspect
					m.inspectMode = 0
					return m, inspectContainer(m.dockerClient, selectedContainer.ID)
				}
			} else if m.activeTab == 1 {
				// Images tab
				filteredImages := filterImages(m.images, m.containers, m.imageFilter)
				if len(filteredImages) > 0 && m.selectedRow < len(filteredImages) {
					selectedImage := filteredImages[m.selectedRow]
					m.selectedImage = &selectedImage
					m.currentView = viewModeInspect
					m.inspectMode = 0
					return m, inspectImage(m.dockerClient, selectedImage.ID)
				}
			} else if m.activeTab == 2 {
				// Volumes tab
				filteredVolumes := filterVolumes(m.volumes, m.containers, m.dockerClient)
				if len(filteredVolumes) > 0 && m.selectedRow < len(filteredVolumes) {
					selectedVolume := filteredVolumes[m.selectedRow]
					m.selectedVolume = &selectedVolume
					m.currentView = viewModeInspect
					m.inspectMode = 0
					return m, inspectVolume(m.dockerClient, selectedVolume.Name)
				}
			}
		case "f", "F":
			// Open filter modal
			if m.currentView == viewModeList {
				m.currentView = viewModeFilter
				m.selectedFilter = 0

				// Set filter options based on active tab
				switch m.activeTab {
				case 0: // Containers
					m.filterOptions = []string{"All", "Running"}
					m.selectedFilter = m.containerFilter
				case 1: // Images
					m.filterOptions = []string{"All", "In Use", "Unused", "Dangling"}
					m.selectedFilter = m.imageFilter
				case 2: // Volumes
					m.filterOptions = []string{"All", "In Use", "Unused"}
					m.selectedFilter = m.volumeFilter
				case 3: // Networks
					m.filterOptions = []string{"All", "In Use", "Unused"}
					m.selectedFilter = m.networkFilter
				}
			}
		case "d", "D":
			// Delete selected resource (works on all tabs)
			if m.currentView == viewModeList {
				switch m.activeTab {
				case 0: // Containers
					filteredContainers := filterContainers(m.containers, m.containerFilter)
					if len(filteredContainers) > 0 && m.selectedRow < len(filteredContainers) {
						selectedContainer := filteredContainers[m.selectedRow]
						m.selectedContainer = &selectedContainer
						m.currentView = viewModeDeleteConfirm
					}
				case 1: // Images
					filteredImages := filterImages(m.images, m.containers, m.imageFilter)
					if len(filteredImages) > 0 && m.selectedRow < len(filteredImages) {
						selectedImage := filteredImages[m.selectedRow]
						m.selectedImage = &selectedImage
						m.currentView = viewModeDeleteConfirm
					}
				case 2: // Volumes
					filteredVolumes := filterVolumes(m.volumes, m.containers, m.dockerClient)
					if len(filteredVolumes) > 0 && m.selectedRow < len(filteredVolumes) {
						selectedVolume := filteredVolumes[m.selectedRow]
						m.selectedVolume = &selectedVolume
						m.currentView = viewModeDeleteConfirm
					}
				case 3: // Networks
					filteredNetworks := filterNetworks(m.networks, m.containers, m.dockerClient)
					if len(filteredNetworks) > 0 && m.selectedRow < len(filteredNetworks) {
						selectedNetwork := filteredNetworks[m.selectedRow]
						m.selectedNetwork = &selectedNetwork
						m.currentView = viewModeDeleteConfirm
					}
				}
			}
		case "p", "P":
			// Pull image (Images tab only)
			if m.activeTab == 1 && m.currentView == viewModeList && !m.actionInProgress {
				m.currentView = viewModePullImage
				m.pullImageName = ""
			}
		case "esc":
			// Exit search mode in logs view, or return to list view
			if m.currentView == viewModeLogs && m.logsSearchMode {
				// Exit search mode but stay in logs view
				m.logsSearchMode = false
				m.logsSearchQuery = ""
				m.logsScrollOffset = 0
			} else if m.currentView != viewModeList {
				// Return to list view
				m.currentView = viewModeList
				m.selectedContainer = nil
				m.selectedImage = nil
				m.selectedVolume = nil
				m.selectedNetwork = nil
				m.logsContent = ""
				m.inspectContent = ""
				m.availablePorts = nil
				m.logsSearchMode = false
				m.logsSearchQuery = ""
			}
		case "enter":
			// In delete confirmation modal, execute delete based on resource type
			if m.currentView == viewModeDeleteConfirm {
				m.currentView = viewModeList
				m.actionInProgress = true

				if m.selectedContainer != nil {
					m.statusMessage = fmt.Sprintf("Deleting container %s...", m.selectedContainer.Name)
					return m, deleteContainer(m.dockerClient, m.selectedContainer.ID, m.selectedContainer.Name)
				} else if m.selectedImage != nil {
					m.statusMessage = fmt.Sprintf("Deleting image %s:%s...", m.selectedImage.Repository, m.selectedImage.Tag)
					return m, deleteImage(m.dockerClient, m.selectedImage.ID)
				} else if m.selectedVolume != nil {
					m.statusMessage = fmt.Sprintf("Deleting volume %s...", m.selectedVolume.Name)
					return m, deleteVolume(m.dockerClient, m.selectedVolume.Name)
				} else if m.selectedNetwork != nil {
					m.statusMessage = fmt.Sprintf("Deleting network %s...", m.selectedNetwork.Name)
					return m, deleteNetwork(m.dockerClient, m.selectedNetwork.ID)
				}
			}

			// In run image modal, execute run container
			if m.currentView == viewModeRunImage {
				if m.selectedImage != nil {
					m.currentView = viewModeList
					m.actionInProgress = true
					m.statusMessage = "Starting container..."
					return m, runContainer(m.dockerClient, m.selectedImage, m.runContainerName, m.runPorts, m.runVolumes, m.runEnvVars)
				}
			}

			// In stop confirmation modal, execute stop
			if m.currentView == viewModeStopConfirm {
				if m.selectedContainer != nil {
					m.currentView = viewModeList
					return m, stopContainer(m.dockerClient, m.selectedContainer.ID, m.selectedContainer.Name)
				}
			}

			// In port selector, open selected port
			if m.currentView == viewModePortSelector && len(m.availablePorts) > 0 {
				selectedPort := m.availablePorts[m.selectedPortIdx]
				m.currentView = viewModeList
				m.availablePorts = nil
				return m, openBrowserPort(selectedPort)
			}

			// In filter modal, apply selected filter
			if m.currentView == viewModeFilter && len(m.filterOptions) > 0 {
				// Update filter for current tab
				switch m.activeTab {
				case 0: // Containers
					m.containerFilter = m.selectedFilter
					if m.selectedFilter == containerFilterRunning {
						m.statusMessage = "Filter: Running containers"
					} else {
						m.statusMessage = "Filter: All containers"
					}
				case 1: // Images
					m.imageFilter = m.selectedFilter
					switch m.selectedFilter {
					case imageFilterInUse:
						m.statusMessage = "Filter: In use images"
					case imageFilterUnused:
						m.statusMessage = "Filter: Unused images"
					case imageFilterDangling:
						m.statusMessage = "Filter: Dangling images"
					default:
						m.statusMessage = "Filter: All images"
					}
				case 2: // Volumes
					m.volumeFilter = m.selectedFilter
					if m.selectedFilter == volumeFilterInUse {
						m.statusMessage = "Filter: In use volumes"
					} else if m.selectedFilter == volumeFilterUnused {
						m.statusMessage = "Filter: Unused volumes"
					} else {
						m.statusMessage = "Filter: All volumes"
					}
				case 3: // Networks
					m.networkFilter = m.selectedFilter
					if m.selectedFilter == networkFilterInUse {
						m.statusMessage = "Filter: In use networks"
					} else if m.selectedFilter == networkFilterUnused {
						m.statusMessage = "Filter: Unused networks"
					} else {
						m.statusMessage = "Filter: All networks"
					}
				}

				// Reset selection and scroll
				m.selectedRow = 0
				m.scrollOffset = 0
				m.currentView = viewModeList
				m.filterOptions = nil
				return m, nil // IMPORTANT: Return here to prevent fall-through to container actions
			}

			// Refresh current tab
			if m.currentView == viewModeList {
				m.statusMessage = "Refreshing..."
				switch m.activeTab {
				case 0:
					return m, fetchContainers(m.dockerClient)
				case 1:
					return m, fetchImages(m.dockerClient)
				case 2:
					return m, fetchVolumes(m.dockerClient)
				case 3:
					return m, fetchNetworks(m.dockerClient)
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate viewport height based on terminal height
		// Fixed UI elements: tabs (3 lines) + table header (2 lines) + action bar (2 lines) = 7 lines
		// Reserve 1 line for safety margin
		fixedLines := 8
		m.viewportHeight = msg.Height - fixedLines
		if m.viewportHeight < 5 {
			m.viewportHeight = 5 // Minimum height to show something
		}

		// Update components with new width
		m.header = m.header.WithWidth(m.width)
		m.tabs = m.tabs.WithWidth(m.width)
		m.actionBar = m.actionBar.WithWidth(m.width)
		m.detailView = m.detailView.WithWidth(m.width)

	case containerListMsg:
		m.containers = msg
		m.loading = false
		m.actionInProgress = false
		// Keep selection in bounds
		if m.activeTab == 0 && m.selectedRow >= len(m.containers) && len(m.containers) > 0 {
			m.selectedRow = len(m.containers) - 1
		}
		return m, nil

	case imageListMsg:
		m.images = msg
		// Keep selection in bounds
		if m.activeTab == 1 && m.selectedRow >= len(m.images) && len(m.images) > 0 {
			m.selectedRow = len(m.images) - 1
		}
		return m, nil

	case volumeListMsg:
		m.volumes = msg
		// Keep selection in bounds
		if m.activeTab == 2 && m.selectedRow >= len(m.volumes) && len(m.volumes) > 0 {
			m.selectedRow = len(m.volumes) - 1
		}
		return m, nil

	case networkListMsg:
		m.networks = msg
		// Keep selection in bounds
		if m.activeTab == 3 && m.selectedRow >= len(m.networks) && len(m.networks) > 0 {
			m.selectedRow = len(m.networks) - 1
		}
		return m, nil

	case errMsg:
		m.err = msg
		m.loading = false
		m.actionInProgress = false
		return m, nil

	case actionSuccessMsg:
		m.statusMessage = string(msg)
		m.actionInProgress = false
		// Refresh container list after successful action
		return m, fetchContainers(m.dockerClient)

	case actionErrorMsg:
		m.statusMessage = "ERROR: " + string(msg)
		m.actionInProgress = false
		return m, nil

	case logsMsg:
		m.logsContent = string(msg)
		return m, nil

	case inspectMsg:
		m.inspectContent = string(msg)
		return m, nil

	case tickMsg:
		// Refresh all data periodically (only if no action in progress)
		if !m.actionInProgress {
			return m, tea.Batch(
				fetchContainers(m.dockerClient),
				fetchImages(m.dockerClient),
				fetchVolumes(m.dockerClient),
				fetchNetworks(m.dockerClient),
				tickCmd(),
			)
		}
		return m, tickCmd()
	}

	return m, nil
}

// Handle input in the Run modal
func (m model) handleRunModalInput(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "esc":
		// Exit modal
		m.currentView = viewModeList
		m.selectedImage = nil
		return nil

	case "enter":
		// Handle enter based on current field
		switch m.runModalField {
		case runFieldPortContainer:
			// Add port mapping if both fields are filled
			if m.runPortHost != "" && m.runPortContainer != "" {
				m.runPorts = append(m.runPorts, PortMapping{
					Host:      m.runPortHost,
					Container: m.runPortContainer,
				})
				m.runPortHost = ""
				m.runPortContainer = ""
				m.runModalField = runFieldPortHost
			} else {
				// Move to next section
				m.runModalField = runFieldVolumeHost
			}
		case runFieldVolumeContainer:
			// Add volume mapping if both fields are filled
			if m.runVolumeHost != "" && m.runVolumeContainer != "" {
				m.runVolumes = append(m.runVolumes, VolumeMapping{
					Host:      m.runVolumeHost,
					Container: m.runVolumeContainer,
					IsNamed:   false,
				})
				m.runVolumeHost = ""
				m.runVolumeContainer = ""
				m.runModalField = runFieldVolumeHost
			} else {
				// Move to next section
				m.runModalField = runFieldEnvKey
			}
		case runFieldEnvValue:
			// Add environment variable if both fields are filled
			if m.runEnvKey != "" && m.runEnvValue != "" {
				m.runEnvVars = append(m.runEnvVars, EnvVar{
					Key:   m.runEnvKey,
					Value: m.runEnvValue,
				})
				m.runEnvKey = ""
				m.runEnvValue = ""
				m.runModalField = runFieldEnvKey
			} else {
				// Submit form if no env var being entered
				if m.selectedImage != nil {
					m.currentView = viewModeList
					m.actionInProgress = true
					m.statusMessage = "Starting container..."
					return runContainer(m.dockerClient, m.selectedImage, m.runContainerName, m.runPorts, m.runVolumes, m.runEnvVars)
				}
			}
		default:
			// Move to next field
			m.runModalField++
			if m.runModalField > runFieldEnvValue {
				// Submit form
				if m.selectedImage != nil {
					m.currentView = viewModeList
					m.actionInProgress = true
					m.statusMessage = "Starting container..."
					return runContainer(m.dockerClient, m.selectedImage, m.runContainerName, m.runPorts, m.runVolumes, m.runEnvVars)
				}
			}
		}

	case "tab":
		// Move to next field
		m.runModalField++
		if m.runModalField > runFieldEnvValue {
			m.runModalField = runFieldContainerName
		}

	case "shift+tab":
		// Move to previous field
		m.runModalField--
		if m.runModalField < 0 {
			m.runModalField = runFieldEnvValue
		}

	case "backspace":
		// Delete character from current field
		switch m.runModalField {
		case runFieldContainerName:
			if len(m.runContainerName) > 0 {
				m.runContainerName = m.runContainerName[:len(m.runContainerName)-1]
			}
		case runFieldPortHost:
			if len(m.runPortHost) > 0 {
				m.runPortHost = m.runPortHost[:len(m.runPortHost)-1]
			}
		case runFieldPortContainer:
			if len(m.runPortContainer) > 0 {
				m.runPortContainer = m.runPortContainer[:len(m.runPortContainer)-1]
			}
		case runFieldVolumeHost:
			if len(m.runVolumeHost) > 0 {
				m.runVolumeHost = m.runVolumeHost[:len(m.runVolumeHost)-1]
			}
		case runFieldVolumeContainer:
			if len(m.runVolumeContainer) > 0 {
				m.runVolumeContainer = m.runVolumeContainer[:len(m.runVolumeContainer)-1]
			}
		case runFieldEnvKey:
			if len(m.runEnvKey) > 0 {
				m.runEnvKey = m.runEnvKey[:len(m.runEnvKey)-1]
			}
		case runFieldEnvValue:
			if len(m.runEnvValue) > 0 {
				m.runEnvValue = m.runEnvValue[:len(m.runEnvValue)-1]
			}
		}

	default:
		// Add character to current field (only if it's a single character)
		if len(key) == 1 {
			switch m.runModalField {
			case runFieldContainerName:
				m.runContainerName += key
			case runFieldPortHost:
				m.runPortHost += key
			case runFieldPortContainer:
				m.runPortContainer += key
			case runFieldVolumeHost:
				m.runVolumeHost += key
			case runFieldVolumeContainer:
				m.runVolumeContainer += key
			case runFieldEnvKey:
				m.runEnvKey += key
			case runFieldEnvValue:
				m.runEnvValue += key
			}
		}
	}

	return nil
}

// Handle input in the Pull Image modal
func (m model) handlePullModalInput(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "esc":
		// Exit modal
		m.currentView = viewModeList
		m.pullImageName = ""
		return nil

	case "enter":
		// Pull image if name is provided
		if m.pullImageName != "" {
			m.currentView = viewModeList
			m.actionInProgress = true
			m.statusMessage = fmt.Sprintf("Pulling image %s...", m.pullImageName)
			return pullImage(m.dockerClient, m.pullImageName)
		}
		return nil

	case "backspace":
		// Delete character from image name
		if len(m.pullImageName) > 0 {
			m.pullImageName = m.pullImageName[:len(m.pullImageName)-1]
		}

	default:
		// Add character to image name (only if it's a single character)
		if len(key) == 1 {
			m.pullImageName += key
		}
	}

	return nil
}

// Get max row count for current tab (accounting for filters)
func (m model) getMaxRow() int {
	switch m.activeTab {
	case 0:
		filteredContainers := filterContainers(m.containers, m.containerFilter)
		return len(filteredContainers)
	case 1:
		filteredImages := filterImages(m.images, m.containers, m.imageFilter)
		return len(filteredImages)
	case 2:
		filteredVolumes := filterVolumes(m.volumes, m.containers, m.dockerClient)
		return len(filteredVolumes)
	case 3:
		filteredNetworks := filterNetworks(m.networks, m.containers, m.dockerClient)
		return len(filteredNetworks)
	}
	return 0
}

// Render tabs with new visual style
func (m model) renderTabs() string {
	var b strings.Builder

	tabs := []struct {
		name     string
		shortcut string
	}{
		{"Containers", "^D"},
		{"Images", "^I"},
		{"Volumes", "^V"},
		{"Networks", "^N"},
	}

	// Calculate tab widths: space + name + space + ^X + space = name.len + 5
	tabWidths := make([]int, len(tabs))
	for i, tab := range tabs {
		tabWidths[i] = 1 + len(tab.name) + 1 + len(tab.shortcut) + 1 // " Name ^X "
	}

	// Top line with rounded corners
	b.WriteString(" ")
	for _, width := range tabWidths {
		b.WriteString(greenStyle.Render("╭"))
		b.WriteString(greenStyle.Render(strings.Repeat("─", width)))
		b.WriteString(greenStyle.Render("╮"))
	}
	b.WriteString("\n")

	// Tab labels: │ space name space ^X space │
	b.WriteString(" ")
	for i, tab := range tabs {
		b.WriteString(greenStyle.Render("│"))
		content := fmt.Sprintf(" %s %s ", tab.name, tab.shortcut)
		if i == m.activeTab {
			b.WriteString(yellowStyle.Render(content))
		} else {
			b.WriteString(greenStyle.Render(content))
		}
		b.WriteString(greenStyle.Render("│"))
	}
	b.WriteString("\n")

	// Bottom line - active tab connects to content
	b.WriteString(greenStyle.Render("─"))
	for i, width := range tabWidths {
		if i == m.activeTab {
			// Active tab connects to bottom line
			b.WriteString(greenStyle.Render("╯"))
			b.WriteString(strings.Repeat(" ", width))
			b.WriteString(greenStyle.Render("╰"))
		} else {
			// Inactive tab disconnects
			b.WriteString(greenStyle.Render("┴"))
			b.WriteString(greenStyle.Render(strings.Repeat("─", width)))
			b.WriteString(greenStyle.Render("┴"))
		}
	}

	// Calculate remaining space and extend line to edge
	totalTabWidth := 1 // Starting dash
	for _, width := range tabWidths {
		totalTabWidth += width + 2 // +2 for left and right borders (╯/╰ or ┴/┴)
	}
	remaining := 85 - totalTabWidth
	b.WriteString(greenStyle.Render(strings.Repeat("─", remaining)))
	b.WriteString("\n")

	return b.String()
}

// Get visible items range for current viewport
func (m model) getVisibleRange() (start, end int) {
	total := m.getMaxRow()
	start = m.scrollOffset
	end = m.scrollOffset + m.viewportHeight

	// Clamp to actual item count
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}

	return start, end
}

// Add filter indicator to tab bar
func (m model) addFilterIndicator(tabsView string, filterName string, width int) string {
	lines := strings.Split(tabsView, "\n")
	if len(lines) < 2 {
		return tabsView
	}

	// Style for filter indicator (no background to inherit from tab bar)
	filterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC"))

	fStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Underline(true)

	// Build filter text: ≡ Filter: {selection}
	filterText := "≡ " + fStyle.Render("F") + filterStyle.Render("ilter: " + filterName)

	// Calculate position for second line (tab labels row)
	labelLine := lines[1]
	// Strip ANSI codes to get actual length
	labelLineClean := stripAnsi(labelLine)
	currentLen := len(labelLineClean)

	// Calculate spaces needed to push filter to the right (leave 1 space padding from edge)
	filterTextClean := "≡ Filter: " + filterName
	spacesNeeded := width - currentLen - len(filterTextClean) - 1

	if spacesNeeded > 0 {
		lines[1] = labelLine + strings.Repeat(" ", spacesNeeded) + filterText
	}

	return strings.Join(lines, "\n")
}

// Render keyboard shortcut with underscored first letter in white
func renderShortcut(key string) string {
	if len(key) == 0 {
		return key
	}

	firstLetterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Underline(true)

	restStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC"))

	firstLetter := string(key[0])
	rest := ""
	if len(key) > 1 {
		rest = key[1:]
	}

	return firstLetterStyle.Render(firstLetter) + restStyle.Render(rest)
}

// Strip ANSI escape codes for length calculation
func stripAnsi(str string) string {
	result := str
	// Remove any ANSI escape sequences
	for strings.Contains(result, "\x1b[") {
		start := strings.Index(result, "\x1b[")
		end := strings.Index(result[start:], "m")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return result
}

// Get scroll position indicator string
func (m model) getScrollIndicator() string {
	total := m.getMaxRow()
	if total == 0 {
		return ""
	}

	start, end := m.getVisibleRange()
	if total <= m.viewportHeight {
		return "" // No scrolling needed
	}

	return fmt.Sprintf(" [%d-%d of %d]", start+1, end, total)
}

func (m model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	// Show error if Docker connection failed
	if m.err != nil {
		return m.renderError()
	}

	// Check current view mode
	switch m.currentView {
	case viewModeLogs:
		return m.renderLogs()
	case viewModeInspect:
		return m.renderInspect()
	case viewModePortSelector:
		return m.renderPortSelector()
	case viewModeStopConfirm:
		return m.renderStopConfirm()
	case viewModeFilter:
		return m.renderFilterModal()
	case viewModeRunImage:
		return m.renderRunImageModal()
	case viewModePullImage:
		return m.renderPullImageModal()
	case viewModeDeleteConfirm:
		return m.renderDeleteConfirm()
	}

	// Render based on active tab (list view)
	switch m.activeTab {
	case 0:
		return m.renderContainers()
	case 1:
		return m.renderImages()
	case 2:
		return m.renderVolumes()
	case 3:
		return m.renderNetworks()
	}

	return ""
}

func (m model) renderContainers() string {
	var b strings.Builder

	// Ensure minimum width
	width := m.width
	if width < 60 {
		width = 60
	}

	// Header component with responsive width
	header := m.header.WithWidth(width)
	b.WriteString(header.View())

	// Tabs component with responsive width
	tabs := m.tabs.SetActiveTab(m.activeTab).WithWidth(width)
	tabsView := tabs.View()

	// Add filter indicator (always visible)
	filterName := "All"
	if m.containerFilter == containerFilterRunning {
		filterName = "Running"
	}
	tabsView = m.addFilterIndicator(tabsView, filterName, width)
	b.WriteString(tabsView)

	// Apply filter to containers
	filteredContainers := filterContainers(m.containers, m.containerFilter)

	// Status line component with responsive width
	runningCount := 0
	for _, c := range m.containers {
		if c.Status == "RUNNING" {
			runningCount++
		}
	}
	statusLabel := fmt.Sprintf("CONTAINERS (%d total, %d running)", len(m.containers), runningCount)
	statusComp := NewStatusLineComponent(statusLabel, len(filteredContainers)).WithWidth(width)
	statusComp = statusComp.SetScrollIndicator(m.getScrollIndicator())
	b.WriteString(statusComp.View())

	// Table component with fixed and fill columns
	// Layout: [empty] [dot] [empty] [name-fill] [empty] [image-fill] [CPU-4] [MEM-10] [PORTS-15]
	emptyWidth := 1
	dotWidth := 1
	cpuWidth := 4
	memWidth := 10
	portsWidth := 15

	// Calculate separators between columns (2 spaces each)
	numSeparators := 8 // Between 9 columns
	separatorWidth := numSeparators * 2

	// Calculate fixed width used
	fixedWidth := (emptyWidth * 3) + dotWidth + cpuWidth + memWidth + portsWidth + separatorWidth

	// Remaining width for fill columns (name and image)
	remainingWidth := width - fixedWidth
	if remainingWidth < 20 {
		remainingWidth = 20 // Minimum for fill columns
	}

	// Split remaining width between name and image (fill mode)
	nameWidth := remainingWidth / 2
	imageWidth := remainingWidth - nameWidth

	headers := []TableHeader{
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "", Width: dotWidth, AlignRight: false},
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "Name", Width: nameWidth, AlignRight: false},
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "Image", Width: imageWidth, AlignRight: false},
		{Label: "CPU", Width: cpuWidth, AlignRight: true},
		{Label: "MEM", Width: memWidth, AlignRight: true},
		{Label: "PORTS", Width: portsWidth, AlignRight: false},
	}

	table := NewTableComponent(headers).WithWidth(width)

	if !m.loading && len(filteredContainers) > 0 {
		var rows []TableRow
		for i, container := range filteredContainers {
			isStopped := container.Status == "STOPPED"
			rowStyle := normalStyle
			if isStopped {
				rowStyle = grayStyle
			}

			// Status dot
			statusDot := getStatusDot(container.Status)

			// Only truncate if content exceeds column width (fill columns handle naturally)
			nameCell := container.Name
			if len(container.Name) > nameWidth {
				nameCell = truncateWithEllipsis(container.Name, nameWidth)
			}

			imageCell := container.Image
			if len(container.Image) > imageWidth {
				imageCell = truncateWithEllipsis(container.Image, imageWidth)
			}

			cpuCell := container.CPU
			if len(container.CPU) > cpuWidth {
				cpuCell = truncateWithEllipsis(container.CPU, cpuWidth)
			}

			memCell := container.Mem
			if len(container.Mem) > memWidth {
				memCell = truncateWithEllipsis(container.Mem, memWidth)
			}

			portsCell := container.Ports
			if len(container.Ports) > portsWidth {
				portsCell = truncateWithEllipsis(container.Ports, portsWidth)
			}

			rows = append(rows, TableRow{
				Cells: []string{
					"",           // Empty column
					statusDot,    // Status dot
					"",           // Empty column
					nameCell,     // Container name (fill)
					"",           // Empty column
					imageCell,    // Image name (fill)
					cpuCell,      // CPU (4 columns)
					memCell,      // MEM (10 columns)
					portsCell,    // PORTS (15 columns)
				},
				IsSelected: i == m.selectedRow,
				Style:      rowStyle,
			})
		}
		table = table.SetRows(rows)
	}

	start, end := m.getVisibleRange()
	table = table.SetVisibleRange(start, end)
	b.WriteString(table.View())

	// Action bar component
	m.actionBar = m.actionBar.SetStatusMessage(m.statusMessage)
	if m.statusMessage == "" && len(filteredContainers) > 0 && m.selectedRow < len(filteredContainers) {
		selectedContainer := filteredContainers[m.selectedRow]
		var actions string
		if selectedContainer.Status == "RUNNING" {
			actions = " " + renderShortcut("Stop") + " | " + renderShortcut("Restart") + " | " + renderShortcut("Console")
		} else if selectedContainer.Status == "STOPPED" {
			actions = " " + renderShortcut("Start")
		}
		if selectedContainer.Ports != "" && selectedContainer.Ports != "--" {
			actions += " | " + renderShortcut("Open")
		}
		actions += " | " + renderShortcut("Logs") + " | " + renderShortcut("Inspect") + " | " + renderShortcut("Delete")
		m.actionBar = m.actionBar.SetActions(actions)
	} else {
		m.actionBar = m.actionBar.SetActions("")
	}
	actionBar := m.actionBar.WithWidth(width)
	b.WriteString(actionBar.View())

	return containerStyle.Render(b.String())
}

func (m model) renderImages() string {
	var b strings.Builder

	// Ensure minimum width
	width := m.width
	if width < 60 {
		width = 60
	}

	// Header component with responsive width
	header := m.header.WithWidth(width)
	b.WriteString(header.View())

	// Tabs component with responsive width
	tabs := m.tabs.SetActiveTab(m.activeTab).WithWidth(width)
	tabsView := tabs.View()

	// Apply filter to images
	filteredImages := filterImages(m.images, m.containers, m.imageFilter)

	// Add filter indicator (always visible)
	filterName := "All"
	switch m.imageFilter {
	case imageFilterInUse:
		filterName = "In Use"
	case imageFilterUnused:
		filterName = "Unused"
	case imageFilterDangling:
		filterName = "Dangling"
	}
	tabsView = m.addFilterIndicator(tabsView, filterName, width)
	b.WriteString(tabsView)

	// Status line component with responsive width
	statusLabel := fmt.Sprintf("IMAGES (%d total)", len(m.images))
	statusComp := NewStatusLineComponent(statusLabel, len(filteredImages)).WithWidth(width)
	statusComp = statusComp.SetScrollIndicator(m.getScrollIndicator())
	b.WriteString(statusComp.View())

	// Table component with fixed and fill columns
	// Layout: [empty] [dot] [empty] [repository-fill] [empty] [tag-12] [size-8] [created-10]
	emptyWidth := 1
	dotWidth := 1
	tagWidth := 12
	sizeWidth := 8
	createdWidth := 10

	// Calculate separators (7 separators * 2 spaces)
	numSeparators := 7
	separatorWidth := numSeparators * 2

	// Calculate fixed width used
	fixedWidth := (emptyWidth * 3) + dotWidth + tagWidth + sizeWidth + createdWidth + separatorWidth

	// Remaining width for fill column (repository)
	remainingWidth := width - fixedWidth
	if remainingWidth < 10 {
		remainingWidth = 10
	}
	repoWidth := remainingWidth

	headers := []TableHeader{
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "", Width: dotWidth, AlignRight: false},
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "Repository", Width: repoWidth, AlignRight: false},
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "Tag", Width: tagWidth, AlignRight: false},
		{Label: "Size", Width: sizeWidth, AlignRight: true},
		{Label: "Created", Width: createdWidth, AlignRight: false},
	}

	table := NewTableComponent(headers).WithWidth(width)

	if !m.loading && len(filteredImages) > 0 {
		var rows []TableRow
		for i, image := range filteredImages {
			isSelected := i == m.selectedRow

			// Status dot - green if in use, yellow if dangling, gray if unused
			var statusDot string
			if image.InUse {
				statusDot = greenStyle.Render("●")
			} else if image.Dangling {
				statusDot = yellowStyle.Render("●")
			} else {
				statusDot = grayStyle.Render("●")
			}

			// Truncate if needed
			repoCell := image.Repository
			if len(image.Repository) > repoWidth {
				repoCell = truncateWithEllipsis(image.Repository, repoWidth)
			}

			tagCell := image.Tag
			if len(image.Tag) > tagWidth {
				tagCell = truncateWithEllipsis(image.Tag, tagWidth)
			}

			sizeCell := image.Size
			if len(image.Size) > sizeWidth {
				sizeCell = truncateWithEllipsis(image.Size, sizeWidth)
			}

			createdCell := image.Created
			if len(image.Created) > createdWidth {
				createdCell = truncateWithEllipsis(image.Created, createdWidth)
			}

			cells := []string{
				"",          // Empty column
				statusDot,   // Status dot
				"",          // Empty column
				repoCell,    // Repository (fill)
				"",          // Empty column
				tagCell,     // Tag (12 columns)
				sizeCell,    // Size (8 columns)
				createdCell, // Created (10 columns)
			}

			rows = append(rows, TableRow{
				Cells:      cells,
				IsSelected: isSelected,
				Style:      normalStyle,
			})
		}
		table = table.SetRows(rows)
	}

	start, end := m.getVisibleRange()
	table = table.SetVisibleRange(start, end)
	b.WriteString(table.View())

	// Action bar component with responsive width
	m.actionBar = m.actionBar.SetStatusMessage(m.statusMessage)
	if m.statusMessage == "" && len(filteredImages) > 0 {
		actions := " " + renderShortcut("Run") + " | " + renderShortcut("Inspect") + " | " + renderShortcut("Delete") + " | " + renderShortcut("Pull") + " | " + renderShortcut("Filter")
		m.actionBar = m.actionBar.SetActions(actions)
	} else {
		m.actionBar = m.actionBar.SetActions("")
	}
	actionBar := m.actionBar.WithWidth(width)
	b.WriteString(actionBar.View())

	return containerStyle.Render(b.String())
}

func (m model) renderVolumes() string {
	var b strings.Builder

	// Ensure minimum width
	width := m.width
	if width < 60 {
		width = 60
	}

	// Header component with responsive width
	header := m.header.WithWidth(width)
	b.WriteString(header.View())

	// Tabs component with responsive width
	tabs := m.tabs.SetActiveTab(m.activeTab).WithWidth(width)
	tabsView := tabs.View()

	// Apply filter to volumes
	filteredVolumes := filterVolumes(m.volumes, m.containers, m.dockerClient)

	// Add filter indicator (always visible)
	filterName := "All"
	switch m.volumeFilter {
	case volumeFilterInUse:
		filterName = "In Use"
	case volumeFilterUnused:
		filterName = "Unused"
	}
	tabsView = m.addFilterIndicator(tabsView, filterName, width)
	b.WriteString(tabsView)

	// Status line component with responsive width
	statusLabel := fmt.Sprintf("VOLUMES (%d total)", len(m.volumes))
	statusComp := NewStatusLineComponent(statusLabel, len(filteredVolumes)).WithWidth(width)
	statusComp = statusComp.SetScrollIndicator(m.getScrollIndicator())
	b.WriteString(statusComp.View())

	// Table component with fixed and fill columns
	// Layout: [empty] [dot] [empty] [name-fill] [empty] [driver-8] [container-fill] [created-10]
	emptyWidth := 1
	dotWidth := 1
	driverWidth := 8
	createdWidth := 10

	// Calculate separators (8 separators * 2 spaces)
	numSeparators := 8
	separatorWidth := numSeparators * 2

	// Calculate fixed width used
	fixedWidth := (emptyWidth * 4) + dotWidth + driverWidth + createdWidth + separatorWidth

	// Remaining width for fill columns (name and containers)
	remainingWidth := width - fixedWidth
	if remainingWidth < 20 {
		remainingWidth = 20
	}

	// Split remaining between name and containers (60% name, 40% containers)
	nameWidth := (remainingWidth * 6) / 10
	containerWidth := remainingWidth - nameWidth

	headers := []TableHeader{
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "", Width: dotWidth, AlignRight: false},
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "Name", Width: nameWidth, AlignRight: false},
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "Driver", Width: driverWidth, AlignRight: false},
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "Container", Width: containerWidth, AlignRight: false},
		{Label: "Created", Width: createdWidth, AlignRight: false},
	}

	table := NewTableComponent(headers).WithWidth(width)

	if !m.loading && len(filteredVolumes) > 0 {
		var rows []TableRow
		for i, volume := range filteredVolumes {
			isSelected := i == m.selectedRow

			// Status dot - green if in use, gray if unused
			var statusDot string
			if volume.InUse {
				statusDot = greenStyle.Render("●")
			} else {
				statusDot = grayStyle.Render("●")
			}

			// Truncate if needed
			nameCell := volume.Name
			if len(volume.Name) > nameWidth {
				nameCell = truncateWithEllipsis(volume.Name, nameWidth)
			}

			driverCell := volume.Driver
			if len(volume.Driver) > driverWidth {
				driverCell = truncateWithEllipsis(volume.Driver, driverWidth)
			}

			containerCell := volume.Containers
			if len(volume.Containers) > containerWidth {
				containerCell = truncateWithEllipsis(volume.Containers, containerWidth)
			}

			createdCell := volume.Created
			if len(volume.Created) > createdWidth {
				createdCell = truncateWithEllipsis(volume.Created, createdWidth)
			}

			cells := []string{
				"",             // Empty column
				statusDot,      // Status dot
				"",             // Empty column
				nameCell,       // Name (fill)
				"",             // Empty column
				driverCell,     // Driver (8 columns)
				"",             // Empty column
				containerCell,  // Container (fill)
				createdCell,    // Created (10 columns)
			}

			rows = append(rows, TableRow{
				Cells:      cells,
				IsSelected: isSelected,
				Style:      normalStyle,
			})
		}
		table = table.SetRows(rows)
	}

	start, end := m.getVisibleRange()
	table = table.SetVisibleRange(start, end)
	b.WriteString(table.View())

	// Action bar component with responsive width
	m.actionBar = m.actionBar.SetStatusMessage(m.statusMessage)
	if m.statusMessage == "" && len(filteredVolumes) > 0 {
		actions := " " + renderShortcut("Inspect") + " | " + renderShortcut("Delete") + " | " + renderShortcut("Filter")
		m.actionBar = m.actionBar.SetActions(actions)
	} else {
		m.actionBar = m.actionBar.SetActions("")
	}
	actionBar := m.actionBar.WithWidth(width)
	b.WriteString(actionBar.View())

	return containerStyle.Render(b.String())
}

func (m model) renderNetworks() string {
	var b strings.Builder

	// Ensure minimum width
	width := m.width
	if width < 60 {
		width = 60
	}

	// Header component with responsive width
	header := m.header.WithWidth(width)
	b.WriteString(header.View())

	// Tabs component with responsive width
	tabs := m.tabs.SetActiveTab(m.activeTab).WithWidth(width)
	tabsView := tabs.View()

	// Apply filter to networks
	filteredNetworks := filterNetworks(m.networks, m.containers, m.dockerClient)

	// Add filter indicator (always visible)
	filterName := "All"
	switch m.networkFilter {
	case networkFilterInUse:
		filterName = "In Use"
	case networkFilterUnused:
		filterName = "Unused"
	}
	tabsView = m.addFilterIndicator(tabsView, filterName, width)
	b.WriteString(tabsView)

	// Status line component with responsive width
	statusLabel := fmt.Sprintf("NETWORKS (%d total)", len(m.networks))
	statusComp := NewStatusLineComponent(statusLabel, len(filteredNetworks)).WithWidth(width)
	statusComp = statusComp.SetScrollIndicator(m.getScrollIndicator())
	b.WriteString(statusComp.View())

	// Table component with fixed and fill columns
	// Layout: [empty] [dot] [empty] [name-fill] [empty] [driver-8] [scope-8] [ipv4-18] [ipv6-18]
	emptyWidth := 1
	dotWidth := 1
	driverWidth := 8
	scopeWidth := 8
	ipv4Width := 18
	ipv6Width := 18

	// Calculate separators (8 separators * 2 spaces)
	numSeparators := 8
	separatorWidth := numSeparators * 2

	// Calculate fixed width used
	fixedWidth := (emptyWidth * 3) + dotWidth + driverWidth + scopeWidth + ipv4Width + ipv6Width + separatorWidth

	// Remaining width for fill column (name)
	remainingWidth := width - fixedWidth
	if remainingWidth < 10 {
		remainingWidth = 10
	}
	nameWidth := remainingWidth

	headers := []TableHeader{
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "", Width: dotWidth, AlignRight: false},
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "Name", Width: nameWidth, AlignRight: false},
		{Label: "", Width: emptyWidth, AlignRight: false},
		{Label: "Driver", Width: driverWidth, AlignRight: false},
		{Label: "Scope", Width: scopeWidth, AlignRight: false},
		{Label: "IPv4", Width: ipv4Width, AlignRight: false},
		{Label: "IPv6", Width: ipv6Width, AlignRight: false},
	}

	table := NewTableComponent(headers).WithWidth(width)

	if !m.loading && len(filteredNetworks) > 0 {
		var rows []TableRow
		for i, network := range filteredNetworks {
			isSelected := i == m.selectedRow

			// Status dot - green if in use, gray if unused
			var statusDot string
			if network.InUse {
				statusDot = greenStyle.Render("●")
			} else {
				statusDot = grayStyle.Render("●")
			}

			// Truncate if needed
			nameCell := network.Name
			if len(network.Name) > nameWidth {
				nameCell = truncateWithEllipsis(network.Name, nameWidth)
			}

			driverCell := network.Driver
			if len(network.Driver) > driverWidth {
				driverCell = truncateWithEllipsis(network.Driver, driverWidth)
			}

			scopeCell := network.Scope
			if len(network.Scope) > scopeWidth {
				scopeCell = truncateWithEllipsis(network.Scope, scopeWidth)
			}

			ipv4Cell := network.IPv4
			if len(network.IPv4) > ipv4Width {
				ipv4Cell = truncateWithEllipsis(network.IPv4, ipv4Width)
			}

			ipv6Cell := network.IPv6
			if len(network.IPv6) > ipv6Width {
				ipv6Cell = truncateWithEllipsis(network.IPv6, ipv6Width)
			}

			cells := []string{
				"",          // Empty column
				statusDot,   // Status dot
				"",          // Empty column
				nameCell,    // Name (fill)
				"",          // Empty column
				driverCell,  // Driver (8 columns)
				scopeCell,   // Scope (8 columns)
				ipv4Cell,    // IPv4 (18 columns)
				ipv6Cell,    // IPv6 (18 columns)
			}

			rows = append(rows, TableRow{
				Cells:      cells,
				IsSelected: isSelected,
				Style:      normalStyle,
			})
		}
		table = table.SetRows(rows)
	}

	start, end := m.getVisibleRange()
	table = table.SetVisibleRange(start, end)
	b.WriteString(table.View())

	// Action bar component with responsive width
	m.actionBar = m.actionBar.SetStatusMessage(m.statusMessage)
	if m.statusMessage == "" && len(filteredNetworks) > 0 {
		actions := " " + renderShortcut("Delete") + " | " + renderShortcut("Filter")
		m.actionBar = m.actionBar.SetActions(actions)
	} else {
		m.actionBar = m.actionBar.SetActions("")
	}
	actionBar := m.actionBar.WithWidth(width)
	b.WriteString(actionBar.View())

	return containerStyle.Render(b.String())
}

func (m model) renderLogs() string {
	var b strings.Builder

	containerName := "Container"
	if m.selectedContainer != nil {
		containerName = m.selectedContainer.Name
	}

	width := m.width
	if width < 60 {
		width = 60
	}

	// Header component with responsive width
	header := m.header.WithWidth(width)
	b.WriteString(header.View())

	// Tabs component with responsive width
	tabs := m.tabs.SetActiveTab(m.activeTab).WithWidth(width)
	b.WriteString(tabs.View())

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	searchButtonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF"))

	searchInputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#303030"))

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	// Header with search
	titleText := fmt.Sprintf("Logs: %s", containerName)
	var searchText string
	if m.logsSearchMode {
		searchText = " [" + searchInputStyle.Render("Search: "+m.logsSearchQuery+"█") + "]"
	} else {
		searchText = " " + searchButtonStyle.Render("[") + searchButtonStyle.Render("S") + helpStyle.Render("earch]")
	}
	headerRight := "[ESC] Back"
	headerSpacing := strings.Repeat(" ", width-len(titleText)-len(stripAnsiCodes(searchText))-len(headerRight))

	b.WriteString(titleStyle.Render(titleText))
	b.WriteString(searchText)
	b.WriteString(headerSpacing)
	b.WriteString(helpStyle.Render(headerRight))
	b.WriteString("\n")

	// Content divider
	b.WriteString(lineStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	// Filter logs if search is active
	logLines := strings.Split(m.logsContent, "\n")
	var filteredLines []string
	if m.logsSearchMode && m.logsSearchQuery != "" {
		query := strings.ToLower(m.logsSearchQuery)
		for _, line := range logLines {
			if strings.Contains(strings.ToLower(line), query) {
				filteredLines = append(filteredLines, line)
			}
		}
	} else {
		filteredLines = logLines
	}

	// Calculate available height accounting for all UI elements:
	// - header (0 lines, empty)
	// - tabs (3 lines)
	// - logs title + search (1 line)
	// - divider (1 line)
	// - action bar (2 lines)
	// Total UI overhead: ~7 lines
	availableLines := m.height - 7
	if availableLines < 5 {
		availableLines = 5
	}

	// Ensure scroll offset is within bounds
	maxScroll := len(filteredLines) - availableLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.logsScrollOffset > maxScroll {
		m.logsScrollOffset = maxScroll
	}

	// Display logs with scrolling
	end := m.logsScrollOffset + availableLines
	if end > len(filteredLines) {
		end = len(filteredLines)
	}

	if len(filteredLines) == 0 {
		if m.logsSearchMode && m.logsSearchQuery != "" {
			b.WriteString(helpStyle.Render(" No matching logs found\n"))
		} else {
			b.WriteString(helpStyle.Render(" No logs available\n"))
		}
	} else {
		for i := m.logsScrollOffset; i < end; i++ {
			line := filteredLines[i]
			if len(line) > width {
				line = line[:width-3] + "..."
			}
			b.WriteString(contentStyle.Render(line))
			b.WriteString("\n")
		}
	}

	return containerStyle.Render(b.String())
}

func (m model) renderInspect() string {
	// Determine what we're inspecting
	resourceName := "Resource"
	if m.selectedContainer != nil {
		resourceName = m.selectedContainer.Name
	} else if m.selectedImage != nil {
		if m.selectedImage.Repository != "<none>" {
			resourceName = fmt.Sprintf("%s:%s", m.selectedImage.Repository, m.selectedImage.Tag)
		} else {
			resourceName = m.selectedImage.ID[:12]
		}
	} else if m.selectedVolume != nil {
		resourceName = m.selectedVolume.Name
	}

	width := m.width
	if width < 60 {
		width = 60
	}

	title := fmt.Sprintf("Inspect: %s", resourceName)
	detailView := NewDetailViewComponent(title, 15).WithWidth(width)
	detailView = detailView.SetContent(m.inspectContent)
	detailView = detailView.SetScroll(0)

	return containerStyle.Render(detailView.View())
}

func (m model) renderPortSelector() string {
	// Use responsive dimensions
	width := m.width
	if width < 60 {
		width = 60
	}
	height := m.height
	if height < 20 {
		height = 20
	}

	// Render base view (containers list)
	baseView := m.renderContainers()

	containerName := "Container"
	if m.selectedContainer != nil {
		containerName = m.selectedContainer.Name
	}

	// Dim the base view
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	dimmedBase := dimStyle.Render(baseView)

	// Modal dimensions
	modalWidth := 50
	if modalWidth > width-10 {
		modalWidth = width - 10
	}

	// Build modal with box-drawing characters
	var modalContent strings.Builder

	borderColor := lipgloss.Color("#666666")
	textColor := lipgloss.Color("#CCCCCC")
	selectedColor := lipgloss.Color("#FFFFFF")

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	textStyle := lipgloss.NewStyle().Foreground(textColor)
	selectedStyle := lipgloss.NewStyle().Foreground(selectedColor).Bold(true)

	// Calculate inner width
	innerWidth := modalWidth - 4

	title := fmt.Sprintf(" Select Port - %s", containerName)
	if len(title) > innerWidth+1 {
		title = title[:innerWidth-2] + "..."
	}

	// Top border
	modalContent.WriteString(borderStyle.Render("╭" + strings.Repeat("─", innerWidth+2) + "╮") + "\n")

	// Title
	titlePadding := innerWidth - len(title) + 2
	modalContent.WriteString(borderStyle.Render("│") + textStyle.Render(title) + strings.Repeat(" ", titlePadding) + borderStyle.Render("│") + "\n")

	// Divider
	modalContent.WriteString(borderStyle.Render("├" + strings.Repeat("─", innerWidth+2) + "┤") + "\n")

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Port list with triangle indicator for single choice
	for i, port := range m.availablePorts {
		portLine := fmt.Sprintf("localhost:%s", port)
		var optionLine string
		if i == m.selectedPortIdx {
			// Selected option with triangle
			optionLine = " ▶ " + portLine
			optionText := selectedStyle.Render(optionLine)
			padding := innerWidth - len(optionLine) + 2
			modalContent.WriteString(borderStyle.Render("│") + optionText + strings.Repeat(" ", padding) + borderStyle.Render("│") + "\n")
		} else {
			// Unselected option with spaces
			optionLine = "   " + portLine
			optionText := textStyle.Render(optionLine)
			padding := innerWidth - len(optionLine) + 2
			modalContent.WriteString(borderStyle.Render("│") + optionText + strings.Repeat(" ", padding) + borderStyle.Render("│") + "\n")
		}
	}

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Footer with keyboard shortcuts
	footerText := " ↑/↓ navigate, " + renderShortcut("Enter") + " select-open, " + renderShortcut("Esc") + " exit"
	footerClean := " ↑/↓ navigate, Enter select-open, Esc exit"
	footerPadding := innerWidth - len(footerClean) + 2
	modalContent.WriteString(borderStyle.Render("│") + footerText + strings.Repeat(" ", footerPadding) + borderStyle.Render("│") + "\n")

	// Bottom border
	modalContent.WriteString(borderStyle.Render("╰" + strings.Repeat("─", innerWidth+2) + "╯"))

	modal := modalContent.String()

	// Create layers
	baseLayer := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(dimmedBase)

	modalPlaced := lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
	)

	// Composite the layers
	baseLines := strings.Split(baseLayer, "\n")
	modalLines := strings.Split(modalPlaced, "\n")

	var result strings.Builder
	for i := 0; i < len(baseLines) && i < len(modalLines); i++ {
		if strings.TrimSpace(modalLines[i]) != "" {
			result.WriteString(modalLines[i])
		} else {
			result.WriteString(baseLines[i])
		}
		result.WriteString("\n")
	}

	return result.String()
}

func (m model) renderFilterModal() string {
	// Use responsive dimensions
	width := m.width
	if width < 60 {
		width = 60
	}
	height := m.height
	if height < 20 {
		height = 20
	}

	// Render base view based on active tab
	var baseView string
	switch m.activeTab {
	case 0:
		baseView = m.renderContainers()
	case 1:
		baseView = m.renderImages()
	case 2:
		baseView = m.renderVolumes()
	case 3:
		baseView = m.renderNetworks()
	}

	// Dim the base view
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	dimmedBase := dimStyle.Render(baseView)

	// Modal dimensions
	modalWidth := 50
	if modalWidth > width-10 {
		modalWidth = width - 10
	}

	// Build modal with box-drawing characters
	var modalContent strings.Builder

	borderColor := lipgloss.Color("#666666")
	textColor := lipgloss.Color("#CCCCCC")
	selectedColor := lipgloss.Color("#FFFFFF")
	checkColor := lipgloss.Color("#00FF00") // Green checkmark

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	textStyle := lipgloss.NewStyle().Foreground(textColor)
	selectedStyle := lipgloss.NewStyle().Foreground(selectedColor).Bold(true)
	checkStyle := lipgloss.NewStyle().Foreground(checkColor)

	// Calculate inner width
	innerWidth := modalWidth - 4

	// Top border
	modalContent.WriteString(borderStyle.Render("╭" + strings.Repeat("─", innerWidth+2) + "╮") + "\n")

	// Title
	title := " Filter "
	titlePadding := innerWidth + 2 - len(title)
	modalContent.WriteString(borderStyle.Render("│") + textStyle.Render(title) + strings.Repeat(" ", titlePadding) + borderStyle.Render("│") + "\n")

	// Divider
	modalContent.WriteString(borderStyle.Render("├" + strings.Repeat("─", innerWidth+2) + "┤") + "\n")

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Filter options with checkbox style
	for i, option := range m.filterOptions {
		if i == m.selectedFilter {
			// Selected option with green checkbox
			checkbox := checkStyle.Render("[✓]")
			optionText := selectedStyle.Render(" " + option)
			// Calculate clean lengths for proper spacing
			cleanLen := 1 + 3 + 1 + len(option) // space + [✓] + space + option
			padding := innerWidth + 2 - cleanLen
			modalContent.WriteString(borderStyle.Render("│") + " " + checkbox + optionText + strings.Repeat(" ", padding) + borderStyle.Render("│") + "\n")
		} else {
			// Unselected option with empty checkbox
			checkbox := textStyle.Render("[ ]")
			optionText := textStyle.Render(" " + option)
			cleanLen := 1 + 3 + 1 + len(option) // space + [ ] + space + option
			padding := innerWidth + 2 - cleanLen
			modalContent.WriteString(borderStyle.Render("│") + " " + checkbox + optionText + strings.Repeat(" ", padding) + borderStyle.Render("│") + "\n")
		}
	}

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Footer with keyboard shortcuts
	footerText := " ↑/↓ navigate, " + renderShortcut("Enter") + " select-apply, " + renderShortcut("Esc") + " exit "
	footerClean := " ↑/↓ navigate, Enter select-apply, Esc exit "
	footerPadding := innerWidth + 2 - len(footerClean)
	modalContent.WriteString(borderStyle.Render("│") + footerText + strings.Repeat(" ", footerPadding) + borderStyle.Render("│") + "\n")

	// Bottom border
	modalContent.WriteString(borderStyle.Render("╰" + strings.Repeat("─", innerWidth+2) + "╯"))

	modal := modalContent.String()

	// Create layers using Lipgloss with responsive dimensions
	baseLayer := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(dimmedBase)

	modalPlaced := lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
	)

	// Composite the layers
	baseLines := strings.Split(baseLayer, "\n")
	modalLines := strings.Split(modalPlaced, "\n")

	var result strings.Builder
	for i := 0; i < len(baseLines) && i < len(modalLines); i++ {
		if strings.TrimSpace(modalLines[i]) != "" {
			result.WriteString(modalLines[i])
		} else {
			result.WriteString(baseLines[i])
		}
		result.WriteString("\n")
	}

	return result.String()
}

func (m model) renderStopConfirm() string {
	// Use responsive dimensions
	width := m.width
	if width < 60 {
		width = 60
	}
	height := m.height
	if height < 20 {
		height = 20
	}

	// Render base view (containers list)
	baseView := m.renderContainers()

	containerName := "Container"
	if m.selectedContainer != nil {
		containerName = m.selectedContainer.Name
	}

	// Dim the base view
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	dimmedBase := dimStyle.Render(baseView)

	// Modal dimensions
	modalWidth := 50
	if modalWidth > width-10 {
		modalWidth = width - 10
	}

	// Build modal with box-drawing characters
	var modalContent strings.Builder

	borderColor := lipgloss.Color("#666666")
	textColor := lipgloss.Color("#CCCCCC")
	warningColor := lipgloss.Color("#FFFFFF")
	subTextColor := lipgloss.Color("#999999")

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	textStyle := lipgloss.NewStyle().Foreground(textColor)
	warningStyle := lipgloss.NewStyle().Foreground(warningColor)
	subStyle := lipgloss.NewStyle().Foreground(subTextColor)

	// Calculate inner width
	innerWidth := modalWidth - 4

	// Top border
	modalContent.WriteString(borderStyle.Render("╭" + strings.Repeat("─", innerWidth+2) + "╮") + "\n")

	// Title
	title := " ⚠ Stop Container"
	titlePadding := innerWidth - len(title) + 2
	modalContent.WriteString(borderStyle.Render("│") + textStyle.Render(title) + strings.Repeat(" ", titlePadding) + borderStyle.Render("│") + "\n")

	// Divider
	modalContent.WriteString(borderStyle.Render("├" + strings.Repeat("─", innerWidth+2) + "┤") + "\n")

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Warning message
	warningText := fmt.Sprintf(" Stop container '%s'?", containerName)
	if len(warningText) > innerWidth+1 {
		warningText = warningText[:innerWidth-2] + "..."
	}
	warningPadding := innerWidth - len(warningText) + 2
	modalContent.WriteString(borderStyle.Render("│") + warningStyle.Render(warningText) + strings.Repeat(" ", warningPadding) + borderStyle.Render("│") + "\n")

	// Sub text
	subText := " This will stop the running container."
	subPadding := innerWidth - len(subText) + 2
	modalContent.WriteString(borderStyle.Render("│") + subStyle.Render(subText) + strings.Repeat(" ", subPadding) + borderStyle.Render("│") + "\n")

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Footer with keyboard shortcuts
	footerText := " " + renderShortcut("Enter") + " confirm-stop, " + renderShortcut("Esc") + " exit"
	footerClean := " Enter confirm-stop, Esc exit"
	footerPadding := innerWidth - len(footerClean) + 2
	modalContent.WriteString(borderStyle.Render("│") + footerText + strings.Repeat(" ", footerPadding) + borderStyle.Render("│") + "\n")

	// Bottom border
	modalContent.WriteString(borderStyle.Render("╰" + strings.Repeat("─", innerWidth+2) + "╯"))

	modal := modalContent.String()

	// Create layers
	baseLayer := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(dimmedBase)

	modalPlaced := lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
	)

	// Composite the layers
	baseLines := strings.Split(baseLayer, "\n")
	modalLines := strings.Split(modalPlaced, "\n")

	var result strings.Builder
	for i := 0; i < len(baseLines) && i < len(modalLines); i++ {
		if strings.TrimSpace(modalLines[i]) != "" {
			result.WriteString(modalLines[i])
		} else {
			result.WriteString(baseLines[i])
		}
		result.WriteString("\n")
	}

	return result.String()
}

func (m model) renderPullImageModal() string {
	// Use responsive dimensions
	width := m.width
	if width < 80 {
		width = 80
	}
	height := m.height
	if height < 30 {
		height = 30
	}

	// Render base view (images list)
	baseView := m.renderImages()

	// Dim the base view
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	dimmedBase := dimStyle.Render(baseView)

	// Modal dimensions
	modalWidth := 60
	if modalWidth > width-10 {
		modalWidth = width - 10
	}

	// Build modal with box-drawing characters
	var modalContent strings.Builder

	borderColor := lipgloss.Color("#666666")
	textColor := lipgloss.Color("#CCCCCC")
	labelColor := lipgloss.Color("#999999")
	inputColor := lipgloss.Color("#FFFFFF")

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	textStyle := lipgloss.NewStyle().Foreground(textColor)
	labelStyle := lipgloss.NewStyle().Foreground(labelColor)
	inputStyle := lipgloss.NewStyle().Foreground(inputColor)

	// Calculate inner width
	innerWidth := modalWidth - 4

	// Top border
	modalContent.WriteString(borderStyle.Render("╭" + strings.Repeat("─", innerWidth+2) + "╮") + "\n")

	// Title
	title := " Pull Docker Image "
	titlePadding := (innerWidth + 2 - len(title)) / 2
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", titlePadding) +
		textStyle.Render(title) + strings.Repeat(" ", innerWidth+2-titlePadding-len(title)) +
		borderStyle.Render("│") + "\n")

	// Separator
	modalContent.WriteString(borderStyle.Render("├" + strings.Repeat("─", innerWidth+2) + "┤") + "\n")

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Image name field label
	label := "  Image Name:"
	modalContent.WriteString(borderStyle.Render("│") + " " + labelStyle.Render(label) +
		strings.Repeat(" ", innerWidth+1-len(label)) + borderStyle.Render("│") + "\n")

	// Image name input field
	inputValue := m.pullImageName + "█" // Cursor
	if len(inputValue) > innerWidth-4 {
		inputValue = inputValue[:innerWidth-4]
	}
	modalContent.WriteString(borderStyle.Render("│") + "  " + inputStyle.Render(inputValue) +
		strings.Repeat(" ", innerWidth-len(inputValue)) + borderStyle.Render("│") + "\n")

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Help text
	helpText := "  Examples: nginx:latest, postgres:15, node:18-alpine"
	if len(helpText) > innerWidth+2 {
		helpText = helpText[:innerWidth-1] + "..."
	}
	modalContent.WriteString(borderStyle.Render("│") + " " + labelStyle.Render(helpText) +
		strings.Repeat(" ", innerWidth+1-len(helpText)) + borderStyle.Render("│") + "\n")

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Bottom border with controls
	modalContent.WriteString(borderStyle.Render("├" + strings.Repeat("─", innerWidth+2) + "┤") + "\n")

	// Controls
	controls := "  [Enter] Pull   [ESC] Cancel"
	modalContent.WriteString(borderStyle.Render("│") + " " + labelStyle.Render(controls) +
		strings.Repeat(" ", innerWidth+1-len(controls)) + borderStyle.Render("│") + "\n")

	// Bottom border
	modalContent.WriteString(borderStyle.Render("╰" + strings.Repeat("─", innerWidth+2) + "╯") + "\n")

	// Split into lines
	modalLines := strings.Split(modalContent.String(), "\n")

	// Calculate vertical position (center)
	modalHeight := len(modalLines)
	topPadding := (height - modalHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// Overlay modal on dimmed base
	baseLines := strings.Split(dimmedBase, "\n")
	var result strings.Builder

	for i := 0; i < height; i++ {
		if i >= topPadding && i < topPadding+modalHeight {
			modalLineIdx := i - topPadding
			if modalLineIdx < len(modalLines) {
				// Overlay modal on the base line
				leftPadding := (width - modalWidth) / 2
				if leftPadding < 0 {
					leftPadding = 0
				}

				if i < len(baseLines) {
					baseLine := baseLines[i]
					// Simply place modal at the calculated position, background is already dimmed
					if leftPadding > 0 && len(baseLine) > leftPadding {
						result.WriteString(baseLine[:leftPadding] + modalLines[modalLineIdx])
					} else {
						result.WriteString(strings.Repeat(" ", leftPadding) + modalLines[modalLineIdx])
					}
				} else {
					result.WriteString(strings.Repeat(" ", leftPadding) + modalLines[modalLineIdx])
				}
			} else {
				if i < len(baseLines) {
					result.WriteString(baseLines[i])
				}
			}
		} else if i < len(baseLines) {
			result.WriteString(baseLines[i])
		}
		result.WriteString("\n")
	}

	return result.String()
}

func (m model) renderDeleteConfirm() string {
	// Use responsive dimensions
	width := m.width
	if width < 60 {
		width = 60
	}
	height := m.height
	if height < 20 {
		height = 20
	}

	// Render base view based on active tab
	var baseView string
	switch m.activeTab {
	case 0:
		baseView = m.renderContainers()
	case 1:
		baseView = m.renderImages()
	case 2:
		baseView = m.renderVolumes()
	case 3:
		baseView = m.renderNetworks()
	}

	// Determine resource type and name
	var resourceType, resourceName string
	if m.selectedContainer != nil {
		resourceType = "Container"
		resourceName = m.selectedContainer.Name
	} else if m.selectedImage != nil {
		resourceType = "Image"
		resourceName = m.selectedImage.Repository + ":" + m.selectedImage.Tag
	} else if m.selectedVolume != nil {
		resourceType = "Volume"
		resourceName = m.selectedVolume.Name
	} else if m.selectedNetwork != nil {
		resourceType = "Network"
		resourceName = m.selectedNetwork.Name
	}

	// Dim the base view
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	dimmedBase := dimStyle.Render(baseView)

	// Modal dimensions
	modalWidth := 50
	if modalWidth > width-10 {
		modalWidth = width - 10
	}

	// Build modal with box-drawing characters
	var modalContent strings.Builder

	borderColor := lipgloss.Color("#666666")
	textColor := lipgloss.Color("#CCCCCC")
	warningColor := lipgloss.Color("#FFFFFF")
	subTextColor := lipgloss.Color("#999999")

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	textStyle := lipgloss.NewStyle().Foreground(textColor)
	warningStyle := lipgloss.NewStyle().Foreground(warningColor)
	subStyle := lipgloss.NewStyle().Foreground(subTextColor)

	// Calculate inner width
	innerWidth := modalWidth - 4

	// Top border
	modalContent.WriteString(borderStyle.Render("╭" + strings.Repeat("─", innerWidth+2) + "╮") + "\n")

	// Title
	title := " Delete " + resourceType + " "
	padding := strings.Repeat(" ", innerWidth+2-len(title))
	modalContent.WriteString(borderStyle.Render("│") + textStyle.Render(title) + padding + borderStyle.Render("│") + "\n")

	// Divider
	modalContent.WriteString(borderStyle.Render("├" + strings.Repeat("─", innerWidth+2) + "┤") + "\n")

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Warning message
	warning := " Are you sure you want to delete?"
	warningPadding := strings.Repeat(" ", innerWidth+2-len(warning))
	modalContent.WriteString(borderStyle.Render("│") + warningStyle.Render(warning) + warningPadding + borderStyle.Render("│") + "\n")

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Resource name
	resourceText := " " + resourceType + ": " + resourceName
	if len(resourceText) > innerWidth {
		resourceText = resourceText[:innerWidth-3] + "..."
	}
	resourcePadding := strings.Repeat(" ", innerWidth+2-len(resourceText))
	modalContent.WriteString(borderStyle.Render("│") + subStyle.Render(resourceText) + resourcePadding + borderStyle.Render("│") + "\n")

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Footer
	modalContent.WriteString(borderStyle.Render("├" + strings.Repeat("─", innerWidth+2) + "┤") + "\n")

	// Keyboard shortcuts
	footerText := " " + renderShortcut("Enter") + " confirm, " + renderShortcut("Esc") + " cancel"
	footerPadding := strings.Repeat(" ", innerWidth+2-len(stripAnsiCodes(footerText)))
	modalContent.WriteString(borderStyle.Render("│") + footerText + footerPadding + borderStyle.Render("│") + "\n")

	// Bottom border
	modalContent.WriteString(borderStyle.Render("╰" + strings.Repeat("─", innerWidth+2) + "╯") + "\n")

	// Overlay modal on base view
	modal := modalContent.String()
	modalLines := strings.Split(modal, "\n")

	baseLines := strings.Split(dimmedBase, "\n")
	var result strings.Builder

	// Center modal vertically and horizontally
	modalHeight := len(modalLines)
	verticalPadding := (height - modalHeight) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	for i := 0; i < len(baseLines); i++ {
		if i >= verticalPadding && i < verticalPadding+modalHeight && i < len(baseLines) {
			modalLineIdx := i - verticalPadding
			if modalLineIdx < len(modalLines) {
				// Overlay modal on the base line
				leftPadding := (width - modalWidth) / 2
				if leftPadding < 0 {
					leftPadding = 0
				}

				baseLine := baseLines[i]
				// Simply place modal at the calculated position, background is already dimmed
				if leftPadding > 0 && len(baseLine) > leftPadding {
					result.WriteString(baseLine[:leftPadding] + modalLines[modalLineIdx])
				} else {
					result.WriteString(strings.Repeat(" ", leftPadding) + modalLines[modalLineIdx])
				}
			} else {
				result.WriteString(baseLines[i])
			}
		} else if i < len(baseLines) {
			result.WriteString(baseLines[i])
		}
		result.WriteString("\n")
	}

	return result.String()
}

func (m model) renderRunImageModal() string {
	// Use responsive dimensions
	width := m.width
	if width < 80 {
		width = 80
	}
	height := m.height
	if height < 30 {
		height = 30
	}

	// Render base view (images list)
	baseView := m.renderImages()

	imageName := "Image"
	if m.selectedImage != nil {
		imageName = m.selectedImage.Repository + ":" + m.selectedImage.Tag
	}

	// Dim the base view
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	dimmedBase := dimStyle.Render(baseView)

	// Modal dimensions
	modalWidth := 70
	if modalWidth > width-10 {
		modalWidth = width - 10
	}

	// Build modal with box-drawing characters
	var modalContent strings.Builder

	borderColor := lipgloss.Color("#666666")
	textColor := lipgloss.Color("#CCCCCC")
	labelColor := lipgloss.Color("#999999")
	inputColor := lipgloss.Color("#FFFFFF")
	activeColor := lipgloss.Color("#00FF00") // Green for active field

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	textStyle := lipgloss.NewStyle().Foreground(textColor)
	labelStyle := lipgloss.NewStyle().Foreground(labelColor)
	inputStyle := lipgloss.NewStyle().Foreground(inputColor)
	activeStyle := lipgloss.NewStyle().Foreground(activeColor).Bold(true)

	// Calculate inner width
	innerWidth := modalWidth - 4

	// Top border
	modalContent.WriteString(borderStyle.Render("╭" + strings.Repeat("─", innerWidth+2) + "╮") + "\n")

	// Title
	title := " Run Container: " + imageName
	if len(title) > innerWidth+2 {
		title = title[:innerWidth-1] + "..."
	}
	padding := strings.Repeat(" ", innerWidth+2-len(title))
	modalContent.WriteString(borderStyle.Render("│") + textStyle.Render(title) + padding + borderStyle.Render("│") + "\n")

	// Divider
	modalContent.WriteString(borderStyle.Render("├" + strings.Repeat("─", innerWidth+2) + "┤") + "\n")

	// Container name
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")
	nameValue := m.runContainerName
	if m.runModalField == runFieldContainerName {
		nameValue += "█" // Cursor
	}
	nameLabel := " Container name: " + nameValue
	if len(nameLabel) > innerWidth {
		nameLabel = nameLabel[:innerWidth-3] + "..."
	}
	namePadding := strings.Repeat(" ", innerWidth+2-len(nameLabel))
	fieldStyle := labelStyle
	if m.runModalField == runFieldContainerName {
		fieldStyle = activeStyle
	}
	modalContent.WriteString(borderStyle.Render("│") + fieldStyle.Render(nameLabel) + namePadding + borderStyle.Render("│") + "\n")

	// Ports section
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")
	portsSectionLabel := " Ports:"
	portsSectionPadding := strings.Repeat(" ", innerWidth+2-len(portsSectionLabel))
	modalContent.WriteString(borderStyle.Render("│") + labelStyle.Render(portsSectionLabel) + portsSectionPadding + borderStyle.Render("│") + "\n")

	// Show existing ports
	if len(m.runPorts) > 0 {
		for _, port := range m.runPorts {
			portLine := "   " + port.Host + ":" + port.Container
			portLinePadding := strings.Repeat(" ", innerWidth+2-len(portLine))
			modalContent.WriteString(borderStyle.Render("│") + inputStyle.Render(portLine) + portLinePadding + borderStyle.Render("│") + "\n")
		}
	}

	// Add port inputs
	portHostValue := m.runPortHost
	if m.runModalField == runFieldPortHost {
		portHostValue += "█"
	}
	portHostLabel := "   Host: " + portHostValue
	portHostPadding := strings.Repeat(" ", innerWidth+2-len(portHostLabel))
	portHostStyle := labelStyle
	if m.runModalField == runFieldPortHost {
		portHostStyle = activeStyle
	}
	modalContent.WriteString(borderStyle.Render("│") + portHostStyle.Render(portHostLabel) + portHostPadding + borderStyle.Render("│") + "\n")

	portContainerValue := m.runPortContainer
	if m.runModalField == runFieldPortContainer {
		portContainerValue += "█"
	}
	portContainerLabel := "   Container: " + portContainerValue
	portContainerPadding := strings.Repeat(" ", innerWidth+2-len(portContainerLabel))
	portContainerStyle := labelStyle
	if m.runModalField == runFieldPortContainer {
		portContainerStyle = activeStyle
	}
	modalContent.WriteString(borderStyle.Render("│") + portContainerStyle.Render(portContainerLabel) + portContainerPadding + borderStyle.Render("│") + "\n")

	// Volumes section
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")
	volumesSectionLabel := " Volumes:"
	volumesSectionPadding := strings.Repeat(" ", innerWidth+2-len(volumesSectionLabel))
	modalContent.WriteString(borderStyle.Render("│") + labelStyle.Render(volumesSectionLabel) + volumesSectionPadding + borderStyle.Render("│") + "\n")

	// Show existing volumes
	if len(m.runVolumes) > 0 {
		for _, vol := range m.runVolumes {
			var volLine string
			if vol.IsNamed {
				volLine = "   " + vol.VolumeName + ":" + vol.Container
			} else {
				volLine = "   " + vol.Host + ":" + vol.Container
			}
			if len(volLine) > innerWidth {
				volLine = volLine[:innerWidth-3] + "..."
			}
			volLinePadding := strings.Repeat(" ", innerWidth+2-len(volLine))
			modalContent.WriteString(borderStyle.Render("│") + inputStyle.Render(volLine) + volLinePadding + borderStyle.Render("│") + "\n")
		}
	}

	// Add volume inputs
	volHostValue := m.runVolumeHost
	if m.runModalField == runFieldVolumeHost {
		volHostValue += "█"
	}
	volHostLabel := "   Host path: " + volHostValue
	if len(volHostLabel) > innerWidth {
		volHostLabel = volHostLabel[:innerWidth-3] + "..."
	}
	volHostPadding := strings.Repeat(" ", innerWidth+2-len(volHostLabel))
	volHostStyle := labelStyle
	if m.runModalField == runFieldVolumeHost {
		volHostStyle = activeStyle
	}
	modalContent.WriteString(borderStyle.Render("│") + volHostStyle.Render(volHostLabel) + volHostPadding + borderStyle.Render("│") + "\n")

	volContainerValue := m.runVolumeContainer
	if m.runModalField == runFieldVolumeContainer {
		volContainerValue += "█"
	}
	volContainerLabel := "   Container path: " + volContainerValue
	if len(volContainerLabel) > innerWidth {
		volContainerLabel = volContainerLabel[:innerWidth-3] + "..."
	}
	volContainerPadding := strings.Repeat(" ", innerWidth+2-len(volContainerLabel))
	volContainerStyle := labelStyle
	if m.runModalField == runFieldVolumeContainer {
		volContainerStyle = activeStyle
	}
	modalContent.WriteString(borderStyle.Render("│") + volContainerStyle.Render(volContainerLabel) + volContainerPadding + borderStyle.Render("│") + "\n")

	// Environment variables section
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")
	envSectionLabel := " Environment Variables:"
	envSectionPadding := strings.Repeat(" ", innerWidth+2-len(envSectionLabel))
	modalContent.WriteString(borderStyle.Render("│") + labelStyle.Render(envSectionLabel) + envSectionPadding + borderStyle.Render("│") + "\n")

	// Show existing env vars
	if len(m.runEnvVars) > 0 {
		for _, env := range m.runEnvVars {
			envLine := "   " + env.Key + "=" + env.Value
			if len(envLine) > innerWidth {
				envLine = envLine[:innerWidth-3] + "..."
			}
			envLinePadding := strings.Repeat(" ", innerWidth+2-len(envLine))
			modalContent.WriteString(borderStyle.Render("│") + inputStyle.Render(envLine) + envLinePadding + borderStyle.Render("│") + "\n")
		}
	}

	// Add env var inputs
	envKeyValue := m.runEnvKey
	if m.runModalField == runFieldEnvKey {
		envKeyValue += "█"
	}
	envKeyLabel := "   Key: " + envKeyValue
	envKeyPadding := strings.Repeat(" ", innerWidth+2-len(envKeyLabel))
	envKeyStyle := labelStyle
	if m.runModalField == runFieldEnvKey {
		envKeyStyle = activeStyle
	}
	modalContent.WriteString(borderStyle.Render("│") + envKeyStyle.Render(envKeyLabel) + envKeyPadding + borderStyle.Render("│") + "\n")

	envValueValue := m.runEnvValue
	if m.runModalField == runFieldEnvValue {
		envValueValue += "█"
	}
	envValueLabel := "   Value: " + envValueValue
	if len(envValueLabel) > innerWidth {
		envValueLabel = envValueLabel[:innerWidth-3] + "..."
	}
	envValuePadding := strings.Repeat(" ", innerWidth+2-len(envValueLabel))
	envValueStyle := labelStyle
	if m.runModalField == runFieldEnvValue {
		envValueStyle = activeStyle
	}
	modalContent.WriteString(borderStyle.Render("│") + envValueStyle.Render(envValueLabel) + envValuePadding + borderStyle.Render("│") + "\n")

	// Empty line
	modalContent.WriteString(borderStyle.Render("│") + strings.Repeat(" ", innerWidth+2) + borderStyle.Render("│") + "\n")

	// Footer
	modalContent.WriteString(borderStyle.Render("├" + strings.Repeat("─", innerWidth+2) + "┤") + "\n")

	// Keyboard shortcuts
	footerText := " " + renderShortcut("Tab") + " next, " + renderShortcut("Enter") + " add/run, " + renderShortcut("Esc") + " cancel"
	footerPadding := strings.Repeat(" ", innerWidth+2-len(stripAnsiCodes(footerText)))
	modalContent.WriteString(borderStyle.Render("│") + footerText + footerPadding + borderStyle.Render("│") + "\n")

	// Bottom border
	modalContent.WriteString(borderStyle.Render("╰" + strings.Repeat("─", innerWidth+2) + "╯") + "\n")

	// Overlay modal on base view
	modal := modalContent.String()
	modalLines := strings.Split(modal, "\n")

	baseLines := strings.Split(dimmedBase, "\n")
	var result strings.Builder

	// Center modal vertically and horizontally
	modalHeight := len(modalLines)
	verticalPadding := (height - modalHeight) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	for i := 0; i < len(baseLines); i++ {
		if i >= verticalPadding && i < verticalPadding+modalHeight {
			modalLineIdx := i - verticalPadding
			if modalLineIdx < len(modalLines) {
				// Center modal horizontally
				leftPadding := (width - modalWidth) / 2
				if leftPadding < 0 {
					leftPadding = 0
				}
				result.WriteString(strings.Repeat(" ", leftPadding) + modalLines[modalLineIdx])
			} else {
				result.WriteString(baseLines[i])
			}
		} else {
			result.WriteString(baseLines[i])
		}
		result.WriteString("\n")
	}

	return result.String()
}

func (m model) renderError() string {
	width := m.width
	if width < 60 {
		width = 60
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Background(lipgloss.Color("#0a0a0a")).
		Bold(true)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Background(lipgloss.Color("#0a0a0a"))

	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")).
		Background(lipgloss.Color("#0a0a0a"))

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#303030")).
		Background(lipgloss.Color("#0a0a0a"))

	title := "Docker TUI - Error"
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(lineStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n\n")

	errMsg := m.err.Error()
	maxErrLen := width - 10
	if len(errMsg) > maxErrLen {
		errMsg = errMsg[:maxErrLen-3] + "..."
	}
	errorLine := "Error: " + errMsg
	b.WriteString(errorStyle.Render(errorLine))
	b.WriteString("\n\n")

	troubleLine := "Troubleshooting:"
	b.WriteString(textStyle.Render(troubleLine))
	b.WriteString("\n")

	tip1 := "  - Make sure Docker is running"
	b.WriteString(textStyle.Render(tip1))
	b.WriteString("\n")

	tip2 := "  - Check Docker socket permissions"
	b.WriteString(textStyle.Render(tip2))
	b.WriteString("\n")

	tip3 := "  - Verify DOCKER_HOST environment variable"
	b.WriteString(textStyle.Render(tip3))
	b.WriteString("\n\n")

	quitLine := "Press 'q' to quit"
	b.WriteString(helpStyle.Render(quitLine))
	b.WriteString("\n")

	return containerStyle.Render(b.String())
}

func (m model) renderHelp() string {
	width := m.width
	if width < 60 {
		width = 60
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0a0a0a")).
		Bold(true)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0a0a0a"))

	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")).
		Background(lipgloss.Color("#0a0a0a"))

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#303030")).
		Background(lipgloss.Color("#0a0a0a"))

	// Helper function to render a line
	renderLine := func(text string, style lipgloss.Style) {
		if len(text) > width {
			text = text[:width]
		}
		b.WriteString(style.Render(text))
		b.WriteString("\n")
	}

	renderLine("Docker TUI - Help", titleStyle)
	b.WriteString(lineStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")
	renderLine("Navigation:", headerStyle)
	renderLine("  ↑/k      - Move selection up (auto-scrolls)", textStyle)
	renderLine("  ↓/j      - Move selection down (auto-scrolls)", textStyle)
	renderLine("  ←/→ or h/l - Switch tabs", textStyle)
	renderLine("  1-4 or ^D/^I/^V/^N - Jump to specific tab", textStyle)
	b.WriteString("\n")
	renderLine("Container Actions:", headerStyle)
	renderLine("  s        - Start/Stop selected container", textStyle)
	renderLine("  r        - Restart selected container", textStyle)
	renderLine("  c        - Open console (interactive shell, altscreen)", textStyle)
	renderLine("  o        - Open container port in browser", textStyle)
	renderLine("  l        - View container logs", textStyle)
	renderLine("  i        - Inspect (stats/image/mounts)", textStyle)
	renderLine("  Enter    - Refresh list", textStyle)
	renderLine("  ESC      - Return from detail views", textStyle)
	b.WriteString("\n")
	renderLine("Filtering:", headerStyle)
	renderLine("  f        - Filter items (All/Running/In Use/etc.)", textStyle)
	b.WriteString("\n")
	renderLine("Other:", headerStyle)
	renderLine("  F1       - Toggle help", textStyle)
	renderLine("  q/Ctrl+C - Quit application", textStyle)
	b.WriteString("\n")
	renderLine("Auto-refreshes every 5 seconds | Press F1 to return", helpStyle)

	return containerStyle.Render(b.String())
}

// Helper function to get status dot
func getStatusDot(status string) string {
	switch status {
	case "RUNNING":
		return greenStyle.Render("●")
	case "STOPPED":
		return grayStyle.Render("●")
	case "PAUSED":
		return yellowStyle.Render("●")
	case "ERROR":
		return redStyle.Render("●")
	default:
		return grayStyle.Render("○")
	}
}

// Filter helper functions

// Filter containers based on selected filter
func filterContainers(containers []Container, filter int) []Container {
	switch filter {
	case containerFilterRunning:
		var filtered []Container
		for _, c := range containers {
			if c.Status == "RUNNING" {
				filtered = append(filtered, c)
			}
		}
		return filtered
	default: // containerFilterAll
		return containers
	}
}

// Filter images based on selected filter
func filterImages(images []Image, containers []Container, filter int) []Image {
	switch filter {
	case imageFilterInUse:
		var filtered []Image
		for _, img := range images {
			if isImageInUse(img, containers) {
				filtered = append(filtered, img)
			}
		}
		return filtered
	case imageFilterUnused:
		var filtered []Image
		for _, img := range images {
			if !isImageInUse(img, containers) {
				filtered = append(filtered, img)
			}
		}
		return filtered
	case imageFilterDangling:
		var filtered []Image
		for _, img := range images {
			if img.Repository == "<none>" && img.Tag == "<none>" {
				filtered = append(filtered, img)
			}
		}
		return filtered
	default: // imageFilterAll
		return images
	}
}

// Filter volumes based on selected filter
func filterVolumes(volumes []Volume, containers []Container, client *client.Client) []Volume {
	// For now, return all volumes since we need to implement volume usage detection
	// TODO: Implement volume usage detection using container inspect
	return volumes
}

// Filter networks based on selected filter
func filterNetworks(networks []Network, containers []Container, client *client.Client) []Network {
	// For now, return all networks since we need to implement network usage detection
	// TODO: Implement network usage detection using container inspect
	return networks
}

// Helper function to check if an image is in use by any container
func isImageInUse(image Image, containers []Container) bool {
	imageFullName := image.Repository + ":" + image.Tag

	for _, container := range containers {
		containerImage := container.Image

		// Check exact match with full name
		if containerImage == imageFullName {
			return true
		}

		// Check if container image contains the repository:tag
		if strings.Contains(containerImage, imageFullName) {
			return true
		}

		// Check repository match (for images with implicit :latest tag)
		if containerImage == image.Repository {
			return true
		}

		// Check if container image ends with repository:tag (handles registry prefixes)
		if strings.HasSuffix(containerImage, "/"+imageFullName) {
			return true
		}

		// Check by image ID
		if strings.HasPrefix(container.ID, image.ID) || strings.HasPrefix(image.ID, container.ID) {
			return true
		}
	}
	return false
}

// Helper function to truncate text with ellipsis if too long
func truncateWithEllipsis(text string, maxWidth int) string {
	if maxWidth < 3 {
		return text
	}
	if len(text) <= maxWidth {
		return text
	}
	return text[:maxWidth-3] + "..."
}

// Helper functions for text alignment
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func padLeft(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return strings.Repeat(" ", width-len(s)) + s
}

func padCenter(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	leftPad := (width - len(s)) / 2
	rightPad := width - len(s) - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Program panicked: %v\n", r)
			os.Exit(1)
		}
	}()

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
