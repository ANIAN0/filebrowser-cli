// Package config provides FileBrowser-specific configuration.
package config

import (
	"fmt"

	"gopkg.in/yaml.v3"

	sharedconfig "github.com/ANIAN0/filebrowser-cli/pkg/config"
)

// Config is the FileBrowser-specific configuration.
// It embeds the shared Config and adds FileBrowser-specific fields.
type Config struct {
	sharedconfig.Config `yaml:",inline"`

	// InstanceURL is the FileBrowser server URL (e.g., "http://localhost:8080").
	InstanceURL string `yaml:"instance_url"`

	// Username is the FileBrowser username.
	Username string `yaml:"username"`

	// Password is the FileBrowser password (supports ${ENV_VAR} interpolation).
	Password string `yaml:"password"`

	// DefaultExpires is the default expiration time for shares.
	DefaultExpires int `yaml:"default_expires"`

	// DefaultUnit is the default time unit for expiration (s, m, h, d).
	DefaultUnit string `yaml:"default_unit"`
}

// LoadFromBytes parses a YAML config into c.
func LoadFromBytes(data []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

// Validate checks if the config is valid.
func (c *Config) Validate() error {
	if c.InstanceURL == "" {
		return fmt.Errorf("instance_url is required")
	}
	return nil
}
