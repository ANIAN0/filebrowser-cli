// Package client provides FileBrowser API clients.
package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
)

// AuthClient handles authentication operations.
type AuthClient struct {
	C *httpclient.Client
}

// LoginRequest is the request body for login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// User represents a FileBrowser user.
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
}

// Login authenticates with FileBrowser and returns the token.
// The token is also stored on a.C.Token for subsequent requests.
func (a *AuthClient) Login(ctx context.Context, username, password string) (string, error) {
	body, err := json.Marshal(LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("marshal login request: %w", err)
	}

	resp, err := a.C.Post(ctx, "/api/login", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed: HTTP %d", resp.StatusCode)
	}

	tokBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read login response: %w", err)
	}

	token := strings.TrimSpace(string(tokBytes))
	a.C.Token = token // Cache in memory
	return token, nil
}

// Renew refreshes the current token. Requires a valid token to be set.
func (a *AuthClient) Renew(ctx context.Context) (string, error) {
	resp, err := a.C.Post(ctx, "/api/renew", "", nil)
	if err != nil {
		return "", fmt.Errorf("renew request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("renew failed: HTTP %d (token may be expired, re-run login)", resp.StatusCode)
	}

	tokBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read renew response: %w", err)
	}

	token := strings.TrimSpace(string(tokBytes))
	a.C.Token = token
	return token, nil
}

// Whoami returns the current username.
// FileBrowser doesn't have a dedicated endpoint, so we parse the JWT token.
func (a *AuthClient) Whoami(ctx context.Context) (string, error) {
	if a.C.Token == "" {
		return "", fmt.Errorf("not logged in")
	}

	// Parse JWT token to get username
	// JWT format: header.payload.signature
	// We need to decode the payload (base64url)
	parts := strings.Split(a.C.Token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode token payload: %w", err)
	}

	// Parse JSON
	var claims struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("parse token claims: %w", err)
	}

	if claims.User.Username == "" {
		return "", fmt.Errorf("username not found in token")
	}

	return claims.User.Username, nil
}