package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestDo_401_RefreshRetriesOnce covers the regression target: when the server
// returns 401, OnAuthFailure is invoked, the new token is installed, and the
// request is retried exactly once — and that retry carries the new token in
// the auth header.
func TestDo_401_RefreshRetriesOnce(t *testing.T) {
	var (
		hits     int32
		gotToken string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		gotToken = r.Header.Get("X-Auth")
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	c := New(server.URL,
		WithToken("old-token"),
		WithAuthFailure(func() (string, error) {
			return "new-token", nil
		}),
	)

	resp, err := c.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("hits = %d, want 2 (one 401 + one retry)", got)
	}
	if gotToken != "new-token" {
		t.Errorf("auth header on retry = %q, want %q", gotToken, "new-token")
	}
	if c.Token != "new-token" {
		t.Errorf("c.Token after refresh = %q, want %q", c.Token, "new-token")
	}
}

// TestDo_401_RefreshFailureSurfacesError: when OnAuthFailure itself fails,
// Do returns a wrapped error and stops — it does NOT silently retry with
// whatever stale token was on hand.
func TestDo_401_RefreshFailureSurfacesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	refreshErr := errors.New("creds missing")
	c := New(server.URL,
		WithToken("any"),
		WithAuthFailure(func() (string, error) {
			return "", refreshErr
		}),
	)

	_, err := c.Get(context.Background(), "/test")
	if err == nil {
		t.Fatal("want error when auth refresh fails, got nil")
	}
	if !errors.Is(err, refreshErr) {
		t.Errorf("err = %v, want wraps %v", err, refreshErr)
	}
	if !strings.Contains(err.Error(), "auth refresh") {
		t.Errorf("err = %q, want prefix \"auth refresh\"", err.Error())
	}
}

// TestDo_401_RefreshOnlyOnce: even if the second attempt also returns 401,
// OnAuthFailure is invoked at most once per Do() call. This caps the cost
// of a misconfigured refresh hook.
func TestDo_401_RefreshOnlyOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	var refreshHits int32
	c := New(server.URL,
		WithToken("any"),
		WithAuthFailure(func() (string, error) {
			atomic.AddInt32(&refreshHits, 1)
			return "still-rejected", nil
		}),
	)

	resp, err := c.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want 401 (final response after one refresh)", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&refreshHits); got != 1 {
		t.Errorf("refresh hits = %d, want 1 (capped)", got)
	}
}

// TestDo_401_NoCallback_Leaves401Untouched pins backward compatibility: when
// OnAuthFailure is nil, 401 is returned to the caller exactly as before the
// new feature was added.
func TestDo_401_NoCallback_Leaves401Untouched(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	c := New(server.URL, WithToken("any"))
	resp, err := c.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want 401 (no callback = no auto-retry)", resp.StatusCode)
	}
}

// TestDo_401_RefreshSkipsBackoff: the auth refresh path must NOT wait through
// the exponential backoff the 5xx retry path uses. Token rejection -> refresh
// -> retry should feel instant, not delayed.
func TestDo_401_RefreshSkipsBackoff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	c := New(server.URL,
		WithToken("any"),
		WithAuthFailure(func() (string, error) {
			return "new", nil
		}),
		WithMaxRetries(3), // would force 2s + 4s + 8s backoff if triggered
	)

	start := time.Now()
	resp, err := c.Get(context.Background(), "/test")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	// After refresh + retry, server still 401s -> Do returns. Total wall
	// time should be well under the 2s backoff threshold for attempt=1.
	if elapsed > 500*time.Millisecond {
		t.Errorf("auth-refresh path took %v, want < 500ms (no backoff expected)", elapsed)
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
