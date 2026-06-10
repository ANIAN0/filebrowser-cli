package config

import (
	"strings"
	"testing"
)

func TestInterpolate_Basic(t *testing.T) {
	env := map[string]string{"X": "hello"}
	result, err := Interpolate("prefix_${X}_suffix", func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "prefix_hello_suffix" {
		t.Errorf("got %q, want %q", result, "prefix_hello_suffix")
	}
}

func TestInterpolate_MultipleVars(t *testing.T) {
	env := map[string]string{"A": "foo", "B": "bar"}
	result, err := Interpolate("${A}_${B}", func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "foo_bar" {
		t.Errorf("got %q, want %q", result, "foo_bar")
	}
}

func TestInterpolate_NoVars(t *testing.T) {
	result, err := Interpolate("no vars here", func(k string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "no vars here" {
		t.Errorf("got %q, want %q", result, "no vars here")
	}
}

func TestInterpolate_MissingVar(t *testing.T) {
	env := map[string]string{}
	_, err := Interpolate("${MISSING}", func(k string) string { return env[k] })
	if err == nil {
		t.Error("expected error for missing env var")
	}
	if !strings.Contains(err.Error(), "MISSING") {
		t.Errorf("error should mention MISSING, got: %v", err)
	}
}

func TestInterpolate_MultipleMissingVars(t *testing.T) {
	env := map[string]string{"A": "ok"}
	_, err := Interpolate("${A}_${B}_${C}", func(k string) string { return env[k] })
	if err == nil {
		t.Error("expected error for missing env vars")
	}
	if !strings.Contains(err.Error(), "B") || !strings.Contains(err.Error(), "C") {
		t.Errorf("error should mention B and C, got: %v", err)
	}
}

func TestInterpolate_EmptyValueIsNotSet(t *testing.T) {
	env := map[string]string{"X": ""}
	_, err := Interpolate("${X}", func(k string) string { return env[k] })
	if err == nil {
		t.Error("expected error for empty env var value")
	}
}

func TestInterpolate_ConsecutiveVars(t *testing.T) {
	env := map[string]string{"A": "hello", "B": "world"}
	result, err := Interpolate("${A}${B}", func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "helloworld" {
		t.Errorf("got %q, want %q", result, "helloworld")
	}
}

func TestInterpolate_SpecialChars(t *testing.T) {
	env := map[string]string{"X": "hello world"}
	result, err := Interpolate("${X}", func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("got %q, want %q", result, "hello world")
	}
}

func TestHasEnvVars_True(t *testing.T) {
	if !HasEnvVars("prefix_${VAR}_suffix") {
		t.Error("should detect ${VAR}")
	}
}

func TestHasEnvVars_False(t *testing.T) {
	if HasEnvVars("no vars here") {
		t.Error("should not detect env vars")
	}
}

func TestExtractEnvVars(t *testing.T) {
	names := ExtractEnvVars("${A}_${B}_nope_${C}")
	if len(names) != 3 {
		t.Fatalf("expected 3 env vars, got %d: %v", len(names), names)
	}
	if names[0] != "A" || names[1] != "B" || names[2] != "C" {
		t.Errorf("got %v, want [A, B, C]", names)
	}
}