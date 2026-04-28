package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"tinyd/internal/components"
	"tinyd/internal/ui"
)

// detectDark probes several reliable signals before falling back to lipgloss.
// When OSC-11 is unsupported (common on macOS Terminal.app), lipgloss returns
// true (dark) by default — incorrectly flagging light terminals. We check
// explicit env vars first and default to light when nothing is conclusive.
func detectDark() bool {
	// 1. Explicit user override — always wins.
	switch strings.ToLower(os.Getenv("TINYD_THEME")) {
	case "light":
		return false
	case "dark":
		return true
	}

	// 2. iTerm2 / WezTerm / Ghostty set TERM_BACKGROUND=dark|light.
	switch strings.ToLower(os.Getenv("TERM_BACKGROUND")) {
	case "dark":
		return true
	case "light":
		return false
	}

	// 3. COLORFGBG="fg;bg" — bg index < 8 means dark ANSI color (dark bg).
	if fgbg := os.Getenv("COLORFGBG"); fgbg != "" {
		parts := strings.Split(fgbg, ";")
		if bg, err := strconv.Atoi(strings.TrimSpace(parts[len(parts)-1])); err == nil {
			return bg < 8
		}
	}

	// 4. Use the lipgloss cached OSC-11 result (bubbletea pre-queries this
	//    before acquiring the tty, so it's the most accurate we can get).
	//    If your light terminal is misdetected as dark, set TINYD_THEME=light.
	return lipgloss.HasDarkBackground()
}

func main() {
	dark := detectDark()

	components.InitTheme(dark)
	ui.InitTheme(dark)

	model, err := ui.NewModel()
	if err != nil {
		fmt.Printf("Error initializing: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
