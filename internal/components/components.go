package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DarkBackground is set by InitTheme. Exported so other packages can read it.
var DarkBackground = true

// Color palette — populated by InitTheme before the program starts.
var (
	ColorNormal       lipgloss.Color
	ColorDim          lipgloss.Color
	ColorBright       lipgloss.Color
	ColorHeader       lipgloss.Color
	ColorBorder       lipgloss.Color
	ColorActiveBorder lipgloss.Color
	ColorHighlight    lipgloss.Color
	ColorEmpty        lipgloss.Color

	// Selection highlight: colored background so it's always visible on any
	// terminal background color, regardless of theme detection accuracy.
	ColorSelectedBg lipgloss.Color
	ColorSelectedFg lipgloss.Color
)

func init() { InitTheme(true) } // default: dark; overridden by main() before NewProgram

// InitTheme selects the color palette for a dark or light terminal.
// Override auto-detection with TINYD_THEME=light or TINYD_THEME=dark.
func InitTheme(dark bool) {
	DarkBackground = dark
	if dark {
		ColorNormal = lipgloss.Color("#AAAAAA")
		ColorDim = lipgloss.Color("#666666")
		ColorBright = lipgloss.Color("#EEEEEE")
		ColorHeader = lipgloss.Color("#CCCCCC")
		ColorBorder = lipgloss.Color("#3A3A3A")
		ColorActiveBorder = lipgloss.Color("#DDDDDD")
		ColorHighlight = lipgloss.Color("#00CCFF")
		ColorEmpty = lipgloss.Color("#3A3A3A")
		ColorSelectedBg = lipgloss.Color("#1D4ED8")
		ColorSelectedFg = lipgloss.Color("#FFFFFF")
	} else {
		// Light mode: high-contrast dark-on-white palette
		ColorNormal = lipgloss.Color("#222222") // near-black body text
		ColorDim = lipgloss.Color("#666666")    // medium gray for inactive items
		ColorBright = lipgloss.Color("#000000") // pure black for emphasis
		ColorHeader = lipgloss.Color("#000000") // black bold headers
		ColorBorder = lipgloss.Color("#CCCCCC") // light gray separators
		ColorActiveBorder = lipgloss.Color("#000000")
		ColorHighlight = lipgloss.Color("#005588") // dark teal
		ColorEmpty = lipgloss.Color("#BBBBBB")
		ColorSelectedBg = lipgloss.Color("#1D4ED8") // blue bg — always pops
		ColorSelectedFg = lipgloss.Color("#FFFFFF")
	}
}

// HeaderComponent renders the top header bar
type HeaderComponent struct {
	title string
	help  string
	width int
}

func NewHeaderComponent(title, help string) HeaderComponent {
	return HeaderComponent{title: title, help: help, width: 80}
}

func (h HeaderComponent) WithWidth(width int) HeaderComponent {
	h.width = width - 2
	return h
}

func (h HeaderComponent) Init() tea.Cmd { return nil }

func (h HeaderComponent) Update(msg tea.Msg) (HeaderComponent, tea.Cmd) { return h, nil }

func (h HeaderComponent) View() string { return "" }

// TabsComponent renders the tab navigation
type TabsComponent struct {
	tabs      []TabItem
	activeTab int
	width     int
}

type TabItem struct {
	Name     string
	Shortcut string
}

func NewTabsComponent(tabs []TabItem, activeTab int) TabsComponent {
	return TabsComponent{tabs: tabs, activeTab: activeTab, width: 80}
}

func (t TabsComponent) WithWidth(width int) TabsComponent {
	t.width = width - 2
	return t
}

func (t TabsComponent) Init() tea.Cmd { return nil }

func (t TabsComponent) Update(msg tea.Msg) (TabsComponent, tea.Cmd) { return t, nil }

func (t TabsComponent) SetActiveTab(index int) TabsComponent {
	t.activeTab = index
	return t
}

func (t TabsComponent) View() string {
	var b strings.Builder

	borderStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	activeBorderStyle := lipgloss.NewStyle().Foreground(ColorActiveBorder).Bold(true)

	// Top row with rounded corners
	b.WriteString(" ")
	for i, tab := range t.tabs {
		tabText := fmt.Sprintf(" %s ", tab.Name)
		tabWidth := len(tabText)
		style := borderStyle
		if i == t.activeTab {
			style = activeBorderStyle
		}
		b.WriteString(style.Render("╭"))
		b.WriteString(style.Render(strings.Repeat("─", tabWidth)))
		b.WriteString(style.Render("╮"))
	}
	b.WriteString("\n")

	// Middle row with tab labels
	b.WriteString(" ")
	for i, tab := range t.tabs {
		tabText := fmt.Sprintf(" %s ", tab.Name)
		bStyle := borderStyle
		if i == t.activeTab {
			bStyle = activeBorderStyle
		}
		b.WriteString(bStyle.Render("│"))
		textStyle := lipgloss.NewStyle().Foreground(ColorDim)
		if i == t.activeTab {
			textStyle = lipgloss.NewStyle().Foreground(ColorBright).Bold(true)
		}
		b.WriteString(textStyle.Render(tabText))
		b.WriteString(bStyle.Render("│"))
	}
	b.WriteString("\n")

	// Bottom row — the main horizontal line spans the full width and should
	// always be bright/visible, so everything here uses activeBorderStyle.
	b.WriteString(activeBorderStyle.Render("─"))
	for i, tab := range t.tabs {
		tabText := fmt.Sprintf(" %s ", tab.Name)
		tabWidth := len(tabText)
		if i == t.activeTab {
			b.WriteString(activeBorderStyle.Render("╯"))
			b.WriteString(strings.Repeat(" ", tabWidth))
			b.WriteString(activeBorderStyle.Render("╰"))
		} else {
			b.WriteString(activeBorderStyle.Render("─"))
			b.WriteString(activeBorderStyle.Render(strings.Repeat("─", tabWidth)))
			b.WriteString(activeBorderStyle.Render("─"))
		}
	}

	totalTabWidth := 1
	for _, tab := range t.tabs {
		totalTabWidth += len(fmt.Sprintf(" %s ", tab.Name)) + 2
	}
	remaining := t.width - totalTabWidth
	if remaining > 0 {
		b.WriteString(activeBorderStyle.Render(strings.Repeat("─", remaining)))
	}
	b.WriteString("\n")

	return b.String()
}

// StatusLineComponent (unused but kept for API compat)
type StatusLineComponent struct {
	label           string
	count           int
	scrollIndicator string
	width           int
}

func NewStatusLineComponent(label string, count int) StatusLineComponent {
	return StatusLineComponent{label: label, count: count, width: 80}
}

func (s StatusLineComponent) WithWidth(width int) StatusLineComponent {
	s.width = width - 2
	return s
}

func (s StatusLineComponent) Init() tea.Cmd { return nil }

func (s StatusLineComponent) Update(msg tea.Msg) (StatusLineComponent, tea.Cmd) { return s, nil }

func (s StatusLineComponent) SetScrollIndicator(indicator string) StatusLineComponent {
	s.scrollIndicator = indicator
	return s
}

func (s StatusLineComponent) View() string { return "" }

// TableComponent renders a table with headers and rows
type TableComponent struct {
	headers []TableHeader
	rows    []TableRow
	start   int
	end     int
	width   int
}

type TableHeader struct {
	Label      string
	Width      int
	AlignRight bool
}

type TableRow struct {
	Cells      []string
	IsSelected bool
	Style      lipgloss.Style
}

func NewTableComponent(headers []TableHeader) TableComponent {
	return TableComponent{headers: headers, rows: []TableRow{}, width: 80}
}

func (t TableComponent) WithWidth(width int) TableComponent {
	t.width = width - 2
	return t
}

func (t TableComponent) Init() tea.Cmd { return nil }

func (t TableComponent) Update(msg tea.Msg) (TableComponent, tea.Cmd) { return t, nil }

func (t TableComponent) SetRows(rows []TableRow) TableComponent {
	t.rows = rows
	return t
}

func (t TableComponent) SetVisibleRange(start, end int) TableComponent {
	t.start = start
	t.end = end
	return t
}

func (t TableComponent) View() string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Foreground(ColorHeader).Bold(true)
	normalCellStyle := lipgloss.NewStyle().Foreground(ColorNormal)
	selectedCellStyle := lipgloss.NewStyle().
		Background(ColorSelectedBg).
		Foreground(ColorSelectedFg).
		Bold(true)
	lineStyle := lipgloss.NewStyle().Foreground(ColorBorder)

	// Table headers
	for j, header := range t.headers {
		var headerText string
		if header.AlignRight {
			headerText = padLeft(header.Label, header.Width)
		} else {
			headerText = padRight(header.Label, header.Width)
		}
		b.WriteString(headerStyle.Render(headerText))
		if j < len(t.headers)-1 {
			b.WriteString(normalCellStyle.Render("  "))
		}
	}
	b.WriteString("\n")

	// Header divider
	totalWidth := 0
	for _, header := range t.headers {
		totalWidth += header.Width
	}
	totalWidth += (len(t.headers) - 1) * 2
	b.WriteString(lineStyle.Render(strings.Repeat("─", totalWidth)))
	b.WriteString("\n")

	// Table rows
	if len(t.rows) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(ColorEmpty)
		b.WriteString(emptyStyle.Render(" No items found"))
		b.WriteString("\n")
	} else {
		for i := t.start; i < t.end && i < len(t.rows); i++ {
			row := t.rows[i]

			// Detect full-width rows (delete confirmation overlay)
			isFullWidthRow := len(row.Cells) > 1
			for j := 1; j < len(row.Cells); j++ {
				if row.Cells[j] != "" {
					isFullWidthRow = false
					break
				}
			}

			if isFullWidthRow && row.Cells[0] != "" {
				b.WriteString(row.Cells[0])
				b.WriteString("\n")
			} else {
				// Pick spacer style once per row so the gap between columns
				// also carries the selection background.
				spacerStyle := normalCellStyle
				if row.IsSelected {
					spacerStyle = selectedCellStyle
				}

				for j, cell := range row.Cells {
					if j < len(t.headers) {
						if j == 0 && isStatusGlyph(cell) {
							// The cell already carries its own fg color (and bg
							// when the row is selected — set by the caller).
							// Write it as-is so the status color stays visible,
							// then pad the column with the row's background.
							b.WriteString(cell)
							if t.headers[j].Width > 1 {
								b.WriteString(spacerStyle.Render(strings.Repeat(" ", t.headers[j].Width-1)))
							}
						} else if isPrestyled(cell) {
							// Cell already carries ANSI styling — preserve it
							// instead of re-wrapping with the selection style
							// (which would override the per-state color).
							b.WriteString(cell)
							visible := len(stripAnsiCodes(cell))
							if visible < t.headers[j].Width {
								b.WriteString(spacerStyle.Render(strings.Repeat(" ", t.headers[j].Width-visible)))
							}
						} else {
							var cellText string
							if t.headers[j].AlignRight {
								cellText = padLeft(cell, t.headers[j].Width)
							} else {
								cellText = padRight(cell, t.headers[j].Width)
							}
							if row.IsSelected {
								b.WriteString(selectedCellStyle.Render(cellText))
							} else {
								b.WriteString(normalCellStyle.Render(cellText))
							}
						}
						if j < len(t.headers)-1 {
							b.WriteString(spacerStyle.Render("  "))
						}
					}
				}
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

// ActionBarComponent renders the action bar at the bottom
type ActionBarComponent struct {
	actions       string
	statusMessage string
	width         int
}

func NewActionBarComponent() ActionBarComponent { return ActionBarComponent{width: 80} }

func (a ActionBarComponent) WithWidth(width int) ActionBarComponent {
	a.width = width - 2
	return a
}

func (a ActionBarComponent) Init() tea.Cmd { return nil }

func (a ActionBarComponent) Update(msg tea.Msg) (ActionBarComponent, tea.Cmd) { return a, nil }

func (a ActionBarComponent) SetActions(actions string) ActionBarComponent {
	a.actions = actions
	return a
}

func (a ActionBarComponent) SetStatusMessage(message string) ActionBarComponent {
	a.statusMessage = message
	return a
}

func (a ActionBarComponent) View() string {
	var b strings.Builder

	lineStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	statusStyle := lipgloss.NewStyle().Foreground(ColorHighlight)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))

	b.WriteString(lineStyle.Render(strings.Repeat("─", a.width)))
	b.WriteString("\n")

	if a.actions != "" {
		b.WriteString(a.actions)
	}

	if a.statusMessage != "" {
		style := statusStyle
		if strings.HasPrefix(a.statusMessage, "ERROR:") {
			style = errorStyle
		}
		actionsLen := len(stripAnsiCodes(a.actions))
		statusLen := len(a.statusMessage)
		spacing := a.width - actionsLen - statusLen - 2
		if spacing < 1 {
			spacing = 1
		}
		b.WriteString(strings.Repeat(" ", spacing))
		b.WriteString(style.Render(a.statusMessage))
	}
	b.WriteString("\n")

	return b.String()
}

// DetailViewComponent renders logs or inspect views
type DetailViewComponent struct {
	title   string
	content string
	scroll  int
	lines   int
	width   int
}

func NewDetailViewComponent(title string, lines int) DetailViewComponent {
	return DetailViewComponent{title: title, lines: lines, width: 80}
}

func (d DetailViewComponent) WithWidth(width int) DetailViewComponent {
	d.width = width - 2
	return d
}

func (d DetailViewComponent) Init() tea.Cmd { return nil }

func (d DetailViewComponent) Update(msg tea.Msg) (DetailViewComponent, tea.Cmd) { return d, nil }

func (d DetailViewComponent) SetContent(content string) DetailViewComponent {
	d.content = content
	return d
}

func (d DetailViewComponent) SetScroll(scroll int) DetailViewComponent {
	d.scroll = scroll
	return d
}

func (d DetailViewComponent) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(ColorBright).Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(ColorNormal)
	lineStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	contentStyle := lipgloss.NewStyle().Foreground(ColorNormal)
	loadingStyle := lipgloss.NewStyle().Foreground(ColorEmpty)

	headerText := d.title
	headerRight := "[ESC] Back"
	headerSpacing := strings.Repeat(" ", d.width-len(headerText)-len(headerRight))
	b.WriteString(titleStyle.Render(headerText))
	b.WriteString(strings.Repeat(" ", len(headerSpacing)))
	b.WriteString(helpStyle.Render(headerRight))
	b.WriteString("\n")

	b.WriteString(lineStyle.Render(strings.Repeat("─", d.width)))
	b.WriteString("\n")

	if d.content == "" {
		b.WriteString(loadingStyle.Render(" Loading..."))
		b.WriteString("\n")
	} else {
		lines := strings.Split(d.content, "\n")
		end := d.scroll + d.lines
		if end > len(lines) {
			end = len(lines)
		}
		for i := d.scroll; i < end; i++ {
			if i < len(lines) {
				line := lines[i]
				if len(line) > d.width {
					line = line[:d.width-3] + "..."
				}
				b.WriteString(contentStyle.Render(line))
				b.WriteString("\n")
			}
		}
		for i := end - d.scroll; i < d.lines; i++ {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// isPrestyled reports whether a cell already contains ANSI escape sequences.
// The TableComponent treats such cells as literal: their styling is preserved
// (not overridden by the selection style) and width is computed via the
// stripped string so multi-byte UTF-8 runes don't get sliced.
func isPrestyled(s string) bool {
	return strings.Contains(s, "\x1b[")
}

// isStatusGlyph reports whether a cell carries one of the single-cell glyphs
// used in the status column (filled/empty dot or any braille spinner frame).
// These cells are multi-byte UTF-8 and must not be passed through padRight,
// which would slice them in the middle of a rune.
func isStatusGlyph(cell string) bool {
	if strings.Contains(cell, "●") || strings.Contains(cell, "○") {
		return true
	}
	for _, frame := range []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"} {
		if strings.Contains(cell, frame) {
			return true
		}
	}
	return false
}

// stripAnsiCodes removes ANSI escape sequences for length calculation
func stripAnsiCodes(str string) string {
	result := str
	for strings.Contains(result, "\x1b[") {
		start := strings.Index(result, "\x1b[")
		end := strings.Index(result[start:], "m")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return result
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func padLeft(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return strings.Repeat(" ", width-len(s)) + s
}
