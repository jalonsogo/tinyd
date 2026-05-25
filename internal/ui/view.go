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

// Status indicator styles — set by InitTheme, not at package init.
var (
	greenStyle  lipgloss.Style
	yellowStyle lipgloss.Style
	redStyle    lipgloss.Style
	grayStyle   lipgloss.Style
)

func init() { InitTheme(true) } // default: dark

// Status palette — kept in sync between InitTheme, statusTextStatic,
// getStatusDot, getImageStatusDot, and getInUseDot so the dot and the text
// label always render in the same color. Tuned to match the reference
// design: a softer leaf green and a coral red instead of neon/full-bright.
var (
	statusGreen  lipgloss.Color
	statusYellow lipgloss.Color
	statusRed    lipgloss.Color
	statusGray   lipgloss.Color
)

// InitTheme sets status-indicator colors for dark or light terminals.
// Call this before tea.NewProgram, after components.InitTheme.
func InitTheme(dark bool) {
	if dark {
		statusGreen = lipgloss.Color("#5FBA60")
		statusYellow = lipgloss.Color("#E5C07B")
		statusRed = lipgloss.Color("#E54545")
		statusGray = lipgloss.Color("#666666")
	} else {
		statusGreen = lipgloss.Color("#2E8B2E")
		statusYellow = lipgloss.Color("#886600")
		statusRed = lipgloss.Color("#C13030")
		statusGray = lipgloss.Color("#888888")
	}
	greenStyle = lipgloss.NewStyle().Foreground(statusGreen)
	yellowStyle = lipgloss.NewStyle().Foreground(statusYellow)
	redStyle = lipgloss.NewStyle().Foreground(statusRed)
	grayStyle = lipgloss.NewStyle().Foreground(statusGray)
}

// View renders the UI
func (m *Model) View() string {
	if m.err != nil {
		return m.renderErrorScreen()
	}

	// Render based on current view mode
	switch m.currentView {
	case types.ViewModeList:
		return m.renderListView()
	case types.ViewModeLogs:
		return m.renderLogsView()
	case types.ViewModeInspect:
		return m.renderInspectView()
	case types.ViewModePullImage, types.ViewModePullModel:
		return m.renderPullView()
	case types.ViewModeRunImage:
		return m.renderRunModal()
	case types.ViewModeRunVolumePicker:
		return m.renderVolumePicker()
	case types.ViewModeRunFileBrowser:
		return m.renderFileBrowser()
	case types.ViewModeChat:
		return m.renderChatView()
	case types.ViewModeModelTagPicker:
		return m.renderTagPicker()
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

	// Render content based on active tab — or an overlay if active
	var contentStr string
	if m.showHelp {
		contentStr = m.renderHelpOverlay()
	} else if m.showColumnPicker {
		contentStr = m.renderColumnPicker()
	} else {
		switch m.activeTab {
		case types.TabContainers:
			contentStr = m.renderContainersTab()
		case types.TabImages:
			contentStr = m.renderImagesTab()
		case types.TabVolumes:
			contentStr = m.renderVolumesTab()
		case types.TabNetworks:
			contentStr = m.renderNetworksTab()
		case types.TabModels:
			contentStr = m.renderModelsTab()
		}
	}
	b.WriteString(contentStr)

	// On the Models tab we pin the API info panel just above the action
	// bar, so the connection details are always visible at the bottom of
	// the screen regardless of how many models are in the table.
	var bottomPanel string
	if !m.showHelp && m.activeTab == types.TabModels && m.dmrAvailable {
		bottomPanel = m.renderModelsAPIPanel()
	}

	// Calculate padding to push action bar to bottom
	// Count actual lines used (tabs + visible content rows + action bar)
	tabsHeight := strings.Count(tabsContent, "\n")
	contentHeight := strings.Count(contentStr, "\n")
	panelHeight := strings.Count(bottomPanel, "\n")
	actionBarHeight := 3

	// Total used height including current content
	usedHeight := tabsHeight + contentHeight + panelHeight + actionBarHeight + 2 // +2 for newlines

	// Add padding to push action bar to bottom
	remainingHeight := m.height - usedHeight
	if remainingHeight > 0 {
		b.WriteString(strings.Repeat("\n", remainingHeight))
	} else if remainingHeight < 0 {
		// If content is too tall, don't add padding
		b.WriteString("\n")
	}

	// Bottom panel sits just above the action bar.
	if bottomPanel != "" {
		b.WriteString(bottomPanel)
	}

	// Render action bar at bottom — always recompute actions for the current
	// tab/selection, and overlay either an in-progress spinner or a
	// last-result status message.
	b.WriteString("\n")
	m.actionBar = m.actionBar.WithWidth(m.width)
	m.actionBar = m.actionBar.SetActions(m.getActionShortcuts())
	m.actionBar = m.actionBar.SetStatusMessage(m.currentStatusMessage())
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

	showStatus := m.IsColumnVisible("status")
	showImage := m.IsColumnVisible("image")
	showCPU := m.IsColumnVisible("cpu")
	showMem := m.IsColumnVisible("mem")
	showPorts := m.IsColumnVisible("ports")

	// Calculate responsive column widths from the visible set. Hidden
	// fixed-width columns return their space to the fill columns.
	totalWidth := m.width - 4
	if totalWidth < 60 {
		totalWidth = 60
	}
	statusW := 11
	fixedWidth := 2 // status dot
	if showStatus {
		fixedWidth += statusW
	}
	if showCPU {
		fixedWidth += 8
	}
	if showMem {
		fixedWidth += 8
	}
	if showPorts {
		fixedWidth += 15
	}
	// Count visible columns to compute gap spacing.
	visibleN := 2 // dot + NAME (both always shown)
	if showStatus {
		visibleN++
	}
	if showImage {
		visibleN++
	}
	if showCPU {
		visibleN++
	}
	if showMem {
		visibleN++
	}
	if showPorts {
		visibleN++
	}
	spacing := (visibleN - 1) * 2
	fillWidth := totalWidth - fixedWidth - spacing
	if fillWidth < 40 {
		fillWidth = 40
	}
	nameFill := fillWidth
	imageFill := 0
	if showImage {
		nameFill = fillWidth / 2
		imageFill = fillWidth - nameFill
		if imageFill < 20 {
			imageFill = 20
		}
	}
	if nameFill < 20 {
		nameFill = 20
	}

	headers := []components.TableHeader{
		{Label: "", Width: 2, AlignRight: false},
		{Label: "NAME", Width: nameFill, AlignRight: false},
	}
	if showStatus {
		headers = append(headers, components.TableHeader{Label: "STATUS", Width: statusW})
	}
	if showImage {
		headers = append(headers, components.TableHeader{Label: "IMAGE", Width: imageFill})
	}
	if showCPU {
		headers = append(headers, components.TableHeader{Label: "CPU", Width: 8, AlignRight: true})
	}
	if showMem {
		headers = append(headers, components.TableHeader{Label: "MEM", Width: 8, AlignRight: true})
	}
	if showPorts {
		headers = append(headers, components.TableHeader{Label: "PORTS", Width: 15})
	}

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

		if m.deleteConfirmMode && i == m.selectedRow {
			confirmText := renderDeleteConfirmation(c.Name, m.deleteConfirmOption)
			emptyCells := make([]string, len(headers)-1)
			rows = append(rows, components.TableRow{
				Cells:      append([]string{confirmText}, emptyCells...),
				IsSelected: true,
			})
			continue
		}

		dotCell := m.getStatusDot(c.Status, i == m.selectedRow)
		if m.actionInProgress && m.actionTargetID == c.ID {
			dotCell = m.spinnerDot(i == m.selectedRow)
		}
		text, color := m.statusText(c.ID, c.Status)
		dimmed := c.Status != "RUNNING" && i != m.selectedRow
		dim := lipgloss.NewStyle().Foreground(components.ColorDim)

		nameStr := truncateWithEllipsis(c.Name, nameFill)
		imageStr := truncateWithEllipsis(c.Image, imageFill)
		cpuStr := c.CPU
		memStr := c.Mem
		portsStr := truncateWithEllipsis(c.Ports, 15)
		if dimmed {
			nameStr = dim.Render(padRightStr(nameStr, nameFill))
			imageStr = dim.Render(padRightStr(imageStr, imageFill))
			cpuStr = dim.Render(padLeftStr(cpuStr, 8))
			memStr = dim.Render(padLeftStr(memStr, 8))
			portsStr = dim.Render(padRightStr(portsStr, 15))
		}

		cells := []string{dotCell, nameStr}
		if showStatus {
			cells = append(cells, m.renderStatusCell(text, color, statusW, i == m.selectedRow))
		}
		if showImage {
			cells = append(cells, imageStr)
		}
		if showCPU {
			cells = append(cells, cpuStr)
		}
		if showMem {
			cells = append(cells, memStr)
		}
		if showPorts {
			cells = append(cells, portsStr)
		}

		rows = append(rows, components.TableRow{
			Cells:      cells,
			IsSelected: i == m.selectedRow,
		})
	}

	table := components.NewTableComponent(headers).
		WithWidth(m.width).
		SetRows(rows).
		SetVisibleRange(0, len(rows))

	return table.View() + m.getScrollIndicator(len(m.containers))
}

// renderImagesTab renders the images tab with proper table formatting
func (m *Model) renderImagesTab() string {
	if len(m.images) == 0 {
		if m.loading {
			return "Loading images..."
		}
		return "No images found"
	}

	showStatus := m.IsColumnVisible("status")
	showSize := m.IsColumnVisible("size")
	showCreated := m.IsColumnVisible("created")

	totalWidth := m.width - 4
	if totalWidth < 50 {
		totalWidth = 50
	}
	statusW := 10
	fixedWidth := 2
	if showStatus {
		fixedWidth += statusW
	}
	if showSize {
		fixedWidth += 10
	}
	if showCreated {
		fixedWidth += 8
	}
	visibleN := 2 // dot + REPO:TAG
	for _, on := range []bool{showStatus, showSize, showCreated} {
		if on {
			visibleN++
		}
	}
	spacing := (visibleN - 1) * 2
	fillWidth := totalWidth - fixedWidth - spacing
	if fillWidth < 20 {
		fillWidth = 20
	}

	headers := []components.TableHeader{
		{Label: "", Width: 2},
		{Label: "REPOSITORY:TAG", Width: fillWidth},
	}
	if showStatus {
		headers = append(headers, components.TableHeader{Label: "STATUS", Width: statusW})
	}
	if showSize {
		headers = append(headers, components.TableHeader{Label: "SIZE", Width: 10, AlignRight: true})
	}
	if showCreated {
		headers = append(headers, components.TableHeader{Label: "CREATED", Width: 8})
	}

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

		if m.deleteConfirmMode && i == m.selectedRow {
			confirmText := renderDeleteConfirmation(img.Repository+":"+img.Tag, m.deleteConfirmOption)
			emptyCells := make([]string, len(headers)-1)
			rows = append(rows, components.TableRow{
				Cells:      append([]string{confirmText}, emptyCells...),
				IsSelected: true,
			})
			continue
		}

		repoTag := img.Repository + ":" + img.Tag
		repoTagCell := repoTag
		if len(repoTag) > fillWidth {
			repoTagCell = truncateWithEllipsis(repoTag, fillWidth)
		}

		dotCell := m.getImageStatusDot(img, i == m.selectedRow)
		if m.actionInProgress && m.actionTargetID == img.ID {
			dotCell = m.spinnerDot(i == m.selectedRow)
		}
		var stateKey string
		switch {
		case img.InUse:
			stateKey = "IMG_IN_USE"
		case img.Dangling:
			stateKey = "IMG_DANGLING"
		default:
			stateKey = "IMG_UNUSED"
		}
		text, color := m.statusText(img.ID, stateKey)

		cells := []string{dotCell, repoTagCell}
		if showStatus {
			cells = append(cells, m.renderStatusCell(text, color, statusW, i == m.selectedRow))
		}
		if showSize {
			cells = append(cells, img.Size)
		}
		if showCreated {
			cells = append(cells, shortenTimeAgo(img.Created))
		}

		rows = append(rows, components.TableRow{
			Cells:      cells,
			IsSelected: i == m.selectedRow,
		})
	}

	table := components.NewTableComponent(headers).
		WithWidth(m.width).
		SetRows(rows).
		SetVisibleRange(0, len(rows))

	return table.View() + m.getScrollIndicator(len(m.images))
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
	showContainers := m.IsColumnVisible("containers")
	showMountpoint := m.IsColumnVisible("mountpoint")

	fixedWidth := 2 // dot
	visibleN := 2   // dot + NAME
	for _, on := range []bool{showContainers, showMountpoint} {
		if on {
			visibleN++
		}
	}
	spacing := (visibleN - 1) * 2
	fillWidth := totalWidth - fixedWidth - spacing
	if fillWidth < 30 {
		fillWidth = 30
	}

	// Distribute the fill width across the visible fill columns.
	fillCount := 1 // NAME always shown
	if showContainers {
		fillCount++
	}
	if showMountpoint {
		fillCount++
	}
	col := fillWidth / fillCount
	nameFill := col
	containersFill := col
	mountFill := fillWidth - nameFill
	if showContainers {
		mountFill = fillWidth - nameFill - containersFill
	}

	headers := []components.TableHeader{
		{Label: "", Width: 2},
		{Label: "NAME", Width: nameFill},
	}
	if showContainers {
		headers = append(headers, components.TableHeader{Label: "CONTAINERS", Width: containersFill})
	}
	if showMountpoint {
		headers = append(headers, components.TableHeader{Label: "MOUNT POINT", Width: mountFill})
	}

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

		if m.deleteConfirmMode && i == m.selectedRow {
			confirmText := renderDeleteConfirmation(vol.Name, m.deleteConfirmOption)
			emptyCells := make([]string, len(headers)-1)
			rows = append(rows, components.TableRow{
				Cells:      append([]string{confirmText}, emptyCells...),
				IsSelected: true,
			})
			continue
		}

		dotCell := m.getInUseDot(vol.InUse, i == m.selectedRow)
		if m.actionInProgress && m.actionTargetID == vol.Name {
			dotCell = m.spinnerDot(i == m.selectedRow)
		}
		containers := vol.Containers
		if containers == "" {
			containers = "-"
		}

		cells := []string{dotCell, truncateWithEllipsis(vol.Name, nameFill)}
		if showContainers {
			cells = append(cells, truncateWithEllipsis(containers, containersFill))
		}
		if showMountpoint {
			cells = append(cells, truncateWithEllipsis(vol.Mountpoint, mountFill))
		}

		rows = append(rows, components.TableRow{
			Cells:      cells,
			IsSelected: i == m.selectedRow,
		})
	}

	table := components.NewTableComponent(headers).
		WithWidth(m.width).
		SetRows(rows).
		SetVisibleRange(0, len(rows))

	return table.View() + m.getScrollIndicator(len(m.volumes))
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

	showContainers := m.IsColumnVisible("containers")
	showDriver := m.IsColumnVisible("driver")
	showScope := m.IsColumnVisible("scope")
	showIPv4 := m.IsColumnVisible("ipv4")

	fixedWidth := 2 // dot
	if showDriver {
		fixedWidth += 10
	}
	if showScope {
		fixedWidth += 8
	}
	if showIPv4 {
		fixedWidth += 18
	}
	visibleN := 2 // dot + NAME
	for _, on := range []bool{showContainers, showDriver, showScope, showIPv4} {
		if on {
			visibleN++
		}
	}
	spacing := (visibleN - 1) * 2
	fillWidth := totalWidth - fixedWidth - spacing
	if fillWidth < 20 {
		fillWidth = 20
	}

	nameFill := fillWidth
	containersFill := 0
	if showContainers {
		nameFill = fillWidth / 2
		containersFill = fillWidth - nameFill
	}

	headers := []components.TableHeader{
		{Label: "", Width: 2},
		{Label: "NAME", Width: nameFill},
	}
	if showContainers {
		headers = append(headers, components.TableHeader{Label: "CONTAINERS", Width: containersFill})
	}
	if showDriver {
		headers = append(headers, components.TableHeader{Label: "DRIVER", Width: 10})
	}
	if showScope {
		headers = append(headers, components.TableHeader{Label: "SCOPE", Width: 8})
	}
	if showIPv4 {
		headers = append(headers, components.TableHeader{Label: "IPv4", Width: 18})
	}

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

		if m.deleteConfirmMode && i == m.selectedRow {
			confirmText := renderDeleteConfirmation(net.Name, m.deleteConfirmOption)
			emptyCells := make([]string, len(headers)-1)
			rows = append(rows, components.TableRow{
				Cells:      append([]string{confirmText}, emptyCells...),
				IsSelected: true,
			})
			continue
		}

		dotCell := m.getInUseDot(net.InUse, i == m.selectedRow)
		if m.actionInProgress && m.actionTargetID == net.ID {
			dotCell = m.spinnerDot(i == m.selectedRow)
		}
		containers := "-"

		cells := []string{dotCell, truncateWithEllipsis(net.Name, nameFill)}
		if showContainers {
			cells = append(cells, truncateWithEllipsis(containers, containersFill))
		}
		if showDriver {
			cells = append(cells, net.Driver)
		}
		if showScope {
			cells = append(cells, net.Scope)
		}
		if showIPv4 {
			cells = append(cells, truncateWithEllipsis(net.IPv4, 18))
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

// renderErrorScreen displays a fatal-ish error in a readable layout.
// Detects the common "Docker daemon unreachable" case and shows a friendly
// diagnosis + remediation; falls back to a wrapped raw message otherwise.
func (m *Model) renderErrorScreen() string {
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444")).Bold(true)
	headingStyle := lipgloss.NewStyle().Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	dimStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	codeStyle := lipgloss.NewStyle().Foreground(components.ColorHighlight)

	errText := m.err.Error()
	dockerDown := looksLikeDockerDown(errText)

	var b strings.Builder
	b.WriteString("\n")

	if dockerDown {
		b.WriteString("  ")
		b.WriteString(titleStyle.Render("✕  Can't reach Docker"))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(bodyStyle.Render("tinyd tried to talk to the Docker daemon and got no response."))
		b.WriteString("\n  ")
		b.WriteString(bodyStyle.Render("Most likely the daemon isn't running yet."))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(headingStyle.Render("Try one of:"))
		b.WriteString("\n  ")
		b.WriteString(bodyStyle.Render("• Start Docker Desktop"))
		b.WriteString("\n  ")
		b.WriteString(bodyStyle.Render("• "))
		b.WriteString(codeStyle.Render("sudo systemctl start docker"))
		b.WriteString(dimStyle.Render("   (Linux)"))
		b.WriteString("\n  ")
		b.WriteString(bodyStyle.Render("• "))
		b.WriteString(codeStyle.Render("export DOCKER_HOST=tcp://…"))
		b.WriteString(dimStyle.Render("   (remote daemon)"))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("Original error:"))
		b.WriteString("\n")
		b.WriteString(wrapForDisplay(errText, m.width-4, "  "))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("Auto-retrying every 5s — leave this window open and tinyd will"))
		b.WriteString("\n  ")
		b.WriteString(dimStyle.Render("pick up the daemon as soon as it's reachable."))
	} else {
		b.WriteString("  ")
		b.WriteString(titleStyle.Render("✕  Something went wrong"))
		b.WriteString("\n\n")
		b.WriteString(wrapForDisplay(errText, m.width-4, "  "))
	}

	b.WriteString("\n\n  ")
	b.WriteString(dimStyle.Render("Press "))
	b.WriteString(codeStyle.Render("q"))
	b.WriteString(dimStyle.Render(" or "))
	b.WriteString(codeStyle.Render("Ctrl+C"))
	b.WriteString(dimStyle.Render(" to quit."))
	b.WriteString("\n")
	return b.String()
}

// looksLikeDockerDown reports whether an error string is the unmistakable
// "I can't reach the daemon" shape, so we can swap the generic stack-trace
// look for a remediation message.
func looksLikeDockerDown(s string) bool {
	low := strings.ToLower(s)
	for _, needle := range []string{
		"failed to connect to the docker api",
		"cannot connect to the docker daemon",
		"dial unix",
		"connection refused",
		"no such file or directory",
		"is the docker daemon running",
	} {
		if strings.Contains(low, needle) {
			return true
		}
	}
	return false
}

// wrapForDisplay wraps a long single-line error across the available width
// with a constant left indent, so we don't dump a 2000-char string onto a
// 100-column terminal and let the renderer truncate it mid-word.
func wrapForDisplay(s string, width int, indent string) string {
	if width < 20 {
		width = 20
	}
	dim := lipgloss.NewStyle().Foreground(components.ColorDim)

	words := strings.Fields(s)
	var b strings.Builder
	line := indent
	for _, w := range words {
		if len(line)+1+len(w) > width && len(line) > len(indent) {
			b.WriteString(dim.Render(line))
			b.WriteString("\n")
			line = indent
		}
		if len(line) > len(indent) {
			line += " "
		}
		line += w
	}
	if len(line) > len(indent) {
		b.WriteString(dim.Render(line))
	}
	return b.String()
}

// renderModelsTab renders the Models tab (Docker Model Runner).
func (m *Model) renderModelsTab() string {
	if !m.dmrAvailable {
		dim := lipgloss.NewStyle().Foreground(components.ColorDim)
		bright := lipgloss.NewStyle().Bold(true)
		var b strings.Builder
		b.WriteString("\n")
		b.WriteString(bright.Render(" Docker Model Runner is not reachable"))
		b.WriteString("\n\n")
		b.WriteString(dim.Render("  tinyd looked at " + m.dmr.BaseURL() + " and got no response."))
		b.WriteString("\n")
		b.WriteString(dim.Render("  Enable it in Docker Desktop → Settings → Beta features → Docker Model Runner,"))
		b.WriteString("\n")
		b.WriteString(dim.Render("  or set DMR_BASE_URL to a reachable endpoint."))
		b.WriteString("\n\n")
		b.WriteString(dim.Render("  https://docs.docker.com/ai/model-runner/"))
		b.WriteString("\n")
		return b.String()
	}

	if len(m.models) == 0 {
		if m.loading {
			return "Loading models..."
		}
		return "No local models. Press P to pull one from Docker Hub (ai/ namespace)."
	}

	showStatus := m.IsColumnVisible("status")
	showParams := m.IsColumnVisible("params")
	showQuant := m.IsColumnVisible("quant")
	showSize := m.IsColumnVisible("size")

	totalWidth := m.width - 4
	if totalWidth < 60 {
		totalWidth = 60
	}
	statusW := 10
	fixedWidth := 2
	if showStatus {
		fixedWidth += statusW
	}
	if showParams {
		fixedWidth += 8
	}
	if showQuant {
		fixedWidth += 8
	}
	if showSize {
		fixedWidth += 8
	}
	visibleN := 2 // dot + REPO:TAG
	for _, on := range []bool{showStatus, showParams, showQuant, showSize} {
		if on {
			visibleN++
		}
	}
	spacing := (visibleN - 1) * 2
	fillWidth := totalWidth - fixedWidth - spacing
	if fillWidth < 25 {
		fillWidth = 25
	}

	headers := []components.TableHeader{
		{Label: "", Width: 2},
		{Label: "REPOSITORY:TAG", Width: fillWidth},
	}
	if showStatus {
		headers = append(headers, components.TableHeader{Label: "STATUS", Width: statusW})
	}
	if showParams {
		headers = append(headers, components.TableHeader{Label: "PARAMS", Width: 8, AlignRight: true})
	}
	if showQuant {
		headers = append(headers, components.TableHeader{Label: "QUANT", Width: 8})
	}
	if showSize {
		headers = append(headers, components.TableHeader{Label: "SIZE", Width: 8, AlignRight: true})
	}

	var rows []components.TableRow
	start := m.scrollOffset
	end := m.scrollOffset + m.viewportHeight
	if end > len(m.models) {
		end = len(m.models)
	}
	if start > len(m.models) {
		start = len(m.models)
	}

	for i := start; i < end; i++ {
		mod := m.models[i]
		ref := mod.Repository + ":" + mod.Tag

		if m.deleteConfirmMode && i == m.selectedRow {
			confirmText := renderDeleteConfirmation(ref, m.deleteConfirmOption)
			emptyCells := make([]string, len(headers)-1)
			rows = append(rows, components.TableRow{
				Cells:      append([]string{confirmText}, emptyCells...),
				IsSelected: true,
			})
			continue
		}

		dot := m.getInUseDot(false, i == m.selectedRow)
		if m.actionInProgress && m.actionTargetID == ref {
			dot = m.spinnerDot(i == m.selectedRow)
		}
		text, color := m.statusText(ref, "MDL_AVAILABLE")

		cells := []string{dot, truncateWithEllipsis(ref, fillWidth)}
		if showStatus {
			cells = append(cells, m.renderStatusCell(text, color, statusW, i == m.selectedRow))
		}
		if showParams {
			cells = append(cells, defaultStr(mod.ParamSize, "--"))
		}
		if showQuant {
			cells = append(cells, defaultStr(mod.Quant, "--"))
		}
		if showSize {
			cells = append(cells, defaultStr(mod.Size, "--"))
		}
		rows = append(rows, components.TableRow{
			Cells:      cells,
			IsSelected: i == m.selectedRow,
		})
	}

	table := components.NewTableComponent(headers).
		WithWidth(m.width).
		SetRows(rows).
		SetVisibleRange(0, len(rows))

	return table.View() + m.getScrollIndicator(len(m.models))
}

func defaultStr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// renderLogsView renders the logs detail view
func (m *Model) renderLogsView() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	lineStyle := lipgloss.NewStyle().Foreground(components.ColorBorder)
	contentStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)

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

	titleStyle := lipgloss.NewStyle().Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	lineStyle := lipgloss.NewStyle().Foreground(components.ColorBorder)
	contentStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)

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

// spinnerDot returns an animated spinner styled like the status dots, so the
// row being acted on shows clear in-progress feedback in the status column.
func (m *Model) spinnerDot(selected bool) string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	ch := frames[m.animationFrame%len(frames)]
	return dotStyle(components.ColorHighlight, selected).Render(ch)
}

// currentStatusMessage returns the live message to show in the action bar:
// an animated spinner + actionLabel while an action is running, otherwise
// the last success/error message.
func (m *Model) currentStatusMessage() string {
	if m.actionInProgress && m.actionLabel != "" {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinner := frames[m.animationFrame%len(frames)]
		return spinner + " " + m.actionLabel + "..."
	}
	return m.statusMessage
}

// renderHelpOverlay renders a list of keybindings inside the tab content area
func (m *Model) renderHelpOverlay() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true)
	// Section headers and key cells use bold-only — no accent color, so
	// the help screen reads as one neutral block instead of a noisy
	// rainbow.
	sectionStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	keyStyle := lipgloss.NewStyle().Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	dimStyle := lipgloss.NewStyle().Foreground(components.ColorDim)

	row := func(key, desc string) {
		b.WriteString("  ")
		b.WriteString(keyStyle.Render(padRightStr(key, 14)))
		b.WriteString("  ")
		b.WriteString(descStyle.Render(desc))
		b.WriteString("\n")
	}
	section := func(name string) {
		b.WriteString("\n")
		b.WriteString(sectionStyle.Render(" " + name))
		b.WriteString("\n")
	}

	b.WriteString(titleStyle.Render(" tinyd — Keybindings"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(" Press ? or ESC to close"))
	b.WriteString("\n")

	section("Navigation")
	row("←/→", "Switch tabs")
	row("1–5", "Jump to tab")
	row("↑/↓ or j/k", "Move selection")
	row("Enter", "Refresh list")

	section("Containers")
	row("S", "Start / Stop")
	row("R", "Restart")
	row("L", "View logs")
	row("E", "Exec shell")
	row("I", "Inspect")
	row("D", "Delete")

	section("Images")
	row("S", "Run image")
	row("U", "Update to latest")
	row("I", "Inspect")
	row("D", "Delete")
	row("P", "Add image (search Docker Hub)")

	section("Volumes / Networks")
	row("I", "Inspect")
	row("D", "Delete")

	section("Models (Docker Model Runner)")
	row("C", "Chat in-app (streaming)")
	row("R", "Drop to docker model run REPL")
	row("P", "Pull model (variant picker → tag / params / quant / size)")
	row("Y", "Yank curl example to clipboard")
	row("I", "Inspect")
	row("D", "Delete")

	section("Global")
	row("?", "Toggle this help")
	row("V", "Customize visible columns (overlay)")
	row("ESC", "Cancel / close overlay")
	row("Ctrl+C ×2", "Quit")

	return b.String()
}

// renderPullView renders the pull-from-Hub search and results view
func (m *Model) renderPullView() string {
	var b strings.Builder

	// Emphasis uses bold + terminal-default fg so it stays readable on
	// either palette (ColorBright collapses to near-white on a light
	// terminal under the dark palette).
	titleStyle := lipgloss.NewStyle().Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	lineStyle := lipgloss.NewStyle().Foreground(components.ColorBorder)
	dimStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	normalStyle := lipgloss.NewStyle()
	brightStyle := lipgloss.NewStyle().Bold(true)
	selectedStyle := lipgloss.NewStyle().
		Background(components.ColorSelectedBg).
		Foreground(components.ColorSelectedFg).
		Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))

	// Header
	headerText := "Add image from Docker Hub"
	if m.currentView == types.ViewModePullModel {
		headerText = "Pull model from Docker Hub"
	}
	var headerRight string
	switch m.pullStage {
	case 0:
		headerRight = "[ENTER] Search  [ESC] Cancel"
	case 1:
		headerRight = "[ESC] Cancel"
	case 2:
		headerRight = "[↑↓] Move  [ENTER] Pull  [ESC] Cancel"
	case 3:
		headerRight = "Pulling..."
	}
	headerSpacing := strings.Repeat(" ", max(1, m.width-len(headerText)-len(headerRight)-4))
	b.WriteString(titleStyle.Render(headerText))
	b.WriteString(headerSpacing)
	b.WriteString(helpStyle.Render(headerRight))
	b.WriteString("\n")
	b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n")

	// Input line — always visible so the user sees the active query
	cursor := ""
	if m.pullStage == 0 {
		cursor = brightStyle.Render("▌")
	}
	queryDisplay := m.pullSearchQuery
	if queryDisplay == "" && m.pullStage == 0 {
		queryDisplay = dimStyle.Render("type to search Docker Hub...")
	} else {
		queryDisplay = normalStyle.Render(queryDisplay)
	}
	b.WriteString(dimStyle.Render(" Search: "))
	b.WriteString(queryDisplay)
	b.WriteString(cursor)
	b.WriteString("\n\n")

	switch m.pullStage {
	case 0:
		if m.pullSearchError != "" {
			b.WriteString(errorStyle.Render(" " + m.pullSearchError))
			b.WriteString("\n")
		} else {
			b.WriteString(dimStyle.Render(" Press ENTER to search."))
			b.WriteString("\n")
		}

	case 1:
		b.WriteString(dimStyle.Render(" Searching..."))
		b.WriteString("\n")

	case 2:
		if len(m.pullSearchResults) == 0 {
			b.WriteString(dimStyle.Render(" No results found."))
			b.WriteString("\n")
			break
		}

		// Column widths: stars(5) official(4) name(fill) arch(badges) description(rest)
		totalWidth := m.width - 4
		starsW, officialW, archW := 7, 5, 20
		nameW := 30
		if totalWidth-starsW-officialW-archW-nameW-8 < 20 {
			nameW = 20
		}
		descW := totalWidth - starsW - officialW - archW - nameW - 8
		if descW < 10 {
			descW = 10
		}

		// Header row
		header := padRightStr("⭐", starsW) + "  " +
			padRightStr("OFF", officialW) + "  " +
			padRightStr("NAME", nameW) + "  " +
			padRightStr("ARCH", archW) + "  " +
			padRightStr("DESCRIPTION", descW)
		b.WriteString(dimStyle.Render(header))
		b.WriteString("\n")
		b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
		b.WriteString("\n")

		visible := m.pullResultsViewportHeight()
		start := m.pullSearchScrollOffset
		end := start + visible
		if end > len(m.pullSearchResults) {
			end = len(m.pullSearchResults)
		}

		for i := start; i < end; i++ {
			r := m.pullSearchResults[i]
			official := ""
			if r.Official {
				official = "yes"
			}

			// Build the plain row so the selection highlight covers the
			// full width — but split out the arch column so we can paint
			// each badge with its own background color independently.
			rowPrefix := padRightStr(fmt.Sprintf("%d", r.Stars), starsW) + "  " +
				padRightStr(official, officialW) + "  " +
				padRightStr(truncateWithEllipsis(r.Name, nameW), nameW) + "  "
			archCell := renderArchBadges(r.Architectures, archW, i == m.pullSearchSelected)
			rowSuffix := "  " + padRightStr(truncateWithEllipsis(r.Description, descW), descW)

			if i == m.pullSearchSelected {
				b.WriteString(selectedStyle.Render(rowPrefix))
				b.WriteString(archCell)
				b.WriteString(selectedStyle.Render(rowSuffix))
			} else {
				b.WriteString(normalStyle.Render(rowPrefix))
				b.WriteString(archCell)
				b.WriteString(normalStyle.Render(rowSuffix))
			}
			b.WriteString("\n")
		}

		// Scroll hint
		if len(m.pullSearchResults) > visible {
			b.WriteString("\n")
			b.WriteString(dimStyle.Render(fmt.Sprintf(" Showing %d-%d of %d",
				start+1, end, len(m.pullSearchResults))))
			b.WriteString("\n")
		}

	case 3:
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinner := frames[m.animationFrame%len(frames)]
		highlightStyle := lipgloss.NewStyle().Foreground(components.ColorHighlight).Bold(true)
		b.WriteString("\n")
		b.WriteString(" ")
		b.WriteString(highlightStyle.Render(spinner))
		b.WriteString("  ")
		b.WriteString(brightStyle.Render("Pulling "))
		b.WriteString(brightStyle.Bold(true).Render(m.pullingImageName))
		b.WriteString(brightStyle.Render("..."))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render(" This may take a minute. The view will return when complete."))
		b.WriteString("\n")
	}

	return b.String()
}

// padLeft and padRight are defined in components/components.go; redefine
// helpers locally only when missing. (They are exported via the components
// package below — keep using those.)

// max returns the larger of two ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// renderArchBadges renders the supported architectures as compact pills
// (`AMD64`, `ARM64`, ...) padded to a fixed visible width. Pads with spaces
// on the selection background so the highlight stays continuous.
func renderArchBadges(archs []string, width int, selected bool) string {
	if width <= 0 {
		return ""
	}
	pad := lipgloss.NewStyle().Foreground(components.ColorNormal)
	if selected {
		pad = lipgloss.NewStyle().Background(components.ColorSelectedBg).Foreground(components.ColorSelectedFg).Bold(true)
	}

	if len(archs) == 0 {
		return pad.Render(strings.Repeat(" ", width))
	}

	var b strings.Builder
	used := 0
	first := true
	for _, a := range archs {
		label := strings.ToUpper(a) // amd64 → AMD64
		// pill is `▕LABEL▏` — 2 framing chars + len(label). Skip if it
		// won't fit.
		pillW := len(label) + 2
		needed := pillW
		if !first {
			needed += 1 // separator space
		}
		if used+needed > width {
			break
		}
		if !first {
			b.WriteString(pad.Render(" "))
			used++
		}
		b.WriteString(archBadgeStyle(label, selected).Render("▕" + label + "▏"))
		used += pillW
		first = false
	}
	if used < width {
		b.WriteString(pad.Render(strings.Repeat(" ", width-used)))
	}
	return b.String()
}

// archBadgeStyle picks a distinct color per common architecture so users
// can scan the list quickly. Falls back to the highlight color for
// anything unrecognized.
func archBadgeStyle(label string, selected bool) lipgloss.Style {
	var fg lipgloss.Color
	switch label {
	case "AMD64", "X86_64":
		fg = lipgloss.Color("#E67E22") // warm orange — matches Hub UI vibe
	case "ARM64", "AARCH64":
		fg = lipgloss.Color("#3498DB")
	case "ARM", "ARMV7":
		fg = lipgloss.Color("#1ABC9C")
	case "S390X", "PPC64LE", "RISCV64", "386", "I386":
		fg = lipgloss.Color("#9B59B6")
	default:
		fg = components.ColorHighlight
	}
	s := lipgloss.NewStyle().Foreground(fg).Bold(true)
	if selected {
		s = s.Background(components.ColorSelectedBg)
	}
	return s
}

// padRightStr right-pads a string with spaces up to width, truncating if longer.
func padRightStr(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// padLeftStr left-pads a string with spaces up to width, truncating if longer.
func padLeftStr(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// Helper functions

// getScrollIndicator returns a scroll indicator showing current position and scroll availability
func (m *Model) getScrollIndicator(totalItems int) string {
	if totalItems == 0 {
		return ""
	}

	var b strings.Builder
	indicatorStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	highlightStyle := lipgloss.NewStyle().Foreground(components.ColorHighlight)

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
	indicatorStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	highlightStyle := lipgloss.NewStyle().Foreground(components.ColorHighlight)
	lineStyle := lipgloss.NewStyle().Foreground(components.ColorBorder)

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
	var (
		keyStyle    lipgloss.Style
		stringStyle lipgloss.Style
		numberStyle lipgloss.Style
		boolStyle   lipgloss.Style
		nullStyle   lipgloss.Style
		punctStyle  lipgloss.Style
	)
	if components.DarkBackground {
		keyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#87CEEB"))
		stringStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))
		numberStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#D19A66"))
		boolStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B"))
		nullStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370"))
		punctStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ABB2BF"))
	} else {
		keyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#005577"))
		stringStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#226622"))
		numberStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#884400"))
		boolStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#774400"))
		nullStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		punctStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	}

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

// dotStyle returns a lipgloss style with the given fg color, applying the
// selection bg when the row is selected so the dot remains visible.
func dotStyle(fg lipgloss.Color, selected bool) lipgloss.Style {
	s := lipgloss.NewStyle().Foreground(fg)
	if selected {
		s = s.Background(components.ColorSelectedBg).Bold(true)
	}
	return s
}

// statusLabel returns the (text, color) pair shown in the STATUS column for
// a given resource. When the resource is currently the target of an action
// (Stopping/Starting/Restarting/Deleting/Pulling), the in-progress verb
// wins over the underlying state so feedback is immediate.
//
// The fallback chain is: action override → real state → "—".
func (m *Model) statusText(targetID, realStatus string) (string, lipgloss.Color) {
	if m.actionInProgress && m.actionTargetID == targetID && m.actionLabel != "" {
		// actionLabel is "Verb <name>"; take the verb, uppercase for the
		// STATUS column so it matches the static labels visually.
		verb := strings.SplitN(m.actionLabel, " ", 2)[0]
		return strings.ToUpper(verb) + "…", components.ColorHighlight
	}
	return statusTextStatic(realStatus)
}

// statusTextStatic maps tinyd's internal status strings to (label, color).
// Centralizing the mapping means every tab renders the same label for the
// same state — no drift between containers/images/models. Pulls colors
// from the shared statusGreen/Yellow/Red/Gray palette set by InitTheme.
func statusTextStatic(s string) (string, lipgloss.Color) {
	switch s {
	// Containers
	case "RUNNING":
		return "RUNNING", statusGreen
	case "STOPPED":
		return "STOPPED", statusRed
	case "PAUSED":
		return "PAUSED", statusYellow
	case "RESTARTING":
		return "RESTARTING", statusYellow
	case "ERROR":
		return "ERROR", statusRed

	// Images
	case "IMG_IN_USE":
		return "IN USE", statusGreen
	case "IMG_UNUSED":
		return "UNUSED", statusGray
	case "IMG_DANGLING":
		return "DANGLING", statusRed

	// Models
	case "MDL_AVAILABLE":
		return "AVAILABLE", statusGray
	case "MDL_LOADED":
		return "LOADED", statusGreen

	default:
		return "—", statusGray
	}
}

// renderStatusCell renders the STATUS column value with the right color,
// applying the selection background when the row is selected so the
// highlight remains continuous.
//
// On the selected row we drop the per-status color (red/yellow/etc.) and
// use the selection foreground instead. Red text on blue background
// causes chromatic aberration on most LCDs and is hard to read; the
// colored dot in the leftmost column already carries the status signal.
func (m *Model) renderStatusCell(text string, fg lipgloss.Color, width int, selected bool) string {
	s := lipgloss.NewStyle().Foreground(fg)
	if selected {
		s = lipgloss.NewStyle().
			Background(components.ColorSelectedBg).
			Foreground(components.ColorSelectedFg).
			Bold(true)
	}
	return s.Render(padRightStr(text, width))
}

// getStatusDot returns a colored status indicator based on container status.
// When `selected` is true, the selection background is composited in so the
// dot stays readable against the row highlight.
func (m *Model) getStatusDot(status string, selected bool) string {
	switch status {
	case "RUNNING":
		return dotStyle(statusGreen, selected).Render("●")
	case "STOPPED":
		return dotStyle(statusGray, selected).Render("○")
	case "PAUSED":
		return dotStyle(statusYellow, selected).Render("●")
	case "ERROR":
		return dotStyle(statusRed, selected).Render("●")
	case "RESTARTING":
		return dotStyle(statusYellow, selected).Render("●")
	default:
		return dotStyle(statusGray, selected).Render("○")
	}
}

// getImageStatusDot returns a colored status indicator based on image status
func (m *Model) getImageStatusDot(img types.Image, selected bool) string {
	if img.InUse {
		return dotStyle(statusGreen, selected).Render("●")
	} else if img.Dangling {
		return dotStyle(statusRed, selected).Render("●")
	}
	return dotStyle(statusGray, selected).Render("○")
}

// getInUseDot returns a green ●/gray ○ for volumes & networks (in-use flag)
func (m *Model) getInUseDot(inUse bool, selected bool) string {
	if inUse {
		return dotStyle(statusGreen, selected).Render("●")
	}
	return dotStyle(statusGray, selected).Render("○")
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
	confirmStyle := lipgloss.NewStyle().Bold(true)

	// Active YES button: black text on green background
	yesActiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#00FF00"))

	// Active NO button: black text on red background
	noActiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#FF0000"))

	inactiveStyle := lipgloss.NewStyle().Foreground(components.ColorDim)

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
	case types.TabContainers: // dynamic based on selected container status
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
	case types.TabImages:
		sep := lipgloss.NewStyle().Foreground(components.ColorBorder).Render("│")
		shortcuts = []string{
			renderShortcut("R", "un"),
			renderShortcut("U", "pdate to latest"),
			renderShortcut("I", "nspect"),
			renderShortcut("D", "elete"),
			sep,
			renderShortcut("P", "ull image"),
		}
	case types.TabVolumes:
		shortcuts = []string{
			renderShortcut("I", "nspect"),
			renderShortcut("D", "elete"),
		}
	case types.TabNetworks:
		shortcuts = []string{
			renderShortcut("I", "nspect"),
			renderShortcut("D", "elete"),
		}
	case types.TabModels: // Docker Model Runner
		shortcuts = []string{
			renderShortcut("C", "hat"),
			renderShortcut("R", "un (shell)"),
			renderShortcut("I", "nspect"),
			renderShortcut("D", "elete"),
			lipgloss.NewStyle().Foreground(components.ColorBorder).Render("│"),
			renderShortcut("P", "ull model"),
			renderShortcut("Y", " Copy curl"),
		}
	}

	// Add common shortcuts
	shortcuts = append(shortcuts,
		renderShortcut("?", " Help"),
	)

	return strings.Join(shortcuts, " ")
}

// renderShortcut formats a keyboard shortcut with the chord letter
// underlined. The letter uses the terminal's default foreground + bold
// (rather than ColorBright) so it stays visible even when the dark
// palette has been loaded onto a light terminal.
func renderShortcut(key string, rest ...string) string {
	keyStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	textStyle := lipgloss.NewStyle().Foreground(components.ColorDim)

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

// --- Run image modal renders ---

// renderRunModal renders the full-screen Run image form.
//
// Layout:
//   Run image — repo:tag                              [TAB] Next  [Ctrl+R] Run  [ESC] Cancel
//   ─────────
//   Container name
//   > _________________
//
//   Ports (host:container)
//     8080:80
//   > 3000:3000
//
//   Volumes
//     data-vol -> /data
//   > [+ Add volume]
//
//   Env vars (KEY=value)
//     NODE_ENV=prod
//   > FOO=bar
//
//   [ Run ]
func (m *Model) renderRunModal() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	lineStyle := lipgloss.NewStyle().Foreground(components.ColorBorder)
	dimStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	normalStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	brightStyle := lipgloss.NewStyle().Bold(true)
	focusStyle := lipgloss.NewStyle().Bold(true)
	buttonStyle := lipgloss.NewStyle().
		Background(components.ColorSelectedBg).
		Foreground(components.ColorSelectedFg).
		Bold(true).
		Padding(0, 2)

	// Header
	imgRef := ""
	if m.selectedImage != nil {
		imgRef = m.selectedImage.Repository + ":" + m.selectedImage.Tag
	}
	header := "Run image — " + imgRef
	right := "[TAB] Next  [ENTER] Add  [Ctrl+R] Run  [ESC] Cancel"
	spacing := strings.Repeat(" ", max(1, m.width-len(header)-len(right)-4))
	b.WriteString(titleStyle.Render(header))
	b.WriteString(spacing)
	b.WriteString(helpStyle.Render(right))
	b.WriteString("\n")
	b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n\n")

	// --- Container name ---
	b.WriteString(sectionLabel("Container name", m.runModalField == types.RunFieldName))
	b.WriteString("\n")
	b.WriteString(renderRunInput(m.runContainerName, "auto-generated if blank",
		m.runModalField == types.RunFieldName, m.animationFrame))
	b.WriteString("\n\n")

	// --- Ports ---
	b.WriteString(sectionLabel("Ports (host:container)", m.runModalField == types.RunFieldPortInput))
	b.WriteString("\n")
	for _, p := range m.runPorts {
		b.WriteString("   ")
		b.WriteString(normalStyle.Render(p.Host + ":" + p.Container))
		b.WriteString("\n")
	}
	b.WriteString(renderRunInput(m.runPortInput, "e.g. 8080:80 then ENTER",
		m.runModalField == types.RunFieldPortInput, m.animationFrame))
	b.WriteString("\n\n")

	// --- Volumes ---
	b.WriteString(sectionLabel("Volumes", m.runModalField == types.RunFieldVolumeAdd))
	b.WriteString("\n")
	for _, v := range m.runVolumes {
		src := v.Host
		kind := "(bind)"
		if v.IsNamed {
			src = v.VolumeName
			kind = "(volume)"
		}
		b.WriteString("   ")
		b.WriteString(normalStyle.Render(src + " → " + v.Container))
		b.WriteString(" ")
		b.WriteString(dimStyle.Render(kind))
		b.WriteString("\n")
	}
	addLabel := "[+ Add volume]"
	if m.runModalField == types.RunFieldVolumeAdd {
		b.WriteString(" > ")
		b.WriteString(focusStyle.Render(addLabel))
		b.WriteString(" ")
		b.WriteString(dimStyle.Render("(ENTER opens picker)"))
	} else {
		b.WriteString("   ")
		b.WriteString(dimStyle.Render(addLabel))
	}
	b.WriteString("\n\n")

	// --- Env vars ---
	b.WriteString(sectionLabel("Env vars (KEY=value)", m.runModalField == types.RunFieldEnvInput))
	b.WriteString("\n")
	for _, e := range m.runEnvVars {
		b.WriteString("   ")
		b.WriteString(normalStyle.Render(e.Key + "=" + e.Value))
		b.WriteString("\n")
	}
	b.WriteString(renderRunInput(m.runEnvInput, "e.g. NODE_ENV=production then ENTER",
		m.runModalField == types.RunFieldEnvInput, m.animationFrame))
	b.WriteString("\n\n")

	// --- Submit button ---
	if m.runModalField == types.RunFieldSubmit {
		b.WriteString(" > ")
		b.WriteString(buttonStyle.Render("Run container"))
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("(ENTER or Ctrl+R)"))
	} else {
		b.WriteString("   ")
		b.WriteString(dimStyle.Render("[ Run container ] — Ctrl+R any time"))
	}
	b.WriteString("\n")

	_ = brightStyle
	return b.String()
}

// sectionLabel renders a section header with a focus indicator.
func sectionLabel(label string, focused bool) string {
	if focused {
		return lipgloss.NewStyle().Bold(true).Render(" " + label)
	}
	return lipgloss.NewStyle().Foreground(components.ColorDim).Render(" " + label)
}

// renderRunInput renders an input row: a prompt "> " when focused (with a
// blinking cursor), or a dim hint when unfocused.
func renderRunInput(value, placeholder string, focused bool, frame int) string {
	normalStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	dimStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	brightStyle := lipgloss.NewStyle().Bold(true)

	if !focused {
		if value == "" {
			return "   " + dimStyle.Render("(none)")
		}
		return "   " + normalStyle.Render(value)
	}
	prefix := " > "
	cursor := "▌"
	if frame%2 == 1 {
		cursor = " "
	}
	if value == "" {
		return prefix + dimStyle.Render(placeholder) + brightStyle.Render(cursor)
	}
	return prefix + normalStyle.Render(value) + brightStyle.Render(cursor)
}

// renderVolumePicker renders the volume-add sub-view.
func (m *Model) renderVolumePicker() string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	lineStyle := lipgloss.NewStyle().Foreground(components.ColorBorder)
	dimStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	normalStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	focusStyle := lipgloss.NewStyle().Bold(true)

	header := "Add volume mount"
	right := "[↑↓] Move  [ENTER] Select  [ESC] Back"
	spacing := strings.Repeat(" ", max(1, m.width-len(header)-len(right)-4))
	b.WriteString(titleStyle.Render(header))
	b.WriteString(spacing)
	b.WriteString(helpStyle.Render(right))
	b.WriteString("\n")
	b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n\n")

	switch m.runVolumePickerMode {
	case types.VolumePickerChoose:
		options := []string{
			"Select an existing volume",
			"Create a new named volume",
			"Bind mount (pick a host path)",
		}
		for i, opt := range options {
			if i == m.runVolumePickerIndex {
				b.WriteString(" > ")
				b.WriteString(focusStyle.Render(opt))
			} else {
				b.WriteString("   ")
				b.WriteString(normalStyle.Render(opt))
			}
			b.WriteString("\n")
		}

	case types.VolumePickerExisting:
		listFocused := m.runVolumePickerSub == 0
		pathFocused := m.runVolumePickerSub == 1
		b.WriteString(sectionLabel("Available volumes", listFocused))
		b.WriteString("\n")
		if len(m.volumes) == 0 {
			b.WriteString("   ")
			b.WriteString(dimStyle.Render("(no volumes — press ESC and pick another option)"))
			b.WriteString("\n")
		} else {
			pickedStyle := lipgloss.NewStyle().Bold(true)
			for i, v := range m.volumes {
				selected := i == m.runVolumePickerIndex
				prefix := "   "
				style := normalStyle
				switch {
				case selected && listFocused:
					prefix = " > "
					style = focusStyle
				case selected:
					prefix = " • "
					style = pickedStyle
				}
				inUse := ""
				if v.InUse {
					inUse = dimStyle.Render(" (in use)")
				}
				b.WriteString(prefix)
				b.WriteString(style.Render(v.Name))
				b.WriteString(inUse)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
		b.WriteString(sectionLabel("Container mount path", pathFocused))
		b.WriteString("\n")
		b.WriteString(renderRunInput(m.runVolumeContInput, "e.g. /data", pathFocused, m.animationFrame))
		b.WriteString("\n\n")
		if listFocused {
			b.WriteString(dimStyle.Render(" ↑/↓ to pick a volume, ENTER (or TAB) to set the path."))
		} else {
			b.WriteString(dimStyle.Render(" Type the path and press ENTER to attach. TAB to go back to list."))
		}

	case types.VolumePickerNew:
		nameFocused := m.runVolumePickerSub == 0
		contFocused := m.runVolumePickerSub == 1
		b.WriteString(sectionLabel("New volume name", nameFocused))
		b.WriteString("\n")
		b.WriteString(renderRunInput(m.runVolumeNameInput, "e.g. my-data", nameFocused, m.animationFrame))
		b.WriteString("\n\n")
		b.WriteString(sectionLabel("Container mount path", contFocused))
		b.WriteString("\n")
		b.WriteString(renderRunInput(m.runVolumeContInput, "e.g. /data", contFocused, m.animationFrame))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render(" Press TAB (or ENTER on name) to switch fields, ENTER on path to attach."))

	case types.VolumePickerBind:
		b.WriteString(sectionLabel("Host path", false))
		b.WriteString("\n")
		b.WriteString("   ")
		b.WriteString(normalStyle.Render(m.runVolumeHostInput))
		b.WriteString("\n\n")
		b.WriteString(sectionLabel("Container mount path", true))
		b.WriteString("\n")
		b.WriteString(renderRunInput(m.runVolumeContInput, "e.g. /data", true, m.animationFrame))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render(" Press ENTER to attach the bind mount."))
	}

	return b.String()
}

// renderFileBrowser renders the host-path picker.
func (m *Model) renderFileBrowser() string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	lineStyle := lipgloss.NewStyle().Foreground(components.ColorBorder)
	dimStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	normalStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	selectedStyle := lipgloss.NewStyle().
		Background(components.ColorSelectedBg).
		Foreground(components.ColorSelectedFg).
		Bold(true)

	header := "Choose host path"
	right := "[↑↓] Move  [ENTER] Open  [F] Select dir  [ESC] Back"
	spacing := strings.Repeat(" ", max(1, m.width-len(header)-len(right)-4))
	b.WriteString(titleStyle.Render(header))
	b.WriteString(spacing)
	b.WriteString(helpStyle.Render(right))
	b.WriteString("\n")
	b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n")
	b.WriteString(" ")
	b.WriteString(dimStyle.Render("Path: "))
	b.WriteString(normalStyle.Render(m.fileBrowserPath))
	b.WriteString("\n\n")

	// Show up to viewport-height entries.
	visible := m.height - 8
	if visible < 5 {
		visible = 5
	}
	// Adjust scroll window around the selected row.
	if m.fileBrowserIndex < m.fileBrowserScroll {
		m.fileBrowserScroll = m.fileBrowserIndex
	}
	if m.fileBrowserIndex >= m.fileBrowserScroll+visible {
		m.fileBrowserScroll = m.fileBrowserIndex - visible + 1
	}
	start := m.fileBrowserScroll
	end := start + visible
	if end > len(m.fileBrowserEntries) {
		end = len(m.fileBrowserEntries)
	}

	for i := start; i < end; i++ {
		entry := m.fileBrowserEntries[i]
		isDir := strings.HasSuffix(entry, "/")
		display := entry
		if i == m.fileBrowserIndex {
			b.WriteString(selectedStyle.Render(" " + padRightStr(display, m.width-4) + " "))
		} else {
			b.WriteString("  ")
			if isDir {
				b.WriteString(lipgloss.NewStyle().Bold(true).Render(display))
			} else {
				b.WriteString(normalStyle.Render(display))
			}
		}
		b.WriteString("\n")
	}

	if len(m.fileBrowserEntries) > visible {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf(" Showing %d-%d of %d", start+1, end, len(m.fileBrowserEntries))))
	}

	return b.String()
}

// --- Chat view ---

// renderChatView renders a full-screen chat with the selected model.
// Layout:
//
//   Chat — ai/qwen2.5-coder:7b-instruct-q4_K_M          [Ctrl+L] Clear  [ESC] Exit
//   ──────────────────────────────────────────────────────────────────────────
//   you ▎ Explain Go interfaces
//   bot ▎ In Go, an interface is a type that ...
//
//   you ▎ <streaming>
//   ...
//
//   > _____________________________________________
func (m *Model) renderChatView() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	lineStyle := lipgloss.NewStyle().Foreground(components.ColorBorder)
	dimStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	normalStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	brightStyle := lipgloss.NewStyle().Bold(true)
	userTag := lipgloss.NewStyle().Foreground(components.ColorHighlight).Bold(true).Render("you")
	botTag := lipgloss.NewStyle().Bold(true).Render("bot")
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))

	// Header
	header := "Chat — " + m.chatModelRef
	right := "[ENTER] Send  [Ctrl+L] Clear  [PgUp/PgDn] Scroll  [ESC] Exit"
	spacing := strings.Repeat(" ", max(1, m.width-len(header)-len(right)-4))
	b.WriteString(titleStyle.Render(header))
	b.WriteString(spacing)
	b.WriteString(helpStyle.Render(right))
	b.WriteString("\n")
	b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n\n")

	// Transcript area
	transcriptWidth := m.width - 8
	if transcriptWidth < 30 {
		transcriptWidth = 30
	}

	renderTurn := func(role, content string) {
		tag := botTag
		style := normalStyle
		if role == "user" {
			tag = userTag
			style = brightStyle
		}
		// First line gets the tag, continuation lines get a blank.
		lines := wrapLines(content, transcriptWidth)
		for i, line := range lines {
			if i == 0 {
				b.WriteString(" ")
				b.WriteString(tag)
				b.WriteString(dimStyle.Render(" ▎ "))
			} else {
				b.WriteString("     " + dimStyle.Render("▎ "))
			}
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	for _, msg := range m.chatMessages {
		renderTurn(msg.Role, msg.Content)
	}
	if m.chatStreaming || m.chatCurrentResponse != "" {
		live := m.chatCurrentResponse
		if m.chatStreaming {
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			live += " " + frames[m.animationFrame%len(frames)]
		}
		renderTurn("assistant", live)
	}
	if m.chatError != "" {
		b.WriteString(" ")
		b.WriteString(errStyle.Render("error: " + m.chatError))
		b.WriteString("\n\n")
	}

	// Input row at the bottom
	cursor := "▌"
	if m.animationFrame%2 == 1 {
		cursor = " "
	}
	inputBox := m.chatInput
	if inputBox == "" {
		inputBox = dimStyle.Render("type a message and press ENTER…")
	} else {
		inputBox = normalStyle.Render(inputBox)
	}
	b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n ")
	if m.chatStreaming {
		b.WriteString(dimStyle.Render("(generating — wait or ESC to cancel)"))
	} else {
		b.WriteString(brightStyle.Render("> "))
		b.WriteString(inputBox)
		b.WriteString(brightStyle.Render(cursor))
	}
	b.WriteString("\n")

	return b.String()
}

// wrapLines splits s into lines no wider than w runes (rough — counts bytes,
// good enough for monospace ASCII; multibyte glyphs may visually overflow).
func wrapLines(s string, w int) []string {
	if w < 10 {
		w = 10
	}
	var out []string
	for _, raw := range strings.Split(s, "\n") {
		for len(raw) > w {
			cut := w
			// Try to break on a space within the last quarter of the window.
			for i := w - 1; i > w*3/4; i-- {
				if raw[i] == ' ' {
					cut = i
					break
				}
			}
			out = append(out, raw[:cut])
			raw = strings.TrimLeft(raw[cut:], " ")
		}
		out = append(out, raw)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

// renderModelsAPIPanel renders the DMR connection info — base URL,
// selected model ref, and a copy-pasteable curl example. Rendered below
// the model table so the action bar still sits at the bottom.
//
// Value text uses the terminal's default foreground + bold (no explicit
// color) so it stays readable regardless of which palette is loaded —
// otherwise ColorBright collapses to near-white on a light terminal.
func (m *Model) renderModelsAPIPanel() string {
	dimStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	keyStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	valStyle := lipgloss.NewStyle().Bold(true)
	codeStyle := lipgloss.NewStyle()
	lineStyle := lipgloss.NewStyle().Foreground(components.ColorBorder)

	endpoint := m.dmr.ChatEndpoint()
	modelRef := m.currentModelRef()
	if modelRef == "" {
		modelRef = "(select a model)"
	}
	curl := m.currentCurlExample()

	var b strings.Builder
	b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n")
	b.WriteString(" ")
	b.WriteString(keyStyle.Render("API:   "))
	b.WriteString(valStyle.Render(endpoint))
	b.WriteString("  ")
	b.WriteString(dimStyle.Render("(OpenAI-compatible)"))
	b.WriteString("\n ")
	b.WriteString(keyStyle.Render("Model: "))
	b.WriteString(valStyle.Render(modelRef))
	b.WriteString("\n ")
	b.WriteString(keyStyle.Render("curl:  "))
	// Truncate curl to terminal width so it doesn't blow up the layout.
	// Reserve a few extra cols for the trailing "(Y to copy)" hint.
	maxCurl := m.width - 22
	if maxCurl < 30 {
		maxCurl = 30
	}
	if len(curl) > maxCurl {
		curl = curl[:maxCurl-1] + "…"
	}
	b.WriteString(codeStyle.Render(curl))
	b.WriteString("  ")
	b.WriteString(dimStyle.Render("(Y to copy)"))
	b.WriteString("\n")
	return b.String()
}

// renderTagPicker renders the intermediate "choose a variant" screen
// shown after the user picks a model repo from search results. Lists the
// available tags with parsed parameters / quantization / size.
func (m *Model) renderTagPicker() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	lineStyle := lipgloss.NewStyle().Foreground(components.ColorBorder)
	dimStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	normalStyle := lipgloss.NewStyle().Foreground(components.ColorNormal)
	selectedStyle := lipgloss.NewStyle().
		Background(components.ColorSelectedBg).
		Foreground(components.ColorSelectedFg).
		Bold(true)
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))

	header := "Choose a variant — " + m.tagPickerRepo
	right := "[↑↓] Move  [ENTER] Pull this tag  [ESC] Back"
	spacing := strings.Repeat(" ", max(1, m.width-len(header)-len(right)-4))
	b.WriteString(titleStyle.Render(header))
	b.WriteString(spacing)
	b.WriteString(helpStyle.Render(right))
	b.WriteString("\n")
	b.WriteString(lineStyle.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n")

	if m.tagPickerLoading {
		b.WriteString("\n ")
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		b.WriteString(dimStyle.Render(frames[m.animationFrame%len(frames)] + " Loading tags…"))
		b.WriteString("\n")
		return b.String()
	}
	if m.tagPickerError != "" {
		b.WriteString("\n ")
		b.WriteString(errStyle.Render(m.tagPickerError))
		b.WriteString("\n")
		return b.String()
	}
	if len(m.tagPickerTags) == 0 {
		b.WriteString("\n ")
		b.WriteString(dimStyle.Render("No tags available."))
		b.WriteString("\n")
		return b.String()
	}

	// Layout: TAG (fill) | PARAMS (6) | QUANT (10) | SIZE (10) | UPDATED (12)
	totalW := m.width - 6
	paramW, quantW, sizeW, updW := 6, 10, 10, 12
	tagW := totalW - paramW - quantW - sizeW - updW - 8
	if tagW < 20 {
		tagW = 20
	}

	header2 := " " + padRightStr("TAG", tagW) + "  " +
		padRightStr("PARAMS", paramW) + "  " +
		padRightStr("QUANT", quantW) + "  " +
		padRightStr("SIZE", sizeW) + "  " +
		padRightStr("UPDATED", updW)
	b.WriteString(dimStyle.Render(header2))
	b.WriteString("\n")

	visible := m.tagPickerViewportHeight()
	start := m.tagPickerScroll
	end := start + visible
	if end > len(m.tagPickerTags) {
		end = len(m.tagPickerTags)
	}

	for i := start; i < end; i++ {
		t := m.tagPickerTags[i]
		row := " " + padRightStr(truncateWithEllipsis(t.Tag, tagW), tagW) + "  " +
			padRightStr(defaultStr(t.Parameters, "--"), paramW) + "  " +
			padRightStr(defaultStr(t.Quant, "--"), quantW) + "  " +
			padRightStr(defaultStr(t.Size, "--"), sizeW) + "  " +
			padRightStr(defaultStr(t.Updated, "--"), updW)
		if i == m.tagPickerIndex {
			b.WriteString(selectedStyle.Render(row))
		} else {
			b.WriteString(normalStyle.Render(row))
		}
		b.WriteString("\n")
	}

	if len(m.tagPickerTags) > visible {
		b.WriteString("\n ")
		b.WriteString(dimStyle.Render(fmt.Sprintf("Showing %d-%d of %d",
			start+1, end, len(m.tagPickerTags))))
	}
	return b.String()
}

// currentModelRef returns the highlighted model's repo:tag, or "" if none.
func (m *Model) currentModelRef() string {
	if m.selectedRow >= len(m.models) {
		return ""
	}
	return m.models[m.selectedRow].Repository + ":" + m.models[m.selectedRow].Tag
}

// currentCurlExample builds the curl snippet shown in the API panel /
// copied to the clipboard via Y.
func (m *Model) currentCurlExample() string {
	ref := m.currentModelRef()
	if ref == "" {
		ref = "<model>"
	}
	return "curl -s " + m.dmr.ChatEndpoint() +
		" -H 'Content-Type: application/json'" +
		" -d '{\"model\":\"" + ref + "\",\"messages\":[{\"role\":\"user\",\"content\":\"hello\"}]}'"
}

// togglableColumnsForTab returns the columns the user can toggle on the
// active tab via the V picker. Order matches the columns' position in
// the rendered table so the picker reads top-to-bottom = left-to-right.
func togglableColumnsForTab(tab int) []types.ColumnDef {
	switch tab {
	case types.TabContainers:
		return []types.ColumnDef{
			{Key: "status", Label: "Status"},
			{Key: "cpu", Label: "CPU"},
			{Key: "mem", Label: "Memory"},
			{Key: "image", Label: "Image"},
			{Key: "ports", Label: "Ports"},
		}
	case types.TabImages:
		return []types.ColumnDef{
			{Key: "status", Label: "Status"},
			{Key: "size", Label: "Size"},
			{Key: "created", Label: "Created"},
		}
	case types.TabModels:
		return []types.ColumnDef{
			{Key: "status", Label: "Status"},
			{Key: "params", Label: "Parameters"},
			{Key: "quant", Label: "Quantization"},
			{Key: "size", Label: "Size"},
		}
	case types.TabVolumes:
		return []types.ColumnDef{
			{Key: "containers", Label: "Used by (containers)"},
			{Key: "mountpoint", Label: "Mount point"},
		}
	case types.TabNetworks:
		return []types.ColumnDef{
			{Key: "containers", Label: "Containers"},
			{Key: "driver", Label: "Driver"},
			{Key: "scope", Label: "Scope"},
			{Key: "ipv4", Label: "IPv4"},
		}
	}
	return nil
}

// renderColumnPicker renders the V overlay listing togglable columns for
// the active tab with a checkbox each. ↑/↓ moves the cursor, Space (or
// Enter) toggles, V/ESC closes.
func (m *Model) renderColumnPicker() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	dimStyle := lipgloss.NewStyle().Foreground(components.ColorDim)
	focusStyle := lipgloss.NewStyle().Bold(true)

	cols := togglableColumnsForTab(m.activeTab)
	b.WriteString(titleStyle.Render(" Columns"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(" Space/Enter toggles, V or ESC closes"))
	b.WriteString("\n\n")

	if len(cols) == 0 {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("(nothing to toggle on this tab)"))
		b.WriteString("\n")
		return b.String()
	}
	for i, c := range cols {
		mark := "[ ]"
		if m.IsColumnVisible(c.Key) {
			mark = "[x]"
		}
		row := "  " + mark + "  " + c.Label
		if i == m.columnPickerCursor {
			b.WriteString(" > ")
			b.WriteString(focusStyle.Render(row[2:]))
		} else {
			b.WriteString(row)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(" Tip: set TINYD_HIDE_COLS=status,size to persist defaults across runs."))
	return b.String()
}
