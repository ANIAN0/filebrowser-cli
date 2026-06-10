package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New("http://localhost:8080")
	if c.BaseURL != "http://localhost:8080" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, "http://localhost:8080")
	}
	if c.HTTP == nil {
		t.Error("HTTP client should not be nil")
	}
	if c.MaxRetries != DefaultMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", c.MaxRetries, DefaultMaxRetries)
	}
	if c.AuthHeader != "X-Auth" {
		t.Errorf("AuthHeader = %q, want %q", c.AuthHeader, "X-Auth")
	}
}

func TestNewWithOptions(t *testing.T) {
	c := New("http://localhost:8080",
		WithToken("test-token"),
		WithVerbose(true),
		WithMaxRetries(5),
		WithAuthHeader("Authorization"),
	)
	if c.Token != "test-token" {
		t.Errorf("Token = %q, want %q", c.Token, "test-token")
	}
	if !c.Verbose {
		t.Error("Verbose should be true")
	}
	if c.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", c.MaxRetries)
	}
	if c.AuthHeader != "Authorization" {
		t.Errorf("AuthHeader = %q, want %q", c.AuthHeader, "Authorization")
	}
}

func TestGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		if r.Header.Get("X-Auth") != "test-token" {
			t.Errorf("X-Auth = %q, want %q", r.Header.Get("X-Auth"), "test-token")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	c := New(server.URL, WithToken("test-token"))
	resp, err := c.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestGet_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	c := New(server.URL)
	resp, err := c.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestGet_Retry5xx(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt <= 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	c := New(server.URL, WithMaxRetries(3))
	resp, err := c.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if atomic.LoadInt32(&attempts) != 4 {
		t.Errorf("attempts = %d, want 4", atomic.LoadInt32(&attempts))
	}
}

func TestGet_Retry5xx_Exhausted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := New(server.URL, WithMaxRetries(2))
	_, err := c.Get(context.Background(), "/test")
	if err == nil {
		t.Error("expected error after exhausting retries")
	}
}

func TestPost_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want %q", r.Header.Get("Content-Type"), "application/json")
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	c := New(server.URL)
	resp, err := c.Post(context.Background(), "/test", "application/json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestContext_Cancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	c := New(server.URL)
	_, err := c.Get(ctx, "/test")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}