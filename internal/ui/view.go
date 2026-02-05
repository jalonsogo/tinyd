package ui

import (
	"fmt"
	"strings"

	"tinyd/internal/types"
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

	// Render tabs
	b.WriteString(m.tabs.View())
	b.WriteString("\n")

	// Render content based on active tab
	switch m.activeTab {
	case 0:
		b.WriteString(m.renderContainersTab())
	case 1:
		b.WriteString(m.renderImagesTab())
	case 2:
		b.WriteString(m.renderVolumesTab())
	case 3:
		b.WriteString(m.renderNetworksTab())
	}

	// Render action bar
	b.WriteString("\n")
	m.actionBar = m.actionBar.SetStatusMessage(m.statusMessage)
	b.WriteString(m.actionBar.View())

	return b.String()
}

// renderContainersTab renders the containers tab
func (m *Model) renderContainersTab() string {
	if len(m.containers) == 0 {
		if m.loading {
			return "Loading containers..."
		}
		return "No containers found"
	}

	var b strings.Builder

	// Simple list rendering (simplified from original)
	start := m.scrollOffset
	end := m.scrollOffset + m.viewportHeight
	if end > len(m.containers) {
		end = len(m.containers)
	}

	for i := start; i < end; i++ {
		c := m.containers[i]
		selected := ""
		if i == m.selectedRow {
			selected = "> "
		} else {
			selected = "  "
		}

		b.WriteString(fmt.Sprintf("%s%-12s %-20s %-10s %s\n",
			selected,
			c.ID,
			truncate(c.Name, 20),
			c.Status,
			c.Image,
		))
	}

	return b.String()
}

// renderImagesTab renders the images tab
func (m *Model) renderImagesTab() string {
	if len(m.images) == 0 {
		if m.loading {
			return "Loading images..."
		}
		return "No images found"
	}

	var b strings.Builder

	start := m.scrollOffset
	end := m.scrollOffset + m.viewportHeight
	if end > len(m.images) {
		end = len(m.images)
	}

	for i := start; i < end; i++ {
		img := m.images[i]
		selected := ""
		if i == m.selectedRow {
			selected = "> "
		} else {
			selected = "  "
		}

		b.WriteString(fmt.Sprintf("%s%-12s %-30s %-10s %s\n",
			selected,
			img.ID,
			truncate(img.Repository, 30),
			img.Tag,
			img.Size,
		))
	}

	return b.String()
}

// renderVolumesTab renders the volumes tab
func (m *Model) renderVolumesTab() string {
	if len(m.volumes) == 0 {
		if m.loading {
			return "Loading volumes..."
		}
		return "No volumes found"
	}

	var b strings.Builder

	start := m.scrollOffset
	end := m.scrollOffset + m.viewportHeight
	if end > len(m.volumes) {
		end = len(m.volumes)
	}

	for i := start; i < end; i++ {
		vol := m.volumes[i]
		selected := ""
		if i == m.selectedRow {
			selected = "> "
		} else {
			selected = "  "
		}

		inUse := "No"
		if vol.InUse {
			inUse = "Yes"
		}

		b.WriteString(fmt.Sprintf("%s%-25s %-10s %s\n",
			selected,
			truncate(vol.Name, 25),
			vol.Driver,
			inUse,
		))
	}

	return b.String()
}

// renderNetworksTab renders the networks tab
func (m *Model) renderNetworksTab() string {
	if len(m.networks) == 0 {
		if m.loading {
			return "Loading networks..."
		}
		return "No networks found"
	}

	var b strings.Builder

	start := m.scrollOffset
	end := m.scrollOffset + m.viewportHeight
	if end > len(m.networks) {
		end = len(m.networks)
	}

	for i := start; i < end; i++ {
		net := m.networks[i]
		selected := ""
		if i == m.selectedRow {
			selected = "> "
		} else {
			selected = "  "
		}

		inUse := "No"
		if net.InUse {
			inUse = "Yes"
		}

		b.WriteString(fmt.Sprintf("%s%-12s %-20s %-10s %s\n",
			selected,
			net.ID,
			truncate(net.Name, 20),
			net.Driver,
			inUse,
		))
	}

	return b.String()
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
