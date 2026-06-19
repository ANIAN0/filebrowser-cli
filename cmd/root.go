package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
	fbconfig "github.com/ANIAN0/filebrowser-cli/internal/config"
	sharedconfig "github.com/ANIAN0/filebrowser-cli/pkg/config"
	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
	"github.com/ANIAN0/filebrowser-cli/pkg/version"
)

var (
	cfgFile    string
	jsonMode   bool
	verbose    bool
	noColor    bool
	timeoutSec int
)

var rootCmd = &cobra.Command{
	Use:     "filebrowser-cli",
	Short:   "CLI for FileBrowser file management",
	Long:    "filebrowser-cli provides a shell-callable interface to the FileBrowser HTTP API.",
	Version: version.String(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize output
		mode := output.ModeText
		if jsonMode {
			mode = output.ModeJSON
		}
		_ = output.New(mode) // Output will be used by subcommands

		// Validate timeout
		if timeoutSec <= 0 {
			return fmt.Errorf("timeout must be positive, got %d", timeoutSec)
		}

		// Config loading will be done by subcommands that need it
		// This keeps the root command lightweight

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and returns the exit code.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		code := classifyExitCode(err)
		mode := output.ModeText
		if jsonMode {
			mode = output.ModeJSON
		}
		_ = output.New(mode).PrintError(err, code)
		return code
	}
	return output.ExitSuccess
}

var httpStatusPattern = regexp.MustCompile(`HTTP ([0-9]{3})`)

func classifyExitCode(err error) int {
	if err == nil {
		return output.ExitSuccess
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return output.ExitNetwork
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return output.ExitNetwork
	}

	msg := err.Error()
	if strings.Contains(msg, "load config:") ||
		strings.Contains(msg, "config") ||
		strings.Contains(msg, "env var(s) not set") {
		return output.ExitConfig
	}

	if match := httpStatusPattern.FindStringSubmatch(msg); len(match) == 2 {
		status, convErr := strconv.Atoi(match[1])
		if convErr == nil {
			if status >= 500 && status < 600 {
				return output.ExitServerError
			}
			return output.ExitClientError
		}
	}

	return output.ExitClientError
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "output JSON format")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colors")
	rootCmd.PersistentFlags().IntVar(&timeoutSec, "timeout", 60, "HTTP timeout in seconds")

	// Set version template
	rootCmd.SetVersionTemplate(`filebrowser-cli {{.Version}}
`)
}

// getTimeout returns the timeout as a time.Duration.
func getTimeout() time.Duration {
	return time.Duration(timeoutSec) * time.Second
}

// loadToken loads the saved authentication token WITHOUT an expiry check.
// It is used by `renew`, whose semantics require sending the on-disk token to
// the server so the server (the authority on validity) can refresh it.
//
// For normal authenticated commands, prefer newAuthedClient, which transparently
// handles token expiry and auto re-login.
func loadToken() string {
	token, _ := fbconfig.LoadRawToken()
	return token
}

// newAuthedClient builds an HTTP client carrying a usable auth token.
//
// If a valid (non-expired) token is on disk it is used directly. If the token
// has expired (or there is none) and the config carries username/password
// credentials, it logs in again transparently, persists the new token, and
// returns a client carrying it. If credentials are missing it returns an error
// directing the user to run `filebrowser-cli login`.
//
// This eliminates the "token expired -> HTTP 401" failure mode reported in the
// token-expiry bug, while degrading gracefully when auto re-login isn't possible.
func newAuthedClient(ctx context.Context, cfg *fbconfig.Config) (*httpclient.Client, error) {
	httpc := httpclient.New(cfg.InstanceURL,
		httpclient.WithTimeout(getTimeout()),
		httpclient.WithVerbose(verbose),
	)

	token, err := fbconfig.LoadToken()
	if err != nil && !errors.Is(err, fbconfig.ErrTokenExpired) {
		return nil, fmt.Errorf("load token: %w", err)
	}

	if token != "" {
		httpc.Token = token
		return httpc, nil
	}

	// Token missing or expired -> attempt auto re-login with configured creds.
	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("not logged in or token expired; run `filebrowser-cli login` (or set username/password in config for automatic re-login)")
	}

	if verbose {
		fmt.Fprintln(os.Stderr, "token expired or missing, logging in...")
	}

	ac := &client.AuthClient{C: httpc}
	newToken, lerr := ac.Login(ctx, cfg.Username, cfg.Password)
	if lerr != nil {
		return nil, fmt.Errorf("auto re-login: %w", lerr)
	}

	if serr := fbconfig.SaveToken(newToken); serr != nil {
		// Non-fatal: the in-memory token is still set below.
		fmt.Fprintf(os.Stderr, "warn: failed to save refreshed token: %v\n", serr)
	}

	httpc.Token = newToken
	return httpc, nil
}

// loadConfig loads the configuration from file.
func loadConfig() (*fbconfig.Config, error) {
	binaryPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("get executable path: %w", err)
	}

	// Build args list with --config flag if specified
	args := os.Args[1:]
	if cfgFile != "" {
		args = append([]string{"--config", cfgFile}, args...)
	}

	// Build env map
	env := make(map[string]string)
	for _, e := range os.Environ() {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				env[e[:i]] = e[i+1:]
				break
			}
		}
	}

	// Load shared config
	result, err := sharedconfig.LoadConfig("filebrowser-cli", args, env, binaryPath, nil)
	if err != nil {
		return nil, err
	}

	cfg, err := fbconfig.LoadFromBytes(result.Data)
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
