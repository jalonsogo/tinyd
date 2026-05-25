// Package ui contains the TUI model, update logic, and view rendering for tinyd.
package ui

import (
	"bufio"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"tinyd/internal/components"
	"tinyd/internal/dmr"
	"tinyd/internal/docker"
	"tinyd/internal/types"
)

// Model represents the application state
type Model struct {
	// Docker client
	docker *docker.Client
	dmr    *dmr.Client

	// Data
	containers   []types.Container
	images       []types.Image
	volumes      []types.Volume
	networks     []types.Network
	models       []types.Model
	dmrAvailable bool // set once on startup; if false we skip refresh + show empty state

	// Navigation state
	activeTab      int
	selectedRow    int
	scrollOffset   int
	viewportHeight int

	// Display state
	width  int
	height int

	// UI state
	showHelp         bool
	loading          bool
	statusMessage    string
	actionInProgress bool
	actionLabel      string // verb+target shown next to spinner while actionInProgress
	actionTargetID   string // ID/name of the row being acted on (for inline spinner)
	err              error
	animationFrame   int // For animated status indicators
	lastCtrlC        time.Time // For double Ctrl+C detection
	showRawJSON      bool      // Toggle for inspect view

	// View mode
	currentView types.ViewMode

	// Detail views
	logsContent      string
	logsScrollOffset int
	logsSearchMode   bool
	logsSearchQuery  string
	inspectContent   string
	inspectMode      int // 0=stats, 1=image, 2=mounts

	// Selection state
	selectedContainer *types.Container
	selectedImage     *types.Image
	selectedVolume    *types.Volume
	selectedNetwork   *types.Network

	// Port selector
	availablePorts  []string
	selectedPortIdx int

	// Filters
	containerFilter int
	imageFilter     int
	volumeFilter    int
	networkFilter   int
	filterOptions   []string
	selectedFilter  int

	// Run image modal
	runContainerName string
	runPorts         []types.PortMapping
	runVolumes       []types.VolumeMapping
	runEnvVars       []types.EnvVar

	// Current input rows (bottom row of each list — Enter commits to the
	// list above and clears the field).
	runPortInput string // "8080:80"
	runEnvInput  string // "KEY=value"

	// Which input field has focus (see RunField* in types).
	runModalField int

	// Volume picker sub-view state. runVolumePickerMode toggles between
	// the type-chooser and the type-specific input.
	runVolumePickerMode  int // VolumePickerChoose / Existing / New / Bind
	runVolumePickerIndex int // cursor in the chooser / existing list
	runVolumePickerSub   int // 0 = primary cursor, 1 = container-path input
	runVolumeNameInput   string
	runVolumeHostInput   string // for bind mount
	runVolumeContInput   string // container path

	// File browser state — used when configuring a bind mount.
	fileBrowserPath    string   // current directory
	fileBrowserEntries []string // sorted entries (dirs end with /)
	fileBrowserIndex   int
	fileBrowserScroll  int

	// Column visibility — keys correspond to togglable column labels
	// (e.g. "status", "cpu", "mem"). Initialised from TINYD_HIDE_COLS or
	// hard defaults; mutated at runtime via the V column picker overlay.
	hiddenColumns      map[string]bool
	showColumnPicker   bool
	columnPickerCursor int

	// Model tag picker (intermediate screen between "select repo" in the
	// model-pull search results and the actual pull).
	tagPickerRepo    string
	tagPickerTags    []types.ModelTagInfo
	tagPickerIndex   int
	tagPickerScroll  int
	tagPickerLoading bool
	tagPickerError   string

	// Chat with a DMR model.
	chatModelRef        string              // repo:tag the chat targets
	chatMessages        []types.ChatMessage // committed conversation history
	chatInput           string              // current user input
	chatStreaming       bool                // true while the assistant is generating
	chatCurrentResponse string              // tokens received so far for the in-flight reply
	chatError           string              // surfaced once on a stream failure
	chatScrollOffset    int                 // viewport scroll position in the transcript
	chatReader          *bufio.Reader       // live SSE reader while streaming
	chatBody            io.Closer           // underlying body to close at end of stream

	// Pull image modal
	pullImageName string

	// Pull image search flow (p in Images tab)
	// pullStage: 0 = input, 1 = searching, 2 = results, 3 = pulling
	pullStage              int
	pullSearchQuery        string
	pullSearchResults      []types.ImageSearchItem
	pullSearchSelected     int
	pullSearchScrollOffset int
	pullSearchError        string
	pullingImageName       string // image currently being pulled (stage 3)

	// List search (inline filter)
	listSearchMode  bool
	listSearchQuery string

	// Inline delete confirmation
	deleteConfirmMode   bool
	deleteConfirmOption int // 0=Yes, 1=No

	// Components
	header     components.HeaderComponent
	tabs       components.TabsComponent
	actionBar  components.ActionBarComponent
	detailView components.DetailViewComponent
}

// NewModel creates an initial model with default state. version is the build
// version (set via -ldflags in releases, "dev" otherwise) and is shown in
// the header. hidden is the initial set of hidden column keys (typically
// derived from TINYD_HIDE_COLS in main).
func NewModel(version string, hidden map[string]bool) (*Model, error) {
	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, err
	}

	// Initialize tab items
	tabs := []components.TabItem{
		{Name: "Containers", Shortcut: "^D"},
		{Name: "Images", Shortcut: "^I"},
		{Name: "Models", Shortcut: "^M"},
		{Name: "Volumes", Shortcut: "^V"},
		{Name: "Networks", Shortcut: "^N"},
	}

	return &Model{
		docker:         dockerClient,
		dmr:            dmr.NewClient(),
		activeTab:      0,
		selectedRow:    0,
		scrollOffset:   0,
		viewportHeight: 10,
		width:          90,
		height:         35,
		loading:        true,
		currentView:    types.ViewModeList,

		// Initialize components
		header:     components.NewHeaderComponent("tinyd "+version, "[F1] Help [Q]uit"),
		tabs:       components.NewTabsComponent(tabs, 0),
		actionBar:  components.NewActionBarComponent(),
		detailView: components.NewDetailViewComponent("", 15),

		// Initialize slices
		containers: []types.Container{},
		images:     []types.Image{},
		volumes:    []types.Volume{},
		networks:   []types.Network{},
		models:     []types.Model{},
		runPorts:   []types.PortMapping{},
		runVolumes: []types.VolumeMapping{},
		runEnvVars: []types.EnvVar{},

		hiddenColumns: hidden,
	}, nil
}

// IsColumnVisible reports whether a given column key should render. Empty
// keys are always visible (used for fixed columns like the status dot or
// NAME / REPOSITORY:TAG which can't be hidden).
func (m *Model) IsColumnVisible(key string) bool {
	if key == "" {
		return true
	}
	return !m.hiddenColumns[strings.ToLower(key)]
}

// ToggleColumn flips the visibility of a column key.
func (m *Model) ToggleColumn(key string) {
	if m.hiddenColumns == nil {
		m.hiddenColumns = map[string]bool{}
	}
	key = strings.ToLower(key)
	if m.hiddenColumns[key] {
		delete(m.hiddenColumns, key)
	} else {
		m.hiddenColumns[key] = true
	}
}

// Init initializes the model and fetches initial data
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchContainersCmd(),
		m.fetchImagesCmd(),
		m.fetchVolumesCmd(),
		m.fetchNetworksCmd(),
		m.probeDMRCmd(),
		tickCmd(),
		animationTickCmd(),
	)
}
