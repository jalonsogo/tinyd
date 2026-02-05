# Changelog

All notable changes to TUIK will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added
- **Full-screen logs view** - Logs now use the entire terminal height instead of fixed 15 lines
- **Fuzzy search in logs** - Press `S` in logs view to search with case-insensitive substring filtering
- **Pull image functionality** - Press `P` on images tab to pull new Docker images with interactive modal
- **Case-insensitive keyboard shortcuts** - All letter key triggers work with both uppercase and lowercase
- **Transparent terminal support** - Removed all background colors for better terminal transparency

### Changed
- Logs view now displays search button `[Search]` with S underscored in header
- When search is activated, input field appears: `[Search: query█]`
- Scroll position resets automatically when search query changes
- Run container modal (`R` key) now context-aware on images tab

### Fixed
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
