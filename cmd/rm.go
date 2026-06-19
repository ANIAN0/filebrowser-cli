package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
)

var rmCmd = &cobra.Command{
	Use:   "rm <path>",
	Short: "Remove a file or directory",
	Long:  `Delete a file or directory from FileBrowser.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		c, err := newAuthedClient(cmd.Context(), cfg)
		if err != nil {
			return err
		}

		rc := &client.ResourceClient{C: c}
		return rc.Remove(cmd.Context(), path)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
