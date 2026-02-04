package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HeaderComponent renders the top header bar
type HeaderComponent struct {
	title   string
	help    string
	width   int
}

func NewHeaderComponent(title, help string) HeaderComponent {
	return HeaderComponent{
		title: title,
		help:  help,
		width: 80, // Default, will be updated
	}
}

func (h HeaderComponent) WithWidth(width int) HeaderComponent {
	h.width = width - 2 // Account for borders
	return h
}

func (h HeaderComponent) Init() tea.Cmd {
	return nil
}

func (h HeaderComponent) Update(msg tea.Msg) (HeaderComponent, tea.Cmd) {
	return h, nil
}

func (h HeaderComponent) View() string {
	// Minimalistic: no header, just return empty
	return ""
}

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
	return TabsComponent{
		tabs:      tabs,
		activeTab: activeTab,
		width:     80,
	}
}

func (t TabsComponent) WithWidth(width int) TabsComponent {
	t.width = width - 2
	return t
}

func (t TabsComponent) Init() tea.Cmd {
	return nil
}

func (t TabsComponent) Update(msg tea.Msg) (TabsComponent, tea.Cmd) {
	return t, nil
}

func (t TabsComponent) SetActiveTab(index int) TabsComponent {
	t.activeTab = index
	return t
}

func (t TabsComponent) View() string {
	var b strings.Builder

	borderColor := lipgloss.Color("#303030")
	bgColor := lipgloss.Color("#0a0a0a")
	activeColor := lipgloss.Color("#FFFFFF")
	inactiveColor := lipgloss.Color("#666666")

	borderStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Background(bgColor)

	// Top row with rounded corners
	b.WriteString(" ")
	for _, tab := range t.tabs {
		tabText := fmt.Sprintf(" %s ", tab.Name)
		tabWidth := len(tabText)

		// Top border with rounded corners
		b.WriteString(borderStyle.Render("╭"))
		b.WriteString(borderStyle.Render(strings.Repeat("─", tabWidth)))
		b.WriteString(borderStyle.Render("╮"))
	}
	b.WriteString("\n")

	// Middle row with tab labels
	b.WriteString(" ")
	for i, tab := range t.tabs {
		tabText := fmt.Sprintf(" %s ", tab.Name)

		// Left border
		b.WriteString(borderStyle.Render("│"))

		// Tab text
		textStyle := lipgloss.NewStyle().
			Foreground(inactiveColor).
			Background(bgColor)
		if i == t.activeTab {
			textStyle = lipgloss.NewStyle().
				Foreground(activeColor).
				Background(bgColor).
				Bold(true)
		}
		b.WriteString(textStyle.Render(tabText))

		// Right border
		b.WriteString(borderStyle.Render("│"))
	}
	b.WriteString("\n")

	// Bottom row with connecting line
	b.WriteString(borderStyle.Render("─"))
	for i, tab := range t.tabs {
		tabText := fmt.Sprintf(" %s ", tab.Name)
		tabWidth := len(tabText)

		if i == t.activeTab {
			// Active tab: no bottom border (open to content)
			b.WriteString(borderStyle.Render("╯"))
			b.WriteString(strings.Repeat(" ", tabWidth))
			b.WriteString(borderStyle.Render("╰"))
		} else {
			// Inactive tab: bottom border connects to line
			b.WriteString(borderStyle.Render("┴"))
			b.WriteString(borderStyle.Render(strings.Repeat("─", tabWidth)))
			b.WriteString(borderStyle.Render("┴"))
		}
	}

	// Calculate remaining width for the horizontal line
	totalTabWidth := 1 // Initial left padding
	for _, tab := range t.tabs {
		totalTabWidth += len(fmt.Sprintf(" %s ", tab.Name)) + 2 // +2 for borders
	}
	remaining := t.width - totalTabWidth
	if remaining > 0 {
		b.WriteString(borderStyle.Render(strings.Repeat("─", remaining)))
	}
	b.WriteString("\n")

	return b.String()
}

// StatusLineComponent renders a status line with item count
type StatusLineComponent struct {
	label           string
	count           int
	scrollIndicator string
	width           int
}

func NewStatusLineComponent(label string, count int) StatusLineComponent {
	return StatusLineComponent{
		label: label,
		count: count,
		width: 80,
	}
}

func (s StatusLineComponent) WithWidth(width int) StatusLineComponent {
	s.width = width - 2
	return s
}

func (s StatusLineComponent) Init() tea.Cmd {
	return nil
}

func (s StatusLineComponent) Update(msg tea.Msg) (StatusLineComponent, tea.Cmd) {
	return s, nil
}

func (s StatusLineComponent) SetScrollIndicator(indicator string) StatusLineComponent {
	s.scrollIndicator = indicator
	return s
}

func (s StatusLineComponent) View() string {
	// Minimalistic: no status line, or very subtle
	return ""
}

// TableComponent renders a table with headers and rows
type TableComponent struct {
	headers []TableHeader
	rows    []TableRow
	start   int
	end     int
	width   int
}

type TableHeader struct {
	Label     string
	Width     int
	AlignRight bool // Right-align for numbers, left-align for text
}

type TableRow struct {
	Cells      []string
	IsSelected bool
	Style      lipgloss.Style
}

func NewTableComponent(headers []TableHeader) TableComponent {
	return TableComponent{
		headers: headers,
		rows:    []TableRow{},
		width:   80,
	}
}

func (t TableComponent) WithWidth(width int) TableComponent {
	t.width = width - 2
	return t
}

func (t TableComponent) Init() tea.Cmd {
	return nil
}

func (t TableComponent) Update(msg tea.Msg) (TableComponent, tea.Cmd) {
	return t, nil
}

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

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Background(lipgloss.Color("#0a0a0a")).
		Bold(true)

	normalCellStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	selectedCellStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0a0a0a"))

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#303030")).
		Background(lipgloss.Color("#0a0a0a"))

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

	// Header bottom line
	totalWidth := 0
	for _, header := range t.headers {
		totalWidth += header.Width
	}
	totalWidth += (len(t.headers) - 1) * 2 // Add spacing between columns
	b.WriteString(lineStyle.Render(strings.Repeat("─", totalWidth)))
	b.WriteString("\n")

	// Table rows
	if len(t.rows) == 0 {
		emptyMsg := " No items found"
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444")).
			Background(lipgloss.Color("#0a0a0a"))
		b.WriteString(emptyStyle.Render(emptyMsg))
		b.WriteString("\n")
	} else {
		for i := t.start; i < t.end && i < len(t.rows); i++ {
			row := t.rows[i]

			for j, cell := range row.Cells {
				if j < len(t.headers) {
					// Second column (index 1) is the status dot - render as-is without styling
					if j == 1 && strings.Contains(cell, "●") {
						b.WriteString(cell)
						if t.headers[j].Width > 1 {
							b.WriteString(normalCellStyle.Render(strings.Repeat(" ", t.headers[j].Width-1)))
						}
					} else {
						// Apply alignment based on header
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
						b.WriteString(normalCellStyle.Render("  "))
					}
				}
			}
			b.WriteString("\n")
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

func NewActionBarComponent() ActionBarComponent {
	return ActionBarComponent{
		width: 80,
	}
}

func (a ActionBarComponent) WithWidth(width int) ActionBarComponent {
	a.width = width - 2
	return a
}

func (a ActionBarComponent) Init() tea.Cmd {
	return nil
}

func (a ActionBarComponent) Update(msg tea.Msg) (ActionBarComponent, tea.Cmd) {
	return a, nil
}

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

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#303030")).
		Background(lipgloss.Color("#0a0a0a"))

	actionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")).
		Background(lipgloss.Color("#0a0a0a"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Background(lipgloss.Color("#0a0a0a"))

	// Top line
	b.WriteString(lineStyle.Render(strings.Repeat("─", a.width)))
	b.WriteString("\n")

	// Action bar content
	if a.statusMessage != "" {
		style := statusStyle
		if strings.HasPrefix(a.statusMessage, "ERROR:") {
			style = errorStyle
		}
		msg := a.statusMessage
		if len(msg) > a.width-2 {
			msg = msg[:a.width-5] + "..."
		}
		b.WriteString(style.Render(msg))
	} else if a.actions != "" {
		msg := a.actions
		if len(msg) > a.width-2 {
			msg = msg[:a.width-5] + "..."
		}
		b.WriteString(actionStyle.Render(msg))
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
	return DetailViewComponent{
		title: title,
		lines: lines,
		width: 80,
	}
}

func (d DetailViewComponent) WithWidth(width int) DetailViewComponent {
	d.width = width - 2
	return d
}

func (d DetailViewComponent) Init() tea.Cmd {
	return nil
}

func (d DetailViewComponent) Update(msg tea.Msg) (DetailViewComponent, tea.Cmd) {
	return d, nil
}

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

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0a0a0a")).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#303030")).
		Background(lipgloss.Color("#0a0a0a"))

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#0a0a0a"))

	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444")).
		Background(lipgloss.Color("#0a0a0a"))

	// Header
	headerText := d.title
	headerRight := "[ESC] Back"
	headerSpacing := strings.Repeat(" ", d.width-len(headerText)-len(headerRight))
	b.WriteString(titleStyle.Render(headerText))
	b.WriteString(strings.Repeat(" ", len(headerSpacing)))
	b.WriteString(helpStyle.Render(headerRight))
	b.WriteString("\n")

	// Content divider
	b.WriteString(lineStyle.Render(strings.Repeat("─", d.width)))
	b.WriteString("\n")

	// Content
	if d.content == "" {
		loadingMsg := " Loading..."
		b.WriteString(loadingStyle.Render(loadingMsg))
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

		// Fill remaining lines
		for i := end - d.scroll; i < d.lines; i++ {
			b.WriteString("\n")
		}
	}

	return b.String()
}
