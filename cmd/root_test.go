package cmd

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

func TestClassifyExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: output.ExitSuccess},
		{name: "config", err: errors.New("load config: no config found"), want: output.ExitConfig},
		{name: "network", err: &net.DNSError{Err: "lookup failed"}, want: output.ExitNetwork},
		{name: "timeout", err: context.DeadlineExceeded, want: output.ExitNetwork},
		{name: "server", err: errors.New("list failed: HTTP 503"), want: output.ExitServerError},
		{name: "client", err: errors.New("list failed: HTTP 404"), want: output.ExitClientError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyExitCode(tt.err); got != tt.want {
				t.Fatalf("classifyExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}
