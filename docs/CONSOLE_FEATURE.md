# Console Feature with Altscreen Support

tinyd provides an interactive console feature that uses **altscreen** technology to seamlessly open container shells without disrupting your TUI session.

## What is Altscreen?

Altscreen is a terminal feature that switches to an alternate screen buffer, allowing you to:
- Run interactive programs (like shells) that take over the full terminal
- Return to your previous terminal state exactly as you left it
- No scrollback pollution or screen clutter

## Altscreen Console Toolbar

When you press `c` and enter the altscreen console, a toolbar is displayed at the top showing key information:

```
╔════════════════════════════════════════════════════════════════════════════════════╗
║ Console: nginx-proxy                Mode: docker exec                             ║
║ Container ID: a3f9d8c2b1e4           Exit: type 'exit' or Ctrl+D                  ║
╚════════════════════════════════════════════════════════════════════════════════════╝

[nginx-proxy] /app $ ls
[nginx-proxy] /app $ ps aux
[nginx-proxy] /app $ exit
```

### Toolbar Information:
- **Container Name** - Prominently displayed (yellow highlight)
- **Console Mode** - Shows "docker exec" or "docker debug" (cyan)
- **Container ID** - Short container ID for reference
- **Exit Instructions** - Reminder on how to exit (red for visibility)

### Custom Prompt:
The shell prompt is customized to show:
- Container name in brackets (cyan): `[nginx-proxy]`
- Current working directory (green)
- Standard `$` prompt

This keeps you aware of which container you're in at all times.

## Console Modes

### 1. Docker Exec (Default)

Uses the standard `docker exec -it` command to open a shell in the running container.

**Features:**
- Automatic shell detection (tries `/bin/bash`, `/bin/sh`, `/bin/ash`)
- Standard container shell access
- Works with all running containers

**Usage:**
1. Select a running container
2. Press `c` to open the console
3. Terminal switches to altscreen
4. Interactive shell opens
5. Exit the shell (type `exit` or press Ctrl+D)
6. Returns to TUI exactly as before

### 2. Docker Debug

Uses `docker debug` for advanced debugging capabilities.

**Features:**
- Extended debugging tools and utilities
- Access to container filesystem and processes
- Enhanced troubleshooting capabilities
- Requires Docker Desktop or Docker CLI with debug support

**Usage:**
1. Press `d` to toggle debug mode
2. Action bar shows: `[C]onsole (debug)`
3. Select a running container
4. Press `c` to open debug console
5. Exit when done to return to TUI

## Keyboard Shortcuts

| Key | Action | Description |
|-----|--------|-------------|
| `c` | Open Console | Opens interactive shell using altscreen |
| `d` | Toggle Debug Mode | Switches between `exec` and `debug` modes |

## Action Bar Indicators

The action bar shows the current console mode:

**Docker Exec Mode (default):**
```
[S]top | [R]estart | [C]onsole (exec) | [D]ebug toggle
```

**Docker Debug Mode:**
```
[S]top | [R]estart | [C]onsole (debug) | [D]ebug toggle
```

## Example Workflow

### Standard Shell Access

```bash
# In TUI
1. Navigate to Containers tab
2. Select "nginx-proxy" container (RUNNING)
3. Press 'c'

# Terminal switches to altscreen
# Toolbar appears at top:
╔════════════════════════════════════════════════════════╗
║ Console: nginx-proxy       Mode: docker exec          ║
║ Container ID: a3f9d8c2b1e4 Exit: type 'exit' or Ctrl+D║
╚════════════════════════════════════════════════════════╝

[nginx-proxy] /app $ ls
[nginx-proxy] /app $ ps aux
[nginx-proxy] /app $ exit

# Back in TUI, exactly where you left off
```

### Debug Session

```bash
# In TUI
1. Navigate to Containers tab
2. Press 'd' to toggle debug mode
3. Status message: "Console mode: docker debug (enabled)"
4. Select "api-server" container
5. Press 'c'

# Terminal switches to altscreen
# Toolbar shows debug mode:
╔════════════════════════════════════════════════════════╗
║ Console: api-server        Mode: docker debug         ║
║ Container ID: b5e3a1f9c8d7 Exit: type 'exit' or Ctrl+D║
╚════════════════════════════════════════════════════════╝

[api-server] /app $ # Debug session with enhanced tools
[api-server] /app $ exit

# Back in TUI, exactly where you left off
```

## Technical Implementation

### Bubble Tea Integration

The console feature uses `tea.ExecProcess` for proper altscreen handling:

```go
func openConsole(containerID, containerName string, useDebug bool) tea.Cmd {
    var cmd *exec.Cmd

    if useDebug {
        cmd = exec.Command("docker", "debug", containerID)
    } else {
        // Auto-detect shell
        cmd = exec.Command("docker", "exec", "-it", containerID, "/bin/bash")
    }

    // Use ExecProcess for altscreen support
    return tea.ExecProcess(cmd, func(err error) tea.Msg {
        // Handle return to TUI
    })
}
```

### State Preservation

When you open the console:
1. TUI saves current state (selection, scroll position, data)
2. Terminal switches to altscreen buffer
3. Interactive shell runs
4. On exit, terminal restores original screen
5. TUI continues exactly where you left off

## Requirements

### Docker Exec Mode
- Docker daemon running
- Container must be in RUNNING state
- Standard Docker CLI

### Docker Debug Mode
- Docker Desktop 4.27+ or Docker CLI with debug plugin
- Container must be in RUNNING state
- May require additional permissions

## Error Handling

### Common Errors

**"Container must be running"**
- You tried to open console on a stopped container
- Start the container first with `s` key

**"Console error: command not found"**
- Docker CLI is not installed or not in PATH
- Verify Docker installation

**"Failed to open console"**
- No valid shell found in container
- Container may be using a minimal base image
- Try debug mode for enhanced shell access

## Comparison with Other Approaches

### Without Altscreen (Old Approach)
```
Problems:
- Clears the TUI when launching shell
- Pollutes terminal scrollback
- Difficult to return to previous state
- User experience is jarring
```

### With Altscreen (Current Implementation)
```
Benefits:
✅ Seamless transition to shell
✅ Clean return to TUI
✅ No scrollback pollution
✅ Preserves all TUI state
✅ Professional user experience
```

## Tips & Best Practices

### 1. Use Exec for Quick Tasks
- Checking logs: `tail -f /var/log/app.log`
- Inspecting configs: `cat /etc/nginx/nginx.conf`
- Running commands: `curl localhost:8080/health`

### 2. Use Debug for Troubleshooting
- Container not responding
- Need advanced debugging tools
- Investigating complex issues
- Container filesystem exploration

### 3. Toggle Between Modes
- Press `d` to switch modes
- Try exec first (faster)
- Switch to debug if needed

### 4. Multiple Sessions
- Each console session is independent
- You can exit and re-enter
- State is always preserved
- No limit on sessions

## Future Enhancements

Potential improvements:
- [ ] History of console sessions
- [ ] Custom shell preferences
- [ ] Session logging
- [ ] Multiple concurrent consoles
- [ ] Console command history

---

**The altscreen console feature provides seamless, professional shell access within the TUI!** 🖥️
