// Package output provides text and JSON formatters for CLI tools.
package output

// Exit codes for CLI tools.
const (
	// ExitSuccess indicates successful operation.
	ExitSuccess = 0

	// ExitClientError indicates a client-side error (HTTP 4xx, invalid arguments).
	ExitClientError = 1

	// ExitServerError indicates a server-side error (HTTP 5xx, internal errors).
	ExitServerError = 2

	// ExitNetwork indicates a network error (DNS, connection refused, timeout).
	ExitNetwork = 3

	// ExitConfig indicates a configuration error (missing config, invalid YAML).
	ExitConfig = 4
)

// ExitCodeNames maps exit codes to human-readable names.
var ExitCodeNames = map[int]string{
	ExitSuccess:     "success",
	ExitClientError: "client error",
	ExitServerError: "server error",
	ExitNetwork:     "network error",
	ExitConfig:      "config error",
}