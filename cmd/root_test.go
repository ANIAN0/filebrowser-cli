package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ANIAN0/filebrowser-cli/internal/errcode"
	fbconfig "github.com/ANIAN0/filebrowser-cli/internal/config"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

// fakeJWT builds a minimal JWT-like string (header.payload.signature) with
// the given claims. The signature is a placeholder — this tool never verifies
// it, so we can use a stable value.
func fakeJWT(claims map[string]any) string {
	header, _ := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	sig := base64.RawURLEncoding.EncodeToString([]byte("signature"))
	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "." + sig
}

// withTempConfigDir points the token file at a per-test temp directory so
// tests don't touch the real user config.
func withTempConfigDir(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	} else {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	}
}

// writeTokenFile drops a token.json into the resolved user config dir under
// "filebrowser-cli" so LoadToken / LoadRawToken can find it.
func writeTokenFile(t *testing.T, token string, expiresAt time.Time) {
	t.Helper()
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	tokenDir := filepath.Join(dir, "filebrowser-cli")
	if err := os.MkdirAll(tokenDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	data := fbconfig.TokenData{
		Token:     token,
		SavedAt:   time.Now(),
		ExpiresAt: expiresAt,
	}
	js, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tokenDir, "token.json"), js, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// minimalConfig builds a Config sufficient for newAuthedClient; the
// InstanceURL is rewritten per-test to point at a fake server.
func minimalConfig(instanceURL string) *fbconfig.Config {
	return &fbconfig.Config{
		InstanceURL: instanceURL,
	}
}

func TestNewAuthedClient_ValidTokenOnDisk(t *testing.T) {
	withTempConfigDir(t)

	// Future exp — should be picked up directly, no login call.
	token := fakeJWT(map[string]any{"exp": time.Now().Add(1 * time.Hour).Unix()})
	writeTokenFile(t, token, time.Time{})

	// If newAuthedClient were to call /api/login, this server would 500 it.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s — valid token should not require login", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, err := newAuthedClient(context.Background(), minimalConfig(srv.URL))
	if err != nil {
		t.Fatalf("newAuthedClient: %v", err)
	}
	if c.Token != token {
		t.Errorf("client.Token = %q, want %q", c.Token, token)
	}
}

func TestNewAuthedClient_NoTokenFile_NoCreds_Errors(t *testing.T) {
	withTempConfigDir(t)

	// No token file, no creds -> caller-friendly error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL.Path)
	}))
	defer srv.Close()

	cfg := minimalConfig(srv.URL)
	_, err := newAuthedClient(context.Background(), cfg)
	if err == nil {
		t.Fatal("newAuthedClient: want error, got nil")
	}
	if !strings.Contains(err.Error(), "login") {
		t.Errorf("error %q should hint at running `login`", err.Error())
	}
}

func TestNewAuthedClient_ExpiredToken_WithCreds_AutoRelogin(t *testing.T) {
	withTempConfigDir(t)

	// Expired token on disk triggers the auto re-login branch.
	expired := fakeJWT(map[string]any{"exp": time.Now().Add(-1 * time.Hour).Unix()})
	writeTokenFile(t, expired, time.Time{})

	var loginCalled bool
	newToken := fakeJWT(map[string]any{"exp": time.Now().Add(2 * time.Hour).Unix()})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/login" && r.Method == http.MethodPost:
			loginCalled = true
			var body struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode login body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if body.Username != "admin" || body.Password != "secret" {
				t.Errorf("login body = %+v, want admin/secret", body)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(newToken))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	cfg := minimalConfig(srv.URL)
	cfg.Username = "admin"
	cfg.Password = "secret"

	c, err := newAuthedClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("newAuthedClient: %v", err)
	}
	if !loginCalled {
		t.Error("/api/login was not called")
	}
	if c.Token != newToken {
		t.Errorf("client.Token = %q, want refreshed %q", c.Token, newToken)
	}

	// The refreshed token should be persisted so the next call skips the login.
	got, err := fbconfig.LoadToken()
	if err != nil {
		t.Fatalf("LoadToken after relogin: %v", err)
	}
	if got != newToken {
		t.Errorf("persisted token = %q, want %q", got, newToken)
	}
}

func TestNewAuthedClient_ExpiredToken_NoCreds_Errors(t *testing.T) {
	withTempConfigDir(t)

	expired := fakeJWT(map[string]any{"exp": time.Now().Add(-1 * time.Hour).Unix()})
	writeTokenFile(t, expired, time.Time{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL.Path)
	}))
	defer srv.Close()

	_, err := newAuthedClient(context.Background(), minimalConfig(srv.URL))
	if err == nil {
		t.Fatal("want error when expired token and no creds, got nil")
	}
	if !strings.Contains(err.Error(), "login") {
		t.Errorf("error %q should hint at running `login`", err.Error())
	}
}

func TestNewAuthedClient_LoginFailure_PropagatesError(t *testing.T) {
	withTempConfigDir(t)

	expired := fakeJWT(map[string]any{"exp": time.Now().Add(-1 * time.Hour).Unix()})
	writeTokenFile(t, expired, time.Time{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate wrong-password.
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("invalid credentials"))
	}))
	defer srv.Close()

	cfg := minimalConfig(srv.URL)
	cfg.Username = "admin"
	cfg.Password = "wrong"

	_, err := newAuthedClient(context.Background(), cfg)
	if err == nil {
		t.Fatal("want error when login fails, got nil")
	}
	if !strings.Contains(err.Error(), "auto re-login") {
		t.Errorf("error %q should mention auto re-login", err.Error())
	}
}

// TestNewAuthedClient_OnAuthFailureTriggersRelogin is the regression target
// for the 401 bug: even when the on-disk token is "valid" by local checks
// (e.g. no exp claim, within maxNoExpLifetime), the server can still reject
// it. The OnAuthFailure hook installed by newAuthedClient must fire and
// transparently re-login, returning a client whose Token field carries the
// refreshed credential.
func TestNewAuthedClient_OnAuthFailureTriggersRelogin(t *testing.T) {
	withTempConfigDir(t)

	// Local checks consider this token fine: no exp claim, just-saved. But
	// the server will reject it with 401 — exactly the opaque-token case
	// the fix targets.
	opaque := fakeJWT(map[string]any{"sub": "admin"})
	writeTokenFile(t, opaque, time.Time{})

	var (
		loginCalls int32
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/login" && r.Method == http.MethodPost {
			atomic.AddInt32(&loginCalls, 1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`new-refreshed-token`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cfg := minimalConfig(srv.URL)
	cfg.Username = "admin"
	cfg.Password = "secret"

	c, err := newAuthedClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("newAuthedClient: %v", err)
	}
	if c.OnAuthFailure == nil {
		t.Fatal("OnAuthFailure should be wired up by newAuthedClient")
	}
	// newAuthedClient must hand the caller a client whose Token is the
	// persisted (still-locally-valid) credential — the 401 rejection is
	// the server's signal, not a pre-emptive local decision.
	if c.Token != opaque {
		t.Errorf("client.Token before refresh = %q, want %q (opaque)", c.Token, opaque)
	}

	// Trigger the hook the way httpclient.Do would after a 401.
	newTok, aerr := c.OnAuthFailure()
	if aerr != nil {
		t.Fatalf("OnAuthFailure: %v", aerr)
	}
	if newTok != "new-refreshed-token" {
		t.Errorf("refreshed token = %q, want %q", newTok, "new-refreshed-token")
	}
	if got := atomic.LoadInt32(&loginCalls); got != 1 {
		t.Errorf("login calls = %d, want 1", got)
	}
}

// TestNewAuthedClient_OnAuthFailureNoCreds_ErrorsFriendly covers the second
// half of the recovery path: when the on-disk token is rejected AND the
// config has no credentials for re-login, OnAuthFailure must return an error
// that points the user at `filebrowser-cli login`.
func TestNewAuthedClient_OnAuthFailureNoCreds_ErrorsFriendly(t *testing.T) {
	withTempConfigDir(t)

	opaque := fakeJWT(map[string]any{"sub": "admin"})
	writeTokenFile(t, opaque, time.Time{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c, err := newAuthedClient(context.Background(), minimalConfig(srv.URL))
	if err != nil {
		t.Fatalf("newAuthedClient: %v", err)
	}

	_, aerr := c.OnAuthFailure()
	if aerr == nil {
		t.Fatal("OnAuthFailure with no creds: want error, got nil")
	}
	if !strings.Contains(aerr.Error(), "login") {
		t.Errorf("error %q should hint at running `login`", aerr.Error())
	}
}

// classifyExitCode must route by sentinel / StatusError, NEVER by string match.
// These tests pin that contract so message rewording (i18n, wording tweaks)
// cannot silently change exit codes.

func TestClassifyExitCode_StatusError5xx(t *testing.T) {
	err := &errcode.StatusError{Op: "list", Code: 500}
	if got := classifyExitCode(err); got != output.ExitServerError {
		t.Errorf("classifyExitCode(500) = %d, want ExitServerError=%d", got, output.ExitServerError)
	}
}

func TestClassifyExitCode_StatusError4xx(t *testing.T) {
	err := &errcode.StatusError{Op: "list", Code: 404}
	if got := classifyExitCode(err); got != output.ExitClientError {
		t.Errorf("classifyExitCode(404) = %d, want ExitClientError=%d", got, output.ExitClientError)
	}
}

func TestClassifyExitCode_ConfigSentinel(t *testing.T) {
	err := fmt.Errorf("load config: %w", errcode.ErrConfigLoad)
	if got := classifyExitCode(err); got != output.ExitConfig {
		t.Errorf("classifyExitCode(ErrConfigLoad) = %d, want ExitConfig=%d", got, output.ExitConfig)
	}
}

func TestClassifyExitCode_NetworkError(t *testing.T) {
	err := &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	if got := classifyExitCode(err); got != output.ExitNetwork {
		t.Errorf("classifyExitCode(net.OpError) = %d, want ExitNetwork=%d", got, output.ExitNetwork)
	}
}

// Regression: changing the error message text must NOT change the exit code.
// This is the whole point of moving away from strings.Contains.
func TestClassifyExitCode_MessageChangePreservesCode(t *testing.T) {
	// Original-style message: "list failed: HTTP 404"  → ExitClientError
	orig := &errcode.StatusError{Op: "list", Code: 404}
	// Hypothetical i18n rewrite
	i18n := &errcode.StatusError{Op: "列表", Code: 404}
	// Wrapped differently
	wrapped := fmt.Errorf("执行失败: %w", orig)

	for label, e := range map[string]error{"orig": orig, "i18n": i18n, "wrapped": wrapped} {
		if got := classifyExitCode(e); got != output.ExitClientError {
			t.Errorf("[%s] classifyExitCode = %d, want ExitClientError=%d", label, got, output.ExitClientError)
		}
	}
}
