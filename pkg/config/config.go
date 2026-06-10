// Package config provides shared config loading for both CLIs.
//
// Config loading follows a 4-layer priority:
// 1) --config flag value
// 2) <CLI>_CONFIG env var
// 3) project-local config (mode B): <binary-dir>/config.yaml or <binary-dir>/../config.yaml
// 4) user dir config (mode A or fallback): ~/.config/<cli>/config.yaml (Unix) or %APPDATA%\<cli>\config.yaml (Windows)
//
// The package never reads skillconfig.json (DECIDE-011).
package config

// Config is the base struct. Specific CLIs embed this and add their fields.
//
// Example YAML:
//
//	version: 1
//	instance_url: "http://localhost:8080"
//	username: "admin"
//	password: "${FB_PASSWORD}"
type Config struct {
	// Version is the YAML schema version. Must be 1.
	Version int `yaml:"version"`
}

// Version1 is the supported schema version.
const Version1 = 1