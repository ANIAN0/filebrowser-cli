package errcode

import (
	"errors"
	"fmt"
	"testing"
)

func TestStatusError_Is(t *testing.T) {
	cases := []struct {
		name string
		err  *StatusError
		is   error
		want bool
	}{
		{"500 is server", &StatusError{Code: 500}, ErrServer, true},
		{"503 is server", &StatusError{Code: 503}, ErrServer, true},
		{"404 is client", &StatusError{Code: 404}, ErrClient, true},
		{"400 is client", &StatusError{Code: 400}, ErrClient, true},
		{"500 is NOT client", &StatusError{Code: 500}, ErrClient, false},
		{"404 is NOT server", &StatusError{Code: 404}, ErrServer, false},
		{"200 is neither", &StatusError{Code: 200}, ErrServer, false},
		{"200 is neither (client)", &StatusError{Code: 200}, ErrClient, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := errors.Is(c.err, c.is); got != c.want {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", c.err, c.is, got, c.want)
			}
		})
	}
}

func TestStatusError_Error(t *testing.T) {
	cases := []struct {
		err  *StatusError
		want string
	}{
		{&StatusError{Code: 404}, "HTTP 404"},
		{&StatusError{Op: "list", Code: 404}, "list failed: HTTP 404"},
		{&StatusError{Op: "list", Code: 404, Path: "/foo"}, "list /foo failed: HTTP 404"},
	}
	for _, c := range cases {
		if got := c.err.Error(); got != c.want {
			t.Errorf("Error() = %q, want %q", got, c.want)
		}
	}
}

func TestStatusError_Unwrap(t *testing.T) {
	if got := (&StatusError{Code: 500}).Unwrap(); got != ErrServer {
		t.Errorf("Unwrap(500) = %v, want ErrServer", got)
	}
	if got := (&StatusError{Code: 404}).Unwrap(); got != ErrClient {
		t.Errorf("Unwrap(404) = %v, want ErrClient", got)
	}
}

func TestSentinel_IsUnchangedByMessage(t *testing.T) {
	// Negative case (regression): errors.Is via sentinel must NOT depend on
	// the wrapped message text.
	wrapped := fmt.Errorf("读取配置: %w", ErrConfigLoad)
	if !errors.Is(wrapped, ErrConfigLoad) {
		t.Fatal("errors.Is lost sentinel through i18n message change")
	}
}
