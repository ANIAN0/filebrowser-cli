package httpclient

import (
	"testing"
)

func TestMapHTTPStatusToExitCode(t *testing.T) {
	tests := []struct {
		status   int
		expected int
	}{
		// 2xx success
		{200, ExitSuccess},
		{201, ExitSuccess},
		{204, ExitSuccess},
		{202, ExitSuccess},

		// 4xx client errors
		{400, ExitClientError},
		{401, ExitClientError},
		{403, ExitClientError},
		{404, ExitClientError},
		{405, ExitClientError},
		{409, ExitClientError},
		{410, ExitClientError},
		{422, ExitClientError},

		// 5xx server errors
		{500, ExitServerError},
		{501, ExitServerError},
		{502, ExitServerError},
		{503, ExitServerError},
		{504, ExitServerError},

		// Edge cases
		{100, ExitClientError}, // Informational (unexpected)
		{301, ExitClientError}, // Redirect (unexpected for API)
		{0, ExitClientError},   // Invalid
	}

	for _, tt := range tests {
		t.Run("status_"+string(rune(tt.status+'0')), func(t *testing.T) {
			got := MapHTTPStatusToExitCode(tt.status)
			if got != tt.expected {
				t.Errorf("MapHTTPStatusToExitCode(%d) = %d, want %d", tt.status, got, tt.expected)
			}
		})
	}
}

func TestMapMemosCodeToExitCode(t *testing.T) {
	tests := []struct {
		code     int32
		expected int
	}{
		// Success
		{0, ExitSuccess},

		// Client errors
		{3, ExitClientError},  // INVALID_ARGUMENT
		{5, ExitClientError},  // NOT_FOUND
		{7, ExitClientError},  // UNAUTHENTICATED
		{16, ExitClientError}, // UNAUTHENTICATED

		// Server errors (other codes)
		{1, ExitServerError},  // CANCELLED
		{2, ExitServerError},  // UNKNOWN
		{4, ExitServerError},  // DEADLINE_EXCEEDED
		{6, ExitServerError},  // ALREADY_EXISTS
		{8, ExitServerError},  // RESOURCE_EXHAUSTED
		{10, ExitServerError}, // ABORTED
		{11, ExitServerError}, // OUT_OF_RANGE
		{13, ExitServerError}, // INTERNAL
		{14, ExitServerError}, // UNAVAILABLE
		{15, ExitServerError}, // DATA_LOSS
	}

	for _, tt := range tests {
		t.Run("code_"+string(rune(tt.code+'0')), func(t *testing.T) {
			got := MapMemosCodeToExitCode(tt.code)
			if got != tt.expected {
				t.Errorf("MapMemosCodeToExitCode(%d) = %d, want %d", tt.code, got, tt.expected)
			}
		})
	}
}

func TestExitCodeConstants(t *testing.T) {
	if ExitSuccess != 0 {
		t.Errorf("ExitSuccess = %d, want 0", ExitSuccess)
	}
	if ExitClientError != 1 {
		t.Errorf("ExitClientError = %d, want 1", ExitClientError)
	}
	if ExitServerError != 2 {
		t.Errorf("ExitServerError = %d, want 2", ExitServerError)
	}
	if ExitNetwork != 3 {
		t.Errorf("ExitNetwork = %d, want 3", ExitNetwork)
	}
	if ExitConfigError != 4 {
		t.Errorf("ExitConfigError = %d, want 4", ExitConfigError)
	}
}