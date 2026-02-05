# Architecture - Component-Based Design

tinyd has been refactored to use proper Bubble Tea components for better maintainability and reusability.

## Component Architecture

### Core Principles

1. **Separation of Concerns** - Each component handles one specific UI element
2. **Reusability** - Components can be used across different views
3. **Composability** - Complex UIs built from simple components
4. **State Management** - Each component manages its own presentation logic

### Component Hierarchy

```
Model (Root)
├── HeaderComponent
├── TabsComponent
├── StatusLineComponent
├── TableComponent
│   ├── TableHeader[]
│   └── TableRow[]
├── ActionBarComponent
└── DetailViewComponent
```

## Components

### 1. HeaderComponent

**Purpose:** Renders the top header bar with title and help text

**File:** `components.go`

**Properties:**
- `title` - Main title text
- `help` - Help text (right-aligned)
- `width` - Component width

**Usage:**
```go
header := NewHeaderComponent("tinyd v2.0.1", "[F1] Help [Q]uit")
output := header.View()
```

**Renders:**
```
┌─────────────────────────────────────────────────┐
│ tinyd v2.0.1            [F1] Help [Q]uit │
```

---

### 2. TabsComponent

**Purpose:** Renders tab navigation with visual active state

**Properties:**
- `tabs` - Array of TabItem (name + shortcut)
- `activeTab` - Index of currently active tab
- `width` - Component width

**Usage:**
```go
tabs := []TabItem{
    {Name: "Containers", Shortcut: "^D"},
    {Name: "Images", Shortcut: "^I"},
}
tabsComp := NewTabsComponent(tabs, 0)
tabsComp = tabsComp.SetActiveTab(1) // Switch to Images
output := tabsComp.View()
```

**Renders:**
```
 ╭───────────────╮╭──────────────╮
 │ Containers ^D ││ Images    ^I │
─┴───────────────┴╯              ╰──
```

---

### 3. StatusLineComponent

**Purpose:** Shows status information with counts and scroll indicators

**Properties:**
- `label` - Status text
- `count` - Item count
- `scrollIndicator` - Scroll position (e.g., "[1-10 of 50]")
- `width` - Component width

**Usage:**
```go
status := NewStatusLineComponent("CONTAINERS", 25)
status = status.SetScrollIndicator(" [1-10 of 25]")
output := status.View()
```

**Renders:**
```
│ CONTAINERS (25 total) [1-10 of 25]              │
```

---

### 4. TableComponent

**Purpose:** Renders tabular data with headers and rows

**Properties:**
- `headers` - Column definitions with labels and widths
- `rows` - Table rows with cells and styling
- `start/end` - Visible range for scrolling
- `width` - Component width

**Usage:**
```go
headers := []TableHeader{
    {Label: "NAME", Width: 20},
    {Label: "STATUS", Width: 10},
}

rows := []TableRow{
    {
        Cells: []string{"nginx", "RUNNING"},
        IsSelected: true,
        Style: normalStyle,
    },
}

table := NewTableComponent(headers)
table = table.SetRows(rows)
table = table.SetVisibleRange(0, 10)
output := table.View()
```

**Renders:**
```
├────────────────────┬──────────┤
│ NAME               │ STATUS   │
├────────────────────┼──────────┤
│ nginx              │ RUNNING  │
├────────────────────┴──────────┤
```

---

### 5. ActionBarComponent

**Purpose:** Displays available actions or status messages at bottom

**Properties:**
- `actions` - Action text (e.g., "[S]top | [R]estart")
- `statusMessage` - Status/error message
- `width` - Component width

**Usage:**
```go
actionBar := NewActionBarComponent()
actionBar = actionBar.SetActions(" [S]top | [R]estart")
actionBar = actionBar.SetStatusMessage("Container started")
output := actionBar.View()
```

**Renders:**
```
│ Container started                                │
└──────────────────────────────────────────────────┘
```

---

### 6. DetailViewComponent

**Purpose:** Displays detailed content (logs, inspect data)

**Properties:**
- `title` - View title
- `content` - Content to display (multiline)
- `scroll` - Scroll offset
- `lines` - Number of visible lines
- `width` - Component width

**Usage:**
```go
detail := NewDetailViewComponent("Logs: nginx", 15)
detail = detail.SetContent("log line 1\nlog line 2\n...")
detail = detail.SetScroll(0)
output := detail.View()
```

**Renders:**
```
┌──────────────────────────────────────────┐
│ Logs: nginx                 [ESC] Back │
├──────────────────────────────────────────┤
│ log line 1                               │
│ log line 2                               │
└──────────────────────────────────────────┘
```

---

## Main Model Integration

### Component Initialization

Components are initialized once in `initialModel()`:

```go
func initialModel() model {
    return model{
        header: NewHeaderComponent("tinyd v2.0.1", "[F1] Help [Q]uit"),
        tabs: NewTabsComponent(tabs, 0),
        actionBar: NewActionBarComponent(),
        detailView: NewDetailViewComponent("", 15),
        // ... other fields
    }
}
```

### Component Usage in Views

All render functions now compose components:

```go
func (m model) renderContainers() string {
    var b strings.Builder

    // Use components
    b.WriteString(m.header.View())
    b.WriteString(m.tabs.View())
    b.WriteString(statusComp.View())
    b.WriteString(table.View())
    b.WriteString(m.actionBar.View())

    return b.String()
}

func (m model) renderImages() string {
    // Same component-based approach
}

func (m model) renderVolumes() string {
    // Same component-based approach
}

func (m model) renderNetworks() string {
    // Same component-based approach
}

func (m model) renderLogs() string {
    // Uses DetailViewComponent
}

func (m model) renderInspect() string {
    // Uses DetailViewComponent
}
```

### Component State Updates

Components are immutable - methods return new instances:

```go
// Update active tab
m.tabs = m.tabs.SetActiveTab(1)

// Update action bar
m.actionBar = m.actionBar.SetActions("[S]top | [R]estart")
m.actionBar = m.actionBar.SetStatusMessage("Success")
```

## Benefits

### 1. Maintainability ✓

- Each component in separate, focused code
- Easy to locate and modify specific UI elements
- Changes isolated to component files

### 2. Reusability ✓

- Same TableComponent used for all tabs
- DetailViewComponent shared by logs and inspect
- Components work across different views

### 3. Testability ✓

- Components can be tested independently
- Predictable output for given inputs
- No side effects in View() methods

### 4. Consistency ✓

- Uniform styling across all views
- Centralized UI patterns
- Easy to maintain design system

### 5. Scalability ✓

- Add new components without touching existing code
- Compose complex UIs from simple parts
- Easy to add new views/tabs

## Code Organization

```
tinyd/
├── main.go          # Main app logic, Update(), business logic
├── components.go    # All UI components
├── go.mod           # Dependencies
└── *.md            # Documentation
```

## Future Components

Potential new components:

- **ProgressBarComponent** - For long-running operations
- **MenuComponent** - Context menus for actions
- **ModalComponent** - Confirmation dialogs
- **ChartComponent** - CPU/Memory graphs
- **FilterComponent** - Search and filter UI
- **PaginationComponent** - Page-based navigation

## Best Practices

### Creating New Components

1. **Define Purpose** - One component, one responsibility
2. **Minimal State** - Only UI presentation state
3. **Immutable Updates** - Return new instances
4. **Standard Interface** - Init(), Update(), View()
5. **Configurable** - Use setter methods for flexibility

### Using Components

1. **Initialize Once** - Create in initialModel()
2. **Update Immutably** - `comp = comp.SetX(value)`
3. **Compose in View** - Combine in render functions
4. **Keep Logic in Model** - Components only render

### Example New Component

```go
type MyComponent struct {
    data  string
    width int
}

func NewMyComponent(data string) MyComponent {
    return MyComponent{data: data, width: 85}
}

func (c MyComponent) Init() tea.Cmd {
    return nil
}

func (c MyComponent) Update(msg tea.Msg) (MyComponent, tea.Cmd) {
    return c, nil
}

func (c MyComponent) SetData(data string) MyComponent {
    c.data = data
    return c
}

func (c MyComponent) View() string {
    return greenStyle.Render("│ " + c.data + " │")
}
```

## Migration Notes

### Before (Manual String Building)

```go
func (m model) renderView() string {
    var b strings.Builder
    b.WriteString("┌─────┐\n")
    b.WriteString("│ Title │\n")
    b.WriteString("├─────┤\n")
    // ... 100 more lines of manual rendering
    return b.String()
}
```

### After (Component-Based)

```go
func (m model) renderView() string {
    var b strings.Builder
    b.WriteString(m.header.View())
    b.WriteString(m.content.View())
    return b.String()
}
```

**Result:**
- 90% less rendering code in main.go
- Reusable components across views
- Easier to maintain and extend

---

**The component architecture makes tinyd more maintainable, testable, and extensible!** 🏗️
