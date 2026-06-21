package client

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// normalizeRemotePath repairs remote FileBrowser paths that Git Bash / MSYS2
// mangles before the Go process even starts.
//
// On Windows under MSYS2, a POSIX path passed on the command line (e.g. "/",
// "/foo/bar") is rewritten to a Windows path rooted at the MSYS install dir
// (e.g. "C:/Program Files/Git/", "C:/Program Files/Git/foo/bar"). Because the
// conversion happens in the shell before argv reaches this program, the env
// var MSYS_NO_PATHCONV cannot prevent it from inside the process. We can only
// detect and reverse it after the fact.
//
// Detection is conservative on three axes:
//   - we only act on Windows;
//   - we only act when MSYSTEM is set (so we know argv came from an MSYS
//     shell rather than a regular Windows command);
//   - we only act when we can identify the MSYS install root (either via
//     MSYSTEM_PREFIX / EXEPATH envs, or — as a last-resort fallback — by
//     probing a small list of well-known default install locations).
//
// Legitimate Windows drive paths and plain POSIX paths are passed through
// unchanged whenever no root matches. Anything ambiguous is left untouched
// (safer to do nothing than to corrupt a real path).
func normalizeRemotePath(p string) string {
	if runtime.GOOS != "windows" || p == "" {
		return p
	}

	if os.Getenv("MSYSTEM") == "" {
		// Not (recognisably) an MSYS environment at all; cannot safely reverse.
		return p
	}

	roots := msysRootCandidates()
	if len(roots) == 0 {
		// MSYSTEM is set but we cannot derive any root — give up rather
		// than guess. Better to forward a Windows path verbatim than to
		// silently rewrite a real one.
		return p
	}

	normalized := normalizeForCompare(p)
	fl := normalizeForward(p)

	for _, root := range roots {
		rootNorm := normalizeForCompare(root)
		if normalized == rootNorm {
			return "/"
		}
		if !strings.HasPrefix(normalized, rootNorm+"/") {
			continue
		}
		// The case-insensitive prefix match already succeeded in compare form.
		// Now strip the same logical prefix from the forward-slashed original,
		// locating the boundary via case-insensitive comparison (the original
		// may have different casing than the detected root).
		prefix := normalizeForward(root)
		if len(fl) < len(prefix) || !strings.EqualFold(fl[:len(prefix)], prefix) {
			// Defensive: should be unreachable given the compare above, but if
			// it ever does, fall through to the next candidate rather than
			// corrupt the path on a single root miss.
			continue
		}
		tail := fl[len(prefix):]
		tail = strings.TrimPrefix(tail, "/")
		if tail == "" {
			return "/"
		}
		// path.Clean collapses "/./", "foo/../bar", and trailing slashes so
		// MSYS-mangled input always becomes a clean POSIX path. We preserve
		// case in the returned segments (path.Clean is case-preserving).
		if cleaned := path.Clean("/" + tail); cleaned != "" {
			return cleaned
		}
		return "/"
	}

	return p
}

// msysRootCandidates returns the ordered list of MSYS install roots to try
// when reversing a mangled path. The first entry is the authoritative one
// (derived from MSYSTEM_PREFIX / EXEPATH); any further entries are
// well-known default locations, used only as a last-resort fallback when
// the env-derived root is unavailable. The list may be empty when nothing
// is recognisable, in which case normalizeRemotePath leaves its input alone.
func msysRootCandidates() []string {
	if r := msysRoot(); r != "" {
		return []string{r}
	}
	return msysDefaultRoots()
}

// msysDefaultRoots is a package-level function variable so tests can stub
// the default MSYS install-root lookup. Production returns a small list of
// well-known Git for Windows / MSYS2 install paths that exist on disk;
// non-existent entries are silently dropped.
var msysDefaultRoots = defaultMsysRoots

// defaultMsysRoots probes a small set of well-known MSYS install locations.
// It returns only paths that currently exist on disk so a stale or
// user-specific install location never silently wins over the env-derived
// root (msysRoot) — that one already short-circuits above.
//
// Why this exists: in some shells (e.g. `bash -c "..."` spawned from CI or
// from this very CLI tool) the MSYS login profile isn't sourced, so neither
// MSYSTEM_PREFIX nor EXEPATH is set, yet MSYSTEM is. Without this fallback,
// `filebrowser-cli ls /` arrives as argv "C:/Program Files/Git/" and goes
// straight to the server, which then 401s / 404s on a Windows path the
// remote FileBrowser has no concept of.
func defaultMsysRoots() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Git"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Git"),
		`C:\Program Files\Git`,
		`C:\Program Files (x86)\Git`,
		`C:\msys64`,
	}
	var found []string
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if _, err := os.Stat(c); err == nil {
			found = append(found, c)
		}
	}
	return found
}

// msysRoot returns the MSYS2 install root when running inside an MSYS shell,
// or "" otherwise.
//
// MSYSTEM_PREFIX points at something like "C:/Program Files/Git/mingw64" (Git
// for Windows) or "/mingw64" (native MSYS2); its parent directory is the MSYS
// root. We fall back to EXEPATH (set by Git for Windows to "<root>/bin") when
// MSYSTEM_PREFIX is empty or unusable, which happens in non-interactive shells
// (e.g. `bash -c "..."` spawned by pi / Claude Code / CI) where the MSYS login
// profile is not sourced.
func msysRoot() string {
	if os.Getenv("MSYSTEM") == "" {
		return ""
	}

	if prefix := os.Getenv("MSYSTEM_PREFIX"); prefix != "" {
		if abs := toWindowsAbs(prefix); abs != "" {
			return filepath.Dir(abs)
		}
	}

	// Fallback: Git for Windows sets EXEPATH="<root>\bin" even in non-interactive
	// shells, while MSYSTEM_PREFIX is empty. Take the parent to get the MSYS root.
	if exePath := os.Getenv("EXEPATH"); exePath != "" {
		if abs := toWindowsAbs(exePath); abs != "" {
			return filepath.Dir(abs)
		}
	}

	return ""
}

// toWindowsAbs resolves a possibly-MSYS-style path to an absolute Windows path.
// Returns "" if it cannot be resolved.
func toWindowsAbs(p string) string {
	if p == "" {
		return ""
	}
	// MSYSTEM_PREFIX is normally already an absolute Windows path under Git
	// for Windows (e.g. "C:/Program Files/Git/mingw64"). Abs cleans it.
	abs, err := filepath.Abs(p)
	if err != nil {
		return ""
	}
	return abs
}

// normalizeForCompare produces a canonical, lowercased, forward-slashed,
// cleaned form suitable only for equality/prefix comparison.
func normalizeForCompare(p string) string {
	s := normalizeForward(p)
	s = path.Clean(s)
	if runtime.GOOS == "windows" {
		s = strings.ToLower(s)
	}
	return s
}

// normalizeForward converts backslashes to forward slashes but does not clean,
// so callers can split/trim with predictable separators.
func normalizeForward(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
