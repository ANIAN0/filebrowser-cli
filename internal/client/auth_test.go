package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
)

func setupMockFB(handler http.HandlerFunc) (*httptest.Server, *httpclient.Client) {
	srv := httptest.NewServer(handler)
	c := httpclient.New(srv.URL)
	return srv, c
}

func TestAuthClient_Login_Success(t *testing.T) {
	srv, c := setupMockFB(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/login" && r.Method == "POST" {
			w.WriteHeader(200)
			w.Write([]byte("test-token-abc"))
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	a := &AuthClient{C: c}
	tok, err := a.Login(context.Background(), "admin", "password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "test-token-abc" {
		t.Errorf("token = %q, want %q", tok, "test-token-abc")
	}
	if c.Token != "test-token-abc" {
		t.Error("token not cached on client")
	}
}

func TestAuthClient_Login_401(t *testing.T) {
	srv, c := setupMockFB(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	})
	defer srv.Close()

	a := &AuthClient{C: c}
	_, err := a.Login(context.Background(), "admin", "wrong")
	if err == nil {
		t.Error("expected error on 401")
	}
}

func TestAuthClient_Login_5xx(t *testing.T) {
	srv, c := setupMockFB(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	defer srv.Close()

	a := &AuthClient{C: c}
	_, err := a.Login(context.Background(), "admin", "password")
	if err == nil {
		t.Error("expected error on 5xx")
	}
}

func TestAuthClient_Renew_Success(t *testing.T) {
	srv, c := setupMockFB(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/renew" && r.Method == "POST" {
			w.WriteHeader(200)
			w.Write([]byte("new-token-xyz"))
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	c.Token = "old-token"
	a := &AuthClient{C: c}
	tok, err := a.Renew(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "new-token-xyz" {
		t.Errorf("token = %q, want %q", tok, "new-token-xyz")
	}
	if c.Token != "new-token-xyz" {
		t.Error("token not updated on client")
	}
}

func TestAuthClient_Renew_401(t *testing.T) {
	srv, c := setupMockFB(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	})
	defer srv.Close()

	c.Token = "expired-token"
	a := &AuthClient{C: c}
	_, err := a.Renew(context.Background())
	if err == nil {
		t.Error("expected error on 401")
	}
}

func TestAuthClient_Whoami_Success(t *testing.T) {
	srv, c := setupMockFB(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/users" && r.Method == "GET" {
			w.WriteHeader(200)
			w.Write([]byte(`[{"id":1,"username":"admin"}]`))
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	a := &AuthClient{C: c}
	username, err := a.Whoami(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "admin" {
		t.Errorf("username = %q, want %q", username, "admin")
	}
}

func TestAuthClient_Whoami_NoUsers(t *testing.T) {
	srv, c := setupMockFB(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
	})
	defer srv.Close()

	a := &AuthClient{C: c}
	_, err := a.Whoami(context.Background())
	if err == nil {
		t.Error("expected error when no users")
	}
}