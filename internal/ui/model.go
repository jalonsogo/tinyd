// Package ui contains the TUI model, update logic, and view rendering for tinyd.
package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"tinyd/internal/components"
	"tinyd/internal/docker"
	"tinyd/internal/types"
)

// Model represents the application state
type Model struct {
	// Docker client
	docker *docker.Client

	// Data
	containers []types.Container
	images     []types.Image
	volumes    []types.Volume
	networks   []types.Network

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
	err              error
	animationFrame   int // For animated status indicators

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
	runContainerName   string
	runPortHost        string
	runPortContainer   string
	runPorts           []types.PortMapping
	runVolumes         []types.VolumeMapping
	runEnvVars         []types.EnvVar
	runSelectedVolume  string
	runVolumeHost      string
	runVolumeContainer string
	runEnvKey          string
	runEnvValue        string
	runModalField      int

	// Pull image modal
	pullImageName string

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

// NewModel creates an initial model with default state
func NewModel() (*Model, error) {
	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, err
	}

	// Initialize tab items
	tabs := []components.TabItem{
		{Name: "Containers", Shortcut: "^D"},
		{Name: "Images", Shortcut: "^I"},
		{Name: "Volumes", Shortcut: "^V"},
		{Name: "Networks", Shortcut: "^N"},
	}

	return &Model{
		docker:         dockerClient,
		activeTab:      0,
		selectedRow:    0,
		scrollOffset:   0,
		viewportHeight: 10,
		width:          90,
		height:         35,
		loading:        true,
		currentView:    types.ViewModeList,

		// Initialize components
		header:     components.NewHeaderComponent("tinyd v2.0.1", "[F1] Help [Q]uit"),
		tabs:       components.NewTabsComponent(tabs, 0),
		actionBar:  components.NewActionBarComponent(),
		detailView: components.NewDetailViewComponent("", 15),

		// Initialize slices
		containers: []types.Container{},
		images:     []types.Image{},
		volumes:    []types.Volume{},
		networks:   []types.Network{},
		runPorts:   []types.PortMapping{},
		runVolumes: []types.VolumeMapping{},
		runEnvVars: []types.EnvVar{},
	}, nil
}

// Init initializes the model and fetches initial data
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchContainersCmd(),
		m.fetchImagesCmd(),
		m.fetchVolumesCmd(),
		m.fetchNetworksCmd(),
		tickCmd(),
		animationTickCmd(),
	)
}
