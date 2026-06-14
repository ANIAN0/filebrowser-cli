package cmd

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ANIAN0/filebrowser-cli/pkg/version"
)

const (
	cliName     = "filebrowser-cli"
	githubOwner = "ANIAN0"
	githubRepo  = "filebrowser-cli"
)

var (
	selfDir   string
	selfForce bool
	selfPurge bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install this binary into a user PATH directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		src, err := os.Executable()
		if err != nil {
			return fmt.Errorf("get executable path: %w", err)
		}
		dir, err := installDir(selfDir)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, executableName())
		if samePath(src, target) {
			fmt.Fprintf(cmd.OutOrStdout(), "%s is already installed at %s\n", cliName, target)
			return nil
		}
		if err := copyExecutable(src, target); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Installed %s to %s\n", cliName, target)
		if !pathContains(dir) {
			fmt.Fprintf(cmd.ErrOrStderr(), "WARN: %s is not in PATH. Add it to your shell profile.\n", dir)
		}
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update this binary from the latest GitHub release",
	RunE: func(cmd *cobra.Command, args []string) error {
		rel, err := latestRelease()
		if err != nil {
			return err
		}
		if !selfForce && version.Version != "" && version.Version != "dev" && rel.TagName == version.Version {
			fmt.Fprintf(cmd.OutOrStdout(), "%s is already up to date (%s)\n", cliName, version.Version)
			return nil
		}

		assetName := releaseAssetName()
		assetURL, ok := rel.assetURL(assetName)
		if !ok {
			return fmt.Errorf("release %s does not contain asset %q", rel.TagName, assetName)
		}

		tmp, err := downloadTemp(assetURL, assetName)
		if err != nil {
			return err
		}

		if checksumURL, ok := rel.assetURL("checksums.txt"); ok {
			if err := verifyChecksum(checksumURL, assetName, tmp); err != nil {
				os.Remove(tmp)
				return err
			}
		}

		target, err := os.Executable()
		if err != nil {
			return fmt.Errorf("get executable path: %w", err)
		}
		if err := replaceExecutable(tmp, target); err != nil {
			os.Remove(tmp)
			return err
		}
		if runtime.GOOS == "windows" {
			fmt.Fprintf(cmd.OutOrStdout(), "Update to %s has been scheduled. Restart your shell before running %s again.\n", rel.TagName, cliName)
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Updated %s to %s\n", cliName, rel.TagName)
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the installed binary",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := installDir(selfDir)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, executableName())
		current, err := os.Executable()
		if err == nil && samePath(filepath.Dir(current), dir) {
			target = current
		}

		if err := removeExecutable(target); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(cmd.OutOrStdout(), "%s is not installed at %s\n", cliName, target)
			} else {
				return err
			}
		} else if runtime.GOOS == "windows" && samePath(current, target) {
			fmt.Fprintf(cmd.OutOrStdout(), "Uninstall has been scheduled for %s\n", target)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", target)
		}

		if selfPurge {
			dir, err := os.UserConfigDir()
			if err != nil {
				return fmt.Errorf("get user config dir: %w", err)
			}
			configDir := filepath.Join(dir, cliName)
			if err := os.RemoveAll(configDir); err != nil {
				return fmt.Errorf("remove config dir: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed config directory %s\n", configDir)
		}
		return nil
	},
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func (r githubRelease) assetURL(name string) (string, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a.URL, true
		}
	}
	return "", false
}

func init() {
	installCmd.Flags().StringVar(&selfDir, "dir", "", "installation directory")
	updateCmd.Flags().BoolVar(&selfForce, "force", false, "update even when the current version matches latest")
	uninstallCmd.Flags().StringVar(&selfDir, "dir", "", "installation directory")
	uninstallCmd.Flags().BoolVar(&selfPurge, "purge", false, "also remove user configuration")

	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(uninstallCmd)
}

func executableName() string {
	if runtime.GOOS == "windows" {
		return cliName + ".exe"
	}
	return cliName
}

func releaseAssetName() string {
	return fmt.Sprintf("%s-%s-%s%s", cliName, runtime.GOOS, runtime.GOARCH, exeSuffix())
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func installDir(override string) (string, error) {
	if override != "" {
		return filepath.Abs(override)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home dir: %w", err)
	}
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "Programs", cliName), nil
		}
		return filepath.Join(home, "bin"), nil
	}
	return filepath.Join(home, ".local", "bin"), nil
}

func copyExecutable(src, target string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source executable: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}

	tmp := target + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("create temp executable: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy executable: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close temp executable: %w", err)
	}
	return replaceExecutable(tmp, target)
}

func replaceExecutable(src, target string) error {
	if runtime.GOOS == "windows" {
		current, _ := os.Executable()
		if samePath(current, target) {
			return scheduleWindowsReplace(src, target)
		}
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove old executable: %w", err)
		}
	}
	if err := os.Rename(src, target); err != nil {
		return fmt.Errorf("replace executable: %w", err)
	}
	return os.Chmod(target, 0755)
}

func removeExecutable(target string) error {
	if runtime.GOOS == "windows" {
		current, _ := os.Executable()
		if samePath(current, target) {
			return scheduleWindowsDelete(target)
		}
	}
	if err := os.Remove(target); err != nil {
		return fmt.Errorf("remove executable: %w", err)
	}
	return nil
}

func scheduleWindowsReplace(src, target string) error {
	script := filepath.Join(os.TempDir(), cliName+"-update.cmd")
	body := fmt.Sprintf("@echo off\r\nping 127.0.0.1 -n 2 > nul\r\nmove /Y %q %q > nul\r\n", src, target)
	if err := os.WriteFile(script, []byte(body), 0600); err != nil {
		return fmt.Errorf("write update script: %w", err)
	}
	return exec.Command("cmd", "/C", "start", "", "/B", script).Start()
}

func scheduleWindowsDelete(target string) error {
	script := filepath.Join(os.TempDir(), cliName+"-uninstall.cmd")
	body := fmt.Sprintf("@echo off\r\nping 127.0.0.1 -n 2 > nul\r\ndel /F /Q %q > nul\r\n", target)
	if err := os.WriteFile(script, []byte(body), 0600); err != nil {
		return fmt.Errorf("write uninstall script: %w", err)
	}
	return exec.Command("cmd", "/C", "start", "", "/B", script).Start()
}

func latestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", cliName+"/"+version.Version)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch latest release: HTTP %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode latest release: %w", err)
	}
	return &rel, nil
}

func downloadTemp(url, name string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", name, resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", name+".*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmp.Close()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}
	return tmp.Name(), nil
}

func verifyChecksum(checksumURL, assetName, path string) error {
	resp, err := http.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download checksums: HTTP %d", resp.StatusCode)
	}

	expected := ""
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && strings.TrimPrefix(fields[1], "*") == assetName {
			expected = fields[0]
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}
	if expected == "" {
		return fmt.Errorf("checksums.txt does not contain %s", assetName)
	}

	actual, err := sha256File(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf("checksum mismatch for %s", assetName)
	}
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open checksum target: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash checksum target: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func pathContains(dir string) bool {
	want := cleanComparable(dir)
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if cleanComparable(p) == want {
			return true
		}
	}
	return false
}

func samePath(a, b string) bool {
	return cleanComparable(a) == cleanComparable(b)
}

func cleanComparable(path string) string {
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}
