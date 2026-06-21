package client

import (
	"path/filepath"
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
	// MSYSTEM set but MSYSTEM_PREFIX and EXEPATH both empty AND no default
	// install-root probes match -> cannot derive any root, must pass through.
	// We stub msysDefaultRoots so the test is hermetic (doesn't depend on
	// whether C:/Program Files/Git exists on the developer's machine).
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", "")
	t.Setenv("EXEPATH", "")
	stubMsysDefaultRoots(t, nil)

	in := "C:/Program Files/Git/foo"
	if got := normalizeRemotePath(in); got != in {
		t.Errorf("normalizeRemotePath(%q) = %q, want %q (no prefix, no default)", in, got, in)
	}
}

// stubMsysDefaultRoots swaps the package-level msysDefaultRoots hook for
// the duration of a test, restoring the production value on cleanup. Tests
// use this to make normalizeRemotePath hermetic — independent of whether
// a real Git-for-Windows install happens to exist on the developer's box.
func stubMsysDefaultRoots(t *testing.T, roots []string) {
	t.Helper()
	orig := msysDefaultRoots
	msysDefaultRoots = func() []string { return roots }
	t.Cleanup(func() { msysDefaultRoots = orig })
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

// TestNormalizeRemotePath_NoEnvRoot_DefaultHit covers the regression this
// fix targets: MSYSTEM_PREFIX and EXEPATH are both empty, yet we still
// recognise the MSYS-mangled root via default-install-location probing.
// On a Windows machine where C:/Program Files/Git exists, / -> that root
// must be reversed back to "/". The stub makes the test independent of
// whether the developer happens to have Git for Windows installed.
func TestNormalizeRemotePath_NoEnvRoot_DefaultHit(t *testing.T) {
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", "")
	t.Setenv("EXEPATH", "")

	fakeRoot := t.TempDir() // arbitrary but stable per-test path
	t.Setenv("ProgramFiles", filepath.Dir(fakeRoot))
	t.Setenv("ProgramFiles(x86)", filepath.Dir(fakeRoot))
	// The production defaultMsysRoots probes "C:/Program Files/Git" etc.,
	// not the temp dir — exercise the candidates list explicitly via the
	// stub so the assertion is about the prefix-matching logic, not OS luck.
	stubMsysDefaultRoots(t, []string{fakeRoot})

	cases := map[string]string{
		fakeRoot:          "/",
		fakeRoot + "/":     "/",
		fakeRoot + `\` + "/": "/",
		fakeRoot + "/foo":  "/foo",
		fakeRoot + "/x/y":  "/x/y",
	}
	for in, want := range cases {
		if got := normalizeRemotePath(in); got != want {
			t.Errorf("normalizeRemotePath(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestNormalizeRemotePath_NoEnvRoot_DefaultMiss confirms that when neither
// env-derived roots nor default candidates match, the path is forwarded
// unchanged. This is the safety net that keeps the fix from silently
// rewriting legitimate Windows paths in unrelated scenarios.
func TestNormalizeRemotePath_NoEnvRoot_DefaultMiss(t *testing.T) {
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", "")
	t.Setenv("EXEPATH", "")
	stubMsysDefaultRoots(t, []string{`C:\somewhere\unrelated`})

	in := `D:/Users/someone/file.txt`
	if got := normalizeRemotePath(in); got != in {
		t.Errorf("normalizeRemotePath(%q) = %q, want %q (no root match)", in, got, in)
	}
}

// TestMsysRootCandidates_EnvRootBeatsDefault pins the priority order:
// when MSYSTEM_PREFIX/EXEPATH successfully derive a root, that root wins
// and default probes are not consulted (preventing stale defaults from
// shadowing the live env-derived value).
func TestMsysRootCandidates_EnvRootBeatsDefault(t *testing.T) {
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", `C:\env-root\mingw64`)
	t.Setenv("EXEPATH", "")
	stubMsysDefaultRoots(t, []string{`C:\default-root`})

	got := msysRootCandidates()
	if len(got) != 1 || got[0] != `C:\env-root` {
		t.Errorf("msysRootCandidates() = %v, want only [C:\\env-root]", got)
	}
}

// TestMsysRootCandidates_NoEnv_FallsBackToDefault confirms that when no
// env-derived root is available, the default-roots hook is consulted.
func TestMsysRootCandidates_NoEnv_FallsBackToDefault(t *testing.T) {
	t.Setenv("MSYSTEM", "MINGW64")
	t.Setenv("MSYSTEM_PREFIX", "")
	t.Setenv("EXEPATH", "")
	stubMsysDefaultRoots(t, []string{`C:\fallback-root`})

	got := msysRootCandidates()
	if len(got) != 1 || got[0] != `C:\fallback-root` {
		t.Errorf("msysRootCandidates() = %v, want [C:\\fallback-root]", got)
	}
}
