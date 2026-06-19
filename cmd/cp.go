package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
)

var cpCmd = &cobra.Command{
	Use:   "cp <src> <dst>",
	Short: "Copy a file or directory",
	Long:  `Copy a file or directory on FileBrowser.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		dst := args[1]

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		c, err := newAuthedClient(cmd.Context(), cfg)
		if err != nil {
			return err
		}

		rc := &client.ResourceClient{C: c}
		return rc.Copy(cmd.Context(), src, dst)
	},
}

func init() {
	rootCmd.AddCommand(cpCmd)
}
