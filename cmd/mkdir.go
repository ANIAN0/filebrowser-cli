package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
)

var mkdirCmd = &cobra.Command{
	Use:   "mkdir <path>",
	Short: "Create a directory",
	Long:  `Create a new directory on FileBrowser.`,
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
		return rc.Mkdir(cmd.Context(), path)
	},
}

func init() {
	rootCmd.AddCommand(mkdirCmd)
}
