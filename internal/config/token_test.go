package config

import (
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// fakeJWT builds a minimal JWT-like string (header.payload.signature) with
// the given claims. Signature is a placeholder — we never verify it.
func fakeJWT(claims map[string]any) string {
	header, _ := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	sig := base64.RawURLEncoding.EncodeToString([]byte("signature"))
	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "." + sig
}

func TestGetTokenFilePathUsesUserConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	} else {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	}

	path, err := getTokenFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(userConfigRootForTest(tmpDir), "filebrowser-cli", tokenFileName)
	if path != want {
		t.Fatalf("token path = %q, want %q", path, want)
	}
}

func userConfigRootForTest(tmpDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(tmpDir, "AppData", "Roaming")
	}
	return filepath.Join(tmpDir, "config")
}

// --- jwtExpiry tests ---

func TestJwtExpiry_ValidExp(t *testing.T) {
	exp := time.Now().Add(2 * time.Hour).Unix()
	token := fakeJWT(map[string]any{"exp": exp})
	got := jwtExpiry(token)
	want := time.Unix(exp, 0)
	if !got.Equal(want) {
		t.Errorf("jwtExpiry = %v, want %v", got, want)
	}
}

func TestJwtExpiry_NoExpClaim(t *testing.T) {
	token := fakeJWT(map[string]any{"sub": "admin"})
	if got := jwtExpiry(token); !got.IsZero() {
		t.Errorf("jwtExpiry with no exp = %v, want zero", got)
	}
}

func TestJwtExpiry_MalformedToken(t *testing.T) {
	cases := []string{
		"",                          // empty
		"not-a-jwt",                 // no dots
		"a.b",                       // only two parts
		"invalid.invalid.invalid",   // non-base64
		"eyJh.bWFsZm9ybWVk.invalid", // valid header, invalid payload
	}
	for _, token := range cases {
		if got := jwtExpiry(token); !got.IsZero() {
			t.Errorf("jwtExpiry(%q) = %v, want zero", token, got)
		}
	}
}

// --- isExpired tests ---

func TestIsExpired_NotExpired(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour).Unix()
	token := fakeJWT(map[string]any{"exp": exp})
	if isExpired(token, time.Time{}, time.Time{}) {
		t.Error("token with future exp should not be expired")
	}
}

func TestIsExpired_JustExpired(t *testing.T) {
	// Token expired 60 seconds ago — well past the 30s skew.
	exp := time.Now().Add(-60 * time.Second).Unix()
	token := fakeJWT(map[string]any{"exp": exp})
	if !isExpired(token, time.Time{}, time.Time{}) {
		t.Error("token expired 60s ago should be expired")
	}
}

func TestIsExpired_WithinSkew(t *testing.T) {
	// Token expires 20 seconds from now — within the 30s skew window, should
	// still be considered expired (we subtract skew from expiry).
	exp := time.Now().Add(20 * time.Second).Unix()
	token := fakeJWT(map[string]any{"exp": exp})
	if !isExpired(token, time.Time{}, time.Time{}) {
		t.Error("token expiring within skew should be considered expired")
	}
}

func TestIsExpired_NoExp_NoPersisted_NoSavedAt(t *testing.T) {
	// Token has no JWT exp, no persisted expiry, no SavedAt — fall through
	// to the "treat as never expiring" branch (kept as the last resort so
	// tokens saved by older tool versions that recorded neither field still
	// work).
	if isExpired("no-exp", time.Time{}, time.Time{}) {
		t.Error("token with no expiry info and no SavedAt should not be expired")
	}
}

func TestIsExpired_FallsBackToPersisted(t *testing.T) {
	// Token has no JWT exp, but persisted ExpiresAt is in the past.
	pastExpiry := time.Now().Add(-1 * time.Hour)
	if !isExpired("no-jwt-exp", pastExpiry, time.Time{}) {
		t.Error("should fall back to persisted ExpiresAt and report expired")
	}
}

func TestIsExpired_PersistedFuture(t *testing.T) {
	futureExpiry := time.Now().Add(1 * time.Hour)
	if isExpired("no-jwt-exp", futureExpiry, time.Time{}) {
		t.Error("persisted future expiry should not be expired")
	}
}

// TestIsExpired_NoExp_OpaqueWithinCap is the regression target for the
// 401 bug: an opaque token (no JWT exp claim) saved recently should still
// be trusted, but one saved longer than maxNoExpLifetime ago must be
// considered expired so we re-login before the server 401s us.
func TestIsExpired_NoExp_OpaqueWithinCap(t *testing.T) {
	savedAt := time.Now().Add(-1 * time.Hour)
	if isExpired("opaque-token", time.Time{}, savedAt) {
		t.Error("opaque token saved 1h ago should still be within maxNoExpLifetime")
	}
}

func TestIsExpired_NoExp_OpaquePastCap(t *testing.T) {
	savedAt := time.Now().Add(-25 * time.Hour)
	if !isExpired("opaque-token", time.Time{}, savedAt) {
		t.Error("opaque token saved 25h ago should be expired (past maxNoExpLifetime)")
	}
}

func TestIsExpired_JwtExpWinsOverSavedAt(t *testing.T) {
	// When the JWT carries an exp, that takes precedence over SavedAt even
	// if SavedAt alone would say "still fresh" — JWT is authoritative.
	exp := time.Now().Add(2 * time.Hour).Unix()
	token := fakeJWT(map[string]any{"exp": exp})
	savedAt := time.Now().Add(-48 * time.Hour) // way past the cap
	if isExpired(token, time.Time{}, savedAt) {
		t.Error("JWT exp should win; future exp means token is not expired regardless of SavedAt")
	}
}

// --- SaveToken / LoadToken integration tests ---

func TestSaveToken_WritesExpiresAt(t *testing.T) {
	tmpDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	} else {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	}

	exp := time.Now().Add(2 * time.Hour).Unix()
	token := fakeJWT(map[string]any{"exp": exp})

	if err := SaveToken(token); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	// Read back and check ExpiresAt was populated.
	data, err := loadTokenData()
	if err != nil {
		t.Fatalf("loadTokenData: %v", err)
	}
	want := time.Unix(exp, 0)
	if !data.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", data.ExpiresAt, want)
	}
}

func TestLoadToken_ValidToken(t *testing.T) {
	tmpDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	} else {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	}

	exp := time.Now().Add(2 * time.Hour).Unix()
	token := fakeJWT(map[string]any{"exp": exp})
	SaveToken(token)

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if got != token {
		t.Errorf("LoadToken = %q, want %q", got, token)
	}
}

func TestLoadToken_ExpiredToken(t *testing.T) {
	tmpDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	} else {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	}

	exp := time.Now().Add(-2 * time.Hour).Unix()
	token := fakeJWT(map[string]any{"exp": exp})
	SaveToken(token)

	_, err := LoadToken()
	if err != ErrTokenExpired {
		t.Errorf("LoadToken err = %v, want ErrTokenExpired", err)
	}
}

func TestLoadToken_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	} else {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	}

	token, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken no file: %v", err)
	}
	if token != "" {
		t.Errorf("LoadToken no file = %q, want empty", token)
	}
}

func TestLoadToken_NoExp_NeverExpires(t *testing.T) {
	tmpDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	} else {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	}

	// Token with no exp claim and no persisted ExpiresAt.
	token := fakeJWT(map[string]any{"sub": "admin"})
	SaveToken(token)

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken no exp: %v", err)
	}
	if got != token {
		t.Errorf("LoadToken no exp = %q, want %q", got, token)
	}
}

func TestLoadRawToken_DoesNotCheckExpiry(t *testing.T) {
	tmpDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	} else {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	}

	exp := time.Now().Add(-2 * time.Hour).Unix()
	token := fakeJWT(map[string]any{"exp": exp})
	SaveToken(token)

	got, err := LoadRawToken()
	if err != nil {
		t.Fatalf("LoadRawToken: %v", err)
	}
	if got != token {
		t.Errorf("LoadRawToken = %q, want %q", got, token)
	}
}
