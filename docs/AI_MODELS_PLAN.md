# AI Models support — design plan

Adding a **Models** tab to tinyd backed by [Docker Model Runner](https://docs.docker.com/ai/model-runner/) (DMR). This doc maps DMR concepts onto existing tinyd patterns so the implementation reuses what's already proven (table component, action bar, pull-search flow, action spinner) instead of inventing parallel UI.

## What DMR gives us

- **CLI**: `docker model list/ls`, `pull`, `push`, `rm`, `run`, `inspect`, `status`, `ps`, `tag`, `configure`, `logs`, `unload`, `df`, `version`.
- **HTTP API** on `localhost:12434` (Docker Desktop also exposes it inside containers as `http://model-runner.docker.internal`):
  - `GET    /models` — list local models
  - `GET    /models/{namespace}/{name}` — inspect a model
  - `POST   /models/create` — pull a model
  - `DELETE /models/{namespace}/{name}` — remove a model
- **Inference proxy** (OpenAI / Anthropic / Ollama compatible) on the same port — not relevant for the management TUI, but needed if we ever embed a chat panel.
- **Registries**: Docker Hub (the `ai/` namespace), arbitrary OCI registries, Hugging Face.

## Mapping DMR concepts onto tinyd patterns

| DMR concept | tinyd analogue | Reuse |
|---|---|---|
| Model | Image | New `types.Model{}` next to `types.Image` |
| Running (loaded) model | Running container | Status dot — green `●` loaded, gray `○` available, yellow `●` pulling |
| `docker model pull` | `docker pull` | The existing pull-search flow (`internal/ui/update.go` `handlePullViewKeys`) is a direct fit |
| `docker model rm` | `docker rmi` | Same delete-confirm overlay |
| `docker model inspect` | `docker inspect` | Same `colorizeJSON` viewer |
| `docker model run` | `docker exec -it` | Same `tea.ExecProcess` pattern from `execContainerCmd` |
| `docker model unload` | `docker stop` | New action key, same `actionInProgress` + spinner flow |
| `docker model df` | (none) | Could surface in a footer or status bar later — not MVP |

## Architecture

Add `internal/dmr/` mirroring `internal/docker/`:

```
internal/dmr/
  client.go       // net/http.Client w/ TimeoutQuick/Medium/Long constants
  models.go       // FetchModels, PullModel, DeleteModel, InspectModel, SearchModels, UnloadModel
```

Rationale for HTTP API over shelling out to `docker model …`:

1. Matches the existing `internal/docker` shape — single `Client` struct, context-bound timeouts, typed return values.
2. Structured responses are easier to render in a table than parsing CLI output.
3. The CLI is still required for **`docker model run`** (interactive chat) — same exception we make for `docker exec`. `tea.ExecProcess` handles the screen takeover.

New types in `internal/types/types.go`:

```go
type Model struct {
    ID          string   // sha or content addressable id
    Repository  string   // e.g. "ai/qwen2.5-coder"
    Tag         string   // e.g. "7b-instruct"
    Format      string   // gguf / safetensors
    Quant       string   // Q4_K_M, F16, ...
    ParamSize   string   // "7B", "1.5B"
    Size        string   // disk footprint
    Loaded      bool     // currently in memory (from /ps)
    Created     string
}

type ModelListMsg []Model
type ModelSearchItem struct{ Name, Description string; Pulls, Stars int }
type ModelSearchMsg  []ModelSearchItem
```

## UI surface

A new tab between Images and Volumes (or appended after Networks — TBD by taste; appending is less disruptive to existing muscle memory).

**Columns:**

```
●  REPOSITORY:TAG               PARAMS  QUANT   SIZE    STATUS
●  ai/qwen2.5-coder:7b          7B      Q4_K_M  4.2GB   loaded
○  ai/llama3.2:3b               3B      Q5_K_M  2.1GB   available
○  ai/smollm2:1.7b              1.7B    F16     3.4GB   available
```

Status dot reuses `getStatusDot`/`getInUseDot`. "loaded" maps to RUNNING-style green; "available" to STOPPED-style gray.

**Action bar (Models tab):**

| Key | Action | Implementation reuse |
|---|---|---|
| `R` | Run / open chat REPL | `execContainerCmd` pattern — `tea.ExecProcess(exec.Command("docker","model","run", repo+":"+tag))` |
| `U` | Unload | Same as start/stop: set `actionInProgress`, `actionLabel = "Unloading X"`, `actionTargetID`, dispatch cmd; spinner shows on row |
| `P` | Pull model | Reuse the existing pull-search view. Swap `SearchImages` for `SearchModels`. The view layer (`renderPullView`) doesn't need to change — same input → results → confirm flow |
| `I` | Inspect | `inspectImageCmd` clone; route `InspectMsg` through `colorizeJSON` (already wired) |
| `D` | Delete | Reuse `deleteConfirmMode` overlay |

**Pull search source.** Two options:

1. Use the same `docker search` endpoint, filtered to `ai/` namespace prefix.
2. Hit Docker Hub's REST API directly (`https://hub.docker.com/v2/search/repositories/?query=…&namespace=ai`).

Option 1 is consistent with how images are searched today; option 2 gives better fields (pulls, stars, last update) but introduces an external HTTP dep. Recommend **option 1** for MVP, option 2 if filtering quality is poor.

## Phasing

**Phase 1 — Read + lifecycle.** New tab, list (`GET /models`), inspect, delete, unload. Covers the diagnostic flow ("what's loaded? clear it"). ~1 day of work if the HTTP shapes are stable.

**Phase 2 — Pull.** Wire `SearchModels` into the existing pull view; `POST /models/create` with progress streamed back as `ActionSuccessMsg` (the same chunked-read pattern as `ImagePull`). The spinner UX is already in place.

**Phase 3 — Run.** `R` opens a chat REPL via `tea.ExecProcess`. Same screen-takeover behavior as `e` (exec) on containers.

**Phase 4 — Polish.** Surface `docker model df` (disk usage footer), `configure --context-size`, log tail. These are nice-to-haves, not blockers.

## Open questions before coding

1. **Detect DMR availability.** `GET /models` returns connection refused if the runner isn't enabled in Docker Desktop. Need a friendly empty state, not an error toast. Probably probe once on startup and grey out the Models tab if absent.
2. **Pull progress.** DMR's `POST /models/create` returns a streaming response similar to image pull. We currently `io.ReadAll` and discard it — fine for MVP, but per-layer progress would be nicer eventually.
3. **Repository:tag parsing.** `ai/qwen2.5-coder:7b-instruct-q4_k_m` is longer than typical image tags; column widths will need an extra few chars vs the Images tab.
4. **Model ID stability.** Use repository:tag as the action-target key (same as we do for volumes by name) rather than digest, since DMR's API may not expose a stable short ID.

## What we should not build

- A built-in chat panel. The `docker model run` REPL already does this well and matches our exec UX. Embedding a chat client means tokenizing streams, scrollback, and prompt history — out of scope for a Docker TUI.
- A separate config/setup wizard. If DMR isn't running, link to the docs from the empty state.
