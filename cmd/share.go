package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/internal/client"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Manage file shares",
	Long:  `Create, list, delete, and view file shares on FileBrowser.`,
}

var shareCreateCmd = &cobra.Command{
	Use:   "create <path>",
	Short: "Create a new share",
	Long:  `Create a new share link for a file or directory.`,
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

		sc := &client.ShareClient{C: c}
		sh, err := sc.Create(cmd.Context(), path, shareExpires, shareUnit, sharePassword)
		if err != nil {
			return err
		}

		mode := output.ModeText
		if jsonMode {
			mode = output.ModeJSON
		}
		out := output.New(mode)

		if jsonMode {
			return out.PrintObject(sh)
		}
		fmt.Fprintf(out.W, "Hash: %s\nPath: %s\nURL: /api/public/dl/%s\n", sh.Hash, sh.Path, sh.Hash)
		return nil
	},
}

var shareListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all shares",
	Long:  `List all active shares on FileBrowser.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		c, err := newAuthedClient(cmd.Context(), cfg)
		if err != nil {
			return err
		}

		sc := &client.ShareClient{C: c}
		shares, err := sc.List(cmd.Context())
		if err != nil {
			return err
		}

		mode := output.ModeText
		if jsonMode {
			mode = output.ModeJSON
		}
		out := output.New(mode)

		if jsonMode {
			return out.PrintList(sharesToAny(shares))
		}

		if len(shares) == 0 {
			fmt.Fprintln(out.W, "No shares found")
			return nil
		}

		for _, sh := range shares {
			fmt.Fprintf(out.W, "%s -> %s\n", sh.Hash, sh.Path)
		}
		return nil
	},
}

var shareDeleteCmd = &cobra.Command{
	Use:   "delete <hash>",
	Short: "Delete a share",
	Long:  `Delete a share by its hash.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hash := args[0]

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		c, err := newAuthedClient(cmd.Context(), cfg)
		if err != nil {
			return err
		}

		sc := &client.ShareClient{C: c}
		return sc.Delete(cmd.Context(), hash)
	},
}

var shareInfoCmd = &cobra.Command{
	Use:   "info <path>",
	Short: "Show share information",
	Long:  `Show share information for a specific path.`,
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

		sc := &client.ShareClient{C: c}
		shares, err := sc.Info(cmd.Context(), path)
		if err != nil {
			return err
		}

		mode := output.ModeText
		if jsonMode {
			mode = output.ModeJSON
		}
		out := output.New(mode)

		if jsonMode {
			return out.PrintList(sharesToAny(shares))
		}

		for _, sh := range shares {
			fmt.Fprintf(out.W, "Hash: %s\nPath: %s\nURL: /api/public/dl/%s\n\n", sh.Hash, sh.Path, sh.Hash)
		}
		return nil
	},
}

// sharesToAny converts []Share to []any for output.
func sharesToAny(shares []client.Share) []any {
	result := make([]any, len(shares))
	for i, sh := range shares {
		result[i] = sh
	}
	return result
}

var (
	shareExpires  string
	shareUnit     string
	sharePassword string
)

func init() {
	shareCreateCmd.Flags().StringVar(&shareExpires, "expires", "", "expiration time")
	shareCreateCmd.Flags().StringVar(&shareUnit, "unit", "hours", "time unit: seconds, minutes, hours, days")
	shareCreateCmd.Flags().StringVar(&sharePassword, "password", "", "share password")

	shareCmd.AddCommand(shareCreateCmd)
	shareCmd.AddCommand(shareListCmd)
	shareCmd.AddCommand(shareDeleteCmd)
	shareCmd.AddCommand(shareInfoCmd)

	rootCmd.AddCommand(shareCmd)
}
