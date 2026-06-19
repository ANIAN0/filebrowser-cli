package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

var renewCmd = &cobra.Command{
	Use:   "renew",
	Short: "Renew the authentication token",
	Long:  `Renew the current authentication token. Requires a valid token from a previous login.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Create HTTP client (no token yet, will be set from previous login in real usage)
		c := httpclient.New(cfg.InstanceURL,
			httpclient.WithTimeout(getTimeout()),
			httpclient.WithVerbose(verbose),
			httpclient.WithToken(loadToken()),
		)

		// Create auth client
		ac := &client.AuthClient{C: c}

		// Renew
		token, err := ac.Renew(cmd.Context())
		if err != nil {
			return err
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
	rootCmd.AddCommand(renewCmd)
}
