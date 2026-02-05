package ui

import (
	"time"

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
		// Show prettified JSON with jq-style color coding
		m.inspectContent = colorizeJSON(string(msg))
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
	// - Scroll indicator: 2 lines
	// - Buffer: 1 line
	fixedLines := 12
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
	case "ctrl+c":
		// Double Ctrl+C to exit
		now := time.Now()
		if now.Sub(m.lastCtrlC) < 500*time.Millisecond {
			return m, tea.Quit
		}
		m.lastCtrlC = now
		m.statusMessage = "Press Ctrl+C again to exit"
		return m, nil
	case "H", "?":
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
func (m *Model) handleListViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle delete confirmation mode
	if m.deleteConfirmMode {
		switch key {
		case "left", "h":
			m.deleteConfirmOption = 0 // Yes
			return m, nil
		case "right", "l":
			m.deleteConfirmOption = 1 // No
			return m, nil
		case "enter":
			if m.deleteConfirmOption == 0 {
				// User confirmed delete
				m.deleteConfirmMode = false
				m.actionInProgress = true

				// Handle delete based on active tab
				switch m.activeTab {
				case 0: // Containers
					if m.selectedRow < len(m.containers) {
						container := m.containers[m.selectedRow]
						return m, m.deleteContainerCmd(container.ID, container.Name)
					}
				case 1: // Images
					if m.selectedRow < len(m.images) {
						image := m.images[m.selectedRow]
						return m, m.deleteImageCmd(image.ID)
					}
				case 2: // Volumes
					if m.selectedRow < len(m.volumes) {
						volume := m.volumes[m.selectedRow]
						return m, m.deleteVolumeCmd(volume.Name)
					}
				case 3: // Networks
					if m.selectedRow < len(m.networks) {
						network := m.networks[m.selectedRow]
						return m, m.deleteNetworkCmd(network.ID)
					}
				}
			}
			// User cancelled or selected No
			m.deleteConfirmMode = false
			return m, nil
		case "esc":
			m.deleteConfirmMode = false
			return m, nil
		}
		return m, nil
	}

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

	case "left", "h", "right", "1", "2", "3", "4":
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

	// Container actions (only on Containers tab)
	case "s", "S":
		if m.activeTab == 0 {
			return m.handleContainerStartStop()
		} else if m.activeTab == 1 {
			return m.handleImageStart()
		}
		return m, nil
	case "r", "R":
		if m.activeTab == 0 {
			return m.handleContainerRestart()
		}
		return m, nil
	case "l", "L":
		if m.activeTab == 0 {
			return m.handleContainerLogs()
		}
		return m, nil
	case "i", "I":
		switch m.activeTab {
		case 0: // Containers
			return m.handleContainerInspect()
		case 1: // Images
			return m.handleImageInspect()
		case 2: // Volumes
			return m.handleVolumeInspect()
		case 3: // Networks
			return m.handleNetworkInspect()
		}
		return m, nil
	case "d", "D":
		switch m.activeTab {
		case 0: // Containers
			return m.handleContainerDelete()
		case 1: // Images
			return m.handleImageDelete()
		case 2: // Volumes
			return m.handleVolumeDelete()
		case 3: // Networks
			return m.handleNetworkDelete()
		}
		return m, nil
	case "e", "E":
		if m.activeTab == 0 {
			return m.handleContainerExec()
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

// Container action handlers

func (m *Model) handleContainerStartStop() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.containers) {
		return m, nil
	}
	container := m.containers[m.selectedRow]

	// Toggle start/stop based on current status
	if container.Status == "RUNNING" {
		m.actionInProgress = true
		return m, m.stopContainerCmd(container.ID, container.Name)
	} else {
		m.actionInProgress = true
		return m, m.startContainerCmd(container.ID, container.Name)
	}
}

func (m *Model) handleContainerRestart() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.containers) {
		return m, nil
	}
	container := m.containers[m.selectedRow]

	// Only restart if running
	if container.Status != "RUNNING" {
		m.statusMessage = "Container must be running to restart"
		return m, nil
	}

	m.actionInProgress = true
	return m, m.restartContainerCmd(container.ID, container.Name)
}

func (m *Model) handleContainerLogs() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.containers) {
		return m, nil
	}
	container := m.containers[m.selectedRow]
	m.selectedContainer = &container
	m.currentView = types.ViewModeLogs
	m.logsContent = ""
	m.logsScrollOffset = 0
	return m, m.getContainerLogsCmd(container.ID)
}

func (m *Model) handleContainerInspect() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.containers) {
		return m, nil
	}
	container := m.containers[m.selectedRow]
	m.selectedContainer = &container
	m.currentView = types.ViewModeInspect
	m.inspectContent = ""
	return m, m.inspectContainerCmd(container.ID)
}

func (m *Model) handleContainerDelete() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.containers) {
		return m, nil
	}
	// Toggle delete confirmation mode
	m.deleteConfirmMode = !m.deleteConfirmMode
	m.deleteConfirmOption = 1 // Default to "No"
	return m, nil
}

func (m *Model) handleContainerExec() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.containers) {
		return m, nil
	}
	container := m.containers[m.selectedRow]

	// Only allow exec on running containers
	if container.Status != "RUNNING" {
		m.statusMessage = "Container must be running to exec"
		return m, nil
	}

	// Create exec command for interactive shell
	// The command will suspend the TUI and run in the foreground
	return m, m.execContainerCmd(container.ID)
}

// Image action handlers

func (m *Model) handleImageStart() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.images) {
		return m, nil
	}
	image := m.images[m.selectedRow]
	m.selectedImage = &image

	// Start the image (create and run a container from it)
	// For now, use simple defaults - can be expanded to a modal later
	return m, m.runContainerCmd()
}

func (m *Model) handleImageInspect() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.images) {
		return m, nil
	}
	image := m.images[m.selectedRow]
	m.selectedImage = &image
	m.currentView = types.ViewModeInspect
	m.inspectContent = ""
	return m, m.inspectImageCmd(image.ID)
}

func (m *Model) handleImageDelete() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.images) {
		return m, nil
	}
	// Toggle delete confirmation mode
	m.deleteConfirmMode = !m.deleteConfirmMode
	m.deleteConfirmOption = 1 // Default to "No"
	return m, nil
}

// Volume action handlers

func (m *Model) handleVolumeInspect() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.volumes) {
		return m, nil
	}
	volume := m.volumes[m.selectedRow]
	m.selectedVolume = &volume
	m.currentView = types.ViewModeInspect
	m.inspectContent = ""
	return m, m.inspectVolumeCmd(volume.Name)
}

func (m *Model) handleVolumeDelete() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.volumes) {
		return m, nil
	}
	// Toggle delete confirmation mode
	m.deleteConfirmMode = !m.deleteConfirmMode
	m.deleteConfirmOption = 1 // Default to "No"
	return m, nil
}

// Network action handlers

func (m *Model) handleNetworkInspect() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.networks) {
		return m, nil
	}
	network := m.networks[m.selectedRow]
	m.selectedNetwork = &network
	m.currentView = types.ViewModeInspect
	m.inspectContent = ""
	return m, m.inspectNetworkCmd(network.ID)
}

func (m *Model) handleNetworkDelete() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.networks) {
		return m, nil
	}
	// Toggle delete confirmation mode
	m.deleteConfirmMode = !m.deleteConfirmMode
	m.deleteConfirmOption = 1 // Default to "No"
	return m, nil
}
