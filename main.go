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

// Populated at build time via -ldflags by GoReleaser (or whichever tool runs
// the build). Defaults to "dev" for plain `go build` / `go run`.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

// parseHiddenColumns turns TINYD_HIDE_COLS ("status,cpu,mem") into a set
// of lowercased column keys. Unset env (empty input) defaults to hiding
// the STATUS column; "-" or "none" disables all hiding.
func parseHiddenColumns(env string) map[string]bool {
	out := map[string]bool{}
	env = strings.TrimSpace(env)
	if env == "" {
		out["status"] = true
		return out
	}
	if env == "-" || strings.EqualFold(env, "none") {
		return out
	}
	for _, k := range strings.Split(env, ",") {
		k = strings.ToLower(strings.TrimSpace(k))
		if k != "" {
			out[k] = true
		}
	}
	return out
}

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
	// --version / -v short-circuit so users can `tinyd --version` from a
	// release tarball without going through the TUI handshake.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("tinyd %s\n", version)
			if commit != "" {
				fmt.Printf("commit %s\n", commit)
			}
			if date != "" {
				fmt.Printf("built  %s\n", date)
			}
			return
		}
	}

	dark := detectDark()

	components.InitTheme(dark)
	ui.InitTheme(dark)

	// Column visibility from env. TINYD_HIDE_COLS is a comma-separated
	// list of column keys (case-insensitive): status, cpu, mem, ports,
	// image, size, created, params, quant, driver, containers, ipv4,
	// scope. Unset = hide "status" by default (the colored dot already
	// encodes state, so the textual STATUS column is redundant for most
	// users). Set to a single dash (-) to start with everything shown.
	hidden := parseHiddenColumns(os.Getenv("TINYD_HIDE_COLS"))

	model, err := ui.NewModel(version, hidden)
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
