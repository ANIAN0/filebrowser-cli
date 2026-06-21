package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	fbconfig "github.com/ANIAN0/filebrowser-cli/internal/config"
)

// TestConfigJSONKeysMatchYAML pins the cross-format contract: the JSON keys
// produced by `config show --json` must match the YAML keys a user writes in
// config.yaml. Without explicit json tags, Go marshals Go field names
// (PascalCase), which breaks that contract.
func TestConfigJSONKeysMatchYAML(t *testing.T) {
	cfg := &fbconfig.Config{
		InstanceURL:    "http://localhost:8080",
		Username:       "admin",
		Password:       "secret",
		DefaultExpires: 24,
		DefaultUnit:    "hours",
	}
	cfg.Version = 1

	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	got := string(b)

	wantKeys := []string{
		`"version":`,
		`"instance_url":`,
		`"username":`,
		`"password":`,
		`"default_expires":`,
		`"default_unit":`,
	}
	for _, k := range wantKeys {
		if !strings.Contains(got, k) {
			t.Errorf("json output missing key %q\noutput: %s", k, got)
		}
	}

	// Negative case: ensure PascalCase Go-field-named output is NOT present.
	badKeys := []string{`"Version":`, `"InstanceURL":`, `"DefaultExpires":`}
	for _, k := range badKeys {
		if strings.Contains(got, k) {
			t.Errorf("json output contains unwanted PascalCase key %q\noutput: %s", k, got)
		}
	}
}

// TestConfigJSONRedact mirrors the cmd/config.go redact behavior: with the
// redacted copy, password must serialize as "***", not the real value.
func TestConfigJSONRedact(t *testing.T) {
	cfg := fbconfig.Config{
		InstanceURL:    "http://localhost:8080",
		Username:       "admin",
		Password:       "supersecret",
		DefaultExpires: 24,
		DefaultUnit:    "hours",
	}
	cfg.Version = 1

	// Mimic cmd/config.go:configShowCmd redact logic.
	shown := cfg
	if shown.Password != "" {
		shown.Password = "***"
	}
	b, err := json.Marshal(&shown)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, "supersecret") {
		t.Errorf("redacted output leaked real password: %s", got)
	}
	if !strings.Contains(got, `"password":"***"`) {
		t.Errorf("redacted output should contain password:***, got: %s", got)
	}
}