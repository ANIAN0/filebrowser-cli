package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

var lsCmd = &cobra.Command{
	Use:   "ls [path]",
	Short: "List directory contents",
	Long:  `List the contents of a directory on FileBrowser.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "/"
		if len(args) > 0 {
			path = args[0]
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
		res, err := rc.List(cmd.Context(), path)
		if err != nil {
			return err
		}

		mode := output.ModeText
		if jsonMode {
			mode = output.ModeJSON
		}
		out := output.New(mode)

		if jsonMode {
			return out.PrintObject(res)
		}

		for _, item := range res.Items {
			if item.IsDir {
				fmt.Fprintf(out.W, "%s/\n", item.Name)
			} else {
				fmt.Fprintf(out.W, "%s\n", item.Name)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}