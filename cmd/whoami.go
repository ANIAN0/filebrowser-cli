package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display the current user",
	Long:  `Display the username of the currently authenticated user.`,
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
			httpclient.WithToken(loadToken()),
		)

		// Create auth client
		ac := &client.AuthClient{C: c}

		// Whoami
		username, err := ac.Whoami(cmd.Context())
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
			return out.PrintObject(map[string]string{"username": username})
		}
		fmt.Fprintln(out.W, username)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}