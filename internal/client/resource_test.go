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

func setupMockResource(handler http.HandlerFunc) (*httptest.Server, *httpclient.Client) {
	srv := httptest.NewServer(handler)
	c := httpclient.New(srv.URL)
	return srv, c
}

func TestResourceClient_List_Success(t *testing.T) {
	srv, c := setupMockResource(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/resources/" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"path":"/","name":"/","isDir":true,"items":[{"name":"test.txt","isDir":false}]}`))
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	rc := &ResourceClient{C: c}
	res, err := rc.List(context.Background(), "/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsDir {
		t.Error("expected IsDir=true")
	}
	if len(res.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(res.Items))
	}
}

func TestResourceClient_Info_Success(t *testing.T) {
	srv, c := setupMockResource(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/resources/test.txt" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"path":"/test.txt","name":"test.txt","isDir":false,"size":100}`))
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	rc := &ResourceClient{C: c}
	res, err := rc.Info(context.Background(), "/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsDir {
		t.Error("expected IsDir=false")
	}
	if res.Size != 100 {
		t.Errorf("expected Size=100, got %d", res.Size)
	}
}

func TestResourceClient_Mkdir_Success(t *testing.T) {
	srv, c := setupMockResource(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/resources/newdir/" && r.Method == "POST" {
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	rc := &ResourceClient{C: c}
	err := rc.Mkdir(context.Background(), "/newdir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResourceClient_Remove_Success(t *testing.T) {
	srv, c := setupMockResource(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/resources/test.txt" && r.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	rc := &ResourceClient{C: c}
	err := rc.Remove(context.Background(), "/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResourceClient_Move_Success(t *testing.T) {
	srv, c := setupMockResource(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/resources/old.txt" && r.Method == "PATCH" {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	rc := &ResourceClient{C: c}
	err := rc.Move(context.Background(), "/old.txt", "/new.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResourceClient_Copy_Success(t *testing.T) {
	srv, c := setupMockResource(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/resources/source.txt" && r.Method == "PATCH" {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	rc := &ResourceClient{C: c}
	err := rc.Copy(context.Background(), "/source.txt", "/copy.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResourceClient_Upload_Success(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "upload.txt")
	os.WriteFile(localFile, []byte("test content"), 0644)

	srv, c := setupMockResource(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/resources/upload.txt" && r.Method == "POST" {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	rc := &ResourceClient{C: c}
	err := rc.Upload(context.Background(), localFile, "/upload.txt", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResourceClient_Download_Success(t *testing.T) {
	srv, c := setupMockResource(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/raw/test.txt" && r.Method == "GET" {
			w.Write([]byte("file content"))
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "download.txt")

	rc := &ResourceClient{C: c}
	err := rc.Download(context.Background(), "/test.txt", localFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(localFile)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != "file content" {
		t.Errorf("unexpected content: %s", string(data))
	}
}