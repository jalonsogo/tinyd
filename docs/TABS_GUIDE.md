# Tab Navigation & Resource Guide

tinyd provides four fully functional tabs for comprehensive Docker management.

## Tab Overview

```
┌───────────────╮╭──────────────╮╭──────────────╮╭──────────────╮
│ Containers ^D ││ Images    ^I ││ Volumes   ^V ││ Networks  ^N │
╰───────────────┴┴──────────────┴┴──────────────┴┴──────────────┴
```

Switch between tabs using:
- Number keys `1-4`
- Arrow keys `←` `→` or `h` `l`
- Ctrl shortcuts `^D` `^I` `^V` `^N`

---

## 1️⃣ Containers Tab

**Quick Access:** Press `1` or `Ctrl+D`

### Display Columns

| Column | Description |
|--------|-------------|
| ● | Status indicator (green=running, gray=stopped) |
| NAME | Container name (leading "/" removed) |
| STATUS | RUNNING / STOPPED / PAUSED |
| CPU% | Live CPU usage percentage (running only) |
| MEM | Current memory usage in MB/GB (running only) |
| IMAGE | Docker image used for container |
| PORTS | Exposed port mappings (comma-separated) |

### Available Actions

| Key | Action | Description |
|-----|--------|-------------|
| `s/S` | Start/Stop | Smart toggle based on current state |
| `r/R` | Restart | Restart running container |
| `c/C` | Console | Open interactive shell (altscreen) |
| `o/O` | Open Browser | Open exposed ports in browser |
| `l/L` | View Logs | Full-screen logs with search |
| `i/I` | Inspect | View container stats and details |
| `d/D` | Delete | Delete with inline confirmation |
| `f/F` | Filter | Open filter modal |
| `/` | Search | Quick inline search/filter |

### Filter Options

- **All** - Show all containers (default)
- **Running** - Show only running containers

### Visual Indicators

- **● Green** - Container is running
- **○ Gray** - Container is stopped
- **◐ Yellow** - Container is paused
- **> Indicator** - Currently selected container
- **Yellow text** - Selected row

### Status Line Example

```
CONTAINERS (25 total, 15 running) [Filter: Running]
```

### Example View

```
┌──────────────────────────────────────────────────────────────────┐
│ CONTAINERS (3 total, 2 running)                                   │
├──┬──────────────────┬─────────┬──────┬──────┬──────────┬────────┤
│● │ nginx-proxy      │ RUNNING │ 2.4% │ 128M │ nginx    │ 80,443 │
│● │ postgres-db      │ RUNNING │ 0.8% │ 512M │ postgres │ 5432   │
│○ │ api-server       │ STOPPED │  --  │  --  │ node:18  │ 3000   │
└──┴──────────────────┴─────────┴──────┴──────┴──────────┴────────┘
[S]tart/Stop | [R]estart | [C]onsole | [O]pen | [L]ogs | [I]nspect
```

### Common Workflows

**Quick Health Check:**
1. View CPU/Memory usage at a glance
2. Green dots show active containers
3. Gray dots show stopped containers

**Open Web Application:**
1. Select container with exposed ports
2. Press `o` to open in browser
3. If multiple ports, select from modal

**Debug Container:**
1. Press `l` to view logs
2. Press `s` to toggle search in logs
3. Press `c` to open interactive shell
4. Press `i` to inspect full stats

---

## 2️⃣ Images Tab

**Quick Access:** Press `2` or `Ctrl+I`

### Display Columns

| Column | Description |
|--------|-------------|
| REPOSITORY | Repository name (e.g., nginx, postgres) |
| TAG | Image tag (e.g., latest, 15, alpine) |
| SIZE | Image size on disk (MB/GB) |
| CREATED | Relative creation time (e.g., "2d ago") |
| STATUS | In-use indicator (● if used by containers) |

### Available Actions

| Key | Action | Description |
|-----|--------|-------------|
| `r/R` | Run | Run container from image (opens modal) |
| `p/P` | Pull | Pull new image from registry |
| `i/I` | Inspect | View layers, architecture, config |
| `d/D` | Delete | Delete image with confirmation |
| `f/F` | Filter | Open filter modal |
| `/` | Search | Quick inline search/filter |

### Filter Options

- **All** - Show all images (default)
- **In Use** - Images currently used by containers
- **Unused** - Images not used by any container
- **Dangling** - Broken/orphaned images (`<none>:<none>`)

### Visual Indicators

- **● Green** - Image in use by at least one container
- **⚠ Yellow** - Dangling image (orphaned)
- **Gray text** - Unused image

### Run Container Modal

Press `r/R` to open the run container configuration:

**Form Sections:**
1. **Container Name** (optional)
2. **Port Mappings** (host:container pairs)
3. **Volume Mounts** (host:container paths)
4. **Environment Variables** (key=value pairs)

**Navigation:**
- Tab/Shift+Tab to move between fields
- Enter to add port/volume/env var pair
- Leave fields empty to skip to next section
- Enter on final field submits form

### Pull Image Modal

Press `p/P` to pull a new image:

**Input:**
- Image name in `repository:tag` format
- Examples: `nginx:latest`, `postgres:15`, `node:18-alpine`
- Press Enter to pull

### Status Line Example

```
IMAGES (15 total) [Filter: Dangling]
```

### Example View

```
┌───────────────────────────────────────────────────────────────┐
│ IMAGES (4 total)                                               │
├────────────────┬──────────┬────────┬──────────┬──────────────┤
│ REPOSITORY     │ TAG      │ SIZE   │ CREATED  │ STATUS       │
├────────────────┼──────────┼────────┼──────────┼──────────────┤
│ nginx          │ alpine   │ 24.1MB │ 2w ago   │ ● In use     │
│ postgres       │ 15       │ 376MB  │ 1mo ago  │ ● In use     │
│ node           │ 18       │ 995MB  │ 3d ago   │              │
│ <none>         │ <none>   │ 1.2GB  │ 5h ago   │ ⚠ Dangling   │
└────────────────┴──────────┴────────┴──────────┴──────────────┘
[R]un | [P]ull | [I]nspect | [D]elete | [F]ilter
```

### Common Workflows

**Run New Container:**
1. Navigate to desired image
2. Press `r` to open run modal
3. Configure name, ports, volumes, env vars
4. Press Enter to create and start

**Clean Up Disk Space:**
1. Press `f` to filter
2. Select "Dangling" or "Unused"
3. Press `d` on each unwanted image
4. Navigate left/right to Yes, press Enter

**Pull Latest Version:**
1. Press `p` to open pull modal
2. Type image name (e.g., `nginx:latest`)
3. Press Enter to pull
4. Check status bar for progress

---

## 3️⃣ Volumes Tab

**Quick Access:** Press `3` or `Ctrl+V`

### Display Columns

| Column | Description |
|--------|-------------|
| NAME | Volume name |
| DRIVER | Volume driver (usually "local") |
| MOUNTPOINT | Host path where volume is stored |
| CREATED | Relative creation time |
| CONTAINERS | Names of containers using this volume |
| STATUS | In-use indicator (● if mounted) |

### Available Actions

| Key | Action | Description |
|-----|--------|-------------|
| `i/I` | Inspect | View volume details and usage |
| `d/D` | Delete | Delete volume with confirmation |
| `f/F` | Filter | Open filter modal |
| `/` | Search | Quick inline search/filter |

### Filter Options

- **All** - Show all volumes (default)
- **In Use** - Volumes currently mounted
- **Unused** - Volumes not mounted by any container

### Visual Indicators

- **● Green** - Volume in use (mounted)
- **Gray text** - Unused volume

### Status Line Example

```
VOLUMES (12 total, 8 in use) [Filter: Unused]
```

### Example View

```
┌──────────────────────────────────────────────────────────────────────┐
│ VOLUMES (3 total, 2 in use)                                           │
├──────────────────┬────────┬──────────────────────┬─────────┬─────────┤
│ NAME             │ DRIVER │ MOUNTPOINT           │ CREATED │ STATUS  │
├──────────────────┼────────┼──────────────────────┼─────────┼─────────┤
│ postgres_data    │ local  │ /var/lib/docker/...  │ 1w ago  │ ● In use│
│ nginx_config     │ local  │ /var/lib/docker/...  │ 3d ago  │ ● In use│
│ old_app_data     │ local  │ /var/lib/docker/...  │ 2mo ago │         │
└──────────────────┴────────┴──────────────────────┴─────────┴─────────┘
[I]nspect | [D]elete | [F]ilter
```

### Volume Details (Inspect)

Press `i` to see:
- Full mountpoint path
- Driver configuration
- Scope (local/global)
- List of containers using the volume
- Mount options

### Common Workflows

**Find Unused Volumes:**
1. Press `f` to filter
2. Select "Unused"
3. Review list for cleanup candidates

**Verify Data Persistence:**
1. Navigate to volume
2. Press `i` to inspect
3. Check which containers use it
4. Verify mountpoint path

**Safe Volume Deletion:**
1. Filter for unused volumes
2. Press `d` on volume to delete
3. Navigate right to "Yes"
4. Press Enter to confirm

---

## 4️⃣ Networks Tab

**Quick Access:** Press `4` or `Ctrl+N`

### Display Columns

| Column | Description |
|--------|-------------|
| NAME | Network name |
| DRIVER | Network driver (bridge, host, overlay, etc.) |
| SCOPE | Network scope (local, global, swarm) |
| IPv4 | IPv4 subnet in CIDR notation |
| IPv6 | IPv6 subnet (if configured) |
| STATUS | In-use indicator (● if containers connected) |

### Available Actions

| Key | Action | Description |
|-----|--------|-------------|
| `i/I` | Inspect | View network details and connections |
| `d/D` | Delete | Delete network with confirmation |
| `f/F` | Filter | Open filter modal |
| `/` | Search | Quick inline search/filter |

### Filter Options

- **All** - Show all networks (default)
- **In Use** - Networks with connected containers
- **Unused** - Networks without any containers

### Network Drivers

| Driver | Description |
|--------|-------------|
| **bridge** | Default isolated network for containers |
| **host** | No network isolation, uses host's network stack |
| **overlay** | Multi-host networking for Docker Swarm |
| **macvlan** | Assigns MAC addresses to containers |
| **none** | Disables networking completely |

### Visual Indicators

- **● Green** - Network in use (has connected containers)
- **Gray text** - Unused network

### Status Line Example

```
NETWORKS (8 total, 4 in use) [Filter: In Use]
```

### Example View

```
┌────────────────────────────────────────────────────────────────────┐
│ NETWORKS (4 total, 2 in use)                                       │
├──────────────┬─────────┬───────┬────────────────┬────────┬────────┤
│ NAME         │ DRIVER  │ SCOPE │ IPv4           │ IPv6   │ STATUS │
├──────────────┼─────────┼───────┼────────────────┼────────┼────────┤
│ bridge       │ bridge  │ local │ 172.17.0.0/16  │ --     │ ● In use│
│ host         │ host    │ local │ --             │ --     │        │
│ none         │ null    │ local │ --             │ --     │        │
│ my-app-net   │ bridge  │ local │ 192.168.1.0/24 │ --     │ ● In use│
└──────────────┴─────────┴───────┴────────────────┴────────┴────────┘
[I]nspect | [D]elete | [F]ilter
```

### Network Details (Inspect)

Press `i` to see:
- Full network configuration
- Connected containers
- Gateway addresses
- Subnet configuration
- Driver options

### Common Workflows

**Check Container Connectivity:**
1. Navigate to network
2. Press `i` to inspect
3. View list of connected containers
4. Verify IP subnet ranges

**Clean Up Networks:**
1. Press `f` to filter "Unused"
2. Review custom networks
3. Press `d` to delete unwanted ones

**Troubleshoot Network Issues:**
1. Find container's network
2. Press `i` to inspect network
3. Check subnet doesn't conflict
4. Verify containers on same network

---

## Tab Switching

### Quick Navigation Methods

**Number Keys (Direct Jump):**
```
1 - Containers tab
2 - Images tab
3 - Volumes tab
4 - Networks tab
```

**Arrow Keys (Sequential):**
```
← or h - Previous tab
→ or l - Next tab
```

**Ctrl Shortcuts (Mnemonic):**
```
Ctrl+D - Containers (Docker)
Ctrl+I - Images
Ctrl+V - Volumes
Ctrl+N - Networks
```

### Tab Behavior

When switching tabs:
- **Selection resets** to first item
- **Scroll position** resets to top
- **Active filter** is preserved per tab
- **Search** is cleared
- **Status messages** are cleared

### Independent Tab States

Each tab maintains:
- Its own filter setting
- Its own scroll position
- Its own selection
- Its own search state

---

## Common Operations Across Tabs

### View All Docker Resources

**Complete System Overview:**
1. Press `1` - Check container status and resource usage
2. Press `2` - Review disk space usage (images)
3. Press `3` - Verify persistent volumes exist
4. Press `4` - Review network configuration

### Troubleshooting Workflow

**Container Issues:**
1. Tab 1 (Containers) - Check status, view logs
2. Tab 2 (Images) - Verify image exists
3. Tab 3 (Volumes) - Check data volumes mounted
4. Tab 4 (Networks) - Verify network connectivity

**Disk Space Issues:**
1. Tab 2 (Images) - Filter "Unused" or "Dangling"
2. Delete large unused images
3. Tab 3 (Volumes) - Filter "Unused"
4. Delete old unused volumes

**Network Issues:**
1. Tab 4 (Networks) - Inspect network config
2. Check IP subnet ranges
3. Verify containers on correct network
4. Tab 1 (Containers) - Restart affected containers

---

## Keyboard Reference

### Global Navigation
```
↑/k         - Move selection up (auto-scroll)
↓/j         - Move selection down (auto-scroll)
←/→ or h/l  - Switch tabs left/right
1-4         - Jump to specific tab
Ctrl+D/I/V/N - Jump to tab by shortcut
Enter       - Refresh current tab
ESC         - Return to list view
F1          - Toggle help screen
q/Ctrl+C    - Quit application
```

### Universal Actions (All Tabs)
```
i/I         - Inspect selected resource
d/D         - Delete with inline confirmation
f/F         - Open filter modal
/           - Toggle inline search
```

### Container-Specific (Tab 1)
```
s/S         - Start/Stop container
r/R         - Restart container
c/C         - Open console/shell
o/O         - Open port in browser
l/L         - View logs
```

### Image-Specific (Tab 2)
```
r/R         - Run container from image
p/P         - Pull new image
```

---

## Tips & Best Practices

### Efficient Tab Navigation

**Use Number Keys for Speed:**
- `1` for quick container check
- `2` for image management
- `3` for volume verification
- `4` for network review

**Use Arrow Keys for Sequential:**
- Browse through tabs with `→`
- Go back with `←`
- Natural left-to-right flow

**Use Ctrl Shortcuts for Muscle Memory:**
- Ctrl+D - Most frequently used (containers)
- Ctrl+I - Second most (images)

### Workflow Optimization

**Daily Health Check:**
```
1. Press 1 - Check running containers
2. Press 2 - Check disk usage
3. Press / - Search for specific resources
```

**Resource Cleanup:**
```
1. Tab 2 - Filter Dangling images, delete
2. Tab 3 - Filter Unused volumes, delete
3. Tab 4 - Filter Unused networks, delete
```

**Container Lifecycle:**
```
1. Tab 2 - Select image, press r to run
2. Tab 1 - Monitor in containers tab
3. Tab 1 - Press o to open in browser
4. Tab 1 - Press l to check logs
5. Tab 1 - Press s to stop when done
```

### Reading the Data

**Relative Times:**
- "2d ago" easier to parse than "2024-01-13"
- "5h ago" shows recent activity
- "3mo ago" indicates old/stale resources

**Color Coding:**
- Green indicators = Active/healthy
- Gray indicators = Inactive
- Yellow warnings = Attention needed
- Red errors = Problems

**Status Indicators:**
- ● (filled) = In use / Running
- ○ (empty) = Not in use / Stopped
- ⚠ = Warning / Dangling

---

## Auto-Refresh

All tabs automatically refresh every **5 seconds** to display:
- New/removed containers, images, volumes, networks
- Updated container stats (CPU/Memory)
- Status changes (started/stopped containers)
- Usage indicators (in-use status)

Manual refresh available anytime with `Enter` key.

---

**tinyd provides complete visibility and control over your Docker environment!** 🐋
