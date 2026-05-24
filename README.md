# 🐋 tinyd - Docker Terminal UI

![tinyd mascot — a tiny geometric whale](tinyd.png)

> A blazingly fast, beautifully minimal Terminal User Interface for Docker management. Built with Go and Bubble Tea.

![tinyd](https://img.shields.io/badge/docker-tinyd-blue?style=for-the-badge&logo=docker)
![Go](https://img.shields.io/badge/go-1.19+-00ADD8?style=for-the-badge&logo=go)
![License](https://img.shields.io/badge/license-LOLcense-purple?style=for-the-badge)

## ✨ Why tinyd?

**tinyd** (Terminal UI Kit) transforms Docker management into a delightful terminal experience. No more memorizing complex CLI commands or switching between browser tabs. Everything you need is right at your fingertips.

### 🎯 Standout Features

**📱 Fully Responsive**
- Adapts seamlessly to any terminal size
- Works beautifully in VSCode terminal splits
- Perfect for small screens and tmux panes
- Minimum dimensions: 60 columns × 13 rows
- Real-time resizing without restart

**🔍 Deep Resource Inspection**
- **Images**: Explore layer-by-layer composition, architecture details, and exposed configurations
- **Volumes**: See exactly which containers are using each volume, driver options, and usage statistics
- **Containers**: Full stats, bind mounts, and runtime configuration at a glance

**⚡ Lightning Fast Operations**
- Start/stop containers with a single keypress
- Restart misbehaving services instantly
- Open exposed ports directly in your browser
- Delete resources with confirmation modals
- Run new containers from images interactively

**🎨 Minimalist Design**
- Clean, distraction-free interface
- Classic terminal aesthetics (green/yellow/red color scheme)
- Smart status indicators (green dots for active, gray for inactive, yellow for dangling)
- Intelligent scrolling for large resource lists
- Box-drawing characters for crisp borders

## 🚀 Quick Start

```bash
# Clone the repository
git clone https://github.com/jalonsogo/tinyd.git
cd tinyd

# Build the binary
go build -o tinyd

# Run it!
./tinyd
```

### Prerequisites

- Go 1.19 or higher
- Docker daemon running (local or remote)
- Terminal with Unicode support

## 🎮 Interactive Features

### Container Management
- **`s`** - Start or stop containers (smart toggle)
- **`r`** - Restart running containers
- **`c`** - Open interactive shell with altscreen (preserves TUI state)
- **`o`** - Open exposed ports in browser (port selector for multiple ports)
- **`l`** - View last 100 lines of logs in scrollable view
- **`i`** - Inspect deep: stats, mounts, configuration
- **`D`** - Delete with confirmation (works across all tabs)

### Image Operations
- **`R`** - Run new containers with interactive modal (name, ports, volumes, env vars)
- **`i`** - Inspect layers, architecture, and configuration
- **`D`** - Remove images (with force option)
- **`f`** - Filter by status: All / In Use / Unused / Dangling

### Volume Management
- **`i`** - Inspect volume details, see which containers are attached
- **`D`** - Delete volumes safely
- **Container column** shows which containers use each volume in real-time

### Network Inspection
- View all networks with connection status
- Filter active vs. unused networks
- See IPv4/IPv6 subnet information

## 📊 All Four Tabs

### 1️⃣ Containers (Default)
Real-time container monitoring with live CPU and memory stats:
```
● nginx-proxy     RUNNING   2.3%    128MB   nginx:latest        80:8080,443:8443
● api-server      RUNNING   15.1%   512MB   node:18-alpine      3000:3000
● postgres-db     RUNNING   8.7%    256MB   postgres:15         5432:5432
```

### 2️⃣ Images
Complete image inventory with layer inspection:
```
● node            18-alpine    1.2GB    2d ago
● nginx           latest       142MB    5d ago
● postgres        15           412MB    1w ago
```

### 3️⃣ Volumes
Volume management with container tracking:
```
● app-data        local    nginx-proxy, api-server    2d ago
● postgres-vol    local    postgres-db                1w ago
```

### 4️⃣ Networks
Network topology at a glance:
```
● bridge          bridge   172.17.0.0/16    local
● app-network     bridge   172.18.0.0/16    local
```

## ⌨️ Keyboard Reference

> **Note:** All letter keys work in both uppercase and lowercase (case-insensitive).

### Navigation
| Key | Action |
|-----|--------|
| `↑` / `k` | Move selection up (with auto-scroll) |
| `↓` / `j` | Move selection down (with auto-scroll) |
| `←` / `→` | Previous / next tab |
| `1-4` | Jump directly to tab |

> Navigation keys remain active even while a Docker action (stop, restart, delete, pull...) is in flight — only the action keys themselves are temporarily blocked to prevent queuing.

### Universal Actions
| Key | Action |
|-----|--------|
| `i` | Inspect selected resource |
| `d` | Delete selected resource |
| `?` / `Shift+H` | Toggle help overlay |
| `ESC` | Close overlay / return to list |
| `Enter` | Refresh / confirm |
| `Ctrl+C` ×2 | Quit application |

### Tab-Specific Actions
| Key | Tab | Action |
|-----|-----|--------|
| `s` | Containers | Start / Stop container (toggles by status) |
| `r` | Containers | Restart container |
| `e` | Containers | Open interactive shell (`docker exec -it`) |
| `l` | Containers | View logs |
| `s` | Images | Run a new container from the selected image |
| `p` | Images | Pull image from Docker Hub (search + select) |

## 🎯 Use Cases

### Perfect For:
- **DevOps Engineers**: Quick container health checks during deployments
- **Backend Developers**: Managing local development environments
- **System Administrators**: Monitoring production Docker hosts
- **Students & Learners**: Visual way to understand Docker concepts
- **Terminal Enthusiasts**: Because GUIs are overrated 😎

### Works Great In:
- ✅ VSCode integrated terminal
- ✅ iTerm2 / Alacritty / Wezterm
- ✅ tmux panes
- ✅ GNU Screen sessions
- ✅ SSH sessions (local or remote Docker)
- ✅ Windows Terminal

## 🔧 Configuration

**Local Docker** (default):
```bash
./tinyd
```

**Remote Docker**:
```bash
export DOCKER_HOST=tcp://remote-host:2376
./tinyd
```

**Docker Desktop** (macOS/Windows): Automatically detected!

## 📚 Documentation

Detailed guides available in the [`docs/`](docs/) folder:
- [Interactive Features](docs/INTERACTIVE_FEATURES.md)
- [Console Feature](docs/CONSOLE_FEATURE.md)
- [Tab Navigation](docs/TAB_NAVIGATION.md)
- [Architecture](docs/ARCHITECTURE.md)
- And more...

##  LOLcense

**All rights reserved.**

For {root} sake I'm a designer. Mostly all the code has been written by Claude and ad latere.

## 🙏 Acknowledgments

- Built with [Charm](https://charm.sh/) libraries (Bubble Tea, Lipgloss)
- Inspired by k9s, lazydocker, and other terminal tools

---

**Made with ❤️ for terminal lovers everywhere**

*Because the best interface is no interface at all.*
