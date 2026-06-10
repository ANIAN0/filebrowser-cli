package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
)

var downloadCmd = &cobra.Command{
	Use:   "download <remote> [local]",
	Short: "Download a file",
	Long:  `Download a file from FileBrowser to local filesystem.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := args[0]
		localPath := ""
		if len(args) > 1 {
			localPath = args[1]
		}

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		c := httpclient.New(cfg.InstanceURL,
			httpclient.WithTimeout(getTimeout()),
			httpclient.WithVerbose(verbose),
			httpclient.WithToken(loadToken()),
		)

		rc := &client.ResourceClient{C: c}
		return rc.Download(cmd.Context(), remotePath, localPath)
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}