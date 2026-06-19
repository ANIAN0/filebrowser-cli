package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
)

func setupMockPreview(handler http.HandlerFunc) (*httptest.Server, *httpclient.Client) {
	srv := httptest.NewServer(handler)
	c := httpclient.New(srv.URL)
	return srv, c
}

func TestResourceClient_Preview_Thumb_Success(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	srv, c := setupMockPreview(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/preview/thumb/test.png" && r.Method == "GET" {
			w.Header().Set("Content-Type", "image/png")
			w.Write(pngData)
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	rc := &ResourceClient{C: c}
	data, err := rc.Preview(context.Background(), "/test.png", "thumb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != len(pngData) {
		t.Errorf("expected %d bytes, got %d", len(pngData), len(data))
	}
}

func TestResourceClient_Preview_Big_Success(t *testing.T) {
	jpgData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header
	srv, c := setupMockPreview(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/preview/big/test.jpg" && r.Method == "GET" {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write(jpgData)
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	rc := &ResourceClient{C: c}
	data, err := rc.Preview(context.Background(), "/test.jpg", "big")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != len(jpgData) {
		t.Errorf("expected %d bytes, got %d", len(jpgData), len(data))
	}
}

func TestResourceClient_Preview_InvalidSize(t *testing.T) {
	srv, c := setupMockPreview(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	defer srv.Close()

	rc := &ResourceClient{C: c}
	_, err := rc.Preview(context.Background(), "/test.png", "invalid")
	if err == nil {
		t.Error("expected error for invalid size")
	}
}

func TestResourceClient_Preview_404(t *testing.T) {
	srv, c := setupMockPreview(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	defer srv.Close()

	rc := &ResourceClient{C: c}
	_, err := rc.Preview(context.Background(), "/nonexistent.png", "thumb")
	if err == nil {
		t.Error("expected error on 404")
	}
}

func TestResourceClient_Preview_WriteToFile(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	srv, c := setupMockPreview(func(w http.ResponseWriter, r *http.Request) {
		w.Write(pngData)
	})
	defer srv.Close()

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "preview.png")

	rc := &ResourceClient{C: c}
	data, err := rc.Preview(context.Background(), "/test.png", "thumb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	savedData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	if len(savedData) != len(pngData) {
		t.Errorf("expected %d bytes, got %d", len(pngData), len(savedData))
	}
}
