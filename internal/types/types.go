// Package types contains all data structures and constants used throughout tinyd.
package types

import "time"

// Container represents a Docker container with display data
type Container struct {
	ID     string
	Name   string
	Status string
	CPU    string
	Mem    string
	Image  string
	Ports  string
}

// Image represents a Docker image
type Image struct {
	ID         string
	Repository string
	Tag        string
	Size       string
	Created    string
	InUse      bool // Whether the image is used by any container
	Dangling   bool // Whether the image has <none> tag/repo
}

// Volume represents a Docker volume
type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Scope      string
	Created    string
	InUse      bool   // Whether the volume is mounted to any container
	Containers string // Comma-separated list of container names using this volume
}

// Network represents a Docker network
type Network struct {
	ID     string
	Name   string
	Driver string
	Scope  string
	IPv4   string
	IPv6   string
	InUse  bool // Whether the network has any connected containers
}

// PortMapping for run modal
type PortMapping struct {
	Host      string
	Container string
}

// VolumeMapping for run modal
type VolumeMapping struct {
	Host       string
	Container  string
	IsNamed    bool
	VolumeName string
}

// EnvVar for run modal
type EnvVar struct {
	Key   string
	Value string
}

// ChatMessage is one turn in a chat conversation with a DMR model.
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// ColumnDef describes a togglable table column. Key is the value stored
// in hiddenColumns; Label is what the picker overlay shows.
type ColumnDef struct {
	Key   string
	Label string
}

// ImageSearchItem is a single Docker Hub search result
type ImageSearchItem struct {
	Name          string
	Description   string
	Stars         int
	Official      bool
	Architectures []string // e.g. ["amd64","arm64"]
	OperatingSys  []string // e.g. ["linux","windows"]
}

// Model represents a local Docker Model Runner model.
type Model struct {
	ID         string
	Repository string // e.g. "ai/qwen2.5-coder"
	Tag        string // e.g. "7b-instruct-q4_K_M"
	Format     string // gguf / safetensors / ...
	Quant      string // Q4_K_M, F16, ...
	ParamSize  string // "7B", "1.5B"
	Size       string // human-readable disk footprint
	Created    string
}

// ModelSearchItem is a hub search result for the ai/ namespace.
type ModelSearchItem struct {
	Name          string
	Description   string
	Stars         int
	Pulls         int
	Architectures []string
}

// ModelTagInfo is one selectable tag for a model repo. Parameters and
// Quantization are best-effort: they're parsed from the tag name when it
// follows the ai/ namespace convention (e.g. "7b-instruct-q4_K_M").
type ModelTagInfo struct {
	Tag        string
	Parameters string // e.g. "7B", "1.5B"
	Quant      string // e.g. "q4_K_M", "f16"
	Size       string // human-readable
	Updated    string // YYYY-MM-DD
}

// ModelTagsMsg carries the result of a Hub tag fetch into the Update loop.
type ModelTagsMsg struct {
	Repo string
	Tags []ModelTagInfo
}

// Message types for Bubble Tea
type ContainerListMsg []Container
type ImageListMsg []Image
type VolumeListMsg []Volume
type NetworkListMsg []Network
type ErrMsg error
type TickMsg time.Time
type AnimationTickMsg time.Time
type ActionSuccessMsg string
type ActionErrorMsg string
type LogsMsg string
type InspectMsg string
type ImageSearchMsg []ImageSearchItem
type ModelListMsg []Model
type ModelSearchMsg []ModelSearchItem
type DMRAvailableMsg bool

// ViewMode represents different UI views
type ViewMode int

const (
	ViewModeList ViewMode = iota
	ViewModeLogs
	ViewModeInspect
	ViewModePortSelector
	ViewModeStopConfirm
	ViewModeFilter
	ViewModeRunImage
	ViewModePullImage
	ViewModePullModel
	ViewModeRunVolumePicker
	ViewModeRunFileBrowser
	ViewModeChat
	ViewModeModelTagPicker
)

// Chat streaming messages — emitted by the chat command chain.
type ChatStartedMsg struct {
	// Reader produces SSE-style "data: {json}" lines from DMR.
	// Caller stores both on the model and closes Body when done.
	Reader interface{} // *bufio.Reader (kept opaque so types/ has no bufio dep)
	Body   interface{} // io.Closer
}

// ChatTokenMsg is one decoded chunk of the assistant's response. Done
// signals end-of-stream; Err is set on a fatal stream error.
type ChatTokenMsg struct {
	Token string
	Done  bool
	Err   string
}

// Run modal field tags — used to know which input row TAB / typing affects.
const (
	RunFieldName       = iota // container name
	RunFieldPortInput         // port host:container input row
	RunFieldVolumeAdd         // "+ Add volume" row (Enter opens picker)
	RunFieldEnvInput          // env key=value input row
	RunFieldSubmit            // [Run] button
	RunFieldCount             // sentinel; keep last
)

// Volume picker mode — which kind of mount the user is configuring.
const (
	VolumePickerChoose = iota // picking type
	VolumePickerExisting      // selecting from m.volumes
	VolumePickerNew           // typing a new named volume
	VolumePickerBind          // bind mount (host path picked via file browser)
)

// Tab indices — declared so the rest of the codebase can stop hard-coding 0..4.
const (
	TabContainers = iota
	TabImages
	TabModels
	TabVolumes
	TabNetworks
)

// Container filter constants
const (
	ContainerFilterAll = iota
	ContainerFilterRunning
)

// Image filter constants
const (
	ImageFilterAll = iota
	ImageFilterInUse
	ImageFilterUnused
	ImageFilterDangling
)

// Volume filter constants
const (
	VolumeFilterAll = iota
	VolumeFilterInUse
	VolumeFilterUnused
)

// Network filter constants
const (
	NetworkFilterAll = iota
	NetworkFilterInUse
	NetworkFilterUnused
)

