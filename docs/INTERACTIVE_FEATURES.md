# Interactive Features Guide

tinyd supports full interactive Docker resource management directly from the terminal interface.

## Available Actions

### Universal Actions (All Tabs)

**Available on all resource types:**
- **i / I** - Inspect selected resource (shows detailed information)
- **d / D** - Delete selected resource with inline confirmation
- **f / F** - Open filter modal to filter resources
- **/** - Toggle inline search/filter across current list
- **ESC** - Return to list view / Cancel current operation
- **Enter** - Refresh current tab (in list view)

## Container Tab Features (Tab 1)

### Container Management Actions

**Available Actions:**
- **s / S** - Start/Stop selected container (smart toggle based on state)
- **r / R** - Restart selected container (running containers only)
- **c / C** - Open interactive console/shell (uses altscreen)
- **o / O** - Open container port in browser
- **l / L** - View container logs with search capability
- **i / I** - Inspect container (shows stats and details)
- **d / D** - Delete container with inline confirmation
- **f / F** - Open filter modal

### 1. Start/Stop Containers (s/S key)

**For Running Containers:**
- Press `s` to stop the selected container
- Container gracefully shuts down (10-second timeout)
- Status updates in real-time

**For Stopped Containers:**
- Press `s` to start the selected container
- Container boots with original configuration
- Status bar shows progress

**Visual Feedback:**
```
Stopping container-name...
✓ Stopped container-name
```

### 2. Restart Containers (r/R key)

**For Running Containers:**
- Press `r` to restart the selected container
- Performs graceful stop followed by start
- 10-second timeout for stopping
- Useful for applying configuration changes

**Status Messages:**
```
Restarting container-name...
✓ Restarted container-name
```

### 3. Console Access (c/C key)

**Interactive Shell in Containers:**
- Press `c` to open interactive shell inside running container
- Uses **altscreen** technology for seamless terminal switching
- Terminal temporarily switches to full-screen shell
- Exit shell (type `exit` or Ctrl+D) returns to TUI
- No scrollback pollution or disruption

**Requirements:**
- Container must be in RUNNING state
- Shell must be available in container (bash, sh, or ash)

**See:** `CONSOLE_FEATURE.md` for detailed documentation.

### 4. Open in Browser (o/O key)

**For Containers with Exposed Ports:**
- Press `o` to open exposed ports in browser
- **Single port**: Opens directly at `http://localhost:PORT`
- **Multiple ports**: Shows port selector modal
- Browser opens in background (no focus steal)
- Works on macOS, Linux, and Windows

**Port Selection Modal:**
When container has multiple exposed ports:
```
┌─────────────────────────────────────────────┐
│ Select Port - nginx-proxy                   │
├─────────────────────────────────────────────┤
│                                              │
│ Select which port to open in browser:       │
│                                              │
│  ▸  http://localhost:80                     │
│     http://localhost:443                    │
│     http://localhost:8080                   │
│                                              │
│ ↑/↓ Navigate  |  ENTER Open  |  ESC Cancel │
└─────────────────────────────────────────────┘
```

**Navigation:**
- `↑` / `↓` or `k` / `j` - Select port
- `Enter` - Open selected port
- `ESC` - Cancel

### 5. View Logs (l/L key)

**Full-Screen Logs Viewer:**
- Press `l` to view container logs
- Shows last 100 lines
- Full terminal height display
- Search capability built-in
- Scroll through logs

**Logs View Features:**
- **s / S** - Toggle search mode
- **↑ / k** - Scroll up
- **↓ / j** - Scroll down
- **ESC** - Return to list view

**Search Mode:**
```
┌──────────────────────────────────────────────┐
│ Logs: nginx-proxy          [Search: error█] │
├──────────────────────────────────────────────┤
│ 2024/01/15 10:23:45 [error] connection lost │
│ 2024/01/15 10:24:12 Starting server...      │
│ 2024/01/15 10:24:13 [error] port in use     │
└──────────────────────────────────────────────┘
```

- Type to filter logs (case-insensitive)
- Backspace to delete characters
- Matching lines shown, others hidden
- Press `s` again to exit search

### 6. Container Filters

**Available Filters:**
- **All** - Show all containers (running and stopped)
- **Running** - Show only running containers

**How to Filter:**
1. Press `f` to open filter modal
2. Use `↑` / `↓` to select filter
3. Press `Enter` to apply
4. Active filter shown in status line

```
┌─────────────────────────────────┐
│ Filter Containers               │
├─────────────────────────────────┤
│  ✓  All                         │
│  ○  Running                     │
├─────────────────────────────────┤
│ ↑/↓ Select  |  ENTER Apply      │
└─────────────────────────────────┘
```

### 7. Container Display Information

**Table Columns:**
- Status indicator (● green=running, ○ gray=stopped)
- Container name
- Status (RUNNING/STOPPED/PAUSED)
- CPU usage % (live, running only)
- Memory usage (MB/GB, running only)
- Image name
- Port mappings

**Status Line:**
```
CONTAINERS (25 total, 15 running) [Filter: Running]
```

## Images Tab Features (Tab 2)

### Image Management Actions

**Available Actions:**
- **r / R** - Run container from image (opens run modal)
- **p / P** - Pull new image from registry (opens pull modal)
- **i / I** - Inspect image (layers, architecture, config)
- **d / D** - Delete image with inline confirmation
- **f / F** - Open filter modal

### 1. Run Container from Image (r/R key)

**Multi-Field Form Modal:**

Press `r` on an image to open the run container configuration modal.

**Form Sections:**
1. **Container Name** - Optional custom name
2. **Port Mappings** - Multiple host:container port pairs
3. **Volume Mounts** - Multiple host:container volume pairs
4. **Environment Variables** - Multiple key=value pairs

**Navigation:**
- `Tab` - Next field
- `Shift+Tab` - Previous field
- Type characters to enter values
- `Backspace` - Delete characters
- `Enter` - Add entry (ports/volumes/env vars) or submit form
- `ESC` - Cancel

**Adding Multiple Entries:**

**Ports:**
```
┌────────────────────────────────────────────┐
│ Run Container: nginx:latest                │
├────────────────────────────────────────────┤
│ Container Name: my-nginx█                  │
│                                             │
│ Ports:                                      │
│   Host Port:      8080                      │
│   Container Port: 80█    [Press Enter]     │
│   Added: 8080:80                            │
│                                             │
│   Host Port:      ___                       │
│   Container Port: ___    [Leave empty]     │
│                                             │
│ [Tab] Next Section                          │
└────────────────────────────────────────────┘
```

- Enter host port, press Tab
- Enter container port, press Enter to add
- Pair is added to list
- Fields clear for next entry
- Leave both empty to move to next section

**Volumes:**
```
│ Volumes:                                    │
│   Host Path:      /home/user/data          │
│   Container Path: /data█    [Press Enter]  │
│   Added: /home/user/data:/data              │
```

- Supports absolute paths
- Supports named volumes (e.g., `my-volume:/data`)

**Environment Variables:**
```
│ Environment Variables:                      │
│   Key:   NODE_ENV                           │
│   Value: production█    [Press Enter]       │
│   Added: NODE_ENV=production                │
```

**Submit:**
- After adding all entries, press Enter on empty field
- Or navigate to submit area and press Enter

### 2. Pull Image (p/P key)

**Single-Field Modal:**

```
┌─────────────────────────────────────────────┐
│ Pull Image                                  │
├─────────────────────────────────────────────┤
│                                              │
│ Image Name: nginx:latest█                   │
│                                              │
│ Examples:                                    │
│   nginx:latest                               │
│   postgres:15                                │
│   ghcr.io/owner/repo:tag                    │
│                                              │
│ ENTER Pull  |  ESC Cancel                   │
└─────────────────────────────────────────────┘
```

**Usage:**
1. Press `p` on images tab
2. Type image name (repository:tag format)
3. Press Enter to pull
4. Status bar shows progress

### 3. Image Filters

**Available Filters:**
- **All** - Show all images
- **In Use** - Images used by running containers
- **Unused** - Images not used by any container
- **Dangling** - Broken/orphaned images (`<none>:<none>`)

**Filter Modal:**
```
┌─────────────────────────────────┐
│ Filter Images                   │
├─────────────────────────────────┤
│  ○  All                         │
│  ✓  In Use                      │
│  ○  Unused                      │
│  ○  Dangling                    │
├─────────────────────────────────┤
│ ↑/↓ Select  |  ENTER Apply      │
└─────────────────────────────────┘
```

### 4. Image Display Information

**Table Columns:**
- Repository name
- Tag
- Image size
- Created date (relative, e.g., "2d ago")
- In-use indicator (● if used by containers)
- Dangling indicator (⚠ if `<none>:<none>`)

## Volumes Tab Features (Tab 3)

### Volume Management Actions

**Available Actions:**
- **i / I** - Inspect volume (shows mountpoint, driver, usage)
- **d / D** - Delete volume with inline confirmation
- **f / F** - Open filter modal

### Volume Filters

**Available Filters:**
- **All** - Show all volumes
- **In Use** - Volumes mounted by containers
- **Unused** - Volumes not mounted by any container

### Volume Display Information

**Table Columns:**
- Volume name
- Driver (usually "local")
- Mountpoint path
- Created date
- In-use indicator (● if mounted)
- Container names using volume

## Networks Tab Features (Tab 4)

### Network Management Actions

**Available Actions:**
- **i / I** - Inspect network (shows driver, scope, IP config)
- **d / D** - Delete network with inline confirmation
- **f / F** - Open filter modal

### Network Filters

**Available Filters:**
- **All** - Show all networks
- **In Use** - Networks with connected containers
- **Unused** - Networks without containers

### Network Display Information

**Table Columns:**
- Network ID (short)
- Network name
- Driver (bridge, host, overlay, etc.)
- Scope (local, global, swarm)
- IPv4 subnet (CIDR notation)
- IPv6 subnet (if configured)
- In-use indicator (● if containers connected)

## Delete Confirmation (All Tabs)

### Inline Delete Confirmation

Press `d` or `D` to delete the selected resource.

**Inline Confirmation UI:**
```
┌──────────────────────────────────────────────┐
│ ● nginx-proxy    RUNNING  2.4%  128M  nginx │
│ ⚠ Delete nginx-proxy?  [Yes] [No]          │ ← Dark red background
│   postgres-db    RUNNING  0.8%  512M  post  │
└──────────────────────────────────────────────┘
```

**Navigation:**
- `←` / `h` - Select "No" (default)
- `→` / `l` - Select "Yes"
- `Enter` - Confirm selection
- `ESC` - Cancel (same as No)

**Selection Indicator:**
- `[Yes]` - Selected (will delete)
- `[No]` - Selected (will cancel)

**After Confirmation:**
- Immediate deletion if "Yes"
- Status message shows result
- List refreshes automatically

**Safety:**
- Requires explicit left/right navigation to Yes
- Default is No (safer choice)
- Clear visual warning (dark red background)

## Inline Search (All Tabs)

### Quick List Filtering

Press `/` to toggle inline search mode on any tab.

**Search UI:**
```
┌──────────────────────────────────────────────┐
│ CONTAINERS (5 matching of 25 total)          │
│ Search: nginx█                                │
├──────────────────────────────────────────────┤
│ ● nginx-proxy    RUNNING  2.4%  128M  nginx │
│ ○ nginx-test     STOPPED   --    --   nginx │
└──────────────────────────────────────────────┘
```

**Usage:**
- Press `/` to activate search
- Type to filter list (case-insensitive, substring match)
- Backspace to delete characters
- Press `/` again or `ESC` to exit search
- Matching count shown in status line

**Search Behavior:**
- Filters in real-time as you type
- Searches across all visible columns
- Selection resets to first match
- Scroll position resets
- Works on all tabs (Containers, Images, Volumes, Networks)

## Visual Feedback System

### Status Messages

**Success (Green):**
```
✓ Started nginx-proxy
✓ Stopped postgres-db
✓ Pulled image nginx:latest
```

**Error (Red):**
```
✗ Failed to stop container: permission denied
✗ Image not found: invalid-image:tag
```

**Progress (Yellow):**
```
⟳ Stopping nginx-proxy...
⟳ Pulling nginx:latest...
⟳ Restarting postgres-db...
```

### Status Indicators

**Container Status:**
- ● Green - Running
- ○ Gray - Stopped
- ◐ Yellow - Paused
- ✗ Red - Error state

**Resource Usage:**
- ● Green dot - Resource in use
- ○ Gray dot - Resource unused
- ⚠ Yellow warning - Dangling/broken

## Keyboard Reference

### Navigation
```
↑/k         - Move selection up (auto-scroll)
↓/j         - Move selection down (auto-scroll)
←/→ or h/l  - Switch tabs
1-4         - Jump to specific tab
Ctrl+D/I/V/N - Jump to tab by shortcut
```

### Universal Actions
```
i/I         - Inspect resource
d/D         - Delete resource (inline confirmation)
f/F         - Open filter modal
/           - Toggle inline search
Enter       - Refresh list / Confirm action
ESC         - Return to list / Cancel
F1          - Toggle help screen
q/Ctrl+C    - Quit application
```

### Container-Specific
```
s/S         - Start/Stop container
r/R         - Restart container
c/C         - Open console (shell)
o/O         - Open port in browser
l/L         - View logs
```

### Image-Specific
```
r/R         - Run container from image
p/P         - Pull new image
```

### Modal Navigation
```
Tab         - Next field (in forms)
Shift+Tab   - Previous field (in forms)
↑/↓ or k/j  - Select option (in lists)
←/→ or h/l  - Select Yes/No (delete confirmation)
Enter       - Confirm / Add entry
ESC         - Cancel / Exit
Backspace   - Delete character (in text fields)
```

## System Requirements

### Browser Opening
- **macOS**: Uses `open` command (built-in)
- **Linux**: Requires `xdg-open` (usually pre-installed)
- **Windows**: Uses `start` command (built-in)

### Docker Permissions
- User must be in `docker` group (Linux)
- Docker daemon must be running
- Proper `DOCKER_HOST` if using remote Docker

## Tips & Best Practices

### Efficient Workflows

**Quick Container Management:**
```
1. Navigate with j/k
2. Press s to toggle start/stop
3. Press o to open in browser
4. Press l to check logs
```

**Rapid Filtering:**
```
1. Press f to filter
2. Select filter with ↑/↓
3. Press Enter
4. Press / for quick search within filter
```

**Multi-Container Operations:**
```
1. Filter to show only running containers
2. Navigate and stop each one with s
3. Clear filter to see all
```

**Image Management:**
```
1. Press f to show dangling images
2. Press d to delete unwanted ones
3. Press p to pull new versions
```

### Troubleshooting

**Container Won't Start:**
1. Press `l` to view logs
2. Check for port conflicts
3. Press `i` to inspect configuration

**Out of Disk Space:**
1. Press `2` for Images tab
2. Press `f` to filter Unused
3. Delete large unused images with `d`

**Network Issues:**
1. Press `4` for Networks tab
2. Press `i` to inspect network config
3. Verify subnet ranges don't conflict

---

**tinyd provides a complete, keyboard-driven Docker management experience!** ⌨️
