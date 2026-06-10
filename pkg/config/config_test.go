package config

import (
	"testing"
)

func TestConfig_DefaultVersion(t *testing.T) {
	cfg := &Config{}
	if cfg.Version != 0 {
		t.Errorf("new Config should have Version=0, got %d", cfg.Version)
	}
}

func TestConfig_Version1(t *testing.T) {
	cfg := &Config{Version: 1}
	if cfg.Version != 1 {
		t.Errorf("Config.Version should be 1, got %d", cfg.Version)
	}
}

func TestVersion1_Constant(t *testing.T) {
	if Version1 != 1 {
		t.Errorf("Version1 constant should be 1, got %d", Version1)
	}
}