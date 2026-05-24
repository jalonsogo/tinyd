package dmr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/docker/go-units"
	"tinyd/internal/types"
)

// apiModel is a tolerant view of whatever GET /models returns. DMR's payload
// is not strictly documented, so every field is optional and we accept both
// the native shape (top-level array of model objects) and the OpenAI-style
// `{ "object": "list", "data": [...] }` envelope.
type apiModel struct {
	ID         string   `json:"id,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Files      []string `json:"files,omitempty"`
	Format     string   `json:"format,omitempty"`
	Created    int64    `json:"created,omitempty"`
	Size       int64    `json:"size,omitempty"`
	Config     struct {
		Format        string `json:"format,omitempty"`
		Quantization  string `json:"quantization,omitempty"`
		Parameters    string `json:"parameters,omitempty"`
		ParameterSize string `json:"parameter_size,omitempty"`
		ContextSize   int    `json:"context_size,omitempty"`
	} `json:"config,omitempty"`
}

type apiModelEnvelope struct {
	Object string     `json:"object,omitempty"`
	Data   []apiModel `json:"data,omitempty"`
}

// FetchModels lists local DMR models.
func (c *Client) FetchModels(ctx context.Context) ([]types.Model, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithTimeout()
		defer cancel()
	}

	body, err := c.get(ctx, "/models")
	if err != nil {
		return nil, err
	}

	// Try envelope first, then bare array.
	var env apiModelEnvelope
	if jerr := json.Unmarshal(body, &env); jerr == nil && len(env.Data) > 0 {
		return mapModels(env.Data), nil
	}
	var raw []apiModel
	if jerr := json.Unmarshal(body, &raw); jerr == nil {
		return mapModels(raw), nil
	}
	return nil, fmt.Errorf("unexpected /models payload: %s", truncate(string(body), 120))
}

// InspectModel returns the raw JSON for a model (repo[:tag]).
func (c *Client) InspectModel(ctx context.Context, ref string) (string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithTimeout()
		defer cancel()
	}
	ns, name := splitRef(ref)
	body, err := c.get(ctx, "/models/"+ns+"/"+name)
	if err != nil {
		return "", err
	}
	// Re-marshal pretty for the inspect view.
	var v interface{}
	if jerr := json.Unmarshal(body, &v); jerr == nil {
		if pretty, perr := json.MarshalIndent(v, "", "  "); perr == nil {
			return string(pretty), nil
		}
	}
	return string(body), nil
}

// DeleteModel removes a local model by ref (repo[:tag]).
func (c *Client) DeleteModel(ctx context.Context, ref string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutMedium)
		defer cancel()
	}
	ns, name := splitRef(ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		c.baseURL+"/models/"+ns+"/"+name, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return wrapErr(err, "delete model")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (%d): %s", resp.StatusCode, truncate(string(body), 200))
	}
	return nil
}

// PullModel pulls a model. The POST body is { "from": "<ref>" } — DMR
// streams progress as newline-delimited JSON; we consume the stream to
// completion and surface any error.
func (c *Client) PullModel(ctx context.Context, ref string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutLong)
		defer cancel()
	}
	payload, _ := json.Marshal(map[string]string{"from": ref})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/models/create", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return wrapErr(err, "pull model")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull failed (%d): %s", resp.StatusCode, truncate(string(body), 200))
	}
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return wrapErr(err, "pull stream")
	}
	return nil
}

// SearchModels searches Docker Hub's ai/ namespace for a term. Uses the
// public Hub v2 search endpoint (no auth required for public listings).
func (c *Client) SearchModels(ctx context.Context, term string, limit int) ([]types.ModelSearchItem, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutMedium)
		defer cancel()
	}
	if limit <= 0 {
		limit = 25
	}

	// Docker Hub search; restrict to the official ai/ namespace.
	q := strings.TrimSpace(term)
	url := fmt.Sprintf("https://hub.docker.com/v2/search/repositories/?query=%s&page_size=%d", urlEscape(q), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, wrapErr(err, "search models")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("hub search failed (%d)", resp.StatusCode)
	}

	var raw struct {
		Results []struct {
			RepoName       string `json:"repo_name"`
			ShortDesc      string `json:"short_description"`
			StarCount      int    `json:"star_count"`
			PullCount      int    `json:"pull_count"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, wrapErr(err, "parse hub search")
	}

	out := make([]types.ModelSearchItem, 0, len(raw.Results))
	for _, r := range raw.Results {
		if !strings.HasPrefix(r.RepoName, "ai/") {
			continue // only AI namespace
		}
		out = append(out, types.ModelSearchItem{
			Name:        r.RepoName,
			Description: r.ShortDesc,
			Stars:       r.StarCount,
			Pulls:       r.PullCount,
		})
	}
	return out, nil
}

// --- helpers ---

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, wrapErr(err, "GET "+path)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET %s failed (%d): %s", path, resp.StatusCode, truncate(string(body), 200))
	}
	return io.ReadAll(resp.Body)
}

func mapModels(in []apiModel) []types.Model {
	out := make([]types.Model, 0, len(in))
	for _, m := range in {
		repo, tag := "<none>", "<none>"
		if len(m.Tags) > 0 {
			repo, tag = splitRef(m.Tags[0])
		}
		size := "--"
		if m.Size > 0 {
			size = units.HumanSize(float64(m.Size))
		}
		format := m.Config.Format
		if format == "" {
			format = m.Format
		}
		params := m.Config.ParameterSize
		if params == "" {
			params = m.Config.Parameters
		}
		created := ""
		if m.Created > 0 {
			created = time.Unix(m.Created, 0).Format("2006-01-02")
		}
		out = append(out, types.Model{
			ID:         m.ID,
			Repository: repo,
			Tag:        tag,
			Format:     format,
			Quant:      m.Config.Quantization,
			ParamSize:  params,
			Size:       size,
			Created:    created,
		})
	}
	return out
}

// splitRef parses "namespace/name:tag" into ("namespace/name", "tag") for
// the API path, and ("namespace/name", "tag") for display purposes.
func splitRef(ref string) (string, string) {
	parts := strings.SplitN(ref, ":", 2)
	repo := parts[0]
	tag := "latest"
	if len(parts) == 2 {
		tag = parts[1]
	}
	return repo, tag
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func wrapErr(err error, op string) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s timed out", op)
	}
	return fmt.Errorf("%s: %w", op, err)
}

func urlEscape(s string) string {
	// minimal escape — `query=` accepts most of what users will type;
	// avoid pulling net/url for one call.
	var b strings.Builder
	for _, r := range s {
		switch r {
		case ' ':
			b.WriteString("+")
		case '&', '?', '#', '=':
			b.WriteString(fmt.Sprintf("%%%02X", r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
