//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	fb "github.com/ANIAN0/filebrowser-cli/internal/client"
	"github.com/ANIAN0/filebrowser-cli/pkg/httpclient"
)

// skipIfNoFB skips the test if FILEBROWSER_TEST_* env vars are not set.
func skipIfNoFB(t *testing.T) (url, user, pwd string) {
	t.Helper()
	url = os.Getenv("FILEBROWSER_TEST_URL")
	user = os.Getenv("FILEBROWSER_TEST_USER")
	pwd = os.Getenv("FILEBROWSER_TEST_PASSWORD")
	if url == "" || user == "" || pwd == "" {
		t.Skip("set FILEBROWSER_TEST_URL, FILEBROWSER_TEST_USER, FILEBROWSER_TEST_PASSWORD to run integration tests")
	}
	return url, user, pwd
}

// newClient creates a new HTTP client for testing.
func newClient(t *testing.T, url string) *httpclient.Client {
	t.Helper()
	return httpclient.New(url,
		httpclient.WithTimeout(60*time.Second),
		httpclient.WithVerbose(true),
	)
}

// TestE2E_LoginListUploadShareDownload tests the full workflow:
// login -> list -> upload -> info -> share -> download -> search -> cleanup
func TestE2E_LoginListUploadShareDownload(t *testing.T) {
	url, user, pwd := skipIfNoFB(t)
	ctx := context.Background()

	// Create client
	c := newClient(t, url)

	// Login
	ac := &fb.AuthClient{C: c}
	token, err := ac.Login(ctx, user, pwd)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if token == "" {
		t.Fatal("login returned empty token")
	}
	t.Logf("login successful, token: %s...", token[:min(10, len(token))])

	// List root directory
	rc := &fb.ResourceClient{C: c}
	root, err := rc.List(ctx, "/")
	if err != nil {
		t.Fatalf("list root failed: %v", err)
	}
	t.Logf("root directory has %d items", len(root.Items))

	// Create test file
	tmpDir := t.TempDir()
	testContent := "integration test content " + time.Now().Format(time.RFC3339Nano)
	testFile := filepath.Join(tmpDir, "test-fixture.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("create test file failed: %v", err)
	}

	// Upload file
	remotePath := "/integration-test-file.txt"
	if err := rc.Upload(ctx, testFile, remotePath, true); err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	t.Logf("uploaded to %s", remotePath)

	// Cleanup: delete remote file after test
	t.Cleanup(func() {
		ctx := context.Background()
		_ = rc.Remove(ctx, remotePath)
		t.Logf("cleanup: removed %s", remotePath)
	})

	// Get file info
	info, err := rc.Info(ctx, remotePath)
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}
	if info.Size != int64(len(testContent)) {
		t.Errorf("info.Size = %d, want %d", info.Size, len(testContent))
	}
	t.Logf("file info: size=%d, modified=%s", info.Size, info.Modified)

	// Create share
	sc := &fb.ShareClient{C: c}
	share, err := sc.Create(ctx, remotePath, "", "", "")
	if err != nil {
		t.Fatalf("share create failed: %v", err)
	}
	if share.Hash == "" {
		t.Fatal("share create returned empty hash")
	}
	t.Logf("share created: hash=%s", share.Hash)

	// Cleanup: delete share after test
	t.Cleanup(func() {
		ctx := context.Background()
		_ = sc.Delete(ctx, share.Hash)
		t.Logf("cleanup: deleted share %s", share.Hash)
	})

	// Download via share
	downloadPath := filepath.Join(tmpDir, "downloaded.txt")
	err = c.Download(ctx, "/api/public/dl/"+share.Hash, downloadPath)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}

	// Verify downloaded content
	downloadedContent, err := os.ReadFile(downloadPath)
	if err != nil {
		t.Fatalf("read downloaded file failed: %v", err)
	}
	if string(downloadedContent) != testContent {
		t.Errorf("downloaded content mismatch:\n  got:  %q\n  want: %q", string(downloadedContent), testContent)
	}
	t.Logf("download verified: %d bytes", len(downloadedContent))

	// Search for the uploaded file
	src := &fb.SearchClient{C: c}
	results, err := src.Search(ctx, "/", "integration-test-file", 100)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	found := false
	for _, r := range results {
		if r.Path == remotePath {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("uploaded file not found in search results (got %d results)", len(results))
	} else {
		t.Logf("search found uploaded file")
	}
}

// TestE2E_MkdirMoveCopyDelete tests directory and file operations.
func TestE2E_MkdirMoveCopyDelete(t *testing.T) {
	url, user, pwd := skipIfNoFB(t)
	ctx := context.Background()

	c := newClient(t, url)

	// Login
	ac := &fb.AuthClient{C: c}
	_, err := ac.Login(ctx, user, pwd)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	rc := &fb.ResourceClient{C: c}

	// Create directory
	dirPath := "/test-integration-dir"
	if err := rc.Mkdir(ctx, dirPath); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	t.Cleanup(func() { rc.Remove(ctx, dirPath) })

	// Create and upload a file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "move-test.txt")
	os.WriteFile(testFile, []byte("move test"), 0644)

	srcPath := dirPath + "/source.txt"
	if err := rc.Upload(ctx, testFile, srcPath, true); err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	t.Cleanup(func() { rc.Remove(ctx, srcPath) })

	// Copy file
	dstPath := dirPath + "/copy.txt"
	if err := rc.Copy(ctx, srcPath, dstPath); err != nil {
		t.Fatalf("copy failed: %v", err)
	}
	t.Cleanup(func() { rc.Remove(ctx, dstPath) })

	// Verify copy exists
	info, err := rc.Info(ctx, dstPath)
	if err != nil {
		t.Fatalf("info copy failed: %v", err)
	}
	if info.Size != 9 { // len("move test")
		t.Errorf("copy size = %d, want 9", info.Size)
	}

	// Move copy to new location
	movedPath := dirPath + "/moved.txt"
	if err := rc.Move(ctx, dstPath, movedPath); err != nil {
		t.Fatalf("move failed: %v", err)
	}
	t.Cleanup(func() { rc.Remove(ctx, movedPath) })

	// Verify moved file exists
	_, err = rc.Info(ctx, movedPath)
	if err != nil {
		t.Fatalf("info moved failed: %v", err)
	}

	// Delete moved file
	if err := rc.Remove(ctx, movedPath); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Verify deleted
	_, err = rc.Info(ctx, movedPath)
	if err == nil {
		t.Error("expected error after delete, but got nil")
	}
}

// min returns the smaller of a or b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}