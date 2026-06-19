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
// Detection is conservative: it only acts when both
//   - we are on Windows, AND
//   - an MSYS environment is present (MSYSTEM set), AND
//   - we can derive the MSYS install root (from MSYSTEM_PREFIX's parent)
//
// so that legitimate Windows drive paths and plain POSIX paths are passed
// through unchanged. Anything ambiguous is left untouched (safer to do nothing
// than to corrupt a real path).
func normalizeRemotePath(p string) string {
	if runtime.GOOS != "windows" || p == "" {
		return p
	}

	root := msysRoot()
	if root == "" {
		// Not (recognisably) an MSYS environment; cannot safely reverse.
		return p
	}

	normalized := normalizeForCompare(p)
	rootNorm := normalizeForCompare(root)

	if normalized == rootNorm {
		return "/"
	}
	if strings.HasPrefix(normalized, rootNorm+"/") {
		// The case-insensitive prefix match already succeeded in compare form.
		// Now strip the same logical prefix from the forward-slashed original,
		// locating the boundary via case-insensitive comparison (the original
		// may have different casing than MSYSTEM_PREFIX).
		fl := normalizeForward(p)
		prefix := normalizeForward(root)
		if len(fl) < len(prefix) || !strings.EqualFold(fl[:len(prefix)], prefix) {
			// Defensive: should be unreachable given the compare above, but if
			// it ever does, fall back to the raw path rather than corrupt it.
			return p
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

// msysRoot returns the MSYS2 install root when running inside an MSYS shell,
// or "" otherwise.
//
// MSYSTEM_PREFIX points at something like "C:/Program Files/Git/mingw64" (Git
// for Windows) or "/mingw64" (native MSYS2); its parent directory is the MSYS
// root. We fall back to MSYSTEM as a last resort only when it is an absolute
// Windows path, since MSYSTEM alone (e.g. "MINGW64") is just a label, not a
// path.
func msysRoot() string {
	if os.Getenv("MSYSTEM") == "" {
		return ""
	}

	if prefix := os.Getenv("MSYSTEM_PREFIX"); prefix != "" {
		abs := toWindowsAbs(prefix)
		if abs != "" {
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
