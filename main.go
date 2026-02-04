package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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
}

// Volume represents a Docker volume
type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Scope      string
	Created    string
}

// Network represents a Docker network
type Network struct {
	ID      string
	Name    string
	Driver  string
	Scope   string
	IPv4    string
	IPv6    string
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
}

// Color palette matching the Pencil design
var (
	green   = lipgloss.Color("#00FF00")
	yellow  = lipgloss.Color("#FFFF00")
	white   = lipgloss.Color("#FFFFFF")
	cyan    = lipgloss.Color("#00FFFF")
	red     = lipgloss.Color("#FF0000")
	gray    = lipgloss.Color("#808080")
	black   = lipgloss.Color("#000000")
)

// Styles
var (
	normalStyle = lipgloss.NewStyle().
			Foreground(white).
			Background(black)

	greenStyle = lipgloss.NewStyle().
			Foreground(green).
			Background(black)

	yellowStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Background(black).
			Bold(true)

	cyanStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Background(black)

	redStyle = lipgloss.NewStyle().
			Foreground(red).
			Background(black)

	grayStyle = lipgloss.NewStyle().
			Foreground(gray).
			Background(black)

	containerStyle = lipgloss.NewStyle().
			Background(black)
)

func initialModel() model {
	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

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

		return containerListMsg(displayContainers)
	}
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

			displayImages = append(displayImages, Image{
				ID:         imageID,
				Repository: repo,
				Tag:        tag,
				Size:       size,
				Created:    createdStr,
			})
		}

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

			displayVolumes = append(displayVolumes, Volume{
				Name:       name,
				Driver:     vol.Driver,
				Mountpoint: mountpoint,
				Scope:      vol.Scope,
				Created:    created,
			})
		}

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

			displayNetworks = append(displayNetworks, Network{
				ID:     networkID,
				Name:   name,
				Driver: net.Driver,
				Scope:  net.Scope,
				IPv4:   ipv4,
				IPv6:   ipv6,
			})
		}

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
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", url)
		default:
			return actionErrorMsg("Unsupported operating system")
		}

		if err := cmd.Start(); err != nil {
			return actionErrorMsg(fmt.Sprintf("Failed to open browser: %v", err))
		}

		return actionSuccessMsg(fmt.Sprintf("Opening %s in browser", url))
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't process keys if action is in progress
		if m.actionInProgress {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.dockerClient != nil {
				m.dockerClient.Close()
			}
			return m, tea.Quit
		case "up", "k":
			if m.selectedRow > 0 {
				m.selectedRow--
				m.statusMessage = "" // Clear status when navigating

				// Scroll up if needed
				if m.selectedRow < m.scrollOffset {
					m.scrollOffset = m.selectedRow
				}
			}
		case "down", "j":
			maxRow := m.getMaxRow()
			if m.selectedRow < maxRow-1 {
				m.selectedRow++
				m.statusMessage = "" // Clear status when navigating

				// Scroll down if needed
				if m.selectedRow >= m.scrollOffset+m.viewportHeight {
					m.scrollOffset = m.selectedRow - m.viewportHeight + 1
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
		case "right", "l":
			// Navigate to next tab
			if m.activeTab < 3 {
				m.activeTab++
				m.selectedRow = 0
				m.scrollOffset = 0
				m.statusMessage = ""
			}
		case "f1":
			m.showHelp = !m.showHelp
		case "r":
			// Restart only works on containers tab
			if m.activeTab == 0 && len(m.containers) > 0 && m.selectedRow < len(m.containers) {
				selectedContainer := m.containers[m.selectedRow]
				m.actionInProgress = true
				m.statusMessage = fmt.Sprintf("Restarting %s...", selectedContainer.Name)
				return m, restartContainer(m.dockerClient, selectedContainer.ID, selectedContainer.Name)
			}
		case "s":
			// Start/Stop only works on containers tab
			if m.activeTab == 0 && len(m.containers) > 0 && m.selectedRow < len(m.containers) {
				selectedContainer := m.containers[m.selectedRow]
				m.actionInProgress = true

				if selectedContainer.Status == "RUNNING" {
					m.statusMessage = fmt.Sprintf("Stopping %s...", selectedContainer.Name)
					return m, stopContainer(m.dockerClient, selectedContainer.ID, selectedContainer.Name)
				} else {
					m.statusMessage = fmt.Sprintf("Starting %s...", selectedContainer.Name)
					return m, startContainer(m.dockerClient, selectedContainer.ID, selectedContainer.Name)
				}
			}
		case "o":
			// Open browser only works on containers tab
			if m.activeTab == 0 && len(m.containers) > 0 && m.selectedRow < len(m.containers) {
				selectedContainer := m.containers[m.selectedRow]
				return m, openBrowser(selectedContainer.Ports)
			}
		case "enter":
			// Refresh current tab
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

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

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

// Get max row count for current tab
func (m model) getMaxRow() int {
	switch m.activeTab {
	case 0:
		return len(m.containers)
	case 1:
		return len(m.images)
	case 2:
		return len(m.volumes)
	case 3:
		return len(m.networks)
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

	// Render based on active tab
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

	// Top border
	b.WriteString(greenStyle.Render("┌─────────────────────────────────────────────────────────────────────────────────────┐"))
	b.WriteString("\n")

	// Header
	headerLeft := greenStyle.Render("│ Docker TUI v2.0.1")
	headerRight := greenStyle.Render("[F1] Help [Q]uit │")
	headerSpacing := strings.Repeat(" ", 85-len("│ Docker TUI v2.0.1")-len("[F1] Help [Q]uit │"))
	b.WriteString(headerLeft + headerSpacing + headerRight)
	b.WriteString("\n")

	// Tabs with new design
	b.WriteString(m.renderTabs())

	// Status line
	runningCount := 0
	for _, c := range m.containers {
		if c.Status == "RUNNING" {
			runningCount++
		}
	}
	statusText := fmt.Sprintf(" CONTAINERS (%d total, %d running)", len(m.containers), runningCount)
	scrollIndicator := m.getScrollIndicator()
	statusText += scrollIndicator
	statusLine := greenStyle.Render("│") + cyanStyle.Render(statusText)
	statusSpacing := strings.Repeat(" ", 85-len(statusText))
	b.WriteString(statusLine + statusSpacing + greenStyle.Render("│"))
	b.WriteString("\n")

	// Table divider
	b.WriteString(greenStyle.Render("├──────────────────────┬─────────┬──────┬──────┬─────────────────┬───────────────────┤"))
	b.WriteString("\n")

	// Table header
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" NAME                 "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" STATUS  "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" CPU% "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" MEM  "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" IMAGE           "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" PORTS               "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString("\n")

	// Header bottom divider
	b.WriteString(greenStyle.Render("├──────────────────────┼─────────┼──────┼──────┼─────────────────┼───────────────────┤"))
	b.WriteString("\n")

	// Table rows or loading/empty state
	if m.loading {
		// Show loading message
		b.WriteString(greenStyle.Render("│"))
		loadingMsg := " Loading containers..."
		b.WriteString(cyanStyle.Render(loadingMsg))
		b.WriteString(strings.Repeat(" ", 85-len(loadingMsg)))
		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	} else if len(m.containers) == 0 {
		// Show no containers message
		b.WriteString(greenStyle.Render("│"))
		noContainersMsg := " No containers found. Press 'r' to refresh."
		b.WriteString(cyanStyle.Render(noContainersMsg))
		b.WriteString(strings.Repeat(" ", 85-len(noContainersMsg)))
		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	}

	// Only render visible items
	start, end := m.getVisibleRange()
	for i := start; i < end && i < len(m.containers); i++ {
		container := m.containers[i]
		isSelected := i == m.selectedRow
		isStopped := container.Status == "STOPPED"

		// Row indicator and name
		b.WriteString(greenStyle.Render("│"))
		if isSelected {
			b.WriteString(yellowStyle.Render(">"))
			b.WriteString(yellowStyle.Render(padRight(container.Name, 21)))
		} else {
			b.WriteString(" ")
			if isStopped {
				b.WriteString(grayStyle.Render(padRight(container.Name, 21)))
			} else {
				b.WriteString(normalStyle.Render(padRight(container.Name, 21)))
			}
		}

		// Status
		b.WriteString(greenStyle.Render("│"))
		statusStyle := greenStyle
		if container.Status == "STOPPED" {
			statusStyle = redStyle
		}
		b.WriteString(statusStyle.Render(padCenter(container.Status, 9)))

		// CPU
		b.WriteString(greenStyle.Render("│"))
		cpuText := padCenter(container.CPU, 6)
		if isStopped {
			b.WriteString(grayStyle.Render(cpuText))
		} else {
			b.WriteString(normalStyle.Render(cpuText))
		}

		// Memory
		b.WriteString(greenStyle.Render("│"))
		memText := padCenter(container.Mem, 6)
		if isStopped {
			b.WriteString(grayStyle.Render(memText))
		} else {
			b.WriteString(normalStyle.Render(memText))
		}

		// Image
		b.WriteString(greenStyle.Render("│"))
		imageText := padRight(container.Image, 17)
		if isStopped {
			b.WriteString(grayStyle.Render(imageText))
		} else {
			b.WriteString(normalStyle.Render(imageText))
		}

		// Ports
		b.WriteString(greenStyle.Render("│"))
		portsText := padRight(container.Ports, 20)
		if isStopped {
			b.WriteString(grayStyle.Render(portsText))
		} else {
			b.WriteString(normalStyle.Render(portsText))
		}

		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	}

	// Table bottom border
	b.WriteString(greenStyle.Render("├──────────────────────┴─────────┴──────┴──────┴─────────────────┴───────────────────┤"))
	b.WriteString("\n")

	// Spacer row or action bar
	b.WriteString(greenStyle.Render("├─────────────────────────────────────────────────────────────────────────────────────┤"))
	b.WriteString("\n")

	// Action/status bar
	if m.statusMessage != "" {
		// Show status message
		statusStyle := cyanStyle
		if strings.HasPrefix(m.statusMessage, "ERROR:") {
			statusStyle = redStyle
		}
		msg := " " + m.statusMessage
		if len(msg) > 83 {
			msg = msg[:80] + "..."
		}
		b.WriteString(greenStyle.Render("│"))
		b.WriteString(statusStyle.Render(msg))
		b.WriteString(strings.Repeat(" ", 85-len(msg)))
		b.WriteString(greenStyle.Render("│"))
	} else if len(m.containers) > 0 && m.selectedRow < len(m.containers) {
		// Show available actions
		selectedContainer := m.containers[m.selectedRow]
		var actions string
		if selectedContainer.Status == "RUNNING" {
			actions = " [S]top | [R]estart"
		} else {
			actions = " [S]tart"
		}
		if selectedContainer.Ports != "" && selectedContainer.Ports != "--" {
			actions += " | [O]pen in browser"
		}
		b.WriteString(greenStyle.Render("│"))
		b.WriteString(cyanStyle.Render(actions))
		b.WriteString(strings.Repeat(" ", 85-len(actions)))
		b.WriteString(greenStyle.Render("│"))
	} else {
		b.WriteString(greenStyle.Render("│"))
		b.WriteString(strings.Repeat(" ", 85))
		b.WriteString(greenStyle.Render("│"))
	}
	b.WriteString("\n")

	// Bottom border
	b.WriteString(greenStyle.Render("└─────────────────────────────────────────────────────────────────────────────────────┘"))

	return containerStyle.Render(b.String())
}

func (m model) renderImages() string {
	var b strings.Builder

	// Top border
	b.WriteString(greenStyle.Render("┌─────────────────────────────────────────────────────────────────────────────────────┐"))
	b.WriteString("\n")

	// Header
	headerLeft := greenStyle.Render("│ Docker TUI v2.0.1")
	headerRight := greenStyle.Render("[F1] Help [Q]uit │")
	headerSpacing := strings.Repeat(" ", 85-len("│ Docker TUI v2.0.1")-len("[F1] Help [Q]uit │"))
	b.WriteString(headerLeft + headerSpacing + headerRight)
	b.WriteString("\n")

	// Tabs with new design
	b.WriteString(m.renderTabs())

	// Status line
	statusText := fmt.Sprintf(" IMAGES (%d total)", len(m.images))
	scrollIndicator := m.getScrollIndicator()
	statusText += scrollIndicator
	statusLine := greenStyle.Render("│") + cyanStyle.Render(statusText)
	statusSpacing := strings.Repeat(" ", 85-len(statusText))
	b.WriteString(statusLine + statusSpacing + greenStyle.Render("│"))
	b.WriteString("\n")

	// Table divider
	b.WriteString(greenStyle.Render("├────────────┬──────────────────────────────┬─────────────┬──────────┬──────────────┤"))
	b.WriteString("\n")

	// Table header
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" IMAGE ID   "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" REPOSITORY                   "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" TAG         "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" SIZE     "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" CREATED      "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString("\n")

	// Header bottom divider
	b.WriteString(greenStyle.Render("├────────────┼──────────────────────────────┼─────────────┼──────────┼──────────────┤"))
	b.WriteString("\n")

	// Table rows
	if m.loading {
		b.WriteString(greenStyle.Render("│"))
		loadingMsg := " Loading images..."
		b.WriteString(cyanStyle.Render(loadingMsg))
		b.WriteString(strings.Repeat(" ", 85-len(loadingMsg)))
		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	} else if len(m.images) == 0 {
		b.WriteString(greenStyle.Render("│"))
		noImagesMsg := " No images found."
		b.WriteString(cyanStyle.Render(noImagesMsg))
		b.WriteString(strings.Repeat(" ", 85-len(noImagesMsg)))
		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	}

	// Only render visible items
	start, end := m.getVisibleRange()
	for i := start; i < end && i < len(m.images); i++ {
		image := m.images[i]
		isSelected := i == m.selectedRow

		b.WriteString(greenStyle.Render("│"))
		if isSelected {
			b.WriteString(yellowStyle.Render(">"))
			b.WriteString(yellowStyle.Render(padRight(image.ID, 11)))
		} else {
			b.WriteString(" ")
			b.WriteString(normalStyle.Render(padRight(image.ID, 11)))
		}

		b.WriteString(greenStyle.Render("│"))
		repoText := padRight(image.Repository, 30)
		if isSelected {
			b.WriteString(yellowStyle.Render(repoText))
		} else {
			b.WriteString(normalStyle.Render(repoText))
		}

		b.WriteString(greenStyle.Render("│"))
		tagText := padRight(image.Tag, 13)
		b.WriteString(normalStyle.Render(tagText))

		b.WriteString(greenStyle.Render("│"))
		sizeText := padRight(image.Size, 10)
		b.WriteString(normalStyle.Render(sizeText))

		b.WriteString(greenStyle.Render("│"))
		createdText := padRight(image.Created, 14)
		b.WriteString(normalStyle.Render(createdText))

		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	}

	// Table bottom border
	b.WriteString(greenStyle.Render("├────────────┴──────────────────────────────┴─────────────┴──────────┴──────────────┤"))
	b.WriteString("\n")

	// Status/action bar
	b.WriteString(greenStyle.Render("│"))
	if m.statusMessage != "" {
		statusStyle := cyanStyle
		if strings.HasPrefix(m.statusMessage, "ERROR:") {
			statusStyle = redStyle
		}
		msg := " " + m.statusMessage
		if len(msg) > 83 {
			msg = msg[:80] + "..."
		}
		b.WriteString(statusStyle.Render(msg))
		b.WriteString(strings.Repeat(" ", 85-len(msg)))
	} else {
		b.WriteString(strings.Repeat(" ", 85))
	}
	b.WriteString(greenStyle.Render("│"))
	b.WriteString("\n")

	// Bottom border
	b.WriteString(greenStyle.Render("└─────────────────────────────────────────────────────────────────────────────────────┘"))

	return containerStyle.Render(b.String())
}

func (m model) renderVolumes() string {
	var b strings.Builder

	// Top border
	b.WriteString(greenStyle.Render("┌─────────────────────────────────────────────────────────────────────────────────────┐"))
	b.WriteString("\n")

	// Header
	headerLeft := greenStyle.Render("│ Docker TUI v2.0.1")
	headerRight := greenStyle.Render("[F1] Help [Q]uit │")
	headerSpacing := strings.Repeat(" ", 85-len("│ Docker TUI v2.0.1")-len("[F1] Help [Q]uit │"))
	b.WriteString(headerLeft + headerSpacing + headerRight)
	b.WriteString("\n")

	// Tabs with new design
	b.WriteString(m.renderTabs())

	// Status line
	statusText := fmt.Sprintf(" VOLUMES (%d total)", len(m.volumes))
	scrollIndicator := m.getScrollIndicator()
	statusText += scrollIndicator
	statusLine := greenStyle.Render("│") + cyanStyle.Render(statusText)
	statusSpacing := strings.Repeat(" ", 85-len(statusText))
	b.WriteString(statusLine + statusSpacing + greenStyle.Render("│"))
	b.WriteString("\n")

	// Table divider
	b.WriteString(greenStyle.Render("├─────────────────────────┬────────┬──────────────────────────────┬──────────────┤"))
	b.WriteString("\n")

	// Table header
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" NAME                     "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" DRIVER "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" MOUNTPOINT                   "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" CREATED      "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString("\n")

	// Header bottom divider
	b.WriteString(greenStyle.Render("├─────────────────────────┼────────┼──────────────────────────────┼──────────────┤"))
	b.WriteString("\n")

	// Table rows
	if m.loading {
		b.WriteString(greenStyle.Render("│"))
		loadingMsg := " Loading volumes..."
		b.WriteString(cyanStyle.Render(loadingMsg))
		b.WriteString(strings.Repeat(" ", 85-len(loadingMsg)))
		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	} else if len(m.volumes) == 0 {
		b.WriteString(greenStyle.Render("│"))
		noVolumesMsg := " No volumes found."
		b.WriteString(cyanStyle.Render(noVolumesMsg))
		b.WriteString(strings.Repeat(" ", 85-len(noVolumesMsg)))
		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	}

	// Only render visible items
	start, end := m.getVisibleRange()
	for i := start; i < end && i < len(m.volumes); i++ {
		volume := m.volumes[i]
		isSelected := i == m.selectedRow

		b.WriteString(greenStyle.Render("│"))
		if isSelected {
			b.WriteString(yellowStyle.Render(">"))
			b.WriteString(yellowStyle.Render(padRight(volume.Name, 24)))
		} else {
			b.WriteString(" ")
			b.WriteString(normalStyle.Render(padRight(volume.Name, 24)))
		}

		b.WriteString(greenStyle.Render("│"))
		driverText := padRight(volume.Driver, 8)
		b.WriteString(normalStyle.Render(driverText))

		b.WriteString(greenStyle.Render("│"))
		mountText := padRight(volume.Mountpoint, 30)
		b.WriteString(normalStyle.Render(mountText))

		b.WriteString(greenStyle.Render("│"))
		createdText := padRight(volume.Created, 14)
		b.WriteString(normalStyle.Render(createdText))

		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	}

	// Table bottom border
	b.WriteString(greenStyle.Render("├─────────────────────────┴────────┴──────────────────────────────┴──────────────┤"))
	b.WriteString("\n")

	// Status/action bar
	b.WriteString(greenStyle.Render("│"))
	if m.statusMessage != "" {
		statusStyle := cyanStyle
		if strings.HasPrefix(m.statusMessage, "ERROR:") {
			statusStyle = redStyle
		}
		msg := " " + m.statusMessage
		if len(msg) > 83 {
			msg = msg[:80] + "..."
		}
		b.WriteString(statusStyle.Render(msg))
		b.WriteString(strings.Repeat(" ", 85-len(msg)))
	} else {
		b.WriteString(strings.Repeat(" ", 85))
	}
	b.WriteString(greenStyle.Render("│"))
	b.WriteString("\n")

	// Bottom border
	b.WriteString(greenStyle.Render("└─────────────────────────────────────────────────────────────────────────────────────┘"))

	return containerStyle.Render(b.String())
}

func (m model) renderNetworks() string {
	var b strings.Builder

	// Top border
	b.WriteString(greenStyle.Render("┌─────────────────────────────────────────────────────────────────────────────────────┐"))
	b.WriteString("\n")

	// Header
	headerLeft := greenStyle.Render("│ Docker TUI v2.0.1")
	headerRight := greenStyle.Render("[F1] Help [Q]uit │")
	headerSpacing := strings.Repeat(" ", 85-len("│ Docker TUI v2.0.1")-len("[F1] Help [Q]uit │"))
	b.WriteString(headerLeft + headerSpacing + headerRight)
	b.WriteString("\n")

	// Tabs with new design
	b.WriteString(m.renderTabs())

	// Status line
	statusText := fmt.Sprintf(" NETWORKS (%d total)", len(m.networks))
	scrollIndicator := m.getScrollIndicator()
	statusText += scrollIndicator
	statusLine := greenStyle.Render("│") + cyanStyle.Render(statusText)
	statusSpacing := strings.Repeat(" ", 85-len(statusText))
	b.WriteString(statusLine + statusSpacing + greenStyle.Render("│"))
	b.WriteString("\n")

	// Table divider
	b.WriteString(greenStyle.Render("├────────────┬────────────────────┬──────────┬────────┬──────────────────┬──────────┤"))
	b.WriteString("\n")

	// Table header
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" NETWORK ID "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" NAME               "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" DRIVER   "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" SCOPE  "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" IPv4             "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString(normalStyle.Render(" IPv6     "))
	b.WriteString(greenStyle.Render("│"))
	b.WriteString("\n")

	// Header bottom divider
	b.WriteString(greenStyle.Render("├────────────┼────────────────────┼──────────┼────────┼──────────────────┼──────────┤"))
	b.WriteString("\n")

	// Table rows
	if m.loading {
		b.WriteString(greenStyle.Render("│"))
		loadingMsg := " Loading networks..."
		b.WriteString(cyanStyle.Render(loadingMsg))
		b.WriteString(strings.Repeat(" ", 85-len(loadingMsg)))
		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	} else if len(m.networks) == 0 {
		b.WriteString(greenStyle.Render("│"))
		noNetworksMsg := " No networks found."
		b.WriteString(cyanStyle.Render(noNetworksMsg))
		b.WriteString(strings.Repeat(" ", 85-len(noNetworksMsg)))
		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	}

	// Only render visible items
	start, end := m.getVisibleRange()
	for i := start; i < end && i < len(m.networks); i++ {
		network := m.networks[i]
		isSelected := i == m.selectedRow

		b.WriteString(greenStyle.Render("│"))
		if isSelected {
			b.WriteString(yellowStyle.Render(">"))
			b.WriteString(yellowStyle.Render(padRight(network.ID, 11)))
		} else {
			b.WriteString(" ")
			b.WriteString(normalStyle.Render(padRight(network.ID, 11)))
		}

		b.WriteString(greenStyle.Render("│"))
		nameText := padRight(network.Name, 20)
		if isSelected {
			b.WriteString(yellowStyle.Render(nameText))
		} else {
			b.WriteString(normalStyle.Render(nameText))
		}

		b.WriteString(greenStyle.Render("│"))
		driverText := padRight(network.Driver, 10)
		b.WriteString(normalStyle.Render(driverText))

		b.WriteString(greenStyle.Render("│"))
		scopeText := padRight(network.Scope, 8)
		b.WriteString(normalStyle.Render(scopeText))

		b.WriteString(greenStyle.Render("│"))
		ipv4Text := padRight(network.IPv4, 18)
		b.WriteString(normalStyle.Render(ipv4Text))

		b.WriteString(greenStyle.Render("│"))
		ipv6Text := padRight(network.IPv6, 10)
		b.WriteString(normalStyle.Render(ipv6Text))

		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	}

	// Table bottom border
	b.WriteString(greenStyle.Render("├────────────┴────────────────────┴──────────┴────────┴──────────────────┴──────────┤"))
	b.WriteString("\n")

	// Status/action bar
	b.WriteString(greenStyle.Render("│"))
	if m.statusMessage != "" {
		statusStyle := cyanStyle
		if strings.HasPrefix(m.statusMessage, "ERROR:") {
			statusStyle = redStyle
		}
		msg := " " + m.statusMessage
		if len(msg) > 83 {
			msg = msg[:80] + "..."
		}
		b.WriteString(statusStyle.Render(msg))
		b.WriteString(strings.Repeat(" ", 85-len(msg)))
	} else {
		b.WriteString(strings.Repeat(" ", 85))
	}
	b.WriteString(greenStyle.Render("│"))
	b.WriteString("\n")

	// Bottom border
	b.WriteString(greenStyle.Render("└─────────────────────────────────────────────────────────────────────────────────────┘"))

	return containerStyle.Render(b.String())
}

func (m model) renderError() string {
	var b strings.Builder

	b.WriteString(greenStyle.Render("┌─────────────────────────────────────────────────────────────────────────────────────┐"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + redStyle.Render(" Docker TUI - Error") + strings.Repeat(" ", 65) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("├─────────────────────────────────────────────────────────────────────────────────────┤"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + strings.Repeat(" ", 85) + greenStyle.Render("│"))
	b.WriteString("\n")

	errMsg := m.err.Error()
	if len(errMsg) > 80 {
		errMsg = errMsg[:77] + "..."
	}
	b.WriteString(greenStyle.Render("│") + redStyle.Render(" Error: "+errMsg) + strings.Repeat(" ", 85-len(" Error: "+errMsg)) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + strings.Repeat(" ", 85) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render(" Troubleshooting:") + strings.Repeat(" ", 67) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("  - Make sure Docker is running") + strings.Repeat(" ", 53) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("  - Check Docker socket permissions") + strings.Repeat(" ", 49) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("  - Verify DOCKER_HOST environment variable") + strings.Repeat(" ", 41) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + strings.Repeat(" ", 85) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + cyanStyle.Render(" Press 'q' to quit") + strings.Repeat(" ", 66) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("└─────────────────────────────────────────────────────────────────────────────────────┘"))

	return containerStyle.Render(b.String())
}

func (m model) renderHelp() string {
	var b strings.Builder

	b.WriteString(greenStyle.Render("┌─────────────────────────────────────────────────────────────────────────────────────┐"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + yellowStyle.Render(" Docker TUI - Help") + strings.Repeat(" ", 66) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("├─────────────────────────────────────────────────────────────────────────────────────┤"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render(" Navigation:") + strings.Repeat(" ", 72) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("   ↑/k      - Move selection up (auto-scrolls)") + strings.Repeat(" ", 37) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("   ↓/j      - Move selection down (auto-scrolls)") + strings.Repeat(" ", 35) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("   ←/→ or h/l - Switch tabs") + strings.Repeat(" ", 56) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("   1-4 or ^D/^I/^V/^N - Jump to specific tab") + strings.Repeat(" ", 39) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + strings.Repeat(" ", 85) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render(" Container Actions:") + strings.Repeat(" ", 65) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("   s        - Start/Stop selected container") + strings.Repeat(" ", 40) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("   r        - Restart selected container") + strings.Repeat(" ", 43) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("   o        - Open container port in browser") + strings.Repeat(" ", 39) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("   Enter    - Refresh container list") + strings.Repeat(" ", 47) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + strings.Repeat(" ", 85) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render(" Other:") + strings.Repeat(" ", 77) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("   F1       - Toggle help") + strings.Repeat(" ", 58) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + normalStyle.Render("   q/Ctrl+C - Quit application") + strings.Repeat(" ", 53) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + strings.Repeat(" ", 85) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("│") + cyanStyle.Render(" Auto-refreshes every 5 seconds | Press F1 to return") + strings.Repeat(" ", 32) + greenStyle.Render("│"))
	b.WriteString("\n")
	b.WriteString(greenStyle.Render("└─────────────────────────────────────────────────────────────────────────────────────┘"))

	return containerStyle.Render(b.String())
}

// Helper functions for text alignment
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
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
