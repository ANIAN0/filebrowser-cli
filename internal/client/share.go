package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
)

// Share represents a FileBrowser share.
type Share struct {
	Hash     string `json:"hash"`
	Path     string `json:"path"`
	Expire   int64  `json:"expire"`
	UserID   int    `json:"userID"`
	Username string `json:"username"`
	Token    string `json:"token,omitempty"`
}

// ShareClient handles share operations.
type ShareClient struct {
	C *httpclient.Client
}

// createShareRequest is the request body for creating a share.
type createShareRequest struct {
	Expires  string `json:"expires,omitempty"`
	Unit     string `json:"unit,omitempty"`
	Password string `json:"password,omitempty"`
}

// Create creates a new share for the given path.
func (s *ShareClient) Create(ctx context.Context, path, expires, unit, password string) (*Share, error) {
	body, err := json.Marshal(createShareRequest{
		Expires:  expires,
		Unit:     unit,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := s.C.Post(ctx, "/api/share"+path, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("create failed: HTTP %d", resp.StatusCode)
	}

	var sh Share
	if err := json.NewDecoder(resp.Body).Decode(&sh); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &sh, nil
}

// List returns all shares.
func (s *ShareClient) List(ctx context.Context) ([]Share, error) {
	resp, err := s.C.Get(ctx, "/api/shares")
	if err != nil {
		return nil, fmt.Errorf("list request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list failed: HTTP %d", resp.StatusCode)
	}

	var shares []Share
	if err := json.NewDecoder(resp.Body).Decode(&shares); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return shares, nil
}

// Delete removes a share by hash.
func (s *ShareClient) Delete(ctx context.Context, hash string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", s.C.BaseURL+"/api/share/"+hash, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := s.C.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Info returns shares for a specific path.
func (s *ShareClient) Info(ctx context.Context, path string) ([]Share, error) {
	resp, err := s.C.Get(ctx, "/api/share"+path)
	if err != nil {
		return nil, fmt.Errorf("info request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("info failed: HTTP %d", resp.StatusCode)
	}

	var shares []Share
	if err := json.NewDecoder(resp.Body).Decode(&shares); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return shares, nil
}