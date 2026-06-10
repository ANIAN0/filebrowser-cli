package config

import (
	"fmt"
	"os"
	"regexp"
)

// envVarPattern matches ${ENV_VAR} patterns.
// Valid env var names: start with letter or underscore, followed by letters, digits, or underscores.
var envVarPattern = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

// Interpolate replaces ${ENV_VAR} with the value from the provided getenv function.
// Only string fields should call this function.
// Returns error if any referenced env var is not set (empty string is treated as "not set").
//
// Examples:
//
//	"${FB_PASSWORD}" with FB_PASSWORD=secret → "secret"
//	"${FB_PASSWORD}" without FB_PASSWORD set → error
//	"prefix_${X}_suffix" with X=hello → "prefix_hello_suffix"
func Interpolate(s string, getenv func(string) string) (string, error) {
	var missing []string

	result := envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract env var name from ${NAME}
		name := match[2 : len(match)-1]
		val := getenv(name)
		if val == "" {
			missing = append(missing, name)
			return match // Keep original on first pass
		}
		return val
	})

	if len(missing) > 0 {
		return "", fmt.Errorf("env var(s) not set: %v", missing)
	}
	return result, nil
}

// InterpolateFromOS uses os.LookupEnv to resolve environment variables.
// Empty values are treated as "not set".
func InterpolateFromOS(s string) (string, error) {
	return Interpolate(s, func(name string) string {
		v, _ := os.LookupEnv(name)
		return v
	})
}

// HasEnvVars checks if a string contains any ${ENV_VAR} patterns.
func HasEnvVars(s string) bool {
	return envVarPattern.MatchString(s)
}

// ExtractEnvVars returns a list of env var names referenced in the string.
func ExtractEnvVars(s string) []string {
	matches := envVarPattern.FindAllStringSubmatch(s, -1)
	var names []string
	for _, m := range matches {
		if len(m) > 1 {
			names = append(names, m[1])
		}
	}
	return names
}