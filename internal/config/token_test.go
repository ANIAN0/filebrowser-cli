package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

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
