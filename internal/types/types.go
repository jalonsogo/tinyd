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

// Message types for Bubble Tea
type ContainerListMsg []Container
type ImageListMsg []Image
type VolumeListMsg []Volume
type NetworkListMsg []Network
type ErrMsg error
type TickMsg time.Time
type ActionSuccessMsg string
type ActionErrorMsg string
type LogsMsg string
type InspectMsg string

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

// Run modal field indices
const (
	RunFieldContainerName = iota
	RunFieldPortHost
	RunFieldPortContainer
	RunFieldVolumeHost
	RunFieldVolumeContainer
	RunFieldEnvKey
	RunFieldEnvValue
)
