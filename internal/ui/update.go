package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"tinyd/internal/types"
)

// Update handles all state transitions
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case types.ContainerListMsg:
		m.containers = msg
		m.loading = false
		m.actionInProgress = false
		// Keep selection in bounds
		if m.activeTab == 0 && m.selectedRow >= len(m.containers) && len(m.containers) > 0 {
			m.selectedRow = len(m.containers) - 1
		}
		return m, nil

	case types.ImageListMsg:
		m.images = msg
		// Keep selection in bounds
		if m.activeTab == 1 && m.selectedRow >= len(m.images) && len(m.images) > 0 {
			m.selectedRow = len(m.images) - 1
		}
		return m, nil

	case types.VolumeListMsg:
		m.volumes = msg
		// Keep selection in bounds
		if m.activeTab == 2 && m.selectedRow >= len(m.volumes) && len(m.volumes) > 0 {
			m.selectedRow = len(m.volumes) - 1
		}
		return m, nil

	case types.NetworkListMsg:
		m.networks = msg
		// Keep selection in bounds
		if m.activeTab == 3 && m.selectedRow >= len(m.networks) && len(m.networks) > 0 {
			m.selectedRow = len(m.networks) - 1
		}
		return m, nil

	case types.ErrMsg:
		m.err = error(msg)
		m.loading = false
		m.actionInProgress = false
		return m, nil

	case types.ActionSuccessMsg:
		m.statusMessage = string(msg)
		m.actionInProgress = false
		// Refresh data after successful action
		return m, m.fetchContainersCmd()

	case types.ActionErrorMsg:
		m.statusMessage = "ERROR: " + string(msg)
		m.actionInProgress = false
		return m, nil

	case types.LogsMsg:
		m.logsContent = string(msg)
		return m, nil

	case types.InspectMsg:
		m.inspectContent = string(msg)
		return m, nil

	case types.TickMsg:
		// Refresh data periodically (only if no action in progress)
		if !m.actionInProgress {
			return m, tea.Batch(
				m.fetchContainersCmd(),
				m.fetchImagesCmd(),
				m.fetchVolumesCmd(),
				m.fetchNetworksCmd(),
				tickCmd(),
			)
		}
		return m, tickCmd()

	case types.AnimationTickMsg:
		// Update animation frame for status indicators
		m.animationFrame = (m.animationFrame + 1) % 4
		return m, animationTickCmd()
	}

	return m, nil
}

// handleResize adjusts viewport when terminal size changes
func (m *Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Calculate viewport height for scrollable content
	// Fixed UI elements:
	// - Tabs: 4 lines
	// - Table header: 2 lines
	// - Action bar: 3 lines
	// - Buffer: 1 line
	fixedLines := 10
	m.viewportHeight = msg.Height - fixedLines
	if m.viewportHeight < 3 {
		m.viewportHeight = 3 // Minimum 3 visible rows
	}

	// Update component dimensions
	m.header = m.header.WithWidth(m.width)
	m.tabs = m.tabs.WithWidth(m.width)
	m.actionBar = m.actionBar.WithWidth(m.width)
	m.detailView = m.detailView.WithWidth(m.width)

	// Keep scroll position valid after resize
	maxRow := m.getMaxRow()
	if m.selectedRow >= maxRow && maxRow > 0 {
		m.selectedRow = maxRow - 1
	}
	if m.scrollOffset > maxRow-m.viewportHeight && maxRow > m.viewportHeight {
		m.scrollOffset = maxRow - m.viewportHeight
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	return m, nil
}

// handleKeyPress routes keypresses based on current state
// This is a simplified version - the full implementation would handle all keys
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Block input during actions
	if m.actionInProgress {
		return m, nil
	}

	key := msg.String()

	// Global keys (work in all modes)
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "F1", "?":
		m.showHelp = !m.showHelp
		return m, nil
	}

	// Route to appropriate handler based on view
	switch m.currentView {
	case types.ViewModeList:
		return m.handleListViewKeys(msg)
	case types.ViewModeLogs:
		return m.handleLogsViewKeys(msg)
	case types.ViewModeInspect:
		return m.handleInspectViewKeys(msg)
	default:
		return m, nil
	}
}

// handleListViewKeys processes input in list view
// This is a simplified version showing the pattern
func (m *Model) handleListViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "up", "k":
		if m.selectedRow > 0 {
			m.selectedRow--
			if m.selectedRow < m.scrollOffset {
				m.scrollOffset = m.selectedRow
			}
		}
		return m, nil

	case "down", "j":
		maxRow := m.getMaxRow()
		if m.selectedRow < maxRow-1 {
			m.selectedRow++
			if m.selectedRow >= m.scrollOffset+m.viewportHeight {
				m.scrollOffset = m.selectedRow - m.viewportHeight + 1
			}
		}
		return m, nil

	case "left", "h", "right", "l", "1", "2", "3", "4":
		return m.handleTabSwitch(key)

	case "enter":
		// Refresh on enter
		switch m.activeTab {
		case 0:
			return m, m.fetchContainersCmd()
		case 1:
			return m, m.fetchImagesCmd()
		case 2:
			return m, m.fetchVolumesCmd()
		case 3:
			return m, m.fetchNetworksCmd()
		}
		return m, nil

	default:
		return m, nil
	}
}

// handleTabSwitch switches between tabs
func (m *Model) handleTabSwitch(key string) (tea.Model, tea.Cmd) {
	oldTab := m.activeTab

	switch key {
	case "left", "h":
		m.activeTab--
		if m.activeTab < 0 {
			m.activeTab = 3
		}
	case "right", "l":
		m.activeTab++
		if m.activeTab > 3 {
			m.activeTab = 0
		}
	case "1":
		m.activeTab = 0
	case "2":
		m.activeTab = 1
	case "3":
		m.activeTab = 2
	case "4":
		m.activeTab = 3
	}

	if m.activeTab != oldTab {
		m.selectedRow = 0
		m.scrollOffset = 0
		m.tabs = m.tabs.SetActiveTab(m.activeTab)
	}

	return m, nil
}

// handleLogsViewKeys processes input in logs view
func (m *Model) handleLogsViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.currentView = types.ViewModeList
		m.logsContent = ""
		return m, nil

	case "up", "k":
		if m.logsScrollOffset > 0 {
			m.logsScrollOffset--
		}
		return m, nil

	case "down", "j":
		m.logsScrollOffset++
		return m, nil

	default:
		return m, nil
	}
}

// handleInspectViewKeys processes input in inspect view
func (m *Model) handleInspectViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.currentView = types.ViewModeList
		m.inspectContent = ""
		return m, nil

	default:
		return m, nil
	}
}

// getMaxRow returns the number of items in the current tab
func (m *Model) getMaxRow() int {
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
