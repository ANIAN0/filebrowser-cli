package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
)

var uploadCmd = &cobra.Command{
	Use:   "upload <local> [remote]",
	Short: "Upload a file",
	Long:  `Upload a local file to FileBrowser.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		localPath := args[0]
		remotePath := "/"
		if len(args) > 1 {
			remotePath = args[1]
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
		return rc.Upload(cmd.Context(), localPath, remotePath, true)
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}