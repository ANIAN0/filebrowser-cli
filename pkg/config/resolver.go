package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolvedMode indicates where the binary was installed.
const (
	// ModeGlobal means the binary is in a user bin directory (e.g., ~/go/bin).
	// Config is loaded from user directory (~/.config/<cli>/config.yaml).
	ModeGlobal = "global"

	// ModeProject means the binary is in a project directory.
	// Config is loaded from project-local first, then user directory as fallback.
	ModeProject = "project"
)

// userBinPaths returns the set of directories considered "user bin" (global install).
// These directories indicate the binary was installed via `make install`.
func userBinPaths() []string {
	var paths []string

	// GOPATH/bin - explicit GOPATH env
	if gp := os.Getenv("GOPATH"); gp != "" {
		paths = append(paths, filepath.Join(gp, "bin"))
	}

	// Default GOPATH locations when GOPATH env is unset
	home := ""
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	} else {
		home = os.Getenv("HOME")
	}

	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}

	if home != "" {
		paths = append(paths, filepath.Join(home, "go", "bin"))     // Default GOPATH
		paths = append(paths, filepath.Join(home, ".local", "bin")) // XDG user bin
		paths = append(paths, filepath.Join(home, "bin"))           // Common user bin
	}

	return paths
}

// Resolve detects the install mode and returns candidate config paths.
//
// binaryPath is typically from os.Executable(). The function resolves symlinks
// before checking against user bin paths.
//
// Returns:
//   - mode: ModeGlobal or ModeProject
//   - candidates: ordered list of config file paths to try
func Resolve(cliName, binaryPath string) (mode string, candidates []string) {
	abs, err := filepath.Abs(binaryPath)
	if err != nil {
		abs = binaryPath
	}

	// Resolve symlinks to get the real path
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}

	binDir := filepath.Dir(abs)
	normalizedBinDir := normalizePath(binDir)

	// Check if binary is in a user bin directory
	for _, ub := range userBinPaths() {
		if normalizePath(ub) == normalizedBinDir {
			mode = ModeGlobal
			candidates = []string{userConfigPath(cliName)}
			return
		}
	}

	// Not in user bin - assume project-local install
	mode = ModeProject

	// Project-local candidates: next to binary, then parent
	candidates = append(candidates, filepath.Join(binDir, "config.yaml"))
	candidates = append(candidates, filepath.Join(binDir, "..", "config.yaml"))

	// Fallback to user dir
	candidates = append(candidates, userConfigPath(cliName))

	return
}

// userConfigPath returns the per-CLI user config file path.
func userConfigPath(cliName string) string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, cliName, "config.yaml")
}

// normalizePath normalizes a path for comparison.
// On Windows, converts to lowercase for case-insensitive comparison.
func normalizePath(p string) string {
	cleaned := filepath.Clean(p)
	if runtime.GOOS == "windows" {
		return strings.ToLower(cleaned)
	}
	return cleaned
}
