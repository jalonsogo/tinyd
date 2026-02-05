# tinyd - Terminal UI for Docker

> A blazingly fast, beautifully minimal Terminal User Interface for Docker management

![tinyd Screenshot](tinyd.png)

## Overview

**tinyd** (Terminal UI Kit) is an interactive terminal-based Docker management tool that transforms Docker operations into a delightful keyboard-driven experience. Built entirely in Go, it provides a clean, minimalist alternative to Docker CLI commands and Docker Desktop GUI.

**Philosophy:** "The best interface is no interface at all"

**Repository:** https://github.com/jalonsogo/tinyd

---

## Tech Stack

### Core Technologies
- **Language:** Go 1.24.5
- **UI Framework:** Bubble Tea (Charm.sh)
- **Styling:** Lipgloss
- **Docker Integration:** Moby Client SDK

### Key Dependencies
- Bubble Tea v1.3.10 - Terminal UI framework
- Lipgloss v1.1.0 - Style and formatting
- Moby Client v0.2.2 - Official Docker SDK
- Charm.sh Suite - Terminal utilities (ansi, cellbuf, term, colorprofile)
- OpenTelemetry - Telemetry and tracing

---

## Features

### Four Main Tabs

#### 1. Containers Tab 🐳
- **Real-time monitoring** with live CPU and memory stats
- **Start/Stop/Restart** containers with single keypress
- **Browser integration** - Open exposed ports directly
- **Interactive logs** - Full-screen scrollable view with fuzzy search
- **Console access** - Interactive shell with altscreen technology
- **Deep inspection** - Stats, mounts, configuration details
- **Safe deletion** - With confirmation modal

#### 2. Images Tab 📦
- Browse all Docker images with size and creation date
- **Layer-by-layer inspection**
- View architecture and configuration
- **Filter by status** (All/In Use/Unused/Dangling)
- **Run containers** from images (interactive modal)
- **Pull images** from registry
- Delete with force option

#### 3. Volumes Tab 💾
- Volume inventory with driver and scope info
- **Real-time container tracking**
- Volume inspection and statistics
- Safe volume deletion

#### 4. Networks Tab 🌐
- View all Docker networks with connection status
- Filter active vs. unused networks
- IPv4/IPv6 subnet information
- Network inspection and container connectivity

---

## Keyboard Shortcuts

### Navigation
| Key | Action |
|-----|--------|
| `↑/↓` or `k/j` | Move selection up/down |
| `←/→` or `h/l` | Switch tabs |
| `1-4` | Jump directly to tab |
| `Enter` | Refresh current view |
| `ESC` | Return to list |
| `q` or `Ctrl+C` | Quit application |

### Actions
| Key | Action |
|-----|--------|
| `s` | Start/Stop containers (toggle) |
| `r` | Restart running containers |
| `c` | Open interactive shell |
| `o` | Open exposed ports in browser |
| `l` | View logs (last 100 lines) |
| `i` | Inspect resource details |
| `D` | Delete with confirmation |
| `R` | Run new container from image |
| `P` | Pull new Docker image |
| `f` | Open filter/search modal |
| `F1` | Toggle help screen |

---

## Architecture

### Component-Based Design

The application follows a clean component-based architecture with six core UI components:

1. **HeaderComponent** - Top header bar with title
2. **TabsComponent** - Tab navigation with visual active state
3. **StatusLineComponent** - Status information and counts
4. **TableComponent** - Tabular data with headers and rows
5. **ActionBarComponent** - Bottom action bar with shortcuts
6. **DetailViewComponent** - Logs and inspect detail views

### Core Data Structures

```go
// Docker Resources
type Container struct {
    ID, Name, Status, CPU, Mem, Image, Ports string
}

type Image struct {
    ID, Repository, Tag, Size, Created string
    InUse, Dangling bool
}

type Volume struct {
    Name, Driver, Mountpoint, Scope, Created, Containers string
    InUse bool
}

type Network struct {
    ID, Name, Driver, Scope, IPv4, IPv6 string
    InUse bool
}

// Application State
type model struct {
    activeTab, selectedRow, scrollOffset, viewportHeight int
    containers, images, volumes, networks []Resource
    dockerClient *client.Client
    // ... modal states, filters, etc.
}
```

### View Modes

- **viewModeList** - Main resource listing
- **viewModeLogs** - Container logs viewer
- **viewModeInspect** - Deep resource inspection
- **viewModePortSelector** - Port selection for browser
- **viewModeStopConfirm** - Delete confirmation
- **viewModeFilter** - Filter/search modal
- **viewModeRunImage** - Run container modal
- **viewModePullImage** - Pull image modal

---

## Design Principles

1. **Component Isolation** - Each UI element is independent and reusable
2. **Immutable Updates** - State changes create new instances
3. **Separation of Concerns** - UI components vs. business logic
4. **Terminal Responsiveness** - Real-time resizing support
5. **Minimalist UI** - Classic terminal aesthetics (unicode, colors)
6. **Efficient Rendering** - String builders for fast output
7. **Modal Overlays** - Clean dimmed background for dialogs

---

## Advanced Capabilities

### Fully Responsive Design
- Adapts to any terminal size (minimum: 60 cols × 13 rows)
- Dynamic layout recalculation on resize

### Altscreen Console
- Interactive shell with toolbar preservation
- Seamless transition between UI and shell

### Modal Dialogs
- Run containers with custom configuration
- Pull images from registry
- Delete confirmations with inline UI

### Inline Filtering
- Real-time search/filter for resources
- Case-insensitive substring matching

### Intelligent Scrolling
- Auto-scroll for large resource lists
- Viewport-aware navigation

### Status Indicators
- 🟢 Green dot - Running containers
- ⚪ Gray dot - Stopped/inactive
- 🟡 Yellow - Dangling images

---

## Terminal Compatibility

Works seamlessly with:
- VSCode integrated terminal
- iTerm2
- Alacrity
- Wezterm
- tmux
- SSH sessions
- Standard terminal emulators

---

## Docker Integration

### Connection Methods
- **Local Docker** - Default connection to local daemon
- **Remote Docker** - Support via `DOCKER_HOST` environment variable
- **Docker Desktop** - Automatic detection on macOS/Windows

### API Integration
Uses official Moby Client SDK for:
- Container lifecycle management
- Image operations and inspection
- Volume management
- Network configuration
- Real-time stats and logs

---

## Recent Updates

### Latest Features
- Full-screen logs view with scrolling
- Fuzzy search in logs (case-insensitive)
- Pull image functionality with interactive modal
- Case-insensitive keyboard shortcuts
- Transparent terminal support

### Recent Improvements
- Delete confirmation modals with inline UI
- Modal overlay system with clean rendering
- Container status display improvements
- Enhanced error handling and panic safety

---

## Project Structure

```
tinyd/
├── main.go                 # Core app logic (4,645 lines)
├── components.go           # Reusable UI components (556 lines)
├── go.mod & go.sum         # Dependency management
├── README.md              # Project documentation
├── CHANGELOG.md           # Feature history
├── tinyd.png              # Screenshot
│
└── docs/                  # Comprehensive documentation
    ├── ARCHITECTURE.md      # Component-based design
    ├── INTERACTIVE_FEATURES.md
    ├── CONSOLE_FEATURE.md
    ├── TAB_NAVIGATION.md
    ├── TABS_GUIDE.md
    ├── SCROLLING_FEATURE.md
    └── DOCKER_INTEGRATION.md
```

---

## Getting Started

### Installation

```bash
git clone https://github.com/jalonsogo/tinyd.git
cd tinyd
go build -o tinyd
./tinyd
```

### Requirements
- Go 1.24.5 or higher
- Docker daemon running
- Terminal with unicode support
- Minimum terminal size: 60 cols × 13 rows

---

## Development Philosophy

tinyd embraces:
- **Minimalism** - Only essential features, no bloat
- **Performance** - Blazingly fast, instant feedback
- **Keyboard-first** - No mouse required
- **Unix Philosophy** - Do one thing well
- **Developer Experience** - Clean code, good documentation

---

## Future Roadmap

Potential enhancements:
- Docker Compose support
- Container statistics graphs
- Custom themes and color schemes
- Multi-host management
- Export/import container configurations
- Kubernetes integration

---

## License & Contribution

Open for contributions and feedback.

**Maintainer:** Javier Alonso
**Repository:** https://github.com/jalonsogo/tinyd
