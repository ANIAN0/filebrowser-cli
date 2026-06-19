// Package config provides FileBrowser-specific configuration.
package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrTokenExpired is returned by LoadToken when the saved token has expired.
// Callers can branch on this to trigger a re-login flow.
var ErrTokenExpired = errors.New("saved token has expired")

// tokenExpirySkew is how far before the real expiry we treat a token as
// expired, to avoid races where the token ages out between the check and the
// first authenticated request.
const tokenExpirySkew = 30 * time.Second

// TokenData represents the saved authentication token.
type TokenData struct {
	// Token is the JWT token.
	Token string `json:"token"`

	// SavedAt is when the token was saved.
	SavedAt time.Time `json:"saved_at"`

	// ExpiresAt is when the token expires (optional, from JWT).
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// tokenFileName is the name of the token file.
const tokenFileName = "token.json"

// getTokenFilePath returns the path to the token file.
// It uses the same directory as the config file.
func getTokenFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", fmt.Errorf("get user config dir: %w", homeErr)
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "filebrowser-cli", tokenFileName), nil
}

// SaveToken saves the authentication token to a file.
func SaveToken(token string) error {
	path, err := getTokenFilePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create token dir: %w", err)
	}

	data := TokenData{
		Token:     token,
		SavedAt:   time.Now(),
		ExpiresAt: jwtExpiry(token),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}

	return nil
}

// LoadToken loads the saved authentication token from file and reports an
// error if it has expired.
//
// It returns:
//   - ("", nil) if no token file exists (caller should prompt the user to log in)
//   - (token, nil) if a non-expired token is on disk
//   - ("", ErrTokenExpired) if the token has expired (caller may auto re-login)
//
// Expiry is decided from the JWT's own "exp" claim when present; the persisted
// ExpiresAt is used as a fallback for tokens whose JWT has no "exp" (treated as
// never expiring if both are zero).
func LoadToken() (string, error) {
	tokenData, err := loadTokenData()
	if err != nil {
		return "", err
	}
	if tokenData == nil {
		return "", nil // No token file
	}

	if isExpired(tokenData.Token, tokenData.ExpiresAt) {
		return "", ErrTokenExpired
	}
	return tokenData.Token, nil
}

// LoadRawToken loads the saved token WITHOUT an expiry check. It is intended
// for commands like `renew`, whose semantics are "ask the server to refresh
// whatever token is on disk" — the server is the authority on validity.
func LoadRawToken() (string, error) {
	tokenData, err := loadTokenData()
	if err != nil {
		return "", err
	}
	if tokenData == nil {
		return "", nil
	}
	return tokenData.Token, nil
}

// loadTokenData reads and parses the token file. It returns (nil, nil) when
// the file does not exist (distinct from a read/parse error).
func loadTokenData() (*TokenData, error) {
	path, err := getTokenFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No token file, not an error
		}
		return nil, fmt.Errorf("read token file: %w", err)
	}

	var tokenData TokenData
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return nil, fmt.Errorf("parse token file: %w", err)
	}

	return &tokenData, nil
}

// isExpired reports whether the token should be considered expired.
//
// The JWT's live "exp" claim takes precedence over the persisted ExpiresAt
// (the claim is authoritative; the persisted copy may be stale). A token with
// no determinable expiry (both zero) is treated as never expiring, preserving
// backward compatibility with tokens saved before expiry tracking existed.
func isExpired(token string, persistedExpiry time.Time) bool {
	if exp := jwtExpiry(token); !exp.IsZero() {
		return time.Now().After(exp.Add(-tokenExpirySkew))
	}
	if !persistedExpiry.IsZero() {
		return time.Now().After(persistedExpiry.Add(-tokenExpirySkew))
	}
	return false
}

// jwtExpiry extracts the "exp" (Unix seconds) claim from a JWT's payload.
//
// It performs no signature verification (this tool is not the token issuer).
// It returns the zero time if the token is malformed, has no payload, or
// carries no "exp" claim — in all those cases callers conservatively treat the
// token as not-expired.
func jwtExpiry(token string) time.Time {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}
	}
	if claims.Exp == 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0)
}

// DeleteToken removes the saved authentication token.
func DeleteToken() error {
	path, err := getTokenFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("delete token file: %w", err)
	}

	return nil
}
