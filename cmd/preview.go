package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

var (
	previewSize   string
	previewOutput string
)

var previewCmd = &cobra.Command{
	Use:   "preview <path>",
	Short: "Get image preview",
	Long:  `Get a preview image of a file (thumb: 256x256, big: 1080x1080).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

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
		data, err := rc.Preview(cmd.Context(), path, previewSize)
		if err != nil {
			return err
		}

		// If output file specified, write to file
		if previewOutput != "" {
			return os.WriteFile(previewOutput, data, 0644)
		}

		// Otherwise write to stdout
		mode := output.ModeText
		if jsonMode {
			mode = output.ModeJSON
		}
		out := output.New(mode)

		_, err = out.W.Write(data)
		return err
	},
}

func init() {
	previewCmd.Flags().StringVar(&previewSize, "size", "thumb", "preview size: thumb (256x256) or big (1080x1080)")
	previewCmd.Flags().StringVar(&previewOutput, "output", "", "output file path (default: stdout)")
	rootCmd.AddCommand(previewCmd)
}