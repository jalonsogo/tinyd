# Changelog

All notable changes to tinyd will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.2.0](https://github.com/jalonsogo/tinyd/compare/v0.1.0...v0.2.0) (2026-05-24)


### Miscellaneous Chores

* cut v0.2.0 ([6d3d3be](https://github.com/jalonsogo/tinyd/commit/6d3d3be0e667dfa65283757d9558e445ef0aba84))

## [Unreleased]

### Added
- **Customizable visible columns** — new `V` key opens a column-picker overlay on top of any tab listing the togglable columns with a `[x]` / `[ ]` checkbox each. ↑/↓ moves the cursor, Space/Enter toggles, V/ESC closes. Changes apply immediately and last for the session.
- **`TINYD_HIDE_COLS` env var** seeds the initial hidden-column set at startup (comma-separated lowercase keys: `status,cpu,mem,ports,image,size,created,params,quant,driver,containers,mountpoint,scope,ipv4`). Default is `status` — the colored dot already encodes that signal so the textual STATUS column is redundant for most users. Set `TINYD_HIDE_COLS=-` (or `none`) to show everything.
- **Copy curl to clipboard** — `Y` on the Models tab yanks the example `curl` command for the selected model to the system clipboard. Uses `pbcopy` (macOS), `wl-copy` / `xclip` / `xsel` (Linux), or `clip.exe` (Windows/WSL).
- **Interactive Run-image modal** — `R` on the Images tab opens a full-screen form for `name`, port mappings (`host:container`), env vars (`KEY=value`), and volumes. Volume entries support three sources via a sub-picker:
  - **Existing volume** picked from the Volumes tab
  - **New named volume** (name + container path)
  - **Bind mount** via a built-in directory browser (↑/↓ navigate, Enter descends, `F` confirms the current directory)
  TAB cycles fields, ENTER commits the current input row to its list, Ctrl+R submits, ESC cancels.
- **Update image to latest** — `U` on the Images tab re-pulls the selected image's `repo:tag` and reports "Updated <ref> to latest". Untagged / dangling images surface a clear "Can't update an untagged image" status instead of failing silently.
- **In-app streaming model chat** — `C` on the Models tab opens a full-screen chat with the selected model via DMR's OpenAI-compatible endpoint (`/engines/llama.cpp/v1/chat/completions`, `stream:true`). Transcript shows `you ▎` / `bot ▎` turns, a spinner indicates generation, PgUp/PgDn scroll, `Ctrl+L` clears history, ESC exits and tears down the stream.
- **Model variant (tag) picker** — pulling a model now detours through an intermediate screen that fetches the tag list from Docker Hub (`/v2/repositories/<repo>/tags/`), parses **Parameters** (e.g. `7B`) and **Quantization** (e.g. `q4_K_M`) from each tag name, and shows a table of `TAG | PARAMS | QUANT | SIZE | UPDATED`. ↑/↓ to navigate, Enter pulls the specific tag.
- **DMR API info panel** — pinned to the bottom of the Models tab, shows the OpenAI-compatible endpoint, the currently-selected model ref, and a copy-pasteable `curl` example.
- **Theme detection + override** — auto-detects light/dark terminal background via `TERM_BACKGROUND`, `COLORFGBG`, and lipgloss OSC-11. Override with `TINYD_THEME=light` or `TINYD_THEME=dark`. Tab labels, table headers, and emphasis text use bold + the terminal's default foreground so they stay readable even when detection guesses wrong.
- **Models tab (Docker Model Runner)** — Fifth tab managing AI models via DMR's HTTP API on `localhost:12434`. List local models, inspect (raw JSON), delete with confirmation, pull from Docker Hub's `ai/` namespace, and chat / REPL with running models. DMR availability is probed on startup; when absent the tab renders a friendly pointer to the docs instead of an error. Configurable via `DMR_BASE_URL`. See `docs/AI_MODELS_PLAN.md` for design notes.
- **Pull from Docker Hub** — `P` on the Images tab to search Docker Hub. Type a query, browse results with `↑/↓`, press Enter to pull, ESC to cancel. The pulling stage shows an animated braille spinner and the image name.
- **In-row action spinner** — while an action runs, the affected row's status dot is replaced with a cyan animated spinner so feedback is visible directly in the list, not just in the action bar.
- **Working help overlay** — `?` (or `Shift+H`) toggles a keybinding panel rendered over the tab content; ESC closes it.
- **Full-screen logs view** — logs now use the entire terminal height instead of fixed 15 lines.
- **Substring search in logs** — `S` in the logs view activates a case-insensitive filter.

### Changed
- **Selection background is now palette-aware violet** — was a saturated `#1D4ED8` blue that fringed against red status text (ERROR / STOPPED). Dark mode uses `#7C3AED` (violet-600) with white text; light mode flips to pale lavender (`#E9D5FF`) with dark-violet text (`#5B21B6`), which sits naturally on a white terminal instead of feeling like a heavy block.
- **Status text on selected rows** drops the per-status color (red / yellow / etc.) and uses `ColorSelectedFg` so the highlight stays clean. The colored status dot still carries the at-a-glance state.
- **Brighter body text in dark mode** — `ColorNormal` bumped `#AAAAAA` → `#DDDDDD` for higher contrast against dark backgrounds. Light mode stays at `#222222`.
- **`docker model run` wrapped in `sh -c`** so a failed launch (model not found, DMR disabled, etc.) pauses for ENTER before tinyd reclaims the alt-screen — previously the error message flashed for a fraction of a second and the user just saw a blink. Also strips the `docker.io/` prefix the CLI doesn't accept.
- **Chat key is `C`, REPL is `R` / `Shift+R`** on the Models tab — the in-app streaming chat lives on `C`; plain `R` (and `Shift+R`) drops to the shell `docker model run` REPL for muscle memory.
- **`R` opens the Run modal on Images** (was `S` "Start"). Shortcut bar shows `Run` with `R` underlined.
- **Pull-image shortcut visually separated** from per-image operations: `R Run  U pdate to latest  I nspect  D elete │ P ull image`.
- **Model delete / inspect URL fixed** — DMR expects `/models/<ns>/<name>:<tag>` (colon preserved), not `/models/<ns>/<name>/<tag>`. Delete and inspect now build the path correctly.
- **Star glyph in pull-search header** changed from `★` to `⭐` for better terminal font coverage.
- **Help-toggle skipped in text-input views** — `H` / `?` no longer trigger the help overlay while typing in chat, the Run modal, the volume picker, or the pull-search input, so capital letters and `?` reach the input field.
- **Action bar always reflects current tab + selection** — recomputes every render.
- **Navigation works during actions** — `↑/↓/j/k`, `←/→`, `1–5`, and `Ctrl+C` are allowed while a Docker call is in flight. Only the action keys are blocked, to prevent queuing a second operation.
- **Stop/restart command timeout bumped to 30 s** — was 10 s, which raced with Docker's own 10 s SIGTERM grace period.

### Removed
- **`c` shortcut for "Open interactive shell with altscreen"** on Containers — never existed in code (the exec shortcut is `e`). `c` is now bound to Chat on Models. Documentation was wrong.
- **`o` shortcut for "Open exposed ports in browser"** — no such handler exists in code; documentation was wrong.
- **`f` filter modal** for Images (All / In Use / Unused / Dangling) and Networks (Active / Unused) — the filter state fields exist on the model but the `f` key is only bound inside the file browser. No filter UI is reachable. Documentation was wrong.

### Fixed
- **Spurious "stop operation timed out after 15s" errors** — Go context (10 s) raced with Docker's hardcoded 10 s SIGTERM grace period. Context is now 30 s; error message no longer hard-codes a misleading duration.
- **Selected-row status dot was invisible** — the table component re-rendered the dot with the selection style, stripping the status color. The dot now composites its status color over the selection background.
- **Inactive tabs showed extra `┴` joins** under the bottom border. Replaced with `─` so the rule reads continuous.
- **Light-terminal contrast** — tab labels, table headers, shortcut keys, the DMR API panel values, and chat/title text now use bold + terminal-default foreground rather than `ColorBright` (which collapses to near-white on a light terminal when the dark palette is mistakenly loaded).
- Delete modal now properly displays in overlay mode.
- Fixed panic when containers have no names (added safety checks).

## [Previous Features]

### Core Features
- **Four tabs**: Containers, Images, Volumes, Networks
- **Container management**: Start, stop, restart, delete containers
- **Interactive console**: Open shell in containers with altscreen preservation
- **Port management**: Open exposed ports in browser with port selector
- **Log viewing**: View last 100 lines of container logs
- **Deep inspection**: View stats, mounts, configuration for all resources
- **Image operations**: Run containers from images, delete images, filter by status
- **Volume tracking**: See which containers use each volume
- **Network inspection**: View network details and connections

### UI/UX
- Fully responsive design adapts to any terminal size
- Minimum width: 60 columns
- Works beautifully in VSCode terminal splits
- Smart status indicators (green/gray/yellow dots)
- Intelligent scrolling for large resource lists
- Clean, minimalist interface with classic terminal aesthetics
- Box-drawing characters for crisp borders

### Navigation
- `↑/↓` or `k/j` - Move selection
- `←/→` or `h/l` - Switch tabs
- `1-4` - Jump directly to tab
- `F1` - Toggle help screen
- `ESC` - Return to list view
- `q` or `Ctrl+C` - Quit

### Actions
- `s/S` - Start/Stop containers (toggle search in logs view)
- `r/R` - Restart containers / Run images
- `c/C` - Open interactive console
- `o/O` - Open port in browser
- `l/L` - View logs
- `i/I` - Inspect resource
- `d/D` - Delete resource
- `f/F` - Open filter modal
- `p/P` - Pull image (images tab only)

---

**Made with ❤️ for terminal lovers everywhere**
