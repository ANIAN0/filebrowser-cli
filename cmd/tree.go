package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

var treeCmd = &cobra.Command{
	Use:   "tree [path]",
	Short: "Display directory tree",
	Long:  `Display the directory structure as a tree.`,
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

		printTree(out.W, res, "")
		return nil
	},
}

func printTree(w io.Writer, res *client.Resource, prefix string) {
	for i, item := range res.Items {
		isLast := i == len(res.Items)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		if item.IsDir {
			fmt.Fprintf(w, "%s%s%s/\n", prefix, connector, item.Name)
			newPrefix := prefix + "│   "
			if isLast {
				newPrefix = prefix + "    "
			}
			if item.Items != nil {
				printTree(w, &client.Resource{Items: item.Items}, newPrefix)
			}
		} else {
			fmt.Fprintf(w, "%s%s%s\n", prefix, connector, item.Name)
		}
	}
}

func init() {
	rootCmd.AddCommand(treeCmd)
}