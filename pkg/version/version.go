// Package version provides build-time version information.
// Values are injected via -ldflags at build time:
//
//	go build -ldflags "-X cli/shared/version.Version=v1.0.0 \
//	                  -X cli/shared/version.Commit=$(git rev-parse HEAD) \
//	                  -X cli/shared/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
package version

import "fmt"

// These variables are set at build time via -ldflags.
var (
	// Version is the semantic version string (e.g., "v1.0.0").
	// Default is "dev" for development builds.
	Version = "dev"

	// Commit is the git commit hash.
	// Default is "unknown" for development builds.
	Commit = "unknown"

	// BuildTime is the UTC build timestamp in RFC 3339 format.
	// Default is empty for development builds.
	BuildTime = ""
)

// String returns a formatted version string for --version output.
// Format: "v1.0.0 (abc123, built 2024-01-01T00:00:00Z)"
func String() string {
	if BuildTime == "" {
		return fmt.Sprintf("%s (%s)", Version, Commit)
	}
	return fmt.Sprintf("%s (%s, built %s)", Version, Commit, BuildTime)
}

// Info returns a struct with version information.
type Info struct {
	Version   string
	Commit    string
	BuildTime string
}

// Get returns the version information as a struct.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
	}
}