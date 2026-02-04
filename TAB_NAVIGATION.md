# Enhanced Tab Navigation

The Docker TUI now features improved tab navigation with a modern visual design and multiple ways to switch between tabs.

## New Tab Design

### Visual Style

The tabs now use a raised, connected design that clearly shows the active tab:

```
 ╭───────────────╮╭──────────────╮╭──────────────╮╭──────────────╮
 │ Containers ^D ││ Images    ^I ││ Volumes   ^V ││ Networks  ^N │
─╯               ╰┴──────────────┴┴──────────────┴┴──────────────┴─
```

**Active Tab (Containers):**
- Appears "raised" above the content
- Connected to the content area below (no line underneath)
- Highlighted in yellow

**Inactive Tabs:**
- Separated from the content area by a line
- Shown in green
- Visually "behind" the active tab

### Keyboard Shortcuts

Each tab displays its Ctrl shortcut:
- `^D` - Containers (Ctrl+D)
- `^I` - Images (Ctrl+I)
- `^V` - Volumes (Ctrl+V)
- `^N` - Networks (Ctrl+N)

## Navigation Methods

### Method 1: Arrow Keys (NEW!)

**Left/Right arrows** cycle through tabs:

```
←  (Left Arrow)  - Previous tab
→  (Right Arrow) - Next tab
```

This provides natural, sequential navigation:
- Containers → Images → Volumes → Networks
- Networks → Volumes → Images → Containers

### Method 2: Vim-Style (NEW!)

**h/l keys** work like arrow keys:

```
h - Previous tab (left)
l - Next tab (right)
```

Perfect for Vim users who prefer home row navigation.

### Method 3: Number Keys

**Direct tab access** with numbers:

```
1 - Containers
2 - Images
3 - Volumes
4 - Networks
```

Quick jump to any tab instantly.

### Method 4: Ctrl Shortcuts (NEW!)

**Keyboard shortcuts** shown in tabs:

```
Ctrl+D - Containers
Ctrl+I - Images
Ctrl+V - Volumes
Ctrl+N - Networks
```

Memorable shortcuts based on tab names.

## Usage Examples

### Example 1: Sequential Navigation

Starting from Containers tab:

```
Press →  → Now on Images
Press →  → Now on Volumes
Press →  → Now on Networks
Press →  → Still on Networks (at end)
Press ←  → Back to Volumes
```

### Example 2: Quick Jump

From any tab:

```
Press 1   → Jump to Containers
Press 3   → Jump to Volumes
Press ^I  → Jump to Images
```

### Example 3: Vim-Style Flow

Using h/l navigation:

```
h h h  → Move left through tabs
l l l  → Move right through tabs
```

## Tab Behavior

### When Switching Tabs

All navigation methods trigger the same behavior:
1. **Reset selection** to first item
2. **Reset scroll** to top of list
3. **Clear status message**
4. **Maintain data** (no refresh needed)

### Visual Feedback

The active tab is always clearly indicated:
- **Yellow text** on the tab label
- **Connected appearance** to content below
- **Raised visual style**

## Comparison: Old vs New

### Old Design
```
┌────────────────────────────────────────────┐
│ [1]Containers  [2]Images  [3]Volumes  [4]Networks
├────────────────────────────────────────────┤
```

**Issues:**
- Flat appearance
- Less visual distinction between active/inactive
- Only number key navigation

### New Design
```
 ╭───────────────╮╭──────────────╮╭──────────────╮╭──────────────╮
 │ Containers ^D ││ Images    ^I ││ Volumes   ^V ││ Networks  ^N │
─╯               ╰┴──────────────┴┴──────────────┴┴──────────────┴─
```

**Improvements:**
✅ Clear active tab indication
✅ Modern, raised appearance
✅ Keyboard shortcuts visible
✅ Multiple navigation methods
✅ Arrow key support
✅ Vim-style h/l support

## Navigation Summary

| Method | Keys | Use Case |
|--------|------|----------|
| **Sequential** | `←` `→` or `h` `l` | Browse tabs in order |
| **Direct Jump** | `1` `2` `3` `4` | Know exactly where to go |
| **Ctrl Shortcuts** | `^D` `^I` `^V` `^N` | Quick access by name |

## Tips

### Efficient Workflows

**Browse All Resources:**
1. Start on Containers (`1` or `^D`)
2. Press `→` three times to see Images, Volumes, Networks
3. Press `←` to go back

**Quick Checks:**
- `^D` - Check container status
- `^I` - Check disk usage (images)
- `^V` - Verify volumes exist
- `^N` - Review networks

**Rapid Switching:**
- Use `←`/`→` for adjacent tabs
- Use numbers for distant tabs
- Use Ctrl for muscle memory access

### Keyboard Combinations

**Navigation + Selection:**
```
→ → ↓ ↓ ↓  - Switch tabs and move through items
1 k k k    - Jump to tab and scroll up
^V j j     - Quick volume access and scroll down
```

**With Actions (Containers tab):**
```
^D j j s   - Go to Containers, select item, stop/start
1 ↓ ↓ o    - Jump to Containers, select item, open browser
```

## Accessibility

### Multiple Input Methods

The variety of navigation options ensures:
- **Arrow keys** - Intuitive for most users
- **Vim keys** - Efficient for power users
- **Numbers** - Fast for direct access
- **Ctrl shortcuts** - Memorable by name

### Visual Clarity

The new tab design provides:
- Clear active state indication
- High contrast (yellow on black)
- Distinct inactive state
- Visible keyboard shortcuts

## Future Enhancements

Potential additions:
- [ ] Tab+Shift to cycle tabs
- [ ] First letter navigation (c/i/v/n)
- [ ] Mouse support for clicking tabs
- [ ] Swipe gestures on touchpads
- [ ] Configurable keyboard shortcuts

---

**Now you can navigate between tabs using your preferred method!** 🎯
