# Interactive Features Guide

## Container Management

The Docker TUI now supports full interactive container management directly from the terminal interface.

## Available Actions

### 1. Start/Stop Containers (`s` key)

**For Running Containers:**
- Press `s` to stop the selected container
- The container will gracefully shut down (10-second timeout)
- Status updates in real-time

**For Stopped Containers:**
- Press `s` to start the selected container
- Container boots up with its original configuration
- Status bar shows progress

### 2. Restart Containers (`r` key)

**For Running Containers:**
- Press `r` to restart the selected container
- Performs a graceful stop followed by start
- Useful for applying configuration changes
- 10-second timeout for stopping

### 3. Open in Browser (`o` key)

**For Containers with Exposed Ports:**
- Press `o` to open exposed ports in your default browser
- **Single port**: Opens directly in browser
- **Multiple ports**: Shows a selection modal
- **Background opening**: Browser opens without stealing focus from TUI
- Automatically constructs `http://localhost:PORT`
- Works on macOS (`open -g`), Linux (`xdg-open`), and Windows (`start /B`)
- Perfect for web applications and APIs

**Port Selection Modal:**
When a container has multiple exposed ports, a modal appears:
```
┌─────────────────────────────────────────────────────────┐
│ Select Port - nginx-proxy                               │
├─────────────────────────────────────────────────────────┤
│                                                          │
│ Select which port to open in browser:                   │
│                                                          │
│  >  http://localhost:80                                 │
│     http://localhost:443                                │
│     http://localhost:8080                               │
│                                                          │
│ ↑/↓ Navigate  |  ENTER Open  |  ESC Cancel             │
└─────────────────────────────────────────────────────────┘
```

**Example:**
- Container exposes ports `80,443,8080`
- Press `o` → Port selector appears
- Use `↑/↓` to navigate
- Press `ENTER` to open selected port
- Press `ESC` to cancel

### 4. Console Access (`c` key) - **NEW with Altscreen!**

**Interactive Shell in Containers:**
- Press `c` to open an interactive shell inside a running container
- Uses **altscreen** technology for seamless terminal switching
- Terminal temporarily switches to full-screen shell
- Exit shell (type `exit` or Ctrl+D) returns to TUI exactly as before
- No scrollback pollution or disruption

**Console Modes:**
- **Docker Exec** (default) - Standard shell access
- **Docker Debug** - Advanced debugging tools (toggle with `d`)

**Usage:**
```
1. Select a RUNNING container
2. Press 'c' to launch altscreen console
3. [Terminal switches to altscreen with toolbar at top]
4. Toolbar shows: container name, mode, ID, exit instructions
5. Run commands in container shell (custom prompt shows container name)
6. Type 'exit' or Ctrl+D to return to TUI
```

**Altscreen Toolbar:**
When the console opens, a persistent toolbar displays:
- Container name (highlighted in yellow)
- Console mode (docker exec or docker debug)
- Container ID
- Exit instructions (how to return to TUI)
- Custom shell prompt with container name

The toolbar provides context at all times without interrupting your workflow.

See `CONSOLE_FEATURE.md` for detailed documentation.

### 5. Toggle Debug Mode (`d` key)

**Switch Console Mode:**
- Press `d` to toggle between `docker exec` and `docker debug`
- Status message shows current mode
- Action bar indicates active mode: `[C]onsole (exec)` or `[C]onsole (debug)`
- Setting persists until changed

### 6. Manual Refresh (`Enter` key)

- Press `Enter` to immediately refresh the container list
- Fetches latest stats and status
- Useful after external Docker operations

## Visual Feedback

### Status Bar

The bottom of the screen shows:

**Available Actions:**
```
[S]top | [R]estart | [O]pen in browser
```

**Action in Progress:**
```
Stopping nginx-proxy...
```

**Success Messages:**
```
Stopped nginx-proxy
Started api-server
Restarted postgres-db
Opening http://localhost:3000 in browser
```

**Error Messages:**
```
ERROR: Failed to stop nginx-proxy: permission denied
ERROR: No ports exposed
```

### Container Status Colors

- 🟢 **Green** - RUNNING status
- 🔴 **Red** - STOPPED status
- 🟡 **Yellow** - Selected container
- ⚪ **Gray** - Stopped container details (CPU, Memory, etc.)

## Usage Examples

### Example 1: Stop a Running Container

1. Use `↑`/`↓` to select a running container
2. Press `s`
3. Status bar shows: "Stopping container-name..."
4. Container stops and list refreshes
5. Status bar shows: "Stopped container-name"

### Example 2: Open Web Application

1. Select a container with exposed ports (e.g., port 3000)
2. Press `o`
3. Browser opens to `http://localhost:3000`
4. Status bar shows: "Opening http://localhost:3000 in browser"

### Example 3: Restart a Misbehaving Container

1. Select the problematic container
2. Press `r`
3. Status bar shows: "Restarting container-name..."
4. Container stops and starts
5. Status bar shows: "Restarted container-name"

## Action States

### Action In Progress

When an action is executing:
- Keyboard input is locked (prevents accidental commands)
- Status message shows progress
- Auto-refresh is paused
- List refreshes automatically when action completes

### Error Handling

If an action fails:
- Error message displayed in red
- Original container state preserved
- You can retry the action immediately
- Check Docker daemon permissions if errors persist

## Tips

1. **Quick Port Opening**: Navigate with `j`/`k`, press `o` for instant browser access
2. **Batch Operations**: Use arrow keys and actions to quickly manage multiple containers
3. **Status Monitoring**: Watch the status bar for confirmation of all actions
4. **Recovery**: If stuck, press `Enter` to force refresh the list

## System Requirements

### Browser Opening

- **macOS**: Uses `open` command (built-in)
- **Linux**: Requires `xdg-open` (usually pre-installed)
- **Windows**: Uses `start` command (built-in)

### Docker Permissions

Container actions require Docker daemon access:
- Local Docker: User must be in `docker` group
- Remote Docker: Proper `DOCKER_HOST` configuration
- Docker Desktop: Runs with full permissions

## Future Enhancements

Planned interactive features:
- [ ] View container logs in real-time
- [ ] Execute commands in containers
- [ ] Remove stopped containers
- [ ] Inspect container details
- [ ] Attach to container terminal
- [ ] Monitor network traffic
- [ ] Manage volumes and networks
- [ ] Docker Compose service management

## Keyboard Reference

Quick reference for all interactive actions:

```
Navigation:        Action Keys:
  ↑/k - Up           s - Start/Stop
  ↓/j - Down         r - Restart
  1-4 - Tabs         o - Open browser
                     ↵ - Refresh
Controls:
  F1  - Help
  q   - Quit
```
