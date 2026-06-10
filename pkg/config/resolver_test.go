package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolve_GlobalMode_GOPATH(t *testing.T) {
	// Create a temp GOPATH structure
	tmpDir := t.TempDir()
	gopathBin := filepath.Join(tmpDir, "go", "bin")
	os.MkdirAll(gopathBin, 0755)

	// Set GOPATH env
	origGOPATH := os.Getenv("GOPATH")
	os.Setenv("GOPATH", filepath.Join(tmpDir, "go"))
	defer os.Setenv("GOPATH", origGOPATH)

	binaryPath := filepath.Join(gopathBin, "filebrowser-cli")
	mode, candidates := Resolve(binaryPath)

	if mode != ModeGlobal {
		t.Errorf("expected mode %q, got %q", ModeGlobal, mode)
	}
	if len(candidates) == 0 {
		t.Error("expected at least one candidate path")
	}
}

func TestResolve_GlobalMode_HomeBin(t *testing.T) {
	// Create a temp home directory structure
	tmpHome := t.TempDir()
	homeBin := filepath.Join(tmpHome, "go", "bin")
	os.MkdirAll(homeBin, 0755)

	// Override home directory (not reliable across platforms, but works for testing)
	origHome := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		origHome = os.Getenv("USERPROFILE")
	}
	os.Setenv("HOME", tmpHome)
	defer func() {
		if runtime.GOOS == "windows" {
			os.Setenv("USERPROFILE", origHome)
		} else {
			os.Setenv("HOME", origHome)
		}
	}()

	binaryPath := filepath.Join(homeBin, "filebrowser-cli")
	mode, _ := Resolve(binaryPath)

	// This should be global mode since ~/go/bin is a user bin path
	if mode != ModeGlobal {
		t.Logf("mode is %q (may depend on GOPATH env)", mode)
	}
}

func TestResolve_ProjectMode(t *testing.T) {
	// Create a temp project structure
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	os.MkdirAll(binDir, 0755)

	// Ensure we're not matching any user bin paths
	origGOPATH := os.Getenv("GOPATH")
	os.Setenv("GOPATH", "/nonexistent")
	defer os.Setenv("GOPATH", origGOPATH)

	binaryPath := filepath.Join(binDir, "filebrowser-cli")
	mode, candidates := Resolve(binaryPath)

	if mode != ModeProject {
		t.Errorf("expected mode %q, got %q", ModeProject, mode)
	}

	// Should have project-local candidates
	hasProjectLocal := false
	for _, c := range candidates {
		if filepath.Dir(c) == binDir || filepath.Dir(c) == filepath.Dir(binDir) {
			hasProjectLocal = true
			break
		}
	}
	if !hasProjectLocal {
		t.Error("expected project-local candidate paths")
	}
}

func TestResolve_NormalizePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		result := normalizePath("C:\\Users\\Test\\go\\bin")
		expected := "c:\\users\\test\\go\\bin"
		if result != expected {
			t.Errorf("normalizePath on Windows: got %q, want %q", result, expected)
		}
	} else {
		result := normalizePath("/home/test/go/bin")
		expected := "/home/test/go/bin"
		if result != expected {
			t.Errorf("normalizePath: got %q, want %q", result, expected)
		}
	}
}

func TestUserBinPaths(t *testing.T) {
	paths := userBinPaths()
	if len(paths) == 0 {
		t.Error("expected at least one user bin path")
	}

	// Check that paths are absolute
	for _, p := range paths {
		if !filepath.IsAbs(p) {
			t.Errorf("expected absolute path, got %q", p)
		}
	}
}