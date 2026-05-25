<h1 align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/assets/logo-dark.png">
    <source media="(prefers-color-scheme: light)" srcset=".github/assets/logo-light.png">
    <img alt="tinyd — tiny Docker TUI" src=".github/assets/logo-light.png" width="420">
  </picture>
</h1>

<p align="center"><strong>tinyd. Tiny Docker TUI.</strong></p>

> A blazingly fast, beautifully minimal Terminal User Interface for Docker management. Built with Go and Bubble Tea.

![tinyd](https://img.shields.io/badge/docker-tinyd-blue?style=for-the-badge&logo=docker)
![Go](https://img.shields.io/badge/go-1.19+-00ADD8?style=for-the-badge&logo=go)
![License](https://img.shields.io/badge/license-LOLcense-purple?style=for-the-badge)

## ✨ Why tinyd?

**tinyd** transforms Docker management into a delightful terminal experience. No more memorizing complex CLI commands or switching between browser tabs. Everything you need is right at your fingertips.

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
- **Models**: Inspect raw DMR metadata for local AI models

**⚡ Lightning Fast Operations**
- Start/stop containers with a single keypress
- Restart misbehaving services instantly
- Delete resources with inline confirmation
- Run new containers from images via an interactive modal (name, ports, volumes, env vars)
- Update local images to their latest tag with one key
- Chat with local AI models directly in the TUI (streaming)

**🎨 Minimalist Design**
- Clean, distraction-free interface
- Auto-detects light/dark terminal background (override with `TINYD_THEME=light|dark`)
- Smart status indicators (green dots for active, gray for inactive, yellow for dangling)
- Intelligent scrolling for large resource lists
- Box-drawing characters for crisp borders

## 🚀 Quick Start

### Download a release

Grab the latest binary from the [Releases page](https://github.com/jalonsogo/tinyd/releases) — prebuilt for macOS, Linux, and Windows on both `amd64` and `arm64`.

```bash
# macOS / Linux
tar -xzf tinyd_*_macOS_arm64.tar.gz
./tinyd

# Windows
# Unzip the .zip and run tinyd.exe
```

### Build from source

```bash
git clone https://github.com/jalonsogo/tinyd.git
cd tinyd
go build -o tinyd
./tinyd
```

### Prerequisites

- Go 1.24+ (only for `go build`; not needed for the prebuilt binary)
- Docker daemon running (local or remote)
- Terminal with Unicode support
- (Optional) [Docker Model Runner](https://docs.docker.com/ai/model-runner/) enabled in Docker Desktop, for the Models tab

## 🎮 Interactive Features

### Container Management
- **`s`** — Start or stop containers (smart toggle by status)
- **`r`** — Restart running containers
- **`e`** — Open an interactive shell in the container (`docker exec -it`, suspends the TUI and restores it on exit)
- **`l`** — View the last 100 lines of logs in a scrollable view (`S` inside logs activates a substring search)
- **`i`** — Inspect: stats, mounts, configuration
- **`d`** — Delete with inline Yes/No confirmation

### Image Operations
- **`r`** — Run a new container from the selected image via an interactive modal: name, port mappings (`host:container`), env vars (`KEY=value`), and volumes. The **Volumes** section opens a sub-picker offering three options:
  - **Select an existing volume** from the Volumes tab
  - **Create a new named volume** (name + container path)
  - **Bind mount** via a built-in directory browser (↑/↓ navigate, Enter descends, `F` confirms the current directory)
- **`u`** — Update the selected image to its latest tag (re-pulls `repo:tag`)
- **`p`** — Pull image from Docker Hub (search → results → pull)
- **`i`** — Inspect layers, architecture, configuration
- **`d`** — Delete

### Models (Docker Model Runner)
- **`c`** — In-app **streaming chat** with the selected model. Full-screen transcript, `Ctrl+L` to clear, PgUp/PgDn to scroll, ESC to exit
- **`r`** / **`Shift+R`** — Drop to the shell `docker model run <ref>` REPL
- **`p`** — Pull model from Docker Hub. A second screen lists the available tags with parsed **Parameters** (e.g. `7B`), **Quantization** (e.g. `q4_K_M`), **Size**, and **Updated** date, so you can pick the specific variant before downloading
- **`i`** / **`d`** — Inspect raw DMR JSON / Delete

The Models tab also shows a **connection panel** pinned to the bottom with the OpenAI-compatible base URL, the selected model ref, and a copy-pasteable `curl` example.

### Volume Management
- **`i`** — Inspect volume details, see which containers are attached
- **`d`** — Delete volumes safely
- The Containers column shows which containers use each volume in real-time

### Network Inspection
- View all networks with connection status
- See IPv4/IPv6 subnet information
- **`i`** / **`d`** — Inspect / Delete

## 📊 Five Tabs

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

### 3️⃣ Models (Docker Model Runner)
Manage local AI models pulled from the `ai/` namespace on Docker Hub:
```
○ ai/qwen2.5-coder:7b-instruct-q4_K_M    7B    Q4_K_M    4.2GB
○ ai/llama3.2:3b                          3B    Q5_K_M    2.1GB
○ ai/smollm2:1.7b                         1.7B  F16       3.4GB
```
Requires [Docker Model Runner](https://docs.docker.com/ai/model-runner/) enabled in Docker Desktop. tinyd probes `http://localhost:12434` on startup (override with `DMR_BASE_URL`). If it's not reachable, the tab still loads with a friendly empty state pointing at the docs.

### 4️⃣ Volumes
Volume management with container tracking:
```
● app-data        local    nginx-proxy, api-server    2d ago
● postgres-vol    local    postgres-db                1w ago
```

### 5️⃣ Networks
Network topology at a glance:
```
● bridge          bridge   172.17.0.0/16    local
● app-network     bridge   172.18.0.0/16    local
```

## ⌨️ Keyboard Reference

> **Note:** Most letter keys work in both uppercase and lowercase (case-insensitive). Exceptions: on the Models tab `r` and `R` both open the shell REPL, while `c` opens the in-app chat.

### Navigation
| Key | Action |
|-----|--------|
| `↑` / `k` | Move selection up (with auto-scroll) |
| `↓` / `j` | Move selection down (with auto-scroll) |
| `←` / `→` | Previous / next tab |
| `1–5` | Jump directly to tab |

> Navigation keys remain active even while a Docker action (stop, restart, delete, pull...) is in flight — only the action keys themselves are temporarily blocked to prevent queuing.

### Universal Actions
| Key | Action |
|-----|--------|
| `i` | Inspect selected resource |
| `d` | Delete selected resource (inline Yes/No confirmation) |
| `?` / `Shift+H` | Toggle help overlay |
| `ESC` | Close overlay / return to list / cancel modal |
| `Enter` | Refresh / confirm |
| `Ctrl+C` ×2 | Quit application |

### Tab-Specific Actions
| Key | Tab | Action |
|-----|-----|--------|
| `s` | Containers | Start / Stop (toggles by status) |
| `r` | Containers | Restart |
| `e` | Containers | Open interactive shell (`docker exec -it`) |
| `l` | Containers | View logs (then `s` to search) |
| `r` | Images | Run image — opens interactive modal (name / ports / volumes / env) |
| `u` | Images | Update to latest (re-pulls `repo:tag`) |
| `p` | Images | Pull image from Docker Hub (search + select) |
| `c` | Models | Chat in-app (streaming) |
| `r` / `Shift+R` | Models | Drop to `docker model run` REPL |
| `p` | Models | Pull model — opens variant picker (tag / params / quant / size) |

## 🎯 Use Cases

### Perfect For:
- **DevOps Engineers**: Quick container health checks during deployments
- **Backend Developers**: Managing local development environments
- **System Administrators**: Monitoring production Docker hosts
- **AI / LLM Tinkerers**: Pulling, inspecting, and chatting with local models via Docker Model Runner
- **Students & Learners**: Visual way to understand Docker concepts
- **Terminal Enthusiasts**: Because GUIs are overrated 😎

### Works Great In:
- ✅ VSCode integrated terminal
- ✅ iTerm2 / Alacritty / Wezterm / Ghostty
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

**Docker Desktop** (macOS/Windows): Automatically detected.

**Theme override** (if auto-detection picks the wrong palette):
```bash
TINYD_THEME=light ./tinyd   # or dark
```

**Docker Model Runner endpoint** (defaults to `http://localhost:12434`):
```bash
DMR_BASE_URL=http://my-host:12434 ./tinyd
```

## 📚 Documentation

Detailed guides available in the [`docs/`](docs/) folder:
- [Interactive Features](docs/INTERACTIVE_FEATURES.md)
- [Console Feature](docs/CONSOLE_FEATURE.md)
- [Tab Navigation](docs/TAB_NAVIGATION.md)
- [Architecture](docs/ARCHITECTURE.md)
- [AI Models plan](docs/AI_MODELS_PLAN.md)

##  LOLcense

**All rights reserved.**

For {root} sake I'm a designer. Mostly all the code has been written by Claude and ad latere.

## 🙏 Acknowledgments

- Built with [Charm](https://charm.sh/) libraries (Bubble Tea, Lipgloss)
- Inspired by k9s, lazydocker, and other terminal tools

---

**Made with ❤️ for terminal lovers everywhere**

*Because the best interface is no interface at all.*
