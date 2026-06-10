package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
)

func setupMockSearch(handler http.HandlerFunc) (*httptest.Server, *httpclient.Client) {
	srv := httptest.NewServer(handler)
	c := httpclient.New(srv.URL)
	return srv, c
}

func TestSearchClient_Search_Success(t *testing.T) {
	sseData := `data: {"dir":false,"path":"/file1.txt"}

data: {"dir":true,"path":"/dir1"}

data: {"dir":false,"path":"/file2.txt"}

`
	srv, c := setupMockSearch(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/search/" && r.Method == "GET" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Write([]byte(sseData))
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	sc := &SearchClient{C: c}
	results, err := sc.Search(context.Background(), "/", "test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Path != "/file1.txt" {
		t.Errorf("results[0].Path = %q, want %q", results[0].Path, "/file1.txt")
	}
	if results[0].Dir {
		t.Error("results[0].Dir should be false")
	}
	if !results[1].Dir {
		t.Error("results[1].Dir should be true")
	}
}

func TestSearchClient_Search_Empty(t *testing.T) {
	srv, c := setupMockSearch(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(""))
	})
	defer srv.Close()

	sc := &SearchClient{C: c}
	results, err := sc.Search(context.Background(), "/", "nonexistent", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchClient_Search_401(t *testing.T) {
	srv, c := setupMockSearch(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	})
	defer srv.Close()

	sc := &SearchClient{C: c}
	_, err := sc.Search(context.Background(), "/", "test", 0)
	if err == nil {
		t.Error("expected error on 401")
	}
}

func TestSearchClient_Search_Limit(t *testing.T) {
	sseData := `data: {"dir":false,"path":"/file1.txt"}

data: {"dir":false,"path":"/file2.txt"}

data: {"dir":false,"path":"/file3.txt"}

data: {"dir":false,"path":"/file4.txt"}

data: {"dir":false,"path":"/file5.txt"}

`
	srv, c := setupMockSearch(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseData))
	})
	defer srv.Close()

	sc := &SearchClient{C: c}
	results, err := sc.Search(context.Background(), "/", "test", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (limited), got %d", len(results))
	}
}

func TestSearchClient_Search_Subpath(t *testing.T) {
	sseData := `data: {"dir":false,"path":"/subdir/file.txt"}

`
	srv, c := setupMockSearch(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/search/subdir" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Write([]byte(sseData))
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	sc := &SearchClient{C: c}
	results, err := sc.Search(context.Background(), "/subdir", "file", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != "/subdir/file.txt" {
		t.Errorf("results[0].Path = %q, want %q", results[0].Path, "/subdir/file.txt")
	}
}