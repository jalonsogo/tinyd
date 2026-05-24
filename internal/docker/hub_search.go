package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"tinyd/internal/types"
)

// hubV3Result mirrors the subset of the Docker Hub v3 catalog-search
// payload we care about. The endpoint returns much more (categories,
// publisher info, dates) — we ignore those fields.
type hubV3Result struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	ShortDescription string `json:"short_description"`
	StarCount        int    `json:"star_count"`
	Publisher        struct {
		Name     string `json:"name"`
		Verified bool   `json:"verified"`
		Badge    string `json:"badge"` // "official" for Docker Official Images
	} `json:"publisher"`
	OperatingSystems []struct {
		Name  string `json:"name"`  // "linux"
		Label string `json:"label"` // "Linux"
	} `json:"operating_systems"`
	Architectures []struct {
		Name  string `json:"name"`  // "amd64"
		Label string `json:"label"` // "x86-64"
	} `json:"architectures"`
}

type hubV3Envelope struct {
	Total   int           `json:"total"`
	Results []hubV3Result `json:"results"`
}

// searchHubV3 queries https://hub.docker.com/api/search/v3/catalog/search,
// the same endpoint Hub's web UI uses. Returns an error if anything
// short of a 200 response with parseable JSON comes back, so the caller
// can fall back to the daemon's `/images/search`.
func searchHubV3(ctx context.Context, term string, limit int) ([]types.ImageSearchItem, error) {
	q := url.Values{}
	q.Set("query", strings.TrimSpace(term))
	q.Set("type", "image")
	q.Set("size", fmt.Sprintf("%d", limit))
	q.Set("from", "0")

	endpoint := "https://hub.docker.com/api/search/v3/catalog/search?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Search-Version", "v3")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hub v3 search returned %d", resp.StatusCode)
	}

	var env hubV3Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, err
	}

	items := make([]types.ImageSearchItem, 0, len(env.Results))
	for _, r := range env.Results {
		archs := make([]string, 0, len(r.Architectures))
		for _, a := range r.Architectures {
			archs = append(archs, a.Name)
		}
		oss := make([]string, 0, len(r.OperatingSystems))
		for _, o := range r.OperatingSystems {
			oss = append(oss, o.Name)
		}
		items = append(items, types.ImageSearchItem{
			Name:          coalesce(r.Slug, r.Name),
			Description:   r.ShortDescription,
			Stars:         r.StarCount,
			Official:      r.Publisher.Badge == "official",
			Architectures: archs,
			OperatingSys:  oss,
		})
	}
	return items, nil
}

func coalesce(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
