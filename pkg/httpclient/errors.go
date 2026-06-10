// Package httpclient provides HTTP client utilities for CLI tools.
package httpclient

// Exit codes for different error types.
const (
	ExitSuccess      = 0
	ExitClientError  = 1
	ExitServerError  = 2
	ExitNetwork      = 3
	ExitConfigError  = 4
)

// MapHTTPStatusToExitCode maps HTTP status codes to CLI exit codes.
//
// FileBrowser mapping:
//   - 2xx (200, 201, 204): ExitSuccess (0)
//   - 401, 403, 404, 409: ExitClientError (1)
//   - 5xx (500, 502, 503, 504): ExitServerError (2)
//   - Other 4xx: ExitClientError (1)
func MapHTTPStatusToExitCode(status int) int {
	switch {
	case status >= 200 && status < 300:
		return ExitSuccess
	case status == 401, status == 403, status == 404, status == 409:
		return ExitClientError
	case status >= 500 && status < 600:
		return ExitServerError
	default:
		// Other 4xx and unexpected codes
		return ExitClientError
	}
}

// MapMemosCodeToExitCode maps Memos gRPC-style error codes to CLI exit codes.
//
// Memos mapping:
//   - 0 (OK): ExitSuccess (0)
//   - 3 (INVALID_ARGUMENT): ExitClientError (1)
//   - 5 (NOT_FOUND): ExitClientError (1)
//   - 7 (UNAUTHENTICATED): ExitClientError (1)
//   - 16 (UNAUTHENTICATED): ExitClientError (1)
//   - Other: ExitServerError (2)
func MapMemosCodeToExitCode(code int32) int {
	switch code {
	case 0:
		return ExitSuccess
	case 3, 5, 7, 16:
		return ExitClientError
	default:
		return ExitServerError
	}
}