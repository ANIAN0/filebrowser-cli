package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	fbconfig "github.com/ANIAN0/filebrowser-cli/internal/config"
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
