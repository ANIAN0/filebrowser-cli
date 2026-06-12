// Package config provides FileBrowser-specific configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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
		Token:   token,
		SavedAt: time.Now(),
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

// LoadToken loads the saved authentication token from file.
func LoadToken() (string, error) {
	path, err := getTokenFilePath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No token file, not an error
		}
		return "", fmt.Errorf("read token file: %w", err)
	}

	var tokenData TokenData
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return "", fmt.Errorf("parse token file: %w", err)
	}

	return tokenData.Token, nil
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
