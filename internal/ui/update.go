package ui

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sort"
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
		m.err = nil // any list arriving means the daemon is back up
		// Keep selection in bounds
		if m.activeTab == types.TabContainers && m.selectedRow >= len(m.containers) && len(m.containers) > 0 {
			m.selectedRow = len(m.containers) - 1
		}
		return m, nil

	case types.ImageListMsg:
		m.images = msg
		m.err = nil
		// Keep selection in bounds
		if m.activeTab == types.TabImages && m.selectedRow >= len(m.images) && len(m.images) > 0 {
			m.selectedRow = len(m.images) - 1
		}
		return m, nil

	case types.VolumeListMsg:
		m.volumes = msg
		m.err = nil
		// Keep selection in bounds
		if m.activeTab == types.TabVolumes && m.selectedRow >= len(m.volumes) && len(m.volumes) > 0 {
			m.selectedRow = len(m.volumes) - 1
		}
		return m, nil

	case types.NetworkListMsg:
		m.networks = msg
		m.err = nil
		// Keep selection in bounds
		if m.activeTab == types.TabNetworks && m.selectedRow >= len(m.networks) && len(m.networks) > 0 {
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
				Name:          r.Name,
				Description:   r.Description,
				Stars:         r.Stars,
				Architectures: r.Architectures,
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

	case types.ModelTagsMsg:
		m.tagPickerLoading = false
		if msg.Repo != m.tagPickerRepo {
			// Late response — user picked a different repo or backed out.
			return m, nil
		}
		m.tagPickerTags = msg.Tags
		m.tagPickerIndex = 0
		m.tagPickerScroll = 0
		return m, nil

	case types.ChatStartedMsg:
		// Store the live reader/body on the model and kick off the
		// chunk-reading loop. Cast back from interface{} (types/ keeps
		// the message struct free of bufio/io imports).
		if r, ok := msg.Reader.(*bufio.Reader); ok {
			m.chatReader = r
		}
		if b, ok := msg.Body.(io.Closer); ok {
			m.chatBody = b
		}
		return m, m.readChatChunkCmd()

	case types.ChatTokenMsg:
		if msg.Err != "" {
			m.closeChatStream()
			m.chatError = msg.Err
			m.chatStreaming = false
			return m, nil
		}
		if msg.Done {
			// Commit the assistant message and reset live state.
			if m.chatCurrentResponse != "" {
				m.chatMessages = append(m.chatMessages, types.ChatMessage{
					Role: "assistant", Content: m.chatCurrentResponse,
				})
			}
			m.chatCurrentResponse = ""
			m.chatStreaming = false
			m.closeChatStream()
			return m, nil
		}
		m.chatCurrentResponse += msg.Token
		return m, m.readChatChunkCmd()

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

	// q quits when the error screen is showing (no other context uses q yet
	// — the regular list view doesn't bind it, so this won't shadow anything).
	if m.err != nil && (key == "q" || key == "Q") {
		return m, tea.Quit
	}

	// Global keys (work in all modes). The help toggle is skipped in
	// text-input views so typing "Hello", "?", etc. doesn't trigger it.
	inputView := m.currentView == types.ViewModeChat ||
		m.currentView == types.ViewModeRunImage ||
		m.currentView == types.ViewModeRunVolumePicker ||
		m.currentView == types.ViewModePullImage ||
		m.currentView == types.ViewModePullModel ||
		(m.currentView == types.ViewModeList && m.listSearchMode)

	if !inputView {
		switch key {
		case "H", "?":
			m.showHelp = !m.showHelp
			return m, nil
		}
	}
	if key == "esc" && m.showHelp {
		m.showHelp = false
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
	case types.ViewModePullImage, types.ViewModePullModel:
		return m.handlePullViewKeys(msg)
	case types.ViewModeRunImage:
		return m.handleRunModalKeys(msg)
	case types.ViewModeRunVolumePicker:
		return m.handleVolumePickerKeys(msg)
	case types.ViewModeRunFileBrowser:
		return m.handleFileBrowserKeys(msg)
	case types.ViewModeChat:
		return m.handleChatViewKeys(msg)
	case types.ViewModeModelTagPicker:
		return m.handleTagPickerKeys(msg)
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
				case types.TabContainers:
					if m.selectedRow < len(m.containers) {
						container := m.containers[m.selectedRow]
						m.actionLabel = "Deleting " + container.Name
						m.actionTargetID = container.ID
						return m, m.deleteContainerCmd(container.ID, container.Name)
					}
				case types.TabImages:
					if m.selectedRow < len(m.images) {
						image := m.images[m.selectedRow]
						m.actionLabel = "Deleting " + image.Repository + ":" + image.Tag
						m.actionTargetID = image.ID
						return m, m.deleteImageCmd(image.ID)
					}
				case types.TabVolumes:
					if m.selectedRow < len(m.volumes) {
						volume := m.volumes[m.selectedRow]
						m.actionLabel = "Deleting " + volume.Name
						m.actionTargetID = volume.Name
						return m, m.deleteVolumeCmd(volume.Name)
					}
				case types.TabNetworks:
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
		}
		return m, nil
	case "l", "L":
		if m.activeTab == types.TabContainers {
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

	case "r":
		if m.activeTab == types.TabContainers {
			return m.handleContainerRestart()
		}
		if m.activeTab == types.TabImages {
			return m.handleImageStart()
		}
		if m.activeTab == types.TabModels {
			// Plain r on Models is the shell REPL fallback (same key as
			// Shift+R, kept for muscle memory). The in-app chat lives on c.
			return m.handleModelRun()
		}
		return m, nil
	case "R":
		if m.activeTab == types.TabContainers {
			return m.handleContainerRestart()
		}
		if m.activeTab == types.TabImages {
			return m.handleImageStart()
		}
		if m.activeTab == types.TabModels {
			return m.handleModelRun()
		}
		return m, nil
	case "c", "C":
		if m.activeTab == types.TabModels {
			return m.handleModelChat()
		}
		return m, nil

	case "u", "U":
		if m.activeTab == types.TabImages {
			return m.handleImageUpdate()
		}
		return m, nil

	case "y", "Y":
		// Yank the curl example to the clipboard. Models tab only.
		if m.activeTab == types.TabModels {
			curl := m.currentCurlExample()
			return m, m.copyToClipboardCmd(curl, "curl example")
		}
		return m, nil

	default:
		return m, nil
	}
}

// handleTabSwitch switches between tabs
func (m *Model) handleTabSwitch(key string) (tea.Model, tea.Cmd) {
	oldTab := m.activeTab

	const lastTab = types.TabNetworks
	switch key {
	case "left", "h":
		m.activeTab--
		if m.activeTab < 0 {
			m.activeTab = lastTab
		}
	case "right", "l":
		m.activeTab++
		if m.activeTab > lastTab {
			m.activeTab = types.TabContainers
		}
	case "1":
		m.activeTab = types.TabContainers
	case "2":
		m.activeTab = types.TabImages
	case "3":
		m.activeTab = types.TabModels
	case "4":
		m.activeTab = types.TabVolumes
	case "5":
		m.activeTab = types.TabNetworks
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

	// Open the interactive Run modal — user fills in name/ports/volumes/env
	// then Ctrl+R submits.
	m.currentView = types.ViewModeRunImage
	m.runContainerName = ""
	m.runPorts = []types.PortMapping{}
	m.runVolumes = []types.VolumeMapping{}
	m.runEnvVars = []types.EnvVar{}
	m.runPortInput = ""
	m.runEnvInput = ""
	m.runModalField = types.RunFieldName
	m.statusMessage = ""
	return m, nil
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

// handleImageUpdate re-pulls the selected image's repo:tag to update it to
// the latest version published under that tag. No Hub search — just a fresh
// pull of what's already there.
func (m *Model) handleImageUpdate() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.images) {
		return m, nil
	}
	image := m.images[m.selectedRow]
	// Dangling images have <none>:<none> — no ref to pull.
	if image.Dangling || image.Repository == "" || image.Repository == "<none>" {
		m.statusMessage = "Can't update an untagged image"
		return m, nil
	}
	ref := image.Repository
	if image.Tag != "" && image.Tag != "<none>" {
		ref = image.Repository + ":" + image.Tag
	}
	m.actionLabel = "Updating " + ref
	m.statusMessage = "Updating " + ref + " to latest..."
	return m, m.updateImageCmd(ref)
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

			// Models: detour through the tag picker so the user can choose
			// a specific quantization variant. Images: pull immediately.
			if m.currentView == types.ViewModePullModel {
				return m.openModelTagPicker(img.Name)
			}

			m.pullStage = 3
			m.pullingImageName = img.Name
			m.actionInProgress = true
			m.statusMessage = "Pulling " + img.Name + "..."
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

// --- Run image modal ---

// handleRunModalKeys handles input in the Run image modal (full-screen form).
// TAB/Shift-TAB cycles fields, Enter commits the current input row to its
// list (or runs when on Submit), Ctrl+R runs from anywhere, Esc cancels.
func (m *Model) handleRunModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.currentView = types.ViewModeList
		return m, nil
	case "ctrl+r":
		return m.submitRunModal()
	case "tab", "down":
		m.runModalField = (m.runModalField + 1) % types.RunFieldCount
		return m, nil
	case "shift+tab", "up":
		m.runModalField = (m.runModalField + types.RunFieldCount - 1) % types.RunFieldCount
		return m, nil
	case "enter":
		switch m.runModalField {
		case types.RunFieldPortInput:
			if p, ok := parsePortMapping(m.runPortInput); ok {
				m.runPorts = append(m.runPorts, p)
				m.runPortInput = ""
			}
			return m, nil
		case types.RunFieldEnvInput:
			if e, ok := parseEnvVar(m.runEnvInput); ok {
				m.runEnvVars = append(m.runEnvVars, e)
				m.runEnvInput = ""
			}
			return m, nil
		case types.RunFieldVolumeAdd:
			return m.openVolumePicker()
		case types.RunFieldSubmit:
			return m.submitRunModal()
		}
		return m, nil
	case "backspace":
		// On an empty input row, remove the last committed entry from the
		// list above (shell-style line erase).
		switch m.runModalField {
		case types.RunFieldName:
			if len(m.runContainerName) > 0 {
				m.runContainerName = m.runContainerName[:len(m.runContainerName)-1]
			}
		case types.RunFieldPortInput:
			if m.runPortInput == "" && len(m.runPorts) > 0 {
				m.runPorts = m.runPorts[:len(m.runPorts)-1]
			} else if len(m.runPortInput) > 0 {
				m.runPortInput = m.runPortInput[:len(m.runPortInput)-1]
			}
		case types.RunFieldEnvInput:
			if m.runEnvInput == "" && len(m.runEnvVars) > 0 {
				m.runEnvVars = m.runEnvVars[:len(m.runEnvVars)-1]
			} else if len(m.runEnvInput) > 0 {
				m.runEnvInput = m.runEnvInput[:len(m.runEnvInput)-1]
			}
		case types.RunFieldVolumeAdd:
			if len(m.runVolumes) > 0 {
				m.runVolumes = m.runVolumes[:len(m.runVolumes)-1]
			}
		}
		return m, nil
	}

	// Printable character → append to focused field's input.
	if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
		switch m.runModalField {
		case types.RunFieldName:
			m.runContainerName += key
		case types.RunFieldPortInput:
			m.runPortInput += key
		case types.RunFieldEnvInput:
			m.runEnvInput += key
		}
	}
	return m, nil
}

func (m *Model) submitRunModal() (tea.Model, tea.Cmd) {
	// Commit any partially-typed input rows so users don't lose them.
	if p, ok := parsePortMapping(m.runPortInput); ok {
		m.runPorts = append(m.runPorts, p)
		m.runPortInput = ""
	}
	if e, ok := parseEnvVar(m.runEnvInput); ok {
		m.runEnvVars = append(m.runEnvVars, e)
		m.runEnvInput = ""
	}
	m.currentView = types.ViewModeList
	m.actionLabel = "Running " + m.selectedImage.Repository + ":" + m.selectedImage.Tag
	m.statusMessage = "Starting container..."
	return m, m.runContainerCmd()
}

// parsePortMapping parses "host:container" (e.g. "8080:80"). Both halves
// must be present and non-empty; whitespace is trimmed.
func parsePortMapping(s string) (types.PortMapping, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return types.PortMapping{}, false
	}
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return types.PortMapping{}, false
	}
	host := strings.TrimSpace(parts[0])
	cont := strings.TrimSpace(parts[1])
	if host == "" || cont == "" {
		return types.PortMapping{}, false
	}
	return types.PortMapping{Host: host, Container: cont}, true
}

// parseEnvVar parses "KEY=value". Value may be empty; key may not.
func parseEnvVar(s string) (types.EnvVar, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return types.EnvVar{}, false
	}
	parts := strings.SplitN(s, "=", 2)
	if len(parts) < 1 || parts[0] == "" {
		return types.EnvVar{}, false
	}
	value := ""
	if len(parts) == 2 {
		value = parts[1]
	}
	return types.EnvVar{Key: parts[0], Value: value}, true
}

// --- Volume picker (sub-view of Run modal) ---

func (m *Model) openVolumePicker() (tea.Model, tea.Cmd) {
	m.currentView = types.ViewModeRunVolumePicker
	m.runVolumePickerMode = types.VolumePickerChoose
	m.runVolumePickerIndex = 0
	m.runVolumePickerSub = 0
	m.runVolumeNameInput = ""
	m.runVolumeHostInput = ""
	m.runVolumeContInput = ""
	return m, nil
}

func (m *Model) handleVolumePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "esc" {
		// Back to the Run modal — drop without committing.
		m.currentView = types.ViewModeRunImage
		return m, nil
	}

	switch m.runVolumePickerMode {
	case types.VolumePickerChoose:
		return m.handleVolumePickerChoose(key)
	case types.VolumePickerExisting:
		return m.handleVolumePickerExisting(key)
	case types.VolumePickerNew:
		return m.handleVolumePickerNew(key)
	case types.VolumePickerBind:
		// Bind mode is the "container path" step after picking a host
		// path in the file browser.
		return m.handleVolumePickerBind(key)
	}
	return m, nil
}

func (m *Model) handleVolumePickerChoose(key string) (tea.Model, tea.Cmd) {
	options := 3 // existing / new / bind
	switch key {
	case "up", "k":
		if m.runVolumePickerIndex > 0 {
			m.runVolumePickerIndex--
		}
	case "down", "j":
		if m.runVolumePickerIndex < options-1 {
			m.runVolumePickerIndex++
		}
	case "enter":
		switch m.runVolumePickerIndex {
		case 0: // existing
			m.runVolumePickerMode = types.VolumePickerExisting
			m.runVolumePickerIndex = 0
			m.runVolumePickerSub = 0 // start with focus on the volume list
			m.runVolumeContInput = ""
		case 1: // new
			m.runVolumePickerMode = types.VolumePickerNew
			m.runVolumePickerSub = 0 // start with focus on the name field
			m.runVolumeNameInput = ""
			m.runVolumeContInput = ""
		case 2: // bind
			return m.openFileBrowser()
		}
	}
	return m, nil
}

// handleVolumePickerExisting drives a two-stage flow: first pick a volume
// from the list (sub=0), then type the container mount path (sub=1). TAB
// toggles focus, Enter on the list advances to the path step, Enter on the
// path commits the mapping.
func (m *Model) handleVolumePickerExisting(key string) (tea.Model, tea.Cmd) {
	if len(m.volumes) == 0 {
		if key == "enter" || key == "esc" {
			m.runVolumePickerMode = types.VolumePickerChoose
			m.runVolumePickerSub = 0
		}
		return m, nil
	}

	// Shared: TAB toggles between list and path field.
	if key == "tab" || key == "shift+tab" {
		m.runVolumePickerSub = 1 - m.runVolumePickerSub
		return m, nil
	}

	if m.runVolumePickerSub == 0 {
		// Stage 1: picking the volume.
		switch key {
		case "up", "k":
			if m.runVolumePickerIndex > 0 {
				m.runVolumePickerIndex--
			}
		case "down", "j":
			if m.runVolumePickerIndex < len(m.volumes)-1 {
				m.runVolumePickerIndex++
			}
		case "enter":
			// Advance to path entry.
			m.runVolumePickerSub = 1
		}
		return m, nil
	}

	// Stage 2: typing the container path.
	switch key {
	case "enter":
		cont := strings.TrimSpace(m.runVolumeContInput)
		if cont == "" {
			return m, nil
		}
		v := m.volumes[m.runVolumePickerIndex]
		m.runVolumes = append(m.runVolumes, types.VolumeMapping{
			IsNamed:    true,
			VolumeName: v.Name,
			Container:  cont,
		})
		m.currentView = types.ViewModeRunImage
		return m, nil
	case "backspace":
		if len(m.runVolumeContInput) > 0 {
			m.runVolumeContInput = m.runVolumeContInput[:len(m.runVolumeContInput)-1]
		}
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.runVolumeContInput += key
		}
	}
	return m, nil
}

func (m *Model) handleVolumePickerNew(key string) (tea.Model, tea.Cmd) {
	// Two fields tracked by runVolumePickerSub: 0 = name, 1 = container path.
	switch key {
	case "tab", "shift+tab", "down", "up":
		m.runVolumePickerSub = 1 - m.runVolumePickerSub
	case "enter":
		// On name field, Enter advances to path; on path field, Enter commits.
		if m.runVolumePickerSub == 0 {
			if strings.TrimSpace(m.runVolumeNameInput) == "" {
				return m, nil
			}
			m.runVolumePickerSub = 1
			return m, nil
		}
		name := strings.TrimSpace(m.runVolumeNameInput)
		cont := strings.TrimSpace(m.runVolumeContInput)
		if name == "" || cont == "" {
			return m, nil
		}
		m.runVolumes = append(m.runVolumes, types.VolumeMapping{
			IsNamed:    true,
			VolumeName: name,
			Container:  cont,
		})
		m.currentView = types.ViewModeRunImage
		return m, nil
	case "backspace":
		if m.runVolumePickerSub == 0 && len(m.runVolumeNameInput) > 0 {
			m.runVolumeNameInput = m.runVolumeNameInput[:len(m.runVolumeNameInput)-1]
		} else if m.runVolumePickerSub == 1 && len(m.runVolumeContInput) > 0 {
			m.runVolumeContInput = m.runVolumeContInput[:len(m.runVolumeContInput)-1]
		}
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			if m.runVolumePickerSub == 0 {
				m.runVolumeNameInput += key
			} else {
				m.runVolumeContInput += key
			}
		}
	}
	return m, nil
}

// handleVolumePickerBind is the post-file-browser step: prompts for the
// container path. The host path was set by the file browser into
// runVolumeHostInput.
func (m *Model) handleVolumePickerBind(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		cont := strings.TrimSpace(m.runVolumeContInput)
		if cont == "" {
			return m, nil
		}
		m.runVolumes = append(m.runVolumes, types.VolumeMapping{
			IsNamed:   false,
			Host:      m.runVolumeHostInput,
			Container: cont,
		})
		m.currentView = types.ViewModeRunImage
		return m, nil
	case "backspace":
		if len(m.runVolumeContInput) > 0 {
			m.runVolumeContInput = m.runVolumeContInput[:len(m.runVolumeContInput)-1]
		}
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.runVolumeContInput += key
		}
	}
	return m, nil
}

// --- File browser (for bind mount host path) ---

func (m *Model) openFileBrowser() (tea.Model, tea.Cmd) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "/"
	}
	m.fileBrowserPath = home
	m.fileBrowserIndex = 0
	m.fileBrowserScroll = 0
	m.fileBrowserEntries = listDir(home)
	m.currentView = types.ViewModeRunFileBrowser
	return m, nil
}

func (m *Model) handleFileBrowserKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		// Back to volume picker chooser.
		m.currentView = types.ViewModeRunVolumePicker
		m.runVolumePickerMode = types.VolumePickerChoose
		return m, nil
	case "f", "F":
		// Confirm: use the *current directory* as the host path.
		m.runVolumeHostInput = m.fileBrowserPath
		m.runVolumePickerMode = types.VolumePickerBind
		m.runVolumeContInput = ""
		m.currentView = types.ViewModeRunVolumePicker
		return m, nil
	case "up", "k":
		if m.fileBrowserIndex > 0 {
			m.fileBrowserIndex--
		}
		return m, nil
	case "down", "j":
		if m.fileBrowserIndex < len(m.fileBrowserEntries)-1 {
			m.fileBrowserIndex++
		}
		return m, nil
	case "enter":
		if len(m.fileBrowserEntries) == 0 {
			return m, nil
		}
		entry := m.fileBrowserEntries[m.fileBrowserIndex]
		// ".." → go up one level.
		if entry == "../" {
			m.fileBrowserPath = filepath.Dir(m.fileBrowserPath)
			m.fileBrowserEntries = listDir(m.fileBrowserPath)
			m.fileBrowserIndex = 0
			m.fileBrowserScroll = 0
			return m, nil
		}
		// Directory entries end in "/" — descend into them.
		if strings.HasSuffix(entry, "/") {
			child := filepath.Join(m.fileBrowserPath, strings.TrimSuffix(entry, "/"))
			m.fileBrowserPath = child
			m.fileBrowserEntries = listDir(child)
			m.fileBrowserIndex = 0
			m.fileBrowserScroll = 0
			return m, nil
		}
		// A file — select it as the host path.
		m.runVolumeHostInput = filepath.Join(m.fileBrowserPath, entry)
		m.runVolumePickerMode = types.VolumePickerBind
		m.runVolumeContInput = ""
		m.currentView = types.ViewModeRunVolumePicker
		return m, nil
	}
	return m, nil
}

// listDir returns sorted entries of a directory. Directories are appended
// with "/" so the renderer can tell them apart from files. ".." is always
// the first entry (so users can navigate up). On error, returns just "..".
func listDir(path string) []string {
	out := []string{"../"}
	entries, err := os.ReadDir(path)
	if err != nil {
		return out
	}
	var dirs, files []string
	for _, e := range entries {
		name := e.Name()
		// Skip dotfiles — they clutter the view; users who need them can
		// type the path manually later.
		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, name+"/")
		} else {
			files = append(files, name)
		}
	}
	sort.Strings(dirs)
	sort.Strings(files)
	out = append(out, dirs...)
	out = append(out, files...)
	return out
}

// --- Chat with a DMR model ---

// handleModelChat opens an in-app streaming chat with the selected model.
func (m *Model) handleModelChat() (tea.Model, tea.Cmd) {
	if m.selectedRow >= len(m.models) {
		return m, nil
	}
	mod := m.models[m.selectedRow]
	ref := mod.Repository + ":" + mod.Tag

	// Start a fresh conversation each time R is pressed — previous
	// history is dropped intentionally so the user isn't surprised by
	// leftover context after navigating away.
	m.chatModelRef = ref
	m.chatMessages = nil
	m.chatInput = ""
	m.chatCurrentResponse = ""
	m.chatStreaming = false
	m.chatError = ""
	m.chatScrollOffset = 0
	m.closeChatStream()
	m.currentView = types.ViewModeChat
	return m, nil
}

func (m *Model) closeChatStream() {
	if m.chatBody != nil {
		_ = m.chatBody.Close()
		m.chatBody = nil
	}
	m.chatReader = nil
}

func (m *Model) handleChatViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		// Bail out — drop any in-flight stream.
		m.closeChatStream()
		m.chatStreaming = false
		m.chatCurrentResponse = ""
		m.currentView = types.ViewModeList
		return m, nil
	case "ctrl+l":
		// Quick clear: wipe history but keep the model selection.
		m.chatMessages = nil
		m.chatCurrentResponse = ""
		m.chatError = ""
		return m, nil
	case "pgup":
		if m.chatScrollOffset > 5 {
			m.chatScrollOffset -= 5
		} else {
			m.chatScrollOffset = 0
		}
		return m, nil
	case "pgdown":
		m.chatScrollOffset += 5
		return m, nil
	}

	// While streaming the only useful input is ESC (handled above) — don't
	// let the user type a new prompt while the previous one is generating.
	if m.chatStreaming {
		return m, nil
	}

	switch key {
	case "enter":
		prompt := strings.TrimSpace(m.chatInput)
		if prompt == "" {
			return m, nil
		}
		m.chatMessages = append(m.chatMessages, types.ChatMessage{
			Role: "user", Content: prompt,
		})
		m.chatInput = ""
		m.chatStreaming = true
		m.chatCurrentResponse = ""
		m.chatError = ""
		return m, m.startChatCmd(m.chatModelRef, m.chatMessages)
	case "backspace":
		if len(m.chatInput) > 0 {
			m.chatInput = m.chatInput[:len(m.chatInput)-1]
		}
		return m, nil
	}

	if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
		m.chatInput += key
	}
	return m, nil
}

// --- Model tag picker ---

func (m *Model) openModelTagPicker(repo string) (tea.Model, tea.Cmd) {
	m.tagPickerRepo = repo
	m.tagPickerTags = nil
	m.tagPickerIndex = 0
	m.tagPickerScroll = 0
	m.tagPickerLoading = true
	m.tagPickerError = ""
	m.currentView = types.ViewModeModelTagPicker
	return m, m.fetchModelTagsCmd(repo)
}

func (m *Model) handleTagPickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		// Back to the model search results (stage 2 of the pull flow).
		m.currentView = types.ViewModePullModel
		m.tagPickerTags = nil
		m.tagPickerError = ""
		return m, nil
	case "up", "k":
		if m.tagPickerIndex > 0 {
			m.tagPickerIndex--
			if m.tagPickerIndex < m.tagPickerScroll {
				m.tagPickerScroll = m.tagPickerIndex
			}
		}
		return m, nil
	case "down", "j":
		if m.tagPickerIndex < len(m.tagPickerTags)-1 {
			m.tagPickerIndex++
			visible := m.tagPickerViewportHeight()
			if m.tagPickerIndex >= m.tagPickerScroll+visible {
				m.tagPickerScroll = m.tagPickerIndex - visible + 1
			}
		}
		return m, nil
	case "enter":
		if m.tagPickerLoading || m.tagPickerIndex >= len(m.tagPickerTags) {
			return m, nil
		}
		tag := m.tagPickerTags[m.tagPickerIndex]
		ref := m.tagPickerRepo + ":" + tag.Tag
		// Reuse the existing pull flow (stage 3 = pulling).
		m.currentView = types.ViewModePullModel
		m.pullStage = 3
		m.pullingImageName = ref
		m.actionInProgress = true
		m.statusMessage = "Pulling " + ref + "..."
		return m, m.pullModelCmd(ref)
	}
	return m, nil
}

func (m *Model) tagPickerViewportHeight() int {
	h := m.height - 10
	if h < 5 {
		h = 5
	}
	return h
}
