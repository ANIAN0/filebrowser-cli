package client

import (
	"runtime"
	"testing"
)

// setMsysEnv sets the environment variables that make normalizeRemotePath
// treat the process as running under MSYS2. The cleanup unsets them.
//
// On non-Windows platforms the MSYS branch is inert (normalizeRemotePath
// returns early for runtime.GOOS != "windows"), so these tests still pass
// but exercise the non-MSYS passthrough path instead.
func setMsysEnv(t *testing.T, prefix string) {
	t.Helper()
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", prefix)
}

func TestNormalizeRemotePath_NotWindows_Passthrough(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-windows behavior test")
	}
	// POSIX paths pass through unchanged on non-Windows.
	cases := []string{"/", "/foo", "/foo/bar", ""}
	for _, in := range cases {
		if got := normalizeRemotePath(in); got != in {
			t.Errorf("normalizeRemotePath(%q) = %q, want %q", in, got, in)
		}
	}
}

func TestNormalizeRemotePath_NoMsysEnv_Passthrough(t *testing.T) {
	// Without MSYSTEM set, nothing should be reversed even on Windows.
	t.Setenv("MSYSTEM", "")
	t.Setenv("MSYSTEM_PREFIX", "")

	cases := []string{
		"C:/Program Files/Git",
		"C:/Program Files/Git/foo",
		"/",
		"/foo/bar",
	}
	for _, in := range cases {
		if got := normalizeRemotePath(in); got != in {
			t.Errorf("normalizeRemotePath(%q) = %q, want %q (no MSYS env)", in, got, in)
		}
	}
}

func TestNormalizeRemotePath_NoPrefix_Passthrough(t *testing.T) {
	// MSYSTEM set but MSYSTEM_PREFIX empty -> cannot derive root, pass through.
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", "")

	in := "C:/Program Files/Git/foo"
	if got := normalizeRemotePath(in); got != in {
		t.Errorf("normalizeRemotePath(%q) = %q, want %q (no prefix)", in, got, in)
	}
}

func TestNormalizeRemotePath_EmptyString(t *testing.T) {
	setMsysEnv(t, `C:\Program Files\Git\mingw64`)
	if got := normalizeRemotePath(""); got != "" {
		t.Errorf("normalizeRemotePath(\"\") = %q, want \"\"", got)
	}
}

func TestNormalizeRemotePath_RootToSlash(t *testing.T) {
	setMsysEnv(t, `C:\Program Files\Git\mingw64`)
	// MSYS root is C:/Program Files/Git (parent of mingw64).
	root := `C:/Program Files/Git`
	if got := normalizeRemotePath(root); got != "/" {
		t.Errorf("normalizeRemotePath(%q) = %q, want \"/\"", root, got)
	}
	// Trailing slash variants should also map to "/".
	for _, in := range []string{`C:/Program Files/Git/`, `C:\Program Files\Git\`} {
		if got := normalizeRemotePath(in); got != "/" {
			t.Errorf("normalizeRemotePath(%q) = %q, want \"/\"", in, got)
		}
	}
}

func TestNormalizeRemotePath_RootChild(t *testing.T) {
	setMsysEnv(t, `C:\Program Files\Git\mingw64`)
	cases := map[string]string{
		`C:/Program Files/Git/foo`:        "/foo",
		`C:/Program Files/Git/foo/bar`:    "/foo/bar",
		`C:\Program Files\Git\foo`:        "/foo",
		`C:/Program Files/Git/foo/`:       "/foo",
		`C:/Program Files/Git/./foo`:      "/foo",
		`C:/Program Files/Git/foo/../bar`: "/bar",
	}
	for in, want := range cases {
		if got := normalizeRemotePath(in); got != want {
			t.Errorf("normalizeRemotePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeRemotePath_PlainPosixPassthrough(t *testing.T) {
	setMsysEnv(t, `C:\Program Files\Git\mingw64`)
	// Real POSIX paths (the desired form) must never be touched.
	cases := []string{"/", "/foo", "/foo/bar", "/a/b/c"}
	for _, in := range cases {
		if got := normalizeRemotePath(in); got != in {
			t.Errorf("normalizeRemotePath(%q) = %q, want %q (plain posix)", in, got, in)
		}
	}
}

func TestNormalizeRemotePath_UnrelatedWindowsPathPassthrough(t *testing.T) {
	setMsysEnv(t, `C:\Program Files\Git\mingw64`)
	// A Windows drive path that is NOT under the MSYS root must not be altered
	// (it could be a legitimately different volume on the remote server).
	cases := []string{
		`D:/some/other/path`,
		`C:/Users/someone/file.txt`,
	}
	for _, in := range cases {
		if got := normalizeRemotePath(in); got != in {
			t.Errorf("normalizeRemotePath(%q) = %q, want %q (unrelated)", in, got, in)
		}
	}
}

func TestNormalizeRemotePath_CaseInsensitiveRootMatch(t *testing.T) {
	setMsysEnv(t, `c:\program files\git\mingw64`)
	// Root detected with different casing should still reverse.
	in := `C:/PROGRAM FILES/Git/foo`
	if got := normalizeRemotePath(in); got != "/foo" {
		t.Errorf("normalizeRemotePath(%q) = %q, want \"/foo\" (case-insensitive)", in, got)
	}
}
