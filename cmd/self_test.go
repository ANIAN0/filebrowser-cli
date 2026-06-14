package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestGithubReleaseAssetURL(t *testing.T) {
	rel := githubRelease{}
	rel.Assets = append(rel.Assets, struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	}{Name: "filebrowser-cli-linux-amd64", URL: "https://example.test/bin"})

	got, ok := rel.assetURL("filebrowser-cli-linux-amd64")
	if !ok || got != "https://example.test/bin" {
		t.Fatalf("assetURL() = %q, %v", got, ok)
	}
	if _, ok := rel.assetURL("missing"); ok {
		t.Fatal("assetURL() found missing asset")
	}
}

func TestSha256File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "payload")
	data := []byte("payload")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	got, err := sha256File(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	want := hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("sha256File() = %q, want %q", got, want)
	}
}

func TestSamePath(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a", "..", "file")
	b := filepath.Join(dir, "file")
	if !samePath(a, b) {
		t.Fatalf("samePath(%q, %q) = false, want true", a, b)
	}
}
