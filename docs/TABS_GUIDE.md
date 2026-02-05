# Tab Navigation Guide

The Docker TUI provides four fully functional tabs for comprehensive Docker management.

## Tab Overview

```
[1]Containers  [2]Images  [3]Volumes  [4]Networks
```

Switch between tabs using number keys `1-4` or by pressing the corresponding number.

---

## 1️⃣ Containers Tab

**Keyboard:** Press `1`

### Display Columns

| Column | Description |
|--------|-------------|
| NAME | Container name (without leading /) |
| STATUS | RUNNING (green), STOPPED (red), PAUSED (yellow) |
| CPU% | CPU usage percentage (live, running containers only) |
| MEM | Memory usage in MB/GB (live, running containers only) |
| IMAGE | Docker image used to create container |
| PORTS | Exposed ports (comma-separated) |

### Interactive Actions

- `s` - Start/Stop selected container
- `r` - Restart selected container
- `o` - Open container's port in browser
- `↑`/`↓` - Navigate containers
- `Enter` - Refresh list

### Visual Indicators

- **Yellow text** - Selected container
- **Green "RUNNING"** - Container is active
- **Red "STOPPED"** - Container is inactive
- **Gray text** - Stopped container details
- **">" indicator** - Current selection

### Example

```
┌─────────────────────────────────────────────────────────────┐
│ NAME                 │ STATUS  │ CPU% │ MEM  │ IMAGE  │ PORTS │
├──────────────────────┼─────────┼──────┼──────┼────────┼───────┤
│> nginx-proxy         │ RUNNING │ 2.4  │ 128M │ nginx  │ 80,443│
│  postgres-db         │ RUNNING │ 0.8  │ 512M │ postgres│ 5432 │
│  api-server          │ STOPPED │  --  │  --  │ node:18│ 3000  │
└──────────────────────────────────────────────────────────────┘
[S]top | [R]estart | [O]pen in browser
```

---

## 2️⃣ Images Tab

**Keyboard:** Press `2`

### Display Columns

| Column | Description |
|--------|-------------|
| IMAGE ID | Short image ID (12 characters) |
| REPOSITORY | Repository name (truncated if long) |
| TAG | Image tag (e.g., "latest", "v1.2.3") |
| SIZE | Image size in human-readable format (MB/GB) |
| CREATED | How long ago the image was created |

### Features

- Lists all images (including dangling images shown as `<none>`)
- Shows compressed size on disk
- Relative time display (e.g., "2d ago", "3w ago", "5mo ago")
- Repository names are shortened for display

### Navigation

- `↑`/`↓` - Navigate images
- `Enter` - Refresh list
- Number keys - Switch tabs

### Example

```
┌────────────────────────────────────────────────────────────────┐
│ IMAGE ID   │ REPOSITORY         │ TAG      │ SIZE   │ CREATED  │
├────────────┼────────────────────┼──────────┼────────┼──────────┤
│> a1b2c3d4e5 │ nginx              │ alpine   │ 24.1MB │ 2w ago   │
│  f6g7h8i9j0 │ postgres           │ 14       │ 376MB  │ 1mo ago  │
│  k1l2m3n4o5 │ node               │ 18       │ 995MB  │ 3d ago   │
│  p6q7r8s9t0 │ <none>             │ <none>   │ 1.2GB  │ 5h ago   │
└────────────────────────────────────────────────────────────────┘
```

---

## 3️⃣ Volumes Tab

**Keyboard:** Press `3`

### Display Columns

| Column | Description |
|--------|-------------|
| NAME | Volume name (truncated if long) |
| DRIVER | Volume driver (usually "local") |
| MOUNTPOINT | Path where volume is mounted on host |
| CREATED | When the volume was created |

### Features

- Shows all Docker volumes
- Displays full mountpoint paths (truncated for display)
- Helps identify unused volumes
- Shows driver information for special volume types

### Navigation

- `↑`/`↓` - Navigate volumes
- `Enter` - Refresh list
- Number keys - Switch tabs

### Use Cases

- Identify persistent data volumes
- Find volumes to backup
- Locate unused volumes for cleanup
- Verify volume creation

### Example

```
┌───────────────────────────────────────────────────────────────────┐
│ NAME                  │ DRIVER │ MOUNTPOINT              │ CREATED │
├───────────────────────┼────────┼─────────────────────────┼─────────┤
│> postgres_data        │ local  │ /var/lib/docker/volume..│ 1w ago  │
│  nginx_config         │ local  │ /var/lib/docker/volume..│ 3d ago  │
│  app_uploads          │ local  │ /var/lib/docker/volume..│ 5h ago  │
└───────────────────────────────────────────────────────────────────┘
```

---

## 4️⃣ Networks Tab

**Keyboard:** Press `4`

### Display Columns

| Column | Description |
|--------|-------------|
| NETWORK ID | Short network ID (12 characters) |
| NAME | Network name |
| DRIVER | Network driver (bridge, host, overlay, macvlan) |
| SCOPE | Network scope (local, global, swarm) |
| IPv4 | IPv4 subnet (CIDR notation) |
| IPv6 | IPv6 subnet if configured |

### Features

- Lists all Docker networks
- Shows default networks (bridge, host, none)
- Displays custom networks
- Shows subnet information for troubleshooting
- Indicates network scope for swarm/multi-host

### Network Drivers

- **bridge** - Default isolated network for containers
- **host** - No network isolation, use host's network
- **overlay** - Multi-host networking for swarm
- **macvlan** - Assign MAC addresses to containers
- **none** - Disable networking

### Navigation

- `↑`/`↓` - Navigate networks
- `Enter` - Refresh list
- Number keys - Switch tabs

### Example

```
┌──────────────────────────────────────────────────────────────────────┐
│ NETWORK ID │ NAME    │ DRIVER  │ SCOPE │ IPv4           │ IPv6      │
├────────────┼─────────┼─────────┼───────┼────────────────┼───────────┤
│> a1b2c3d4e5 │ bridge  │ bridge  │ local │ 172.17.0.0/16  │ --        │
│  f6g7h8i9j0 │ host    │ host    │ local │ --             │ --        │
│  k1l2m3n4o5 │ none    │ null    │ local │ --             │ --        │
│  p6q7r8s9t0 │ my-net  │ bridge  │ local │ 192.168.1.0/24 │ --        │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Tab Switching

### Quick Tab Access

- `1` - Jump to Containers
- `2` - Jump to Images
- `3` - Jump to Volumes
- `4` - Jump to Networks

### Tab Behavior

- **Auto-reset selection**: Switching tabs resets to first item
- **Status message cleared**: Previous status messages are cleared
- **Independent scrolling**: Each tab maintains its own view
- **Live updates**: All tabs refresh every 5 seconds

---

## Common Operations

### View All Docker Resources

1. Press `1` - View containers and their resource usage
2. Press `2` - Check which images are consuming disk space
3. Press `3` - Verify persistent volumes exist
4. Press `4` - Review network configuration

### Troubleshooting Workflow

1. **Container not starting?**
   - Press `1` to see container status
   - Check logs externally if needed
   - Try restart with `r`

2. **Out of disk space?**
   - Press `2` to see image sizes
   - Identify large or unused images
   - Clean up old images

3. **Volume data missing?**
   - Press `3` to verify volume exists
   - Check mountpoint path
   - Verify driver is correct

4. **Network connectivity issues?**
   - Press `4` to see network configuration
   - Verify container is on correct network
   - Check IP subnet ranges

---

## Tips & Tricks

### Efficient Navigation

- Use `j`/`k` (Vim keys) for quick up/down navigation
- Press number keys to jump between tabs instantly
- Use `Enter` to force refresh if data seems stale

### Workflow Optimization

1. **Start with Containers** - See what's running
2. **Check Images** - Verify image availability
3. **Inspect Volumes** - Ensure data persistence
4. **Review Networks** - Validate connectivity setup

### Reading the Data

- **Relative times** easier to parse than absolute dates
- **Color coding** helps identify issues at a glance
- **Truncated names** keep table readable (hover/select for full info)
- **"--" indicators** show unavailable data (e.g., stopped containers)

---

## Keyboard Reference

```
Tab Switching:      Navigation:        Actions (Containers):
  1 - Containers      ↑/k - Up           s - Start/Stop
  2 - Images          ↓/j - Down         r - Restart
  3 - Volumes         ↵   - Refresh      o - Open browser
  4 - Networks

Global:
  F1  - Help
  q   - Quit
```

---

## Auto-Refresh

All tabs automatically refresh every **5 seconds** to show:
- New containers/images/volumes/networks
- Updated container stats (CPU/Memory)
- Status changes (started/stopped containers)
- Removed resources

Manual refresh available with `Enter` key on any tab.
