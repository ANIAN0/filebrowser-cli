package client

import (
	"net/url"
	"testing"
)

func TestBuildResourceURL_Positive(t *testing.T) {
	q := url.Values{}
	q.Set("checksum", "sha256")

	cases := []struct {
		name, endpoint, path, want string
		query                      url.Values
	}{
		{"root slash", "/api/resources", "/", "/api/resources/", nil},
		{"root empty", "/api/resources", "", "/api/resources/", nil},
		{"subpath", "/api/resources", "/foo", "/api/resources/foo", nil},
		{"subpath with query", "/api/resources", "/foo", "/api/resources/foo?checksum=sha256", q},
		{"space encoded", "/api/resources", "/a b c", "/api/resources/a%20b%20c", nil},
		{"question mark encoded", "/api/resources", "/a?b", "/api/resources/a%3Fb", nil},
		{"hash encoded", "/api/resources", "/a#b", "/api/resources/a%23b", nil},
		// Note: url.PathEscape leaves '&' unescaped (sub-delim, allowed in path per RFC 3986).
		// The server parser handles it correctly since the path is bounded by '?' on the query side.
		{"ampersand left as-is", "/api/resources", "/a&b", "/api/resources/a&b", nil},
		{"percent encoded", "/api/resources", "/a%b", "/api/resources/a%25b", nil},
		{"chinese", "/api/resources", "/文件", "/api/resources/%E6%96%87%E4%BB%B6", nil},
		{"dot collapsed", "/api/resources", "/foo/./bar", "/api/resources/foo/bar", nil},
		{"dotdot collapsed", "/api/resources", "/foo/../bar", "/api/resources/bar", nil},
		{"multi segment query", "/api/resources", "/foo",
			"/api/resources/foo?action=rename&destination=%2Fbar", func() url.Values {
				q := url.Values{}
				q.Set("action", "rename")
				q.Set("destination", "/bar")
				return q
			}()},
		// url.Values.Encode() uses "+" for spaces (form-style), unlike
		// url.QueryEscape which uses "%20". FileBrowser's Go server accepts
		// both in query values, so this is the documented output. Pin it so
		// any future change to a strict %20 encoder (e.g. switching to
		// url.QueryEscape) is caught.
		{"destination with space", "/api/resources", "/foo", "/api/resources/foo?action=rename&destination=%2Ffoo+bar", func() url.Values {
			q := url.Values{}
			q.Set("action", "rename")
			q.Set("destination", "/foo bar")
			return q
		}()},
		{"preview endpoint with size in path handled by callers", "/api/preview/thumb", "/foo.png", "/api/preview/thumb/foo.png", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := buildResourceURL(c.endpoint, c.path, c.query)
			if got != c.want {
				t.Errorf("buildResourceURL(%q, %q, _) = %q, want %q", c.endpoint, c.path, got, c.want)
			}
		})
	}
}

// TestBuildResourceURL_NoConcatBug is the negative case proving the builder
// fixes the concatenation bug: raw "/api/resources"+path must never produce
// "/api/resourcesfoo". The builder must guarantee a "/" separator.
func TestBuildResourceURL_NoConcatBug(t *testing.T) {
	u := buildResourceURL("/api/resources", "foo", nil)
	if u == "/api/resourcesfoo" {
		t.Fatalf("regression: builder produced concatenated URL %q", u)
	}
	if u != "/api/resources/foo" {
		t.Errorf("buildResourceURL(..., \"foo\") = %q, want %q (leading slash forced)", u, "/api/resources/foo")
	}
}

// TestEnsureLeadingSlash documents the leading-slash guarantee.
func TestEnsureLeadingSlash(t *testing.T) {
	cases := map[string]string{
		"":       "/",
		"/":      "/",
		"foo":    "/foo",
		"/foo":   "/foo",
		"//":     "/",
		"/a/./b": "/a/b",
		"/foo/":  "/foo/", // trailing slash preserved (Mkdir needs this)
		"/a/b/":  "/a/b/",
	}
	for in, want := range cases {
		if got := ensureLeadingSlash(in); got != want {
			t.Errorf("ensureLeadingSlash(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestEscapePathSegments confirms each segment is escaped but "/" is preserved.
func TestEscapePathSegments(t *testing.T) {
	cases := map[string]string{
		"/":        "/",
		"/foo":     "/foo",
		"/a b":     "/a%20b",
		"/a/b/c":   "/a/b/c",
		"/a b/c d": "/a%20b/c%20d",
	}
	for in, want := range cases {
		if got := escapePathSegments(in); got != want {
			t.Errorf("escapePathSegments(%q) = %q, want %q", in, got, want)
		}
	}
}
