package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	sharedconfig "github.com/ANIAN0/filebrowser-cli/pkg/config"
	"github.com/ANIAN0/filebrowser-cli/pkg/output"
)

var (
	configInitPath string
	configForce    bool
	configRedact   bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration search paths",
	RunE: func(cmd *cobra.Command, args []string) error {
		binaryPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("get executable path: %w", err)
		}
		mode, candidates := sharedconfig.Resolve(cliName, binaryPath)
		envPath := os.Getenv("FILEBROWSER_CLI_CONFIG")
		data := map[string]any{
			"mode":       mode,
			"explicit":   cfgFile,
			"env":        envPath,
			"candidates": candidates,
		}
		if jsonMode {
			return output.New(output.ModeJSON).PrintObject(data)
		}
		w := cmd.OutOrStdout()
		if cfgFile != "" {
			fmt.Fprintf(w, "explicit: %s\n", cfgFile)
		}
		if envPath != "" {
			fmt.Fprintf(w, "env: %s\n", envPath)
		}
		fmt.Fprintf(w, "mode: %s\n", mode)
		for _, p := range candidates {
			fmt.Fprintf(w, "%s\n", p)
		}
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a sample configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := configInitPath
		if path == "" {
			path = defaultConfigPath()
		}
		if !configForce {
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("config already exists at %s; use --force to overwrite", path)
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("stat config: %w", err)
			}
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
		if err := os.WriteFile(path, []byte(sampleConfig), 0600); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created config at %s\n", path)
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the active configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := loadConfig(); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Config is valid")
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the active configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		shown := *cfg
		if configRedact && shown.Password != "" {
			shown.Password = "***"
		}
		mode := output.ModeText
		if jsonMode {
			mode = output.ModeJSON
		}
		return output.New(mode).PrintObject(shown)
	},
}

const sampleConfig = `version: 1
instance_url: "http://localhost:8080"
username: "admin"
password: "${FB_PASSWORD}"
default_expires: 24
default_unit: "hours"
`

func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, cliName, "config.yaml")
}

func init() {
	configInitCmd.Flags().StringVar(&configInitPath, "path", "", "config file path")
	configInitCmd.Flags().BoolVar(&configForce, "force", false, "overwrite an existing config file")
	configShowCmd.Flags().BoolVar(&configRedact, "redact", true, "redact secrets")

	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
