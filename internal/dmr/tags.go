package dmr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/docker/go-units"
	"tinyd/internal/types"
)

// FetchModelTags lists the tags available for a Hub repo (e.g. "ai/qwen3").
// Uses Docker Hub's v2 API directly (no auth — works for public repos);
// returns up to 100 tags. Parameters and quantization are parsed from each
// tag name when it follows the ai/ convention.
func (c *Client) FetchModelTags(ctx context.Context, repo string) ([]types.ModelTagInfo, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = c.WithCustomTimeout(TimeoutMedium)
		defer cancel()
	}
	// Strip a registry prefix like "docker.io/" if present — Hub's v2 API
	// expects bare "namespace/name".
	if i := strings.Index(repo, "/"); i > 0 && strings.Contains(repo[:i], ".") {
		repo = repo[i+1:]
	}

	endpoint := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/tags/?page_size=100", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, wrapErr(err, "fetch tags")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tag fetch failed (%d): %s", resp.StatusCode, truncate(string(body), 200))
	}

	var env struct {
		Results []struct {
			Name        string `json:"name"`
			FullSize    int64  `json:"full_size"`
			LastUpdated string `json:"last_updated"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, wrapErr(err, "parse tag list")
	}

	out := make([]types.ModelTagInfo, 0, len(env.Results))
	for _, t := range env.Results {
		params, quant := parseTagName(t.Name)
		size := "--"
		if t.FullSize > 0 {
			size = units.HumanSize(float64(t.FullSize))
		}
		updated := ""
		if t.LastUpdated != "" {
			if parsed, perr := time.Parse(time.RFC3339Nano, t.LastUpdated); perr == nil {
				updated = parsed.Format("2006-01-02")
			}
		}
		out = append(out, types.ModelTagInfo{
			Tag:        t.Name,
			Parameters: params,
			Quant:      quant,
			Size:       size,
			Updated:    updated,
		})
	}
	// Order: "latest" first, then by size desc as a rough proxy for "more
	// capable variants near the top". Falls back to alpha when sizes match.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Tag == "latest" {
			return true
		}
		if out[j].Tag == "latest" {
			return false
		}
		return out[i].Tag < out[j].Tag
	})
	return out, nil
}

// parseTagName extracts parameter size and quantization from an ai/-style
// tag like "7b-instruct-q4_K_M". Returns empty strings when fields can't be
// matched — the renderer shows "--" in that case.
var (
	reParams = regexp.MustCompile(`(?i)(?:^|[-_])(\d+(?:\.\d+)?(?:[MmBb]))(?:[-_]|$)`)
	reQuant  = regexp.MustCompile(`(?i)(?:^|[-_])(q\d+(?:_[a-z0-9]+)*|f\d+|bf\d+|int\d+)(?:[-_]|$)`)
)

func parseTagName(tag string) (params, quant string) {
	if m := reParams.FindStringSubmatch(tag); len(m) > 1 {
		params = strings.ToUpper(m[1])
	}
	if m := reQuant.FindStringSubmatch(tag); len(m) > 1 {
		quant = strings.ToLower(m[1])
	}
	return params, quant
}
