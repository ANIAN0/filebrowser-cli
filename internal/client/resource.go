package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ANIAN0/filebrowser-cli/internal/errcode"
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
	resp, err := r.C.Get(ctx, buildResourceURL("/api/resources", path, nil))
	if err != nil {
		return nil, fmt.Errorf("list request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &errcode.StatusError{Op: "list", Code: resp.StatusCode, Path: path}
	}

	var res Resource
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &res, nil
}

// Info returns detailed information about a resource.
func (r *ResourceClient) Info(ctx context.Context, path string) (*Resource, error) {
	q := url.Values{}
	q.Set("checksum", "sha256")
	resp, err := r.C.Get(ctx, buildResourceURL("/api/resources", path, q))
	if err != nil {
		return nil, fmt.Errorf("info request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &errcode.StatusError{Op: "info", Code: resp.StatusCode, Path: path}
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
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer f.Close()

	q := url.Values{}
	if override {
		q.Set("override", "true")
	} else {
		q.Set("override", "false")
	}

	u := buildResourceURL("/api/resources", remotePath, q)
	req, err := http.NewRequestWithContext(ctx, "POST", r.C.BaseURL+u, f)
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
		return &errcode.StatusError{Op: "upload", Code: resp.StatusCode, Path: remotePath}
	}
	return nil
}

// Download downloads a remote file to a local path.
func (r *ResourceClient) Download(ctx context.Context, remotePath, localPath string) error {
	if localPath == "" {
		// Use POSIX path.Base (not filepath.Base) since remotePath is always
		// a POSIX-style remote path; on Windows filepath.Base misinterprets it.
		localPath = path.Base(remotePath)
	}

	resp, err := r.C.Get(ctx, buildResourceURL("/api/raw", remotePath, nil))
	if err != nil {
		return fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &errcode.StatusError{Op: "download", Code: resp.StatusCode, Path: remotePath}
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
	// The FileBrowser server requires mkdir URLs to end in "/"; force it here
	// so callers don't have to remember.
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	resp, err := r.C.Post(ctx, buildResourceURL("/api/resources", path, nil), "", nil)
	if err != nil {
		return fmt.Errorf("mkdir request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return &errcode.StatusError{Op: "mkdir", Code: resp.StatusCode, Path: path}
	}
	return nil
}

// Remove deletes a resource at the remote path.
func (r *ResourceClient) Remove(ctx context.Context, path string) error {
	u := buildResourceURL("/api/resources", path, nil)
	req, err := http.NewRequestWithContext(ctx, "DELETE", r.C.BaseURL+u, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := r.C.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("remove request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return &errcode.StatusError{Op: "remove", Code: resp.StatusCode, Path: path}
	}
	return nil
}

// Move moves or renames a resource.
func (r *ResourceClient) Move(ctx context.Context, src, dst string) error {
	q := url.Values{}
	q.Set("action", "rename")
	q.Set("destination", dst)
	u := buildResourceURL("/api/resources", src, q)
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
		return &errcode.StatusError{Op: "move", Code: resp.StatusCode, Path: src}
	}
	return nil
}

// Copy copies a resource.
func (r *ResourceClient) Copy(ctx context.Context, src, dst string) error {
	q := url.Values{}
	q.Set("action", "copy")
	q.Set("destination", dst)
	u := buildResourceURL("/api/resources", src, q)
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
		return &errcode.StatusError{Op: "copy", Code: resp.StatusCode, Path: src}
	}
	return nil
}

// Preview returns an image preview of a resource.
// size must be "thumb" (256x256) or "big" (1080x1080).
func (r *ResourceClient) Preview(ctx context.Context, path, size string) ([]byte, error) {
	if size != "thumb" && size != "big" {
		return nil, fmt.Errorf("size must be 'thumb' or 'big', got %q", size)
	}

	resp, err := r.C.Get(ctx, buildResourceURL("/api/preview/"+size, path, nil))
	if err != nil {
		return nil, fmt.Errorf("preview request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &errcode.StatusError{Op: "preview", Code: resp.StatusCode, Path: path}
	}

	return io.ReadAll(resp.Body)
}
