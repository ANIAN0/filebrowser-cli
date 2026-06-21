package client

import (
	"net/url"
	"path"
	"strings"
)

// buildResourceURL constructs the relative URL for a FileBrowser resource API
// call. It is the single entry point for all resource endpoints (List/Info/
// Upload/Download/Mkdir/Remove/Move/Copy/Preview), preventing each command
// from re-implementing path-joining and encoding.
//
// Behavior:
//   - The remote path is MSYS-normalized via normalizeRemotePath so that
//     `filebrowser-cli ls /` works under Git Bash non-interactive shells.
//   - The path is canonicalized (path.Clean) and forced to start with "/".
//   - Each path segment is url.PathEscape'd; "/" separators are preserved so
//     the request targets the correct sub-resource rather than collapsing
//     into a single URL segment.
//   - When query is non-empty it is appended via query.Encode (url.Values),
//     producing the right query-string encoding for action/destination/
//     override/etc.
//
// The endpoint should be the bare API prefix (e.g. "/api/resources"); the
// leading "/" between endpoint and path is always inserted here, so callers
// must NOT include a trailing slash on endpoint.
func buildResourceURL(endpoint, remotePath string, query url.Values) string {
	cleaned := normalizeRemotePath(remotePath)
	cleaned = ensureLeadingSlash(cleaned)
	cleaned = escapePathSegments(cleaned)

	u := endpoint + cleaned
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return u
}

// ensureLeadingSlash guarantees the path begins with "/", preventing
// concatenation bugs like "/api/resources" + "foo" → "/api/resourcesfoo".
// If the input had a single trailing "/", that is preserved after Clean
// (used by Mkdir, which the FileBrowser server requires to end in "/").
func ensureLeadingSlash(p string) string {
	if p == "" {
		return "/"
	}
	trailing := strings.HasSuffix(p, "/")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = path.Clean(p)
	if trailing && p != "/" {
		p += "/"
	}
	return p
}

// escapePathSegments splits on "/", url.PathEscape's each segment, then
// rejoins. This preserves the "/" structure of the path while escaping
// reserved characters inside each segment (spaces, ?, &, #, %, etc.).
func escapePathSegments(p string) string {
	trimmed := strings.TrimPrefix(p, "/")
	if trimmed == "" {
		return "/"
	}
	segments := strings.Split(trimmed, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return "/" + strings.Join(segments, "/")
}