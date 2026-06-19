package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

var searchLimit int

var searchCmd = &cobra.Command{
	Use:   "search <path> <query>",
	Short: "Search files in FileBrowser",
	Long:  `Search for files matching a query in FileBrowser using SSE stream.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		query := args[1]

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		c, err := newAuthedClient(cmd.Context(), cfg)
		if err != nil {
			return err
		}

		sc := &client.SearchClient{C: c}
		results, err := sc.Search(cmd.Context(), path, query, searchLimit)
		if err != nil {
			return err
		}

		mode := output.ModeText
		if jsonMode {
			mode = output.ModeJSON
		}
		out := output.New(mode)

		items := make([]any, len(results))
		for i, r := range results {
			items[i] = r
		}

		return out.PrintList(items)
	},
}

func init() {
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10000, "max number of results")
	rootCmd.AddCommand(searchCmd)
}
