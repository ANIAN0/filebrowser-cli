package config

import (
	"os"
	"path/filepath"
	"testing"
)

// mockFS implements FS for testing.
type mockFS struct {
	files map[string][]byte
}

func (m *mockFS) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (m *mockFS) Stat(path string) (os.FileInfo, error) {
	if _, ok := m.files[path]; ok {
		return nil, nil // Simplified
	}
	return nil, os.ErrNotExist
}

func TestLoadConfig_ExplicitFlag(t *testing.T) {
	configContent := []byte("version: 1\n")
	fs := &mockFS{
		files: map[string][]byte{
			"/tmp/config.yaml": configContent,
		},
	}

	result, err := LoadConfig("filebrowser-cli", []string{"--config", "/tmp/config.yaml"}, nil, "/fake/binary", fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourcePath != "/tmp/config.yaml" {
		t.Errorf("expected SourcePath /tmp/config.yaml, got %q", result.SourcePath)
	}
	if result.Mode != "explicit" {
		t.Errorf("expected Mode %q, got %q", "explicit", result.Mode)
	}
	if result.Config.Version != 1 {
		t.Errorf("expected Version 1, got %d", result.Config.Version)
	}
}

func TestLoadConfig_ExplicitFlagEquals(t *testing.T) {
	configContent := []byte("version: 1\n")
	fs := &mockFS{
		files: map[string][]byte{
			"/tmp/config.yaml": configContent,
		},
	}

	result, err := LoadConfig("filebrowser-cli", []string{"--config=/tmp/config.yaml"}, nil, "/fake/binary", fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourcePath != "/tmp/config.yaml" {
		t.Errorf("expected SourcePath /tmp/config.yaml, got %q", result.SourcePath)
	}
}

func TestLoadConfig_EnvVar(t *testing.T) {
	configContent := []byte("version: 1\n")
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, configContent, 0644)

	env := map[string]string{
		"FILEBROWSER_CLI_CONFIG": configPath,
	}

	result, err := LoadConfig("filebrowser-cli", nil, env, "/fake/binary", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Mode != "env" {
		t.Errorf("expected Mode %q, got %q", "env", result.Mode)
	}
}

func TestLoadConfig_ProjectLocal(t *testing.T) {
	configContent := []byte("version: 1\n")
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "myproject", "bin")
	os.MkdirAll(binDir, 0755)
	configPath := filepath.Join(binDir, "config.yaml")
	os.WriteFile(configPath, configContent, 0644)

	// Use a path that won't match any user bin paths
	binaryPath := filepath.Join(binDir, "filebrowser-cli")

	// Ensure we're in project mode by using a non-standard path
	origGOPATH := os.Getenv("GOPATH")
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("GOPATH", "/nonexistent")
	os.Setenv("HOME", "/nonexistent")
	// Use a path that definitely won't match any user bin paths
	os.Setenv("USERPROFILE", filepath.Join(tmpDir, "fakehome"))
	defer func() {
		os.Setenv("GOPATH", origGOPATH)
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	// Debug: print env vars
	t.Logf("GOPATH: %s", os.Getenv("GOPATH"))
	t.Logf("HOME: %s", os.Getenv("HOME"))
	t.Logf("USERPROFILE: %s", os.Getenv("USERPROFILE"))
	t.Logf("binaryPath: %s", binaryPath)
	t.Logf("configPath: %s", configPath)

	result, err := LoadConfig("filebrowser-cli", nil, nil, binaryPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("result.Mode: %s", result.Mode)
	t.Logf("result.SourcePath: %s", result.SourcePath)
	// Mode should be "project" since we found config in project-local path
	if result.Mode != "project" {
		t.Errorf("expected Mode %q, got %q", "project", result.Mode)
	}
	if result.SourcePath != configPath {
		t.Errorf("expected SourcePath %q, got %q", configPath, result.SourcePath)
	}
}

func TestLoadConfig_ProjectLocalParent(t *testing.T) {
	configContent := []byte("version: 1\n")
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	os.MkdirAll(binDir, 0755)

	// Config is in parent directory
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, configContent, 0644)

	origGOPATH := os.Getenv("GOPATH")
	os.Setenv("GOPATH", "/nonexistent")
	defer os.Setenv("GOPATH", origGOPATH)

	result, err := LoadConfig("filebrowser-cli", nil, nil, filepath.Join(binDir, "filebrowser-cli"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourcePath != configPath {
		t.Errorf("expected SourcePath %q, got %q", configPath, result.SourcePath)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	fs := &mockFS{files: map[string][]byte{}}

	origGOPATH := os.Getenv("GOPATH")
	os.Setenv("GOPATH", "/nonexistent")
	defer os.Setenv("GOPATH", origGOPATH)

	_, err := LoadConfig("filebrowser-cli", nil, nil, "/nonexistent/binary", fs)
	if err == nil {
		t.Error("expected error when no config found")
	}
}

func TestLoadConfig_VersionMismatch(t *testing.T) {
	configContent := []byte("version: 2\n")
	fs := &mockFS{
		files: map[string][]byte{
			"/tmp/config.yaml": configContent,
		},
	}

	_, err := LoadConfig("filebrowser-cli", []string{"--config", "/tmp/config.yaml"}, nil, "/fake/binary", fs)
	if err == nil {
		t.Error("expected error for version mismatch")
	}
}

func TestLoadConfig_DefaultVersion(t *testing.T) {
	// Config without version field should default to 1
	configContent := []byte("instance_url: http://localhost:8080\n")
	fs := &mockFS{
		files: map[string][]byte{
			"/tmp/config.yaml": configContent,
		},
	}

	result, err := LoadConfig("filebrowser-cli", []string{"--config", "/tmp/config.yaml"}, nil, "/fake/binary", fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Config.Version != 1 {
		t.Errorf("expected default Version 1, got %d", result.Config.Version)
	}
}

func TestLoadConfig_EnvInterpolation(t *testing.T) {
	configContent := []byte("version: 1\npassword: ${TEST_PASSWORD}\n")
	fs := &mockFS{
		files: map[string][]byte{
			"/tmp/config.yaml": configContent,
		},
	}

	// Set env var
	os.Setenv("TEST_PASSWORD", "secret123")
	defer os.Unsetenv("TEST_PASSWORD")

	result, err := LoadConfig("filebrowser-cli", []string{"--config", "/tmp/config.yaml"}, nil, "/fake/binary", fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The config is parsed as a generic Config, so we can't check the password field
	// But the interpolation should have succeeded
	if result.Config == nil {
		t.Error("expected non-nil Config")
	}
}

func TestLoadConfig_NeverReadsSkillconfigJson(t *testing.T) {
	// This is a behavioral test - we verify that skillconfig.json is never in the candidates
	// The actual verification is that loader.go doesn't reference skillconfig.json
	// This test just ensures the loader works without it
	fs := &mockFS{files: map[string][]byte{}}

	_, err := LoadConfig("filebrowser-cli", nil, nil, "/fake/binary", fs)
	if err == nil {
		t.Error("expected error when no config found")
	}
	// Error should not mention skillconfig.json
	if err != nil && contains(err.Error(), "skillconfig") {
		t.Errorf("loader should not reference skillconfig.json, got error: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}