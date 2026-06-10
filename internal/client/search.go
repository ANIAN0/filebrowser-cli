package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
)

// SearchResult represents a single search result.
type SearchResult struct {
	Dir  bool   `json:"dir"`
	Path string `json:"path"`
}

// SearchClient handles search operations.
type SearchClient struct {
	C *httpclient.Client
}

// Search searches for files matching the query.
// path is the directory to search in, query is the search term.
// limit is the maximum number of results (0 = no limit).
func (s *SearchClient) Search(ctx context.Context, path, query string, limit int) ([]SearchResult, error) {
	if path == "" {
		path = "/"
	}

	u := fmt.Sprintf("/api/search%s?query=%s", path, url.QueryEscape(query))
	resp, err := s.C.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search failed: HTTP %d", resp.StatusCode)
	}

	// Parse SSE stream
	maps, err := httpclient.ParseSSEStream(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse SSE: %w", err)
	}

	// Convert to SearchResult
	results := make([]SearchResult, 0, len(maps))
	for _, m := range maps {
		dir, _ := m["dir"].(bool)
		path, _ := m["path"].(string)
		results = append(results, SearchResult{
			Dir:  dir,
			Path: path,
		})
	}

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}