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
	// MSYSTEM set but MSYSTEM_PREFIX and EXEPATH both empty -> cannot derive root,
	// pass through (defensive: with EXEPATH fallback added, must keep this
	// contract so unknown environments stay inert).
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", "")
	t.Setenv("EXEPATH", "")

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

// TestMsysRoot_ExepathFallback covers the non-interactive bash case (pi /
// Claude Code / CI): MSYSTEM is set but MSYSTEM_PREFIX is empty because the
// MSYS login profile is not sourced. Git for Windows still exports
// EXEPATH="<root>\bin", so msysRoot must derive the install root from it.
func TestMsysRoot_ExepathFallback(t *testing.T) {
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", "")
	t.Setenv("EXEPATH", `C:\Program Files\Git\bin`)

	if got := msysRoot(); got != `C:\Program Files\Git` {
		t.Errorf("msysRoot() = %q, want %q", got, `C:\Program Files\Git`)
	}
}

func TestNormalizeRemotePath_ExepathMangled_Root(t *testing.T) {
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", "")
	t.Setenv("EXEPATH", `C:\Program Files\Git\bin`)

	// This is the exact shape of the 404 bug: `/` mangled to MSYS install root.
	if got := normalizeRemotePath(`C:\Program Files\Git`); got != "/" {
		t.Errorf("normalizeRemotePath(<root>) = %q, want \"/\"", got)
	}
	if got := normalizeRemotePath(`C:\Program Files\Git\`); got != "/" {
		t.Errorf("normalizeRemotePath(<root>\\) = %q, want \"/\"", got)
	}
}

func TestNormalizeRemotePath_ExepathMangled_Subpaths(t *testing.T) {
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", "")
	t.Setenv("EXEPATH", `C:\Program Files\Git\bin`)

	cases := map[string]string{
		`C:\Program Files\Git\foo`:        "/foo",
		`C:/Program Files/Git/foo/bar`:    "/foo/bar",
		`C:/Program Files/Git/foo/`:       "/foo",
		`C:/Program Files/Git/foo/../bar`: "/bar",
	}
	for in, want := range cases {
		if got := normalizeRemotePath(in); got != want {
			t.Errorf("normalizeRemotePath(%q) = %q, want %q", in, got, want)
		}
	}
}

// Negative-case companions to TestMsysRoot_ExepathFallback: pin the
// "conservative no-op" contract. When MSYS is absent OR when both
// MSYSTEM_PREFIX and EXEPATH are empty, msysRoot must return "" so that
// normalizeRemotePath leaves input untouched. This guards against future
// "helpful" fallbacks that would silently rewrite legitimate paths.

// TestMsysRoot_NoMsystem_ReturnsEmpty: outside any MSYS shell (Linux, macOS,
// or Windows cmd.exe), MSYSTEM is unset, so msysRoot must short-circuit.
func TestMsysRoot_NoMsystem_ReturnsEmpty(t *testing.T) {
	t.Setenv("MSYSTEM", "")
	t.Setenv("MSYSTEM_PREFIX", "")
	t.Setenv("EXEPATH", "")

	if got := msysRoot(); got != "" {
		t.Errorf("msysRoot() with no MSYS env = %q, want \"\"", got)
	}
}

// TestMsysRoot_MsystemSetNoRootEnv_ReturnsEmpty: MSYSTEM is set (so we know
// we are in MSYS) but neither MSYSTEM_PREFIX nor EXEPATH is exported. This
// can happen on minimal MSYS2 installs or sandboxed CI. We cannot safely
// derive the root, so we must return "" and let normalizeRemotePath pass
// the input through unchanged.
func TestMsysRoot_MsystemSetNoRootEnv_ReturnsEmpty(t *testing.T) {
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", "")
	t.Setenv("EXEPATH", "")

	if got := msysRoot(); got != "" {
		t.Errorf("msysRoot() with MSYSTEM but no root envs = %q, want \"\"", got)
	}
}
