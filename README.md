# Docker TUI

A Terminal User Interface (TUI) for Docker management built with Go and Bubble Tea, designed from a Pencil specification.

## Features

- 📦 **Live Docker Integration** - Connects to Docker API for real container data
- 🔄 **Auto-Refresh** - Updates container stats every 5 seconds
- 📊 **Real-time Metrics** - CPU and memory usage for running containers
- 🎮 **Interactive Container Management**:
  - Start/Stop containers with a keypress
  - Restart running containers
  - Open container ports in your browser
- 📜 **Smart Scrolling** - Handles large lists with viewport scrolling
  - Shows 10 items at a time
  - Auto-scrolls when navigating beyond visible area
  - Scroll indicator shows position (e.g., "[1-10 of 50]")
- 🎨 Classic terminal aesthetics with green/yellow color scheme
- ⌨️  Fully keyboard-driven interface
- 🐳 **Docker Socket Support** - Works with local and remote Docker daemons
- 🔌 **Full Tab Support** - Complete views for Containers, Images, Volumes, and Networks

## Prerequisites

- Go 1.19 or higher
- Docker daemon running (locally or remotely)
- Access to Docker socket (usually `/var/run/docker.sock` on Linux/macOS)

## Installation

```bash
go mod download
go build -o docker-tui
```

## Docker Connection

The application connects to Docker using the standard Docker environment variables:

- **Local Docker**: Works out of the box if Docker is running
- **Remote Docker**: Set `DOCKER_HOST` environment variable
  ```bash
  export DOCKER_HOST=tcp://remote-host:2376
  ```
- **Docker Desktop**: Automatically detected on macOS and Windows

### Troubleshooting Connection Issues

If you see a connection error:
1. Verify Docker is running: `docker ps`
2. Check Docker socket permissions
3. Ensure `DOCKER_HOST` is set correctly (if using remote Docker)

## Usage

Run the application:

```bash
./docker-tui
```

Or run directly with Go:

```bash
go run main.go
```

## Keyboard Shortcuts

### Navigation
- `↑` or `k` - Move selection up (auto-scrolls)
- `↓` or `j` - Move selection down (auto-scrolls)
- `←` or `h` - Previous tab
- `→` or `l` - Next tab
- `1-4` - Jump to specific tab
- `Ctrl+D` / `Ctrl+I` / `Ctrl+V` / `Ctrl+N` - Quick tab access

### Container Actions
- `s` - Start/Stop selected container (toggles based on current state)
- `r` - Restart selected container
- `c` - Open console (launches altscreen with info header)
- `o` - Open container's exposed port in browser (shows selector for multiple ports)
- `l` - View container logs (last 100 lines)
- `i` - Inspect container (stats, image, bind mounts)
- `f` - Filter containers/images/volumes/networks
- `Enter` - Refresh list / Select port (in port selector)
- `ESC` - Return from detail views (logs/inspect/port selector)

### Controls
- `F1` - Toggle help screen
- `q` or `Ctrl+C` - Quit application

**Note:** Container data auto-refreshes every 5 seconds. Available actions are shown in the status bar at the bottom.

## Tab Views

### 1. Containers Tab (Default)
Displays all Docker containers with live data:
- Container ID and name
- Status (RUNNING/STOPPED/PAUSED/ERROR) - color-coded
- CPU usage percentage (for running containers)
- Memory usage in human-readable format
- Docker image name
- Exposed ports (public and private)
- **Interactive actions**: Start/Stop, Restart, Open in browser
- **Smart sorting**: Containers ordered by status (Running → Paused → Error → Stopped)

### 2. Images Tab
Lists all Docker images on the system:
- Image ID (short format)
- Repository name
- Tag
- Size (human-readable)
- Created time (relative, e.g., "2d ago")

### 3. Volumes Tab
Shows all Docker volumes:
- Volume name
- Driver type
- Mountpoint path
- Scope (local/global)
- Created time

### 4. Networks Tab
Displays Docker networks:
- Network ID
- Network name
- Driver (bridge, host, overlay, etc.)
- Scope (local/global/swarm)
- IPv4 subnet
- IPv6 subnet (if configured)

All data is fetched live from the Docker API and refreshes automatically every 5 seconds.

## Architecture

Built using:
- **Bubble Tea** - TUI framework with The Elm Architecture
- **Lipgloss** - Style definitions and terminal styling
- **Go** - System programming language

## Design

This TUI was generated from a Pencil design specification, ensuring pixel-perfect rendering that matches the original design mockup.

The interface uses:
- Monospace font (JetBrains Mono recommended)
- Classic terminal colors (Green, Yellow, Red, Cyan)
- Box-drawing characters for borders
- Selected row highlighting
