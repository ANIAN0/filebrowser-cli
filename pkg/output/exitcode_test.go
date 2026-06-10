package output

import (
	"testing"
)

func TestExitCodeConstants(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"ExitSuccess", ExitSuccess, 0},
		{"ExitClientError", ExitClientError, 1},
		{"ExitServerError", ExitServerError, 2},
		{"ExitNetwork", ExitNetwork, 3},
		{"ExitConfig", ExitConfig, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.expected)
			}
		})
	}
}

func TestExitCodeNames(t *testing.T) {
	tests := []struct {
		code         int
		expectedName string
	}{
		{ExitSuccess, "success"},
		{ExitClientError, "client error"},
		{ExitServerError, "server error"},
		{ExitNetwork, "network error"},
		{ExitConfig, "config error"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedName, func(t *testing.T) {
			name, ok := ExitCodeNames[tt.code]
			if !ok {
				t.Errorf("ExitCodeNames[%d] not found", tt.code)
			}
			if name != tt.expectedName {
				t.Errorf("ExitCodeNames[%d] = %q, want %q", tt.code, name, tt.expectedName)
			}
		})
	}
}