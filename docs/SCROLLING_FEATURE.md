# Scrolling Feature

tinyd now includes intelligent viewport scrolling to handle large lists without losing the UI.

## Problem Solved

Previously, with many containers/images/volumes/networks, the list would overflow and you'd lose:
- Header information
- Tab navigation
- Action buttons
- Status messages

## Solution: Viewport Scrolling

The application now displays **10 items at a time** in a scrollable viewport.

## How It Works

### Automatic Scrolling

As you navigate with `↑`/`↓` or `j`/`k`:
- **Scroll Down**: When you move past the last visible item, the viewport scrolls down automatically
- **Scroll Up**: When you move before the first visible item, the viewport scrolls up automatically
- **Smooth Experience**: Navigation feels natural and continuous

### Visual Feedback

#### Scroll Indicator

The status line shows your position when scrolling is active:

```
CONTAINERS (25 total, 15 running) [1-10 of 25]
```

This means:
- **Total items**: 25 containers
- **Currently showing**: Items 1 through 10
- **More below**: Navigate down to see items 11-25

As you scroll:
```
[1-10 of 25]  → First page
[11-20 of 25] → Second page
[21-25 of 25] → Last page
```

#### No Indicator When Not Needed

If there are 10 or fewer items, no scroll indicator appears - everything fits on screen.

## Viewport Size

**Default**: 10 items per view

This provides:
- Enough context to compare items
- Room for header, tabs, and action bar
- Fast navigation through lists

## Example Usage

### Scenario: 50 Containers

1. **Start**: Shows containers 1-10, indicator: `[1-10 of 50]`
2. **Press `↓` 9 times**: Still on first page, now at container #10
3. **Press `↓` once more**: Auto-scrolls! Now shows containers 2-11, selection on #11
4. **Continue down**: Viewport smoothly scrolls through all 50
5. **Press `↑` to go back**: Viewport scrolls up automatically

### Scenario: 7 Images

- Shows all 7 images at once
- No scroll indicator needed
- Full list visible

## Navigation Tips

### Quick Scrolling

1. **Hold down `j`** - Rapidly scroll down through long lists
2. **Hold down `k`** - Rapidly scroll up
3. **Switch tabs** - Resets scroll position to top

### Finding Items

For very long lists (e.g., 100+ images):
1. Navigate with `j`/`k` while watching the scroll indicator
2. Status line shows your position: `[41-50 of 127]`
3. Each tab remembers its own scroll position independently

## All Tabs Support Scrolling

### Containers Tab
```
CONTAINERS (75 total, 42 running) [1-10 of 75]
```

### Images Tab
```
IMAGES (120 total) [31-40 of 120]
```

### Volumes Tab
```
VOLUMES (45 total) [11-20 of 45]
```

### Networks Tab
```
NETWORKS (25 total) [1-10 of 25]
```

## Technical Details

### Viewport Implementation

- **Scroll Offset**: Tracks the first visible item
- **Viewport Height**: Always shows 10 items
- **Auto-adjustment**: Scrolls when selection moves out of view
- **Bounds checking**: Prevents scrolling past the end

### Memory Efficient

- Only renders visible items (10 at a time)
- Full list kept in memory for fast access
- Scrolling is instant with no lag

### Independent Per Tab

Each tab maintains its own:
- Scroll position
- Selection
- Viewport

Switching tabs resets the view to the top.

## Benefits

### 1. UI Always Visible ✓
- Header always shown
- Tab bar accessible
- Action buttons available
- Status messages visible

### 2. Performance ✓
- Renders only 10 items regardless of total
- Fast rendering even with 1000+ items
- No slowdown with large lists

### 3. Better Navigation ✓
- Clear position indicator
- Smooth auto-scrolling
- Natural keyboard feel
- Easy to find items in long lists

### 4. Maintains Design ✓
- Original design preserved
- Same visual aesthetics
- Classic terminal look
- Clean and minimal

## Keyboard Reference

```
Navigation (with scrolling):
  ↑/k - Move up (auto-scroll when needed)
  ↓/j - Move down (auto-scroll when needed)
  1-4 - Switch tabs (resets scroll to top)
```

## Example Views

### View 1: Beginning of List
```
┌──────────────────────────────────────────────┐
│ CONTAINERS (25 total, 15 running) [1-10 of 25]│
├──────────────────────────────────────────────┤
│> container-1                                 │
│  container-2                                 │
│  container-3                                 │
│  ...                                         │
│  container-10                                │
└──────────────────────────────────────────────┘
```

### View 2: Middle of List
```
┌──────────────────────────────────────────────┐
│ CONTAINERS (25 total, 15 running) [11-20 of 25]│
├──────────────────────────────────────────────┤
│  container-11                                │
│  container-12                                │
│> container-13                                │
│  ...                                         │
│  container-20                                │
└──────────────────────────────────────────────┘
```

### View 3: End of List
```
┌──────────────────────────────────────────────┐
│ CONTAINERS (25 total, 15 running) [21-25 of 25]│
├──────────────────────────────────────────────┤
│  container-21                                │
│  container-22                                │
│  container-23                                │
│  container-24                                │
│> container-25                                │
└──────────────────────────────────────────────┘
```

## Future Enhancements

Potential improvements:
- [ ] Page Up/Page Down support (jump 10 items)
- [ ] Home/End keys (jump to start/end)
- [ ] Configurable viewport size
- [ ] Search/filter to reduce list size
- [ ] Jump to item by number

---

**Now you can manage hundreds of Docker resources without losing the UI!** 🎉
