package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
)

func setupMockShare(handler http.HandlerFunc) (*httptest.Server, *httpclient.Client) {
	srv := httptest.NewServer(handler)
	c := httpclient.New(srv.URL)
	return srv, c
}

func TestShareClient_Create_Success(t *testing.T) {
	srv, c := setupMockShare(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/share/test.txt" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"hash":"abc123","path":"/test.txt","expire":1234567890,"userID":1,"username":"admin"}`))
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	sc := &ShareClient{C: c}
	sh, err := sc.Create(context.Background(), "/test.txt", "24", "hours", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sh.Hash != "abc123" {
		t.Errorf("hash = %q, want %q", sh.Hash, "abc123")
	}
}

func TestShareClient_Create_WithPassword(t *testing.T) {
	var receivedBody string
	srv, c := setupMockShare(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"hash":"def456","path":"/secret.txt","expire":0,"userID":1,"username":"admin"}`))
	})
	defer srv.Close()

	sc := &ShareClient{C: c}
	_, err := sc.Create(context.Background(), "/secret.txt", "1", "days", "mypassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(receivedBody, "mypassword") {
		t.Errorf("request body should contain password, got: %s", receivedBody)
	}
}

func TestShareClient_Create_401(t *testing.T) {
	srv, c := setupMockShare(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	})
	defer srv.Close()

	sc := &ShareClient{C: c}
	_, err := sc.Create(context.Background(), "/test.txt", "24", "hours", "")
	if err == nil {
		t.Error("expected error on 401")
	}
}

func TestShareClient_List_Success(t *testing.T) {
	srv, c := setupMockShare(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/shares" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"hash":"abc123","path":"/file1.txt"},{"hash":"def456","path":"/file2.txt"}]`))
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	sc := &ShareClient{C: c}
	shares, err := sc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shares) != 2 {
		t.Errorf("expected 2 shares, got %d", len(shares))
	}
}

func TestShareClient_Delete_Success(t *testing.T) {
	srv, c := setupMockShare(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/share/abc123" && r.Method == "DELETE" {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	sc := &ShareClient{C: c}
	err := sc.Delete(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShareClient_Info_Success(t *testing.T) {
	srv, c := setupMockShare(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/share/test.txt" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"hash":"abc123","path":"/test.txt"}]`))
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	sc := &ShareClient{C: c}
	shares, err := sc.Info(context.Background(), "/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shares) != 1 {
		t.Errorf("expected 1 share, got %d", len(shares))
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
