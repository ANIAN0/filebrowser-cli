// Package errcode defines sentinel errors and typed errors used to classify
// failures in filebrowser-cli without depending on error-message strings.
//
// The goal is to let cmd/root.go:classifyExitCode route failures to the right
// exit code via errors.Is / errors.As, while leaving individual error
// messages free to be rewritten (i18n, wording tweaks) without silently
// breaking the exit-code contract.
package errcode

import (
	"errors"
	"fmt"
)

// Sentinel errors. Wrap these in higher-level errors with fmt.Errorf("%w", err)
// so callers can classify with errors.Is.
var (
	// ErrConfigLoad indicates the configuration could not be read (missing file,
	// unreadable permissions, invalid YAML, unresolved ${ENV_VAR}).
	ErrConfigLoad = errors.New("config load failed")

	// ErrConfigInvalid indicates the configuration was loaded but failed
	// validation (missing required fields, out-of-range values).
	ErrConfigInvalid = errors.New("config invalid")

	// ErrNetwork indicates a network-level failure (DNS, connection refused,
	// timeout, TLS handshake). Wrap with %w when returning.
	ErrNetwork = errors.New("network error")

	// ErrServer indicates the upstream FileBrowser server returned HTTP 5xx.
	// Use StatusError{Code >= 500} as a more specific alternative.
	ErrServer = errors.New("server error")

	// ErrClient indicates the upstream FileBrowser server returned HTTP 4xx.
	// Use StatusError{Code} as a more specific alternative.
	ErrClient = errors.New("client error")
)

// StatusError carries the HTTP status code returned by the upstream server.
// It enables classifyExitCode to distinguish 4xx (client) from 5xx (server)
// without parsing error messages.
type StatusError struct {
	// Code is the HTTP status code (e.g. 404, 500).
	Code int
	// Op is the operation that produced the error (e.g. "list", "upload").
	Op string
	// Path is the resource path (optional, for context).
	Path string
}

// Error implements the error interface.
func (e *StatusError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s %s failed: HTTP %d", e.Op, e.Path, e.Code)
	}
	if e.Op != "" {
		return fmt.Sprintf("%s failed: HTTP %d", e.Op, e.Code)
	}
	return fmt.Sprintf("HTTP %d", e.Code)
}

// Is enables errors.Is(err, ErrServer) / errors.Is(err, ErrClient) for
// status-based classification.
func (e *StatusError) Is(target error) bool {
	switch target {
	case ErrServer:
		return e.Code >= 500 && e.Code < 600
	case ErrClient:
		return e.Code >= 400 && e.Code < 500
	}
	return false
}

// Unwrap makes StatusError compatible with errors.Is/As.
// Only HTTP error codes (4xx/5xx) map to sentinels; non-error codes (e.g. 200)
// return nil so errors.Is does not falsely match ErrClient/ErrServer.
func (e *StatusError) Unwrap() error {
	switch {
	case e.Code >= 500 && e.Code < 600:
		return ErrServer
	case e.Code >= 400 && e.Code < 500:
		return ErrClient
	}
	return nil
}
