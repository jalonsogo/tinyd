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
	var b strings.Builder

	// Top border
	b.WriteString(greenStyle.Render("┌" + strings.Repeat("─", h.width) + "┐"))
	b.WriteString("\n")

	// Header content
	headerLeft := greenStyle.Render("│ " + h.title)
	headerRight := greenStyle.Render(h.help + " │")
	headerSpacing := strings.Repeat(" ", h.width-len(" "+h.title)-len(h.help+" │"))
	b.WriteString(headerLeft + headerSpacing + headerRight)
	b.WriteString("\n")

	return b.String()
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

	// Calculate tab widths
	tabWidths := make([]int, len(t.tabs))
	for i, tab := range t.tabs {
		tabWidths[i] = 1 + len(tab.Name) + 1 + len(tab.Shortcut) + 1
	}

	// Top line with rounded corners
	b.WriteString(" ")
	for _, width := range tabWidths {
		b.WriteString(greenStyle.Render("╭"))
		b.WriteString(greenStyle.Render(strings.Repeat("─", width)))
		b.WriteString(greenStyle.Render("╮"))
	}
	b.WriteString("\n")

	// Tab labels
	b.WriteString(" ")
	for i, tab := range t.tabs {
		b.WriteString(greenStyle.Render("│"))
		content := fmt.Sprintf(" %s %s ", tab.Name, tab.Shortcut)
		if i == t.activeTab {
			b.WriteString(yellowStyle.Render(content))
		} else {
			b.WriteString(greenStyle.Render(content))
		}
		b.WriteString(greenStyle.Render("│"))
	}
	b.WriteString("\n")

	// Bottom line
	b.WriteString(greenStyle.Render("─"))
	for i, width := range tabWidths {
		if i == t.activeTab {
			b.WriteString(greenStyle.Render("╯"))
			b.WriteString(strings.Repeat(" ", width))
			b.WriteString(greenStyle.Render("╰"))
		} else {
			b.WriteString(greenStyle.Render("┴"))
			b.WriteString(greenStyle.Render(strings.Repeat("─", width)))
			b.WriteString(greenStyle.Render("┴"))
		}
	}

	// Extend line to edge
	totalTabWidth := 1
	for _, width := range tabWidths {
		totalTabWidth += width + 2
	}
	remaining := t.width - totalTabWidth
	b.WriteString(greenStyle.Render(strings.Repeat("─", remaining)))
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
	statusText := fmt.Sprintf(" %s (%d total)%s", s.label, s.count, s.scrollIndicator)
	statusLine := greenStyle.Render("│") + cyanStyle.Render(statusText)
	statusSpacing := strings.Repeat(" ", s.width-len(statusText))
	return statusLine + statusSpacing + greenStyle.Render("│") + "\n"
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
	Label string
	Width int
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

	// Table divider
	divider := "├"
	for i, header := range t.headers {
		divider += strings.Repeat("─", header.Width)
		if i < len(t.headers)-1 {
			divider += "┼"
		}
	}
	divider += "┤"
	b.WriteString(greenStyle.Render(divider))
	b.WriteString("\n")

	// Table headers
	b.WriteString(greenStyle.Render("│"))
	for _, header := range t.headers {
		b.WriteString(normalStyle.Render(padCenter(header.Label, header.Width)))
		b.WriteString(greenStyle.Render("│"))
	}
	b.WriteString("\n")

	// Header bottom divider
	headerDivider := "├"
	for i, header := range t.headers {
		headerDivider += strings.Repeat("─", header.Width)
		if i < len(t.headers)-1 {
			headerDivider += "┼"
		}
	}
	headerDivider += "┤"
	b.WriteString(greenStyle.Render(headerDivider))
	b.WriteString("\n")

	// Table rows
	if len(t.rows) == 0 {
		b.WriteString(greenStyle.Render("│"))
		emptyMsg := " No items found"
		b.WriteString(cyanStyle.Render(emptyMsg))
		b.WriteString(strings.Repeat(" ", t.width-len(emptyMsg)))
		b.WriteString(greenStyle.Render("│"))
		b.WriteString("\n")
	} else {
		for i := t.start; i < t.end && i < len(t.rows); i++ {
			row := t.rows[i]
			b.WriteString(greenStyle.Render("│"))

			for j, cell := range row.Cells {
				if j < len(t.headers) {
					cellText := padRight(cell, t.headers[j].Width)
					if row.IsSelected {
						b.WriteString(yellowStyle.Render(cellText))
					} else {
						b.WriteString(row.Style.Render(cellText))
					}
					b.WriteString(greenStyle.Render("│"))
				}
			}
			b.WriteString("\n")
		}
	}

	// Table bottom border
	bottomDivider := "├"
	for i, header := range t.headers {
		bottomDivider += strings.Repeat("─", header.Width)
		if i < len(t.headers)-1 {
			bottomDivider += "┴"
		}
	}
	bottomDivider += "┤"
	b.WriteString(greenStyle.Render(bottomDivider))
	b.WriteString("\n")

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

	b.WriteString(greenStyle.Render("│"))

	if a.statusMessage != "" {
		statusStyle := cyanStyle
		if strings.HasPrefix(a.statusMessage, "ERROR:") {
			statusStyle = redStyle
		}
		msg := " " + a.statusMessage
		if len(msg) > a.width-2 {
			msg = msg[:a.width-5] + "..."
		}
		b.WriteString(statusStyle.Render(msg))
		b.WriteString(strings.Repeat(" ", a.width-len(msg)))
	} else if a.actions != "" {
		if len(a.actions) > a.width-2 {
			a.actions = a.actions[:a.width-5] + "..."
		}
		b.WriteString(cyanStyle.Render(a.actions))
		b.WriteString(strings.Repeat(" ", a.width-len(a.actions)))
	} else {
		b.WriteString(strings.Repeat(" ", a.width))
	}

	b.WriteString(greenStyle.Render("│"))
	b.WriteString("\n")

	// Bottom border
	b.WriteString(greenStyle.Render("└" + strings.Repeat("─", a.width) + "┘"))

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

	// Top border
	b.WriteString(greenStyle.Render("┌" + strings.Repeat("─", d.width) + "┐"))
	b.WriteString("\n")

	// Header
	headerText := fmt.Sprintf("│ %s", d.title)
	headerRight := "[ESC] Back │"
	headerSpacing := strings.Repeat(" ", d.width-len(d.title)-len(headerRight))
	b.WriteString(greenStyle.Render(headerText))
	b.WriteString(headerSpacing)
	b.WriteString(greenStyle.Render(headerRight))
	b.WriteString("\n")

	// Content divider
	b.WriteString(greenStyle.Render("├" + strings.Repeat("─", d.width) + "┤"))
	b.WriteString("\n")

	// Content
	if d.content == "" {
		b.WriteString(greenStyle.Render("│"))
		loadingMsg := " Loading..."
		b.WriteString(cyanStyle.Render(loadingMsg))
		b.WriteString(strings.Repeat(" ", d.width-len(loadingMsg)))
		b.WriteString(greenStyle.Render("│"))
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
				if len(line) > d.width-2 {
					line = line[:d.width-5] + "..."
				}
				b.WriteString(greenStyle.Render("│"))
				b.WriteString(normalStyle.Render(line))
				b.WriteString(strings.Repeat(" ", d.width-len(line)))
				b.WriteString(greenStyle.Render("│"))
				b.WriteString("\n")
			}
		}

		// Fill remaining lines
		for i := end - d.scroll; i < d.lines; i++ {
			b.WriteString(greenStyle.Render("│"))
			b.WriteString(strings.Repeat(" ", d.width))
			b.WriteString(greenStyle.Render("│"))
			b.WriteString("\n")
		}
	}

	// Bottom border
	b.WriteString(greenStyle.Render("└" + strings.Repeat("─", d.width) + "┘"))

	return b.String()
}
