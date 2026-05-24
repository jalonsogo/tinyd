# Changelog

All notable changes to tinyd will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added
- **Models tab (Docker Model Runner)** — Fifth tab managing AI models via DMR's HTTP API on `localhost:12434`. List local models, inspect (raw JSON), delete with confirmation, pull from Docker Hub's `ai/` namespace using the existing pull-search flow, and `R` to open the chat REPL (`docker model run`). DMR availability is probed on startup; when absent the tab renders a friendly pointer to the docs instead of an error. Configurable via `DMR_BASE_URL`. See `docs/AI_MODELS_PLAN.md` for design notes.
- **Pull from Docker Hub** — Press `P` on the Images tab to search Docker Hub (`docker search` via the API). Type a query, browse results with `↑/↓`, press `Enter` to pull, `Esc` to cancel. The pulling stage shows an animated braille spinner and the image name.
- **In-row action spinner** — While an action runs, the affected row's status dot is replaced with a cyan animated spinner so feedback is visible directly in the list, not just in the action bar.
- **Working help overlay** — `?` (or `Shift+H`) toggles a keybinding panel rendered over the tab content; `Esc` closes it. Previously the keybinding existed but rendered nothing.
- **Full-screen logs view** - Logs now use the entire terminal height instead of fixed 15 lines
- **Fuzzy search in logs** - Press `S` in logs view to search with case-insensitive substring filtering
- **Case-insensitive keyboard shortcuts** - All letter key triggers work with both uppercase and lowercase
- **Transparent terminal support** - Removed all background colors for better terminal transparency

### Changed
- **Help shortcut moved from `H` to `?`** in the action bar — lowercase `h` was always intercepted by the vim-style left binding before reaching the help handler. The on-screen label and accepted key are now consistent.
- **Action bar always reflects current tab + selection** — when an action posted a status message, the shortcut list used to stay stuck on the previously-rendered tab's shortcuts. It now recomputes every render.
- **Navigation works during actions** — `↑/↓/j/k`, `←/→`, `1–4` and `Ctrl+C` are now allowed while a Docker call is in flight. Only the action keys (`s/r/l/e/i/d/p/Enter`) are blocked, to prevent queuing a second operation.
- **`h` no longer switches tabs** — it kept colliding with the delete-confirm Yes selector and the help intent; arrows and `1–4` are the canonical tab nav.
- **Stop/restart command timeout bumped to 30 s** — was 10 s, which raced with Docker's own 10 s SIGTERM grace period (see Fixed).
- **Pull image action bar label** — `P`ull → `P`ull image, to make the destructive nature clearer.
- Logs view now displays search button `[Search]` with S underscored in header
- When search is activated, input field appears: `[Search: query█]`
- Scroll position resets automatically when search query changes
- Run container modal (`R` key) now context-aware on images tab

### Fixed
- **Spurious "stop operation timed out after 15s" errors** — the Go context (10 s default) raced with Docker's hardcoded 10 s SIGTERM grace period, so a normal `docker stop` could surface `DeadlineExceeded` even though the container actually stopped in the background. The context is now 30 s, and the error message no longer hard-codes a misleading duration.
- **Selected-row status dot was invisible** — the table component re-rendered the dot with the selection style, stripping the status color. The dot now composites its status color over the selection background and stays readable.
- **Inactive tabs showed extra `┴` joins** under the bottom border, producing visible ticks. Replaced with `─` so the rule reads as a continuous line.
- Delete modal now properly displays in overlay mode
- Fixed panic when containers have no names (added safety checks)
- Status display correctly shows container states

## [Previous Features]

### Core Features
- **Four tabs**: Containers, Images, Volumes, Networks
- **Container management**: Start, stop, restart, delete containers
- **Interactive console**: Open shell in containers with altscreen preservation
- **Port management**: Open exposed ports in browser with port selector
- **Log viewing**: View last 100 lines of container logs
- **Deep inspection**: View stats, mounts, configuration for all resources
- **Image operations**: Run containers from images, delete images, filter by status
- **Volume tracking**: See which containers use each volume
- **Network inspection**: View network details and connections

### UI/UX
- Fully responsive design adapts to any terminal size
- Minimum width: 60 columns
- Works beautifully in VSCode terminal splits
- Smart status indicators (green/gray/yellow dots)
- Intelligent scrolling for large resource lists
- Clean, minimalist interface with classic terminal aesthetics
- Box-drawing characters for crisp borders

### Navigation
- `↑/↓` or `k/j` - Move selection
- `←/→` or `h/l` - Switch tabs
- `1-4` - Jump directly to tab
- `F1` - Toggle help screen
- `ESC` - Return to list view
- `q` or `Ctrl+C` - Quit

### Actions
- `s/S` - Start/Stop containers (toggle search in logs view)
- `r/R` - Restart containers / Run images
- `c/C` - Open interactive console
- `o/O` - Open port in browser
- `l/L` - View logs
- `i/I` - Inspect resource
- `d/D` - Delete resource
- `f/F` - Open filter modal
- `p/P` - Pull image (images tab only)

---

**Made with ❤️ for terminal lovers everywhere**
