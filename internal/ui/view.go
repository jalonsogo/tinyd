package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/go-units"
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

	// Add scroll indicator
	scrollInfo := m.getScrollIndicator(len(m.containers))
	return table.View() + scrollInfo
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

		// Only truncate if actually needed
		repoTagCell := repoTag
		if len(repoTag) > headers[1].Width {
			repoTagCell = truncateWithEllipsis(repoTag, headers[1].Width)
		}

		cells := []string{
			m.getImageStatusDot(img),
			repoTagCell,
			img.Size,                    // Fixed column - short values
			shortenTimeAgo(img.Created), // Fixed column - already short
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

	// Add scroll indicator
	scrollInfo := m.getScrollIndicator(len(m.images))
	return table.View() + scrollInfo
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

	// Add scroll indicator
	scrollInfo := m.getScrollIndicator(len(m.volumes))
	return table.View() + scrollInfo
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

	// Add scroll indicator
	scrollInfo := m.getScrollIndicator(len(m.networks))
	return table.View() + scrollInfo
}

// renderLogsView renders the logs detail view
func (m *Model) renderLogsView() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0a0a0a")).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Background(lipgloss.Color("#0a0a0a"))

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Background(lipgloss.Color("#0a0a0a"))

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Background(lipgloss.Color("#0a0a0a"))

	// Header
	headerText := "Logs"
	if m.selectedContainer != nil {
		headerText = "Logs: " + m.selectedContainer.Name
	}
	headerRight := "[ESC] Back"
	headerSpacing := strings.Repeat(" ", m.width-len(headerText)-len(headerRight)-4)
	b.WriteString(titleStyle.Render(headerText))
	b.WriteString(headerSpacing)
	b.WriteString(helpStyle.Render(headerRight))
	b.WriteString("\n")

	// Content divider
	b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n")

	// Calculate available lines for content
	// Height - tabs(4) - header(1) - divider(1) - action bar(3) - scroll indicator(2)
	availableLines := m.height - 11
	if availableLines < 5 {
		availableLines = 5
	}

	// Render content with scrolling
	if m.logsContent == "" {
		b.WriteString(contentStyle.Render(" Loading..."))
		b.WriteString("\n")
	} else {
		lines := strings.Split(m.logsContent, "\n")
		totalLines := len(lines)

		end := m.logsScrollOffset + availableLines
		if end > totalLines {
			end = totalLines
		}

		for i := m.logsScrollOffset; i < end; i++ {
			if i < len(lines) {
				b.WriteString(lines[i])
				b.WriteString("\n")
			}
		}

		// Fill remaining lines
		for i := end - m.logsScrollOffset; i < availableLines; i++ {
			b.WriteString("\n")
		}

		// Add scroll indicator
		b.WriteString(m.getInspectScrollIndicator(totalLines, availableLines))
	}

	return b.String()
}

// renderInspectView renders the inspect detail view
func (m *Model) renderInspectView() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0a0a0a")).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Background(lipgloss.Color("#0a0a0a"))

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Background(lipgloss.Color("#0a0a0a"))

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Background(lipgloss.Color("#0a0a0a"))

	// Header
	headerText := "Inspect"
	headerRight := "[ESC] Back"
	headerSpacing := strings.Repeat(" ", m.width-len(headerText)-len(headerRight)-4)
	b.WriteString(titleStyle.Render(headerText))
	b.WriteString(headerSpacing)
	b.WriteString(helpStyle.Render(headerRight))
	b.WriteString("\n")

	// Content divider
	b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n")

	// Calculate available lines for content
	// Height - tabs(4) - header(1) - divider(1) - action bar(3) - scroll indicator(2)
	availableLines := m.height - 11
	if availableLines < 5 {
		availableLines = 5
	}

	// Render content with scrolling
	if m.inspectContent == "" {
		b.WriteString(contentStyle.Render(" Loading..."))
		b.WriteString("\n")
	} else {
		lines := strings.Split(m.inspectContent, "\n")
		totalLines := len(lines)

		end := m.logsScrollOffset + availableLines
		if end > totalLines {
			end = totalLines
		}

		for i := m.logsScrollOffset; i < end; i++ {
			if i < len(lines) {
				b.WriteString(lines[i])
				b.WriteString("\n")
			}
		}

		// Fill remaining lines
		for i := end - m.logsScrollOffset; i < availableLines; i++ {
			b.WriteString("\n")
		}

		// Add scroll indicator
		b.WriteString(m.getInspectScrollIndicator(totalLines, availableLines))
	}

	return b.String()
}

// Helper functions

// getScrollIndicator returns a scroll indicator showing current position and scroll availability
func (m *Model) getScrollIndicator(totalItems int) string {
	if totalItems == 0 {
		return ""
	}

	var b strings.Builder
	indicatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")).
		Background(lipgloss.Color("#0a0a0a"))

	b.WriteString("\n")

	// Show scroll indicators and position info
	canScrollUp := m.scrollOffset > 0
	canScrollDown := m.scrollOffset+m.viewportHeight < totalItems

	start := m.scrollOffset + 1
	end := m.scrollOffset + m.viewportHeight
	if end > totalItems {
		end = totalItems
	}

	// Build indicator line
	var parts []string

	if canScrollUp {
		parts = append(parts, highlightStyle.Render("↑ More above"))
	}

	parts = append(parts, indicatorStyle.Render(fmt.Sprintf("Showing %d-%d of %d", start, end, totalItems)))

	if canScrollDown {
		parts = append(parts, highlightStyle.Render("↓ More below"))
	}

	b.WriteString(strings.Join(parts, "  "))

	return b.String()
}

// getInspectScrollIndicator returns scroll indicator for inspect view showing line positions
func (m *Model) getInspectScrollIndicator(totalLines, visibleLines int) string {
	if totalLines == 0 {
		return ""
	}

	var b strings.Builder
	indicatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")).
		Background(lipgloss.Color("#0a0a0a"))

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Background(lipgloss.Color("#0a0a0a"))

	// Separator line
	b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n")

	// Show scroll indicators and position info
	canScrollUp := m.logsScrollOffset > 0
	canScrollDown := m.logsScrollOffset+visibleLines < totalLines

	start := m.logsScrollOffset + 1
	end := m.logsScrollOffset + visibleLines
	if end > totalLines {
		end = totalLines
	}

	// Build indicator line
	var parts []string

	if canScrollUp {
		parts = append(parts, highlightStyle.Render("↑ Scroll up"))
	}

	parts = append(parts, indicatorStyle.Render(fmt.Sprintf("Lines %d-%d of %d", start, end, totalLines)))

	if canScrollDown {
		parts = append(parts, highlightStyle.Render("↓ Scroll down"))
	}

	b.WriteString(strings.Join(parts, "  "))

	return b.String()
}

// colorizeJSON adds jq-style syntax highlighting to JSON output
func colorizeJSON(jsonStr string) string {
	// Color styles for JSON syntax highlighting
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#87CEEB"))      // Light blue for keys
	stringStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))   // Green for strings
	numberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D19A66"))   // Orange for numbers
	boolStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B"))     // Yellow for booleans
	nullStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370"))     // Gray for null
	punctStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ABB2BF"))    // Light gray for punctuation

	var result strings.Builder
	var inString bool
	var isKey bool
	var buffer strings.Builder

	for i := 0; i < len(jsonStr); i++ {
		ch := jsonStr[i]

		if ch == '"' && (i == 0 || jsonStr[i-1] != '\\') {
			if inString {
				// End of string
				buffer.WriteByte(ch)
				str := buffer.String()
				if isKey {
					result.WriteString(keyStyle.Render(str))
					isKey = false
				} else {
					result.WriteString(stringStyle.Render(str))
				}
				buffer.Reset()
				inString = false
			} else {
				// Start of string - check if it's a key (followed by :)
				inString = true
				buffer.WriteByte(ch)
				// Look ahead to see if this is a key
				j := i + 1
				for j < len(jsonStr) && jsonStr[j] != '"' {
					if jsonStr[j] == '\\' && j+1 < len(jsonStr) {
						j++ // Skip escaped character
					}
					j++
				}
				if j < len(jsonStr) {
					j++ // Skip closing quote
					for j < len(jsonStr) && (jsonStr[j] == ' ' || jsonStr[j] == '\t') {
						j++
					}
					if j < len(jsonStr) && jsonStr[j] == ':' {
						isKey = true
					}
				}
			}
		} else if inString {
			buffer.WriteByte(ch)
		} else if ch >= '0' && ch <= '9' || ch == '-' || ch == '.' {
			// Number
			numStart := i
			for i < len(jsonStr) && (jsonStr[i] >= '0' && jsonStr[i] <= '9' ||
				jsonStr[i] == '.' || jsonStr[i] == '-' || jsonStr[i] == 'e' ||
				jsonStr[i] == 'E' || jsonStr[i] == '+') {
				i++
			}
			i-- // Back up one
			result.WriteString(numberStyle.Render(jsonStr[numStart : i+1]))
		} else if i+4 <= len(jsonStr) && jsonStr[i:i+4] == "true" {
			result.WriteString(boolStyle.Render("true"))
			i += 3
		} else if i+5 <= len(jsonStr) && jsonStr[i:i+5] == "false" {
			result.WriteString(boolStyle.Render("false"))
			i += 4
		} else if i+4 <= len(jsonStr) && jsonStr[i:i+4] == "null" {
			result.WriteString(nullStyle.Render("null"))
			i += 3
		} else if ch == '{' || ch == '}' || ch == '[' || ch == ']' || ch == ',' || ch == ':' {
			result.WriteString(punctStyle.Render(string(ch)))
		} else {
			result.WriteByte(ch)
		}
	}

	return result.String()
}

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

// getImageStatusDot returns a colored status indicator based on image status
func (m *Model) getImageStatusDot(img types.Image) string {
	if img.InUse {
		return greenStyle.Render("●") // Green filled circle for in-use images
	} else if img.Dangling {
		return redStyle.Render("●") // Red filled circle for dangling images (warning)
	}
	return grayStyle.Render("○") // Gray empty circle for unused images
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
			renderShortcut("S", "tart"),
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
		renderShortcut("H", "elp"),
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

// formatImageInspectOutput formats image inspect JSON into a readable tree
func (m *Model) formatImageInspectOutput(jsonStr string) string {
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

	// Extract data
	id := getStr(data, "ID", "Id")
	if len(id) > 19 {
		id = id[7:19] // Skip "sha256:" prefix
	}

	// Get tags
	tags := "-"
	if tagsArr, ok := data["RepoTags"]; ok {
		if arr, ok := tagsArr.([]interface{}); ok && len(arr) > 0 {
			tags = fmt.Sprintf("%v", arr[0])
		}
	}

	// Size
	size := "-"
	if sizeVal, ok := data["Size"]; ok {
		if s, ok := sizeVal.(float64); ok {
			size = units.BytesSize(s)
		}
	}

	// Created
	created := getStr(data, "Created")
	if created != "" && len(created) > 10 {
		created = created[:10] // Just the date part
	}

	// Architecture
	os := getStr(data, "Os")
	arch := getStr(data, "Architecture")
	variant := getStr(data, "Variant")
	platform := os + "/" + arch
	if variant != "" && variant != "-" {
		platform += "/" + variant
	}

	// Config
	config := getMap(data, "Config")
	entrypoint := "-"
	if ep, ok := config["Entrypoint"]; ok && ep != nil {
		if arr, ok := ep.([]interface{}); ok && len(arr) > 0 {
			entrypoint = fmt.Sprintf("%v", arr[0])
		}
	}

	cmd := "-"
	if cmdVal, ok := config["Cmd"]; ok && cmdVal != nil {
		if arr, ok := cmdVal.([]interface{}); ok && len(arr) > 0 {
			cmd = fmt.Sprintf("%v", arr[0])
		}
	}

	// Layers
	layerCount := 0
	rootFS := getMap(data, "RootFS")
	if layers, ok := rootFS["Layers"]; ok {
		if arr, ok := layers.([]interface{}); ok {
			layerCount = len(arr)
		}
	}

	// Build tree structure
	b.WriteString("Press [J] to toggle raw JSON view\n\n")
	b.WriteString(fmt.Sprintf("Image\n"))
	b.WriteString(fmt.Sprintf("├─ ID       : %s\n", id))
	b.WriteString(fmt.Sprintf("├─ Tag      : %s\n", tags))
	b.WriteString(fmt.Sprintf("├─ Size     : %s\n", size))
	b.WriteString(fmt.Sprintf("├─ Created  : %s\n", created))
	b.WriteString(fmt.Sprintf("│\n"))
	b.WriteString(fmt.Sprintf("├─ Platform\n"))
	b.WriteString(fmt.Sprintf("│   └─ %s\n", platform))
	b.WriteString(fmt.Sprintf("│\n"))
	b.WriteString(fmt.Sprintf("├─ Layers\n"))
	b.WriteString(fmt.Sprintf("│   └─ Count : %d\n", layerCount))
	b.WriteString(fmt.Sprintf("│\n"))
	b.WriteString(fmt.Sprintf("└─ Config\n"))
	b.WriteString(fmt.Sprintf("    ├─ Entrypoint : %s\n", entrypoint))
	b.WriteString(fmt.Sprintf("    └─ Cmd        : %s\n", cmd))

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
