package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	fbconfig "github.com/ANIAN0/filebrowser-cli/internal/config"
	sharedconfig "github.com/ANIAN0/filebrowser-cli/pkg/config"
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
		fmt.Fprintln(os.Stderr, "error:", err)
		return output.ExitClientError
	}
	return output.ExitSuccess
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

// loadToken loads the saved authentication token.
func loadToken() string {
	token, _ := fbconfig.LoadToken()
	return token
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
