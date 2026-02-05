package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"tinyd/internal/components"
	"tinyd/internal/types"
)

// Color styles for status indicators
var (
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	grayStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#999999"))
)

// View renders the UI
func (m *Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit", m.err)
	}

	// Render based on current view mode
	switch m.currentView {
	case types.ViewModeList:
		return m.renderListView()
	case types.ViewModeLogs:
		return m.renderLogsView()
	case types.ViewModeInspect:
		return m.renderInspectView()
	default:
		return "Unknown view mode\n\nPress q to quit"
	}
}

// renderListView renders the main list view
func (m *Model) renderListView() string {
	var b strings.Builder

	// Update and render tabs
	m.tabs = m.tabs.SetActiveTab(m.activeTab).WithWidth(m.width)
	tabsContent := m.tabs.View()
	b.WriteString(tabsContent)

	// Render content based on active tab
	var contentStr string
	switch m.activeTab {
	case 0:
		contentStr = m.renderContainersTab()
	case 1:
		contentStr = m.renderImagesTab()
	case 2:
		contentStr = m.renderVolumesTab()
	case 3:
		contentStr = m.renderNetworksTab()
	}
	b.WriteString(contentStr)

	// Calculate padding to push action bar to bottom
	// Count actual lines used (tabs + visible content rows + action bar)
	tabsHeight := strings.Count(tabsContent, "\n")
	contentHeight := strings.Count(contentStr, "\n")
	actionBarHeight := 3

	// Total used height including current content
	usedHeight := tabsHeight + contentHeight + actionBarHeight + 2 // +2 for newlines

	// Add padding to push action bar to bottom
	remainingHeight := m.height - usedHeight
	if remainingHeight > 0 {
		b.WriteString(strings.Repeat("\n", remainingHeight))
	} else if remainingHeight < 0 {
		// If content is too tall, don't add padding
		b.WriteString("\n")
	}

	// Render action bar at bottom
	b.WriteString("\n")
	m.actionBar = m.actionBar.WithWidth(m.width)
	if m.statusMessage != "" {
		m.actionBar = m.actionBar.SetStatusMessage(m.statusMessage)
	} else {
		m.actionBar = m.actionBar.SetActions(m.getActionShortcuts())
	}
	b.WriteString(m.actionBar.View())

	return b.String()
}

// renderContainersTab renders the containers tab with proper table formatting
func (m *Model) renderContainersTab() string {
	if len(m.containers) == 0 {
		if m.loading {
			return "Loading containers..."
		}
		return "No containers found"
	}

	// Calculate responsive column widths using full terminal width
	totalWidth := m.width - 4 // Account for padding
	if totalWidth < 60 {
		totalWidth = 60 // Minimum width for reasonable display
	}

	// Fixed columns: Status(2) + CPU(8) + MEM(8) + Ports(15)
	// Spacing: 5 gaps * 2 spaces = 10
	fixedWidth := 2 + 8 + 8 + 15
	spacing := 5 * 2 // (6 columns - 1) * 2 spaces per gap
	fillWidth := totalWidth - fixedWidth - spacing

	// Ensure minimum width for fill columns
	if fillWidth < 40 {
		fillWidth = 40
	}

	// Two fill columns: Name and Image (distribute equally)
	nameFill := fillWidth / 2
	imageFill := fillWidth - nameFill

	// Ensure each fill column has reasonable minimum
	if nameFill < 20 {
		nameFill = 20
	}
	if imageFill < 20 {
		imageFill = 20
	}

	headers := []components.TableHeader{
		{Label: "", Width: 2, AlignRight: false},          // Status dot
		{Label: "NAME", Width: nameFill, AlignRight: false},
		{Label: "IMAGE", Width: imageFill, AlignRight: false},
		{Label: "CPU", Width: 8, AlignRight: true},
		{Label: "MEM", Width: 8, AlignRight: true},
		{Label: "PORTS", Width: 15, AlignRight: false},
	}

	// Build table rows (only visible ones based on scroll position)
	var rows []components.TableRow
	start := m.scrollOffset
	end := m.scrollOffset + m.viewportHeight
	if end > len(m.containers) {
		end = len(m.containers)
	}
	if start > len(m.containers) {
		start = len(m.containers)
	}

	for i := start; i < end; i++ {
		c := m.containers[i]

		// Handle delete confirmation overlay
		if m.deleteConfirmMode && i == m.selectedRow {
			confirmText := renderDeleteConfirmation(c.Name, m.deleteConfirmOption)
			emptyCells := make([]string, len(headers)-1)
			rows = append(rows, components.TableRow{
				Cells:      append([]string{confirmText}, emptyCells...),
				IsSelected: true,
			})
			continue
		}

		cells := []string{
			m.getStatusDot(c.Status),
			truncateWithEllipsis(c.Name, headers[1].Width),   // Fill column - truncate
			truncateWithEllipsis(c.Image, headers[2].Width),  // Fill column - truncate
			c.CPU,                                             // Fixed column - short values
			c.Mem,                                             // Fixed column - short values
			truncateWithEllipsis(c.Ports, 15),                // Can be long
		}

		rows = append(rows, components.TableRow{
			Cells:      cells,
			IsSelected: i == m.selectedRow,
		})
	}

	// Create and render table
	table := components.NewTableComponent(headers).
		WithWidth(m.width).
		SetRows(rows).
		SetVisibleRange(0, len(rows))

	return table.View()
}

// renderImagesTab renders the images tab with proper table formatting
func (m *Model) renderImagesTab() string {
	if len(m.images) == 0 {
		if m.loading {
			return "Loading images..."
		}
		return "No images found"
	}

	// Calculate responsive column widths using full terminal width
	totalWidth := m.width - 4
	if totalWidth < 50 {
		totalWidth = 50
	}

	// Fixed columns: Status(2) + Size(10) + Created(8)
	// Spacing: 3 gaps * 2 spaces = 6
	fixedWidth := 2 + 10 + 8
	spacing := 3 * 2 // (4 columns - 1) * 2 spaces per gap
	fillWidth := totalWidth - fixedWidth - spacing
	if fillWidth < 20 {
		fillWidth = 20
	}

	// One fill column: Repository:Tag
	headers := []components.TableHeader{
		{Label: "", Width: 2, AlignRight: false},              // Status
		{Label: "REPOSITORY:TAG", Width: fillWidth, AlignRight: false},
		{Label: "SIZE", Width: 10, AlignRight: true},
		{Label: "CREATED", Width: 8, AlignRight: false},
	}

	// Build table rows (only visible ones based on scroll position)
	var rows []components.TableRow
	start := m.scrollOffset
	end := m.scrollOffset + m.viewportHeight
	if end > len(m.images) {
		end = len(m.images)
	}
	if start > len(m.images) {
		start = len(m.images)
	}

	for i := start; i < end; i++ {
		img := m.images[i]

		// Handle delete confirmation overlay
		if m.deleteConfirmMode && i == m.selectedRow {
			confirmText := renderDeleteConfirmation(img.Repository+":"+img.Tag, m.deleteConfirmOption)
			emptyCells := make([]string, len(headers)-1)
			rows = append(rows, components.TableRow{
				Cells:      append([]string{confirmText}, emptyCells...),
				IsSelected: true,
			})
			continue
		}

		// Combine repository:tag
		repoTag := img.Repository + ":" + img.Tag

		cells := []string{
			grayStyle.Render("○"),
			truncateWithEllipsis(repoTag, headers[1].Width), // Fill column - truncate
			img.Size,                                         // Fixed column - short values
			shortenTimeAgo(img.Created),                     // Fixed column - already short
		}

		rows = append(rows, components.TableRow{
			Cells:      cells,
			IsSelected: i == m.selectedRow,
		})
	}

	// Create and render table
	table := components.NewTableComponent(headers).
		WithWidth(m.width).
		SetRows(rows).
		SetVisibleRange(0, len(rows))

	return table.View()
}

// renderVolumesTab renders the volumes tab with proper table formatting
func (m *Model) renderVolumesTab() string {
	if len(m.volumes) == 0 {
		if m.loading {
			return "Loading volumes..."
		}
		return "No volumes found"
	}

	// Calculate responsive column widths using full terminal width
	totalWidth := m.width - 4
	if totalWidth < 50 {
		totalWidth = 50
	}

	// Fixed columns: Status(2)
	// Spacing: 3 gaps * 2 spaces = 6
	fixedWidth := 2
	spacing := 3 * 2 // (4 columns - 1) * 2 spaces per gap
	fillWidth := totalWidth - fixedWidth - spacing
	if fillWidth < 30 {
		fillWidth = 30
	}

	// Three fill columns: Name, Containers, Mount Point (distribute equally)
	nameFill := fillWidth / 3
	containersFill := fillWidth / 3
	mountFill := fillWidth - nameFill - containersFill

	headers := []components.TableHeader{
		{Label: "", Width: 2, AlignRight: false},                    // Status
		{Label: "NAME", Width: nameFill, AlignRight: false},
		{Label: "CONTAINERS", Width: containersFill, AlignRight: false},
		{Label: "MOUNT POINT", Width: mountFill, AlignRight: false},
	}

	// Build table rows (only visible ones based on scroll position)
	var rows []components.TableRow
	start := m.scrollOffset
	end := m.scrollOffset + m.viewportHeight
	if end > len(m.volumes) {
		end = len(m.volumes)
	}
	if start > len(m.volumes) {
		start = len(m.volumes)
	}

	for i := start; i < end; i++ {
		vol := m.volumes[i]

		// Handle delete confirmation overlay
		if m.deleteConfirmMode && i == m.selectedRow {
			confirmText := renderDeleteConfirmation(vol.Name, m.deleteConfirmOption)
			emptyCells := make([]string, len(headers)-1)
			rows = append(rows, components.TableRow{
				Cells:      append([]string{confirmText}, emptyCells...),
				IsSelected: true,
			})
			continue
		}

		statusDot := grayStyle.Render("○")
		if vol.InUse {
			statusDot = greenStyle.Render("●")
		}

		// Show container names or "-" if not in use
		containers := vol.Containers
		if containers == "" {
			containers = "-"
		}

		cells := []string{
			statusDot,
			truncateWithEllipsis(vol.Name, headers[1].Width),       // Fill column - truncate
			truncateWithEllipsis(containers, headers[2].Width),     // Fill column - truncate
			truncateWithEllipsis(vol.Mountpoint, headers[3].Width), // Fill column - truncate
		}

		rows = append(rows, components.TableRow{
			Cells:      cells,
			IsSelected: i == m.selectedRow,
		})
	}

	// Create and render table
	table := components.NewTableComponent(headers).
		WithWidth(m.width).
		SetRows(rows).
		SetVisibleRange(0, len(rows))

	return table.View()
}

// renderNetworksTab renders the networks tab with proper table formatting
func (m *Model) renderNetworksTab() string {
	if len(m.networks) == 0 {
		if m.loading {
			return "Loading networks..."
		}
		return "No networks found"
	}

	// Calculate responsive column widths using full terminal width
	totalWidth := m.width - 4
	if totalWidth < 50 {
		totalWidth = 50
	}

	// Fixed columns: Status(2) + Driver(10) + Scope(8) + IPv4(18)
	// Spacing: 5 gaps * 2 spaces = 10
	fixedWidth := 2 + 10 + 8 + 18
	spacing := 5 * 2 // (6 columns - 1) * 2 spaces per gap
	fillWidth := totalWidth - fixedWidth - spacing
	if fillWidth < 20 {
		fillWidth = 20
	}

	// Two fill columns: Name and Containers (distribute equally)
	nameFill := fillWidth / 2
	containersFill := fillWidth - nameFill

	headers := []components.TableHeader{
		{Label: "", Width: 2, AlignRight: false},                        // Status
		{Label: "NAME", Width: nameFill, AlignRight: false},
		{Label: "CONTAINERS", Width: containersFill, AlignRight: false},
		{Label: "DRIVER", Width: 10, AlignRight: false},
		{Label: "SCOPE", Width: 8, AlignRight: false},
		{Label: "IPv4", Width: 18, AlignRight: false},
	}

	// Build table rows (only visible ones based on scroll position)
	var rows []components.TableRow
	start := m.scrollOffset
	end := m.scrollOffset + m.viewportHeight
	if end > len(m.networks) {
		end = len(m.networks)
	}
	if start > len(m.networks) {
		start = len(m.networks)
	}

	for i := start; i < end; i++ {
		net := m.networks[i]

		// Handle delete confirmation overlay
		if m.deleteConfirmMode && i == m.selectedRow {
			confirmText := renderDeleteConfirmation(net.Name, m.deleteConfirmOption)
			emptyCells := make([]string, len(headers)-1)
			rows = append(rows, components.TableRow{
				Cells:      append([]string{confirmText}, emptyCells...),
				IsSelected: true,
			})
			continue
		}

		statusDot := grayStyle.Render("○")
		if net.InUse {
			statusDot = greenStyle.Render("●")
		}

		// TODO: Add Containers field to Network type to show connected container names
		containers := "-"

		cells := []string{
			statusDot,
			truncateWithEllipsis(net.Name, headers[1].Width),       // Fill column - truncate
			truncateWithEllipsis(containers, headers[2].Width),     // Fill column - truncate
			net.Driver,                                              // Fixed column - short values
			net.Scope,                                               // Fixed column - short values
			truncateWithEllipsis(net.IPv4, 18),                     // Can be long
		}

		rows = append(rows, components.TableRow{
			Cells:      cells,
			IsSelected: i == m.selectedRow,
		})
	}

	// Create and render table
	table := components.NewTableComponent(headers).
		WithWidth(m.width).
		SetRows(rows).
		SetVisibleRange(0, len(rows))

	return table.View()
}

// renderLogsView renders the logs detail view
func (m *Model) renderLogsView() string {
	m.detailView = m.detailView.SetContent(m.logsContent)
	m.detailView = m.detailView.SetScroll(m.logsScrollOffset)
	return m.detailView.View()
}

// renderInspectView renders the inspect detail view
func (m *Model) renderInspectView() string {
	m.detailView = m.detailView.SetContent(m.inspectContent)
	return m.detailView.View()
}

// Helper functions

// getStatusDot returns a colored status indicator based on container status
func (m *Model) getStatusDot(status string) string {
	switch status {
	case "RUNNING":
		return greenStyle.Render("●") // Green filled circle for running
	case "STOPPED":
		return grayStyle.Render("○") // Gray empty circle for stopped
	case "PAUSED":
		return yellowStyle.Render("●") // Yellow filled circle for paused (warning state)
	case "ERROR":
		return redStyle.Render("●") // Red filled circle for error (attention needed)
	case "RESTARTING":
		return yellowStyle.Render("●") // Yellow filled circle for restarting (warning state)
	default:
		return grayStyle.Render("○") // Gray empty circle for unknown
	}
}

// truncateWithEllipsis truncates a string to max length with ellipsis
func truncateWithEllipsis(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// renderDeleteConfirmation renders an inline delete confirmation message
func renderDeleteConfirmation(name string, selectedOption int) string {
	// Delete message in white
	confirmStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0a0a0a")).
		Bold(true)

	// Active YES button: black text on green background
	yesActiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#00FF00"))

	// Active NO button: black text on red background
	noActiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#FF0000"))

	// Inactive button: gray text, no background
	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	var b strings.Builder
	b.WriteString(confirmStyle.Render("Delete " + truncateWithEllipsis(name, 30) + "? "))

	if selectedOption == 0 {
		// YES is active
		b.WriteString(yesActiveStyle.Render(" YES "))
		b.WriteString(inactiveStyle.Render(" NO "))
	} else {
		// NO is active
		b.WriteString(inactiveStyle.Render(" YES "))
		b.WriteString(noActiveStyle.Render(" NO "))
	}

	return b.String()
}

// getActionShortcuts returns the keyboard shortcuts for the current tab
func (m *Model) getActionShortcuts() string {
	var shortcuts []string

	switch m.activeTab {
	case 0: // Containers - dynamic based on selected container status
		if m.selectedRow < len(m.containers) {
			container := m.containers[m.selectedRow]

			// Show appropriate actions based on container status
			if container.Status == "RUNNING" {
				shortcuts = []string{
					renderShortcut("S", "top"),
					renderShortcut("R", "estart"),
					renderShortcut("L", "ogs"),
					renderShortcut("E", "xec"),
					renderShortcut("I", "nspect"),
					renderShortcut("D", "elete"),
				}
			} else {
				// Stopped, Error, or other non-running states
				shortcuts = []string{
					renderShortcut("S", "tart"),
					renderShortcut("L", "ogs"),
					renderShortcut("I", "nspect"),
					renderShortcut("D", "elete"),
				}
			}
		} else {
			// No container selected, show all options
			shortcuts = []string{
				renderShortcut("S", "tart/Stop"),
				renderShortcut("R", "estart"),
				renderShortcut("L", "ogs"),
				renderShortcut("E", "xec"),
				renderShortcut("I", "nspect"),
				renderShortcut("D", "elete"),
			}
		}
	case 1: // Images
		shortcuts = []string{
			renderShortcut("R", "un"),
			renderShortcut("P", "ull"),
			renderShortcut("I", "nspect"),
			renderShortcut("D", "elete"),
		}
	case 2: // Volumes
		shortcuts = []string{
			renderShortcut("I", "nspect"),
			renderShortcut("D", "elete"),
		}
	case 3: // Networks
		shortcuts = []string{
			renderShortcut("I", "nspect"),
			renderShortcut("D", "elete"),
		}
	}

	// Add common shortcuts
	shortcuts = append(shortcuts,
		renderShortcut("F1", " Help"),
	)

	return strings.Join(shortcuts, " ")
}

// renderShortcut formats a keyboard shortcut with underscored first letter
func renderShortcut(key string, rest ...string) string {
	// First letter: white with underline
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0a0a0a")).
		Underline(true)

	// Rest of word: dimmed gray
	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	var b strings.Builder
	b.WriteString(keyStyle.Render(key))
	if len(rest) > 0 {
		b.WriteString(textStyle.Render(strings.Join(rest, "")))
	}

	return b.String()
}

// formatInspectOutput formats container inspect JSON into a readable tree
func (m *Model) formatInspectOutput(jsonStr string) string {
	var b strings.Builder

	// Parse JSON to extract key information
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return "Error parsing inspect data: " + err.Error()
	}

	// Helper to safely get string values (tries multiple keys)
	getStr := func(m map[string]interface{}, keys ...string) string {
		for _, key := range keys {
			if v, ok := m[key]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
		return "-"
	}

	// Helper to safely get nested map
	getMap := func(m map[string]interface{}, key string) map[string]interface{} {
		if v, ok := m[key]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				return m
			}
		}
		return make(map[string]interface{})
	}

	// Extract data (try both "ID" and "Id" for compatibility)
	name := getStr(data, "Name")
	if strings.HasPrefix(name, "/") {
		name = name[1:] // Remove leading slash
	}
	id := getStr(data, "ID", "Id")
	if len(id) > 12 {
		id = id[:12]
	}

	config := getMap(data, "Config")
	state := getMap(data, "State")
	hostConfig := getMap(data, "HostConfig")

	// Image info
	imageName := getStr(config, "Image")
	platform := getStr(data, "Platform")
	if platform == "" {
		platform = "linux/amd64" // Default
	}

	// Process info
	entrypoint := "-"
	if ep, ok := config["Entrypoint"]; ok && ep != nil {
		if arr, ok := ep.([]interface{}); ok && len(arr) > 0 {
			entrypoint = fmt.Sprintf("%v", arr[0])
		}
	}
	workdir := getStr(config, "WorkingDir")
	if workdir == "" {
		workdir = "/"
	}

	// Lifecycle info
	startedAt := getStr(state, "StartedAt")
	finishedAt := getStr(state, "FinishedAt")
	if startedAt != "" && len(startedAt) > 10 {
		// Parse and format time (just show HH:MM:SS)
		if t, err := time.Parse(time.RFC3339, startedAt); err == nil {
			startedAt = t.Format("15:04:05")
		}
	}
	if finishedAt != "" && len(finishedAt) > 10 {
		if t, err := time.Parse(time.RFC3339, finishedAt); err == nil {
			finishedAt = t.Format("15:04:05")
		}
	}

	// State info
	status := getStr(state, "Status")
	exitCode := "0"
	if ec, ok := state["ExitCode"]; ok {
		exitCode = fmt.Sprintf("%v", ec)
	}
	oomKilled := "false"
	if oom, ok := state["OOMKilled"]; ok {
		oomKilled = fmt.Sprintf("%v", oom)
	}
	restartPolicy := getStr(hostConfig, "RestartPolicy")
	if restartPolicy == "" {
		if rp, ok := hostConfig["RestartPolicy"]; ok {
			if rpm, ok := rp.(map[string]interface{}); ok {
				restartPolicy = getStr(rpm, "Name")
			}
		}
	}
	if restartPolicy == "" {
		restartPolicy = "no"
	}

	// Build tree structure
	b.WriteString("Press [J] to toggle raw JSON view\n\n")
	b.WriteString(fmt.Sprintf("Container\n"))
	b.WriteString(fmt.Sprintf("├─ Name        : %s\n", name))
	b.WriteString(fmt.Sprintf("├─ ID          : %s\n", id))
	b.WriteString(fmt.Sprintf("│\n"))
	b.WriteString(fmt.Sprintf("├─ Image\n"))
	b.WriteString(fmt.Sprintf("│   ├─ Name     : %s\n", imageName))
	b.WriteString(fmt.Sprintf("│   └─ Platform : %s\n", platform))
	b.WriteString(fmt.Sprintf("│\n"))
	b.WriteString(fmt.Sprintf("├─ Process\n"))
	b.WriteString(fmt.Sprintf("│   ├─ Entrypoint : %s\n", entrypoint))
	b.WriteString(fmt.Sprintf("│   └─ Workdir    : %s\n", workdir))
	b.WriteString(fmt.Sprintf("│\n"))
	b.WriteString(fmt.Sprintf("├─ Lifecycle\n"))
	b.WriteString(fmt.Sprintf("│   ├─ Started  : %s\n", startedAt))
	b.WriteString(fmt.Sprintf("│   └─ Finished : %s\n", finishedAt))
	b.WriteString(fmt.Sprintf("│\n"))
	b.WriteString(fmt.Sprintf("└─ State\n"))
	b.WriteString(fmt.Sprintf("    ├─ Status   : %s\n", status))
	b.WriteString(fmt.Sprintf("    ├─ ExitCode : %s\n", exitCode))
	b.WriteString(fmt.Sprintf("    ├─ OOMKill  : %s\n", oomKilled))
	b.WriteString(fmt.Sprintf("    └─ Restart  : %s\n", restartPolicy))

	return b.String()
}

// shortenTimeAgo converts "4 hours ago" to "4h ago" format
func shortenTimeAgo(timeStr string) string {
	s := strings.TrimSpace(timeStr)
	s = strings.Replace(s, " hours ago", "h ago", 1)
	s = strings.Replace(s, " hour ago", "h ago", 1)
	s = strings.Replace(s, " minutes ago", "m ago", 1)
	s = strings.Replace(s, " minute ago", "m ago", 1)
	s = strings.Replace(s, " seconds ago", "s ago", 1)
	s = strings.Replace(s, " second ago", "s ago", 1)
	s = strings.Replace(s, " days ago", "d ago", 1)
	s = strings.Replace(s, " day ago", "d ago", 1)
	s = strings.Replace(s, " weeks ago", "w ago", 1)
	s = strings.Replace(s, " week ago", "w ago", 1)
	s = strings.Replace(s, " months ago", "mo ago", 1)
	s = strings.Replace(s, " month ago", "mo ago", 1)
	s = strings.Replace(s, " years ago", "y ago", 1)
	s = strings.Replace(s, " year ago", "y ago", 1)
	return s
}
