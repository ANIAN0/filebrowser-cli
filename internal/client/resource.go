package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
)

// Resource represents a FileBrowser file or directory.
type Resource struct {
	Path      string     `json:"path"`
	Name      string     `json:"name"`
	IsDir     bool       `json:"isDir"`
	Size      int64      `json:"size"`
	Modified  string     `json:"modified"`
	Created   string     `json:"created"`
	Type      string     `json:"type"`
	Extension string     `json:"extension"`
	Items     []Resource `json:"items"`
	NumDirs   int        `json:"numDirs"`
	NumFiles  int        `json:"numFiles"`
	Content   string     `json:"content,omitempty"`
	Checksum  string     `json:"checksum,omitempty"`
}

// ResourceClient handles resource operations.
type ResourceClient struct {
	C *httpclient.Client
}

// List returns the contents of a directory.
func (r *ResourceClient) List(ctx context.Context, path string) (*Resource, error) {
	if path == "" {
		path = "/"
	}
	path = normalizeRemotePath(path)
	resp, err := r.C.Get(ctx, "/api/resources"+path)
	if err != nil {
		return nil, fmt.Errorf("list request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list failed: HTTP %d", resp.StatusCode)
	}

	var res Resource
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &res, nil
}

// Info returns detailed information about a resource.
func (r *ResourceClient) Info(ctx context.Context, path string) (*Resource, error) {
	if path == "" {
		path = "/"
	}
	path = normalizeRemotePath(path)
	resp, err := r.C.Get(ctx, "/api/resources"+path+"?checksum=sha256")
	if err != nil {
		return nil, fmt.Errorf("info request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("info failed: HTTP %d", resp.StatusCode)
	}

	var res Resource
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &res, nil
}

// Upload uploads a local file to the remote path.
func (r *ResourceClient) Upload(ctx context.Context, localPath, remotePath string, override bool) error {
	// localPath is a filesystem path and MUST NOT be normalized.
	remotePath = normalizeRemotePath(remotePath)
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer f.Close()

	overrideStr := "false"
	if override {
		overrideStr = "true"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.C.BaseURL+"/api/resources"+remotePath+"?override="+overrideStr, f)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := r.C.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("upload failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Download downloads a remote file to a local path.
func (r *ResourceClient) Download(ctx context.Context, remotePath, localPath string) error {
	// localPath is a filesystem path and MUST NOT be normalized.
	remotePath = normalizeRemotePath(remotePath)
	if localPath == "" {
		localPath = filepath.Base(remotePath)
	}

	resp, err := r.C.Get(ctx, "/api/raw"+remotePath)
	if err != nil {
		return fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	f, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// Mkdir creates a directory at the remote path.
func (r *ResourceClient) Mkdir(ctx context.Context, path string) error {
	path = normalizeRemotePath(path)
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	resp, err := r.C.Post(ctx, "/api/resources"+path, "", nil)
	if err != nil {
		return fmt.Errorf("mkdir request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("mkdir failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Remove deletes a resource at the remote path.
func (r *ResourceClient) Remove(ctx context.Context, path string) error {
	path = normalizeRemotePath(path)
	req, err := http.NewRequestWithContext(ctx, "DELETE", r.C.BaseURL+"/api/resources"+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := r.C.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("remove request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remove failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Move moves or renames a resource.
func (r *ResourceClient) Move(ctx context.Context, src, dst string) error {
	src = normalizeRemotePath(src)
	dst = normalizeRemotePath(dst)
	u := fmt.Sprintf("/api/resources%s?action=rename&destination=%s", src, url.QueryEscape(dst))
	req, err := http.NewRequestWithContext(ctx, "PATCH", r.C.BaseURL+u, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := r.C.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("move request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("move failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Copy copies a resource.
func (r *ResourceClient) Copy(ctx context.Context, src, dst string) error {
	src = normalizeRemotePath(src)
	dst = normalizeRemotePath(dst)
	u := fmt.Sprintf("/api/resources%s?action=copy&destination=%s", src, url.QueryEscape(dst))
	req, err := http.NewRequestWithContext(ctx, "PATCH", r.C.BaseURL+u, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := r.C.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("copy request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("copy failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Preview returns an image preview of a resource.
// size must be "thumb" (256x256) or "big" (1080x1080).
func (r *ResourceClient) Preview(ctx context.Context, path, size string) ([]byte, error) {
	path = normalizeRemotePath(path)
	if size != "thumb" && size != "big" {
		return nil, fmt.Errorf("size must be 'thumb' or 'big', got %q", size)
	}

	resp, err := r.C.Get(ctx, fmt.Sprintf("/api/preview/%s%s", size, path))
	if err != nil {
		return nil, fmt.Errorf("preview request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("preview failed: HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
