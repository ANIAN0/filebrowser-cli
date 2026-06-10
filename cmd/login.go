package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
	fbconfig "github.com/ANIAN0/filebrowser-cli/internal/config"
	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with FileBrowser",
	Long:  `Login to FileBrowser and obtain an authentication token.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Create HTTP client
		c := httpclient.New(cfg.InstanceURL,
			httpclient.WithTimeout(getTimeout()),
			httpclient.WithVerbose(verbose),
		)

		// Create auth client
		ac := &client.AuthClient{C: c}

		// Login
		token, err := ac.Login(cmd.Context(), cfg.Username, cfg.Password)
		if err != nil {
			return err
		}

		// Save token to file
		if err := fbconfig.SaveToken(token); err != nil {
			// Non-fatal: just warn
			fmt.Fprintf(os.Stderr, "warn: failed to save token: %v\n", err)
		}

		// Output
		mode := output.ModeText
		if jsonMode {
			mode = output.ModeJSON
		}
		out := output.New(mode)

		if jsonMode {
			return out.PrintObject(map[string]string{"token": token})
		}
		fmt.Fprintln(out.W, token)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}