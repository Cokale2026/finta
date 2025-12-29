package config

import (
	"os"
	"regexp"
)

// envVarPattern matches ${VAR} and $VAR patterns
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}|\$([A-Za-z0-9_]+)`)

// ExpandEnv replaces ${VAR} and $VAR with environment variables
// Example: "Bearer ${GITHUB_TOKEN}" → "Bearer ghp_abc123..."
func ExpandEnv(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR} or $VAR
		varName := ""
		if match[1] == '{' {
			// ${VAR} format
			varName = match[2 : len(match)-1]
		} else {
			// $VAR format
			varName = match[1:]
		}

		// Return environment variable value, or empty string if not set
		return os.Getenv(varName)
	})
}

// ExpandEnvMap expands all values in a map
func ExpandEnvMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}

	expanded := make(map[string]string, len(m))
	for key, value := range m {
		expanded[key] = ExpandEnv(value)
	}
	return expanded
}
