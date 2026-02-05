package ui

import (
	"fmt"
	"strings"

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

	// Calculate responsive column widths
	availWidth := m.width - 4 // Account for padding
	if availWidth < 50 {
		availWidth = 50 // Minimum width
	}

	// For narrow terminals, simplify the layout
	var headers []components.TableHeader
	if availWidth >= 90 {
		// Full layout for wide terminals
		headers = []components.TableHeader{
			{Label: "", Width: 2, AlignRight: false},
			{Label: "NAME", Width: availWidth * 30 / 100, AlignRight: false},
			{Label: "IMAGE", Width: availWidth * 25 / 100, AlignRight: false},
			{Label: "STATUS", Width: 10, AlignRight: false},
			{Label: "CPU", Width: 8, AlignRight: true},
			{Label: "MEM", Width: 8, AlignRight: true},
			{Label: "PORTS", Width: 15, AlignRight: false},
		}
	} else {
		// Compact layout for narrow terminals (60 cols)
		headers = []components.TableHeader{
			{Label: "", Width: 2, AlignRight: false},
			{Label: "NAME", Width: availWidth * 40 / 100, AlignRight: false},
			{Label: "IMAGE", Width: availWidth * 35 / 100, AlignRight: false},
			{Label: "STATUS", Width: availWidth * 25 / 100, AlignRight: false},
		}
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

		var cells []string
		if availWidth >= 90 {
			// Full layout
			cells = []string{
				m.getStatusDot(c.Status),
				truncateWithEllipsis(c.Name, headers[1].Width),
				truncateWithEllipsis(c.Image, headers[2].Width),
				c.Status,
				c.CPU,
				c.Mem,
				truncateWithEllipsis(c.Ports, 15),
			}
		} else {
			// Compact layout
			cells = []string{
				m.getStatusDot(c.Status),
				truncateWithEllipsis(c.Name, headers[1].Width),
				truncateWithEllipsis(c.Image, headers[2].Width),
				c.Status,
			}
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

	// Calculate responsive column widths
	availWidth := m.width - 4
	if availWidth < 50 {
		availWidth = 50
	}

	var headers []components.TableHeader
	if availWidth >= 90 {
		headers = []components.TableHeader{
			{Label: "", Width: 2, AlignRight: false},
			{Label: "REPOSITORY", Width: availWidth * 40 / 100, AlignRight: false},
			{Label: "TAG", Width: availWidth * 20 / 100, AlignRight: false},
			{Label: "SIZE", Width: 10, AlignRight: true},
			{Label: "CREATED", Width: 15, AlignRight: false},
		}
	} else {
		headers = []components.TableHeader{
			{Label: "", Width: 2, AlignRight: false},
			{Label: "REPOSITORY", Width: availWidth * 50 / 100, AlignRight: false},
			{Label: "TAG", Width: availWidth * 30 / 100, AlignRight: false},
			{Label: "SIZE", Width: availWidth * 20 / 100, AlignRight: true},
		}
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

		var cells []string
		if availWidth >= 90 {
			cells = []string{
				grayStyle.Render("○"),
				truncateWithEllipsis(img.Repository, headers[1].Width),
				truncateWithEllipsis(img.Tag, headers[2].Width),
				img.Size,
				truncateWithEllipsis(img.Created, 15),
			}
		} else {
			cells = []string{
				grayStyle.Render("○"),
				truncateWithEllipsis(img.Repository, headers[1].Width),
				truncateWithEllipsis(img.Tag, headers[2].Width),
				img.Size,
			}
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

	availWidth := m.width - 4
	if availWidth < 50 {
		availWidth = 50
	}

	var headers []components.TableHeader
	if availWidth >= 90 {
		headers = []components.TableHeader{
			{Label: "", Width: 2, AlignRight: false},
			{Label: "NAME", Width: availWidth * 35 / 100, AlignRight: false},
			{Label: "DRIVER", Width: 12, AlignRight: false},
			{Label: "IN USE", Width: 8, AlignRight: false},
			{Label: "MOUNT POINT", Width: availWidth * 40 / 100, AlignRight: false},
		}
	} else {
		headers = []components.TableHeader{
			{Label: "", Width: 2, AlignRight: false},
			{Label: "NAME", Width: availWidth * 60 / 100, AlignRight: false},
			{Label: "DRIVER", Width: availWidth * 20 / 100, AlignRight: false},
			{Label: "IN USE", Width: availWidth * 20 / 100, AlignRight: false},
		}
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

		inUse := "No"
		if vol.InUse {
			inUse = "Yes"
		}

		var cells []string
		if availWidth >= 90 {
			cells = []string{
				statusDot,
				truncateWithEllipsis(vol.Name, headers[1].Width),
				vol.Driver,
				inUse,
				truncateWithEllipsis(vol.Mountpoint, headers[4].Width),
			}
		} else {
			cells = []string{
				statusDot,
				truncateWithEllipsis(vol.Name, headers[1].Width),
				vol.Driver,
				inUse,
			}
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

	availWidth := m.width - 4
	if availWidth < 50 {
		availWidth = 50
	}

	var headers []components.TableHeader
	if availWidth >= 90 {
		headers = []components.TableHeader{
			{Label: "", Width: 2, AlignRight: false},
			{Label: "NAME", Width: availWidth * 30 / 100, AlignRight: false},
			{Label: "DRIVER", Width: 12, AlignRight: false},
			{Label: "SCOPE", Width: 10, AlignRight: false},
			{Label: "IN USE", Width: 8, AlignRight: false},
			{Label: "IPv4", Width: availWidth * 25 / 100, AlignRight: false},
		}
	} else {
		headers = []components.TableHeader{
			{Label: "", Width: 2, AlignRight: false},
			{Label: "NAME", Width: availWidth * 50 / 100, AlignRight: false},
			{Label: "DRIVER", Width: availWidth * 30 / 100, AlignRight: false},
			{Label: "IN USE", Width: availWidth * 20 / 100, AlignRight: false},
		}
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

		inUse := "No"
		if net.InUse {
			inUse = "Yes"
		}

		var cells []string
		if availWidth >= 90 {
			cells = []string{
				statusDot,
				truncateWithEllipsis(net.Name, headers[1].Width),
				net.Driver,
				net.Scope,
				inUse,
				truncateWithEllipsis(net.IPv4, headers[5].Width),
			}
		} else {
			cells = []string{
				statusDot,
				truncateWithEllipsis(net.Name, headers[1].Width),
				net.Driver,
				inUse,
			}
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
	animFrames := []string{"◴", "◷", "◶", "◵"}

	switch status {
	case "RUNNING":
		return greenStyle.Render("●") // Green filled circle for running
	case "STOPPED":
		return grayStyle.Render("○") // Gray empty circle for stopped
	case "PAUSED":
		return yellowStyle.Render("○") // Yellow empty circle for paused
	case "ERROR":
		return redStyle.Render("◌") // Red empty circle for error
	case "RESTARTING":
		// Animated pulsating green circles
		return greenStyle.Render(animFrames[m.animationFrame])
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
	confirmStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFF00")).
		Background(lipgloss.Color("#0a0a0a")).
		Bold(true)

	yesStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Background(lipgloss.Color("#0a0a0a"))

	noStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Background(lipgloss.Color("#0a0a0a"))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	var b strings.Builder
	b.WriteString(confirmStyle.Render("Delete " + truncateWithEllipsis(name, 30) + "? "))

	if selectedOption == 0 {
		b.WriteString(yesStyle.Render("[Yes]"))
		b.WriteString(normalStyle.Render(" No"))
	} else {
		b.WriteString(normalStyle.Render("Yes "))
		b.WriteString(noStyle.Render("[No]"))
	}

	return b.String()
}

// getActionShortcuts returns the keyboard shortcuts for the current tab
func (m *Model) getActionShortcuts() string {
	var shortcuts []string

	switch m.activeTab {
	case 0: // Containers
		shortcuts = []string{
			renderShortcut("S", "tart"),
			renderShortcut("S", "t", "op"),
			renderShortcut("R", "estart"),
			renderShortcut("L", "ogs"),
			renderShortcut("E", "xec"),
			renderShortcut("I", "nspect"),
			renderShortcut("D", "elete"),
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
		renderShortcut("Q", "uit"),
	)

	return strings.Join(shortcuts, " ")
}

// renderShortcut formats a keyboard shortcut with highlighted key
func renderShortcut(key string, rest ...string) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")).
		Background(lipgloss.Color("#0a0a0a")).
		Bold(true)

	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	var b strings.Builder
	b.WriteString("[")
	b.WriteString(keyStyle.Render(key))
	b.WriteString("]")
	if len(rest) > 0 {
		b.WriteString(textStyle.Render(strings.Join(rest, "")))
	}

	return b.String()
}
