package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// FS is the minimal interface needed for testing file system operations.
type FS interface {
	ReadFile(path string) ([]byte, error)
	Stat(path string) (os.FileInfo, error)
}

// osFS implements FS using the real file system.
type osFS struct{}

func (osFS) ReadFile(p string) ([]byte, error) { return os.ReadFile(p) }
func (osFS) Stat(p string) (os.FileInfo, error) { return os.Stat(p) }

// LoadResult contains the result of loading a config file.
type LoadResult struct {
	// Config is the parsed configuration.
	Config *Config

	// SourcePath is the path to the config file that was loaded.
	SourcePath string

	// Mode is "explicit", "env", "global", or "project".
	Mode string
}

// LoadConfig loads config for the given CLI with 4-layer priority:
// 1) --config flag value (if set in args)
// 2) <CLI>_CONFIG env var
// 3) project-local config (mode B): <binary-dir>/config.yaml or <binary-dir>/../config.yaml
// 4) user dir config (mode A or fallback): ~/.config/<cli>/config.yaml
//
// The function NEVER reads skillconfig.json (DECIDE-011).
//
// Parameters:
//   - cliName: CLI name (e.g., "filebrowser-cli", "memos-cli")
//   - args: command line arguments (to extract --config flag)
//   - env: environment variables (map takes precedence over os env)
//   - binaryPath: path to the binary (from os.Executable())
//   - fs: file system interface (nil uses real fs)
func LoadConfig(cliName string, args []string, env map[string]string, binaryPath string, fs FS) (*LoadResult, error) {
	if fs == nil {
		fs = osFS{}
	}

	getenv := func(k string) string {
		if v, ok := env[k]; ok {
			return v
		}
		return os.Getenv(k)
	}

	// 1) --config flag
	explicit := extractFlagValue(args, "--config")
	if explicit != "" {
		data, err := fs.ReadFile(explicit)
		if err != nil {
			return nil, fmt.Errorf("read --config %q: %w", explicit, err)
		}
		return parseConfig(data, explicit, "explicit")
	}

	// 2) <CLI>_CONFIG env
	envKey := cliNameToEnvKey(cliName)
	if envPath := getenv(envKey); envPath != "" {
		data, err := fs.ReadFile(envPath)
		if err != nil {
			return nil, fmt.Errorf("read %s %q: %w", envKey, envPath, err)
		}
		return parseConfig(data, envPath, "env")
	}

	// 3) + 4) Resolver
	mode, candidates := Resolve(binaryPath)
	for _, p := range candidates {
		// In global mode, only try user config paths
		if mode == ModeGlobal && !isUserConfigPath(p) {
			continue
		}

		// Skip user config paths in project mode - try them last
		if mode == ModeProject && isUserConfigPath(p) {
			continue
		}

		if data, err := fs.ReadFile(p); err == nil {
			return parseConfig(data, p, mode)
		}
	}

	// In project mode, try user config paths as fallback
	if mode == ModeProject {
		for _, p := range candidates {
			if isUserConfigPath(p) {
				if data, err := fs.ReadFile(p); err == nil {
					return parseConfig(data, p, "global")
				}
			}
		}
	}

	return nil, fmt.Errorf("no config found; tried: %v", candidates)
}

// extractFlagValue extracts the value of a flag from command line arguments.
// Supports both "--flag value" and "--flag=value" formats.
func extractFlagValue(args []string, flag string) string {
	for i, a := range args {
		// --flag value
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
		// --flag=value
		if strings.HasPrefix(a, flag+"=") {
			return strings.TrimPrefix(a, flag+"=")
		}
	}
	return ""
}

// cliNameToEnvKey converts a CLI name to its config env var key.
// Example: "filebrowser-cli" -> "FILEBROWSER_CLI_CONFIG"
func cliNameToEnvKey(cliName string) string {
	upper := strings.ToUpper(cliName)
	// Replace hyphens with underscores
	upper = strings.ReplaceAll(upper, "-", "_")
	return upper + "_CONFIG"
}

// parseConfig parses YAML config data with env var interpolation.
func parseConfig(data []byte, path, mode string) (*LoadResult, error) {
	// Interpolate env vars first (YAML is text)
	interpolated, err := Interpolate(string(data), os.Getenv)
	if err != nil {
		return nil, fmt.Errorf("interpolate %q: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(interpolated), cfg); err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}

	// Default version to 1 if not specified
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	// Validate version
	if cfg.Version != Version1 {
		return nil, fmt.Errorf("unsupported config schema version %d in %q (expected %d)", cfg.Version, path, Version1)
	}

	return &LoadResult{Config: cfg, SourcePath: path, Mode: mode}, nil
}

// isUserConfigPath checks if a path is a user config path.
// User config paths are in ~/.config/<cli>/ or %APPDATA%\<cli>\
func isUserConfigPath(p string) bool {
	lower := strings.ToLower(p)
	// Check for .config/<cli> pattern (Unix)
	if strings.Contains(lower, ".config/filebrowser-cli") || strings.Contains(lower, ".config/memos-cli") {
		return true
	}
	// Check for APPDATA\<cli> pattern (Windows)
	if strings.Contains(lower, "appdata/filebrowser-cli") || strings.Contains(lower, "appdata/memos-cli") {
		return true
	}
	if strings.Contains(lower, "appdata\\filebrowser-cli") || strings.Contains(lower, "appdata\\memos-cli") {
		return true
	}
	return false
}