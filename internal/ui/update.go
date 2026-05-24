package ui

import (
	"strings"
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

	case types.ModelListMsg:
		m.models = msg
		if m.activeTab == types.TabModels && m.selectedRow >= len(m.models) && len(m.models) > 0 {
			m.selectedRow = len(m.models) - 1
		}
		return m, nil

	case types.DMRAvailableMsg:
		m.dmrAvailable = bool(msg)
		if m.dmrAvailable {
			return m, m.fetchModelsCmd()
		}
		return m, nil

	case types.ModelSearchMsg:
		// Reuse the existing pull-search UI state for models. Convert
		// model results into the same ImageSearchItem shape so the
		// renderer doesn't need a second code path.
		items := make([]types.ImageSearchItem, 0, len(msg))
		for _, r := range msg {
			items = append(items, types.ImageSearchItem{
				Name:        r.Name,
				Description: r.Description,
				Stars:       r.Stars,
			})
		}
		m.pullSearchResults = items
		m.pullSearchSelected = 0
		m.pullSearchScrollOffset = 0
		m.pullStage = 2
		return m, nil

	case types.ErrMsg:
		m.err = error(msg)
		m.loading = false
		m.actionInProgress = false
		return m, nil

	case types.ActionSuccessMsg:
		m.statusMessage = string(msg)
		m.actionInProgress = false
		m.actionLabel = ""
		m.actionTargetID = ""
		// If we were in the pull flow, return to the list view
		if m.currentView == types.ViewModePullImage || m.currentView == types.ViewModePullModel {
			wasModel := m.currentView == types.ViewModePullModel
			m.currentView = types.ViewModeList
			m.pullStage = 0
			m.pullSearchQuery = ""
			m.pullSearchResults = nil
			m.pullingImageName = ""
			if wasModel {
				return m, m.fetchModelsCmd()
			}
			return m, tea.Batch(m.fetchContainersCmd(), m.fetchImagesCmd())
		}
		// Refresh the active tab — different surfaces have different ground truth
		switch m.activeTab {
		case types.TabModels:
			return m, m.fetchModelsCmd()
		default:
			return m, m.fetchContainersCmd()
		}

	case types.ActionErrorMsg:
		m.statusMessage = "ERROR: " + string(msg)
		m.actionInProgress = false
		m.actionLabel = ""
		m.actionTargetID = ""
		inPullFlow := m.currentView == types.ViewModePullImage || m.currentView == types.ViewModePullModel
		// If a search failed, drop back to the input stage so the user can retry
		if inPullFlow && m.pullStage == 1 {
			m.pullStage = 0
			m.pullSearchError = string(msg)
		}
		// If a pull failed, also return to list view
		if inPullFlow && m.pullStage == 3 {
			m.currentView = types.ViewModeList
			m.pullStage = 0
			m.pullingImageName = ""
		}
		return m, nil

	case types.LogsMsg:
		m.logsContent = string(msg)
		return m, nil

	case types.InspectMsg:
		// Show prettified JSON with jq-style color coding
		m.inspectContent = colorizeJSON(string(msg))
		return m, nil

	case types.ImageSearchMsg:
		m.pullSearchResults = []types.ImageSearchItem(msg)
		m.pullSearchSelected = 0
		m.pullSearchScrollOffset = 0
		m.pullStage = 2
		return m, nil

	case types.TickMsg:
		// Refresh data periodically (only if no action in progress)
		if !m.actionInProgress {
			cmds := []tea.Cmd{
				m.fetchContainersCmd(),
				m.fetchImagesCmd(),
				m.fetchVolumesCmd(),
				m.fetchNetworksCmd(),
				tickCmd(),
			}
			if m.dmrAvailable {
				cmds = append(cmds, m.fetchModelsCmd())
			}
			return m, tea.Batch(cmds...)
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
	key := msg.String()

	// Ctrl+C always works, even while an action is in flight, so the user
	// can quit if a Docker call hangs.
	if key == "ctrl+c" {
		now := time.Now()
		if now.Sub(m.lastCtrlC) < 500*time.Millisecond {
			return m, tea.Quit
		}
		m.lastCtrlC = now
		m.statusMessage = "Press Ctrl+C again to exit"
		return m, nil
	}

	// Global keys (work in all modes)
	switch key {
	case "H", "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "esc":
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
	}

	// Route to appropriate handler based on view
	switch m.currentView {
	case types.ViewModeList:
		return m.handleListViewKeys(msg)
	case types.ViewModeLogs:
		return m.handleLogsViewKeys(msg)
	case types.ViewModeInspect:
		return m.handleInspectViewKeys(msg)
	case types.ViewModePullImage, types.ViewModePullModel:
		return m.handlePullViewKeys(msg)
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
				m.statusMessage = ""

				// Handle delete based on active tab
				switch m.activeTab {
				case 0: // Containers
					if m.selectedRow < len(m.containers) {
						container := m.containers[m.selectedRow]
						m.actionLabel = "Deleting " + container.Name
						m.actionTargetID = container.ID
						return m, m.deleteContainerCmd(container.ID, container.Name)
					}
				case 1: // Images
					if m.selectedRow < len(m.images) {
						image := m.images[m.selectedRow]
						m.actionLabel = "Deleting " + image.Repository + ":" + image.Tag
						m.actionTargetID = image.ID
						return m, m.deleteImageCmd(image.ID)
					}
				case 2: // Volumes
					if m.selectedRow < len(m.volumes) {
						volume := m.volumes[m.selectedRow]
						m.actionLabel = "Deleting " + volume.Name
						m.actionTargetID = volume.Name
						return m, m.deleteVolumeCmd(volume.Name)
					}
				case 3: // Networks
					if m.selectedRow < len(m.networks) {
						network := m.networks[m.selectedRow]
						m.actionLabel = "Deleting " + network.Name
						m.actionTargetID = network.ID
						return m, m.deleteNetworkCmd(network.ID)
					}
				case types.TabModels:
					if m.selectedRow < len(m.models) {
						mod := m.models[m.selectedRow]
						ref := mod.Repository + ":" + mod.Tag
						m.actionLabel = "Deleting " + ref
						m.actionTargetID = ref
						return m, m.deleteModelCmd(ref)
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
	// Navigation — always allowed, even while an action is in progress
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

	case "left", "right", "1", "2", "3", "4", "5":
		return m.handleTabSwitch(key)

	case "enter":
		if m.actionInProgress {
			return m, nil
		}
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
	}

	// All remaining keys trigger actions — block them while one is running
	if m.actionInProgress {
		return m, nil
	}

	switch key {
	// Container actions (only on Containers tab)
	case "s", "S":
		if m.activeTab == types.TabContainers {
			return m.handleContainerStartStop()
		} else if m.activeTab == types.TabImages {
			return m.handleImageStart()
		}
		return m, nil
	case "l", "L":
		if m.activeTab == 0 {
			return m.handleContainerLogs()
		}
		return m, nil
	case "i", "I":
		switch m.activeTab {
		case types.TabContainers:
			return m.handleContainerInspect()
		case types.TabImages:
			return m.handleImageInspect()
		case types.TabVolumes:
			return m.handleVolumeInspect()
		case types.TabNetworks:
			return m.handleNetworkInspect()
		case types.TabModels:
			return m.handleModelInspect()
		}
		return m, nil
	case "d", "D":
		switch m.activeTab {
		case types.TabContainers:
			return m.handleContainerDelete()
		case types.TabImages:
			return m.handleImageDelete()
		case types.TabVolumes:
			return m.handleVolumeDelete()
		case types.TabNetworks:
			return m.handleNetworkDelete()
		case types.TabModels:
			return m.handleModelDelete()
		}
		return m, nil
	case "e", "E":
		if m.activeTab == types.TabContainers {
			return m.handleContainerExec()
		}
		return m, nil

	case "p", "P":
		if m.activeTab == types.TabImages {
			return m.handleImagePullSearch()
		}
		if m.activeTab == types.TabModels {
			return m.handleModelPullSearch()
		}
		return m, nil

	case "r", "R":
		if m.activeTab == types.TabContainers {
			return m.handleContainerRestart()
		}
		if m.activeTab == types.TabModels {
			return m.handleModelRun()
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
			m.activeTab = types.TabModels
		}
	case "right", "l":
		m.activeTab++
		if m.activeTab > types.TabModels {
			m.activeTab = 0
		}
	case "1":
		m.activeTab = types.TabContainers
	case "2":
		m.activeTab = types.TabImages
	case "3":
		m.activeTab = types.TabVolumes
	case "4":
		m.activeTab = types.TabNetworks
	case "5":
		m.activeTab = types.TabModels
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
	case types.TabContainers:
		return len(m.containers)
	case types.TabImages:
		return len(m.images)
	case types.TabVolumes:
		return len(m.volumes)
	case types.TabNetworks:
		return len(m.networks)
	case types.TabModels:
		return len(m.models)
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
		m.actionLabel = "Stopping " + container.Name
		m.actionTargetID = container.ID
		m.statusMessage = ""
		return m, m.stopContainerCmd(container.ID, container.Name)
	} else {
		m.actionInProgress = true
		m.actionLabel = "Starting " + container.Name
		m.actionTargetID = container.ID
		m.statusMessage = ""
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
	m.actionLabel = "Restarting " + container.Name
	m.actionTargetID = container.ID
	m.statusMessage = ""
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

// --- Model handlers (Docker Model Runner) ---

func (m *Model) handleModelInspect() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.models) {
		return m, nil
	}
	mod := m.models[m.selectedRow]
	m.currentView = types.ViewModeInspect
	m.inspectContent = ""
	return m, m.inspectModelCmd(mod.Repository + ":" + mod.Tag)
}

func (m *Model) handleModelDelete() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.models) {
		return m, nil
	}
	m.deleteConfirmMode = !m.deleteConfirmMode
	m.deleteConfirmOption = 1
	return m, nil
}

func (m *Model) handleModelRun() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.models) {
		return m, nil
	}
	mod := m.models[m.selectedRow]
	return m, m.runModelCmd(mod.Repository + ":" + mod.Tag)
}

func (m *Model) handleModelPullSearch() (tea.Model, tea.Cmd) {
	m.currentView = types.ViewModePullModel
	m.pullStage = 0
	m.pullSearchQuery = ""
	m.pullSearchResults = nil
	m.pullSearchSelected = 0
	m.pullSearchScrollOffset = 0
	m.pullSearchError = ""
	m.statusMessage = ""
	return m, nil
}

// handleImagePullSearch enters the pull-from-Hub flow
func (m *Model) handleImagePullSearch() (tea.Model, tea.Cmd) {
	m.currentView = types.ViewModePullImage
	m.pullStage = 0
	m.pullSearchQuery = ""
	m.pullSearchResults = nil
	m.pullSearchSelected = 0
	m.pullSearchScrollOffset = 0
	m.pullSearchError = ""
	m.statusMessage = ""
	return m, nil
}

// handlePullViewKeys handles input in the pull image search flow
func (m *Model) handlePullViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Esc always returns to the list view (but not mid-pull — Docker API
	// can't cancel an in-progress pull, so don't pretend we did)
	if key == "esc" && m.pullStage != 3 {
		m.currentView = types.ViewModeList
		m.pullStage = 0
		m.pullSearchQuery = ""
		m.pullSearchResults = nil
		m.pullSearchError = ""
		return m, nil
	}

	switch m.pullStage {
	case 0: // input
		switch key {
		case "enter":
			q := strings.TrimSpace(m.pullSearchQuery)
			if q == "" {
				return m, nil
			}
			m.pullStage = 1
			m.pullSearchError = ""
			if m.currentView == types.ViewModePullModel {
				return m, m.searchModelsCmd(q)
			}
			return m, m.searchImagesCmd(q)
		case "backspace":
			if len(m.pullSearchQuery) > 0 {
				m.pullSearchQuery = m.pullSearchQuery[:len(m.pullSearchQuery)-1]
			}
			return m, nil
		default:
			// Accept printable characters
			if len(key) == 1 {
				r := key[0]
				if r >= 0x20 && r < 0x7f {
					m.pullSearchQuery += key
				}
			}
			return m, nil
		}

	case 1: // searching — only esc handled above
		return m, nil

	case 2: // results
		switch key {
		case "up", "k":
			if m.pullSearchSelected > 0 {
				m.pullSearchSelected--
				if m.pullSearchSelected < m.pullSearchScrollOffset {
					m.pullSearchScrollOffset = m.pullSearchSelected
				}
			}
			return m, nil
		case "down", "j":
			if m.pullSearchSelected < len(m.pullSearchResults)-1 {
				m.pullSearchSelected++
				visible := m.pullResultsViewportHeight()
				if m.pullSearchSelected >= m.pullSearchScrollOffset+visible {
					m.pullSearchScrollOffset = m.pullSearchSelected - visible + 1
				}
			}
			return m, nil
		case "enter":
			if m.pullSearchSelected >= len(m.pullSearchResults) {
				return m, nil
			}
			img := m.pullSearchResults[m.pullSearchSelected]
			m.pullStage = 3
			m.pullingImageName = img.Name
			m.actionInProgress = true
			m.statusMessage = "Pulling " + img.Name + "..."
			if m.currentView == types.ViewModePullModel {
				return m, m.pullModelCmd(img.Name)
			}
			return m, m.pullSearchCompleteCmd(img.Name)
		}
		return m, nil

	case 3: // pulling — ignore input
		return m, nil
	}

	return m, nil
}

// pullResultsViewportHeight returns how many result rows can fit on screen
func (m *Model) pullResultsViewportHeight() int {
	// height - header(1) - divider(1) - query line(1) - results header(2) - action bar(3) - margin(2)
	h := m.height - 10
	if h < 3 {
		h = 3
	}
	return h
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
