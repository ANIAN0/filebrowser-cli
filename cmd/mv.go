package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
)

var mvCmd = &cobra.Command{
	Use:   "mv <src> <dst>",
	Short: "Move or rename a file or directory",
	Long:  `Move or rename a file or directory on FileBrowser.`,
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
		return rc.Move(cmd.Context(), src, dst)
	},
}

func init() {
	rootCmd.AddCommand(mvCmd)
}
