package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	fbconfig "github.com/ANIAN0/filebrowser-cli/internal/config"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved authentication token",
	Long:  `Remove the saved authentication token from the local file system.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Delete saved token
		if err := fbconfig.DeleteToken(); err != nil {
			return fmt.Errorf("logout failed: %w", err)
		}

		// Output
		mode := output.ModeText
		if jsonMode {
			mode = output.ModeJSON
		}
		out := output.New(mode)

		if jsonMode {
			return out.PrintObject(map[string]string{"status": "logged out"})
		}
		fmt.Fprintln(out.W, "Logged out successfully. Token removed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
