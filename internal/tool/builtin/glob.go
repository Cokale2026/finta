package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"finta/internal/tool"
)

type GlobTool struct{}

func NewGlobTool() *GlobTool {
	return &GlobTool{}
}

func (t *GlobTool) Name() string {
	return "glob"
}

func (t *GlobTool) Description() string {
	return "Find files matching a glob pattern"
}

func (t *GlobTool) BestPractices() string {
	return `**Glob Tool Best Practices**:

1. **Use specific patterns** - Narrow patterns are faster and more precise
   - Good: {"pattern": "**/*.go"} (only Go files)
   - Avoid: {"pattern": "**"} (all files, can be slow in large directories)

2. **Use recursive ** for deep searches** - Search all subdirectories efficiently
   - Recursive: {"pattern": "internal/**/*.go"} (all Go files under internal/)
   - Non-recursive: {"pattern": "*.go"} (only current directory)

3. **Combine with path parameter** - Limit search scope for better performance
   - Scoped: {"pattern": "**/*.ts", "path": "src"} (only search src/)
   - Unscoped: {"pattern": "src/**/*.ts"} (equivalent but less clear)

4. **Use specific prefixes** - Reduce search space with directory prefixes
   - Efficient: {"pattern": "cmd/**/*.go"} (only search cmd/)
   - Inefficient: {"pattern": "**/*.go"} then filter results manually

5. **Match file extensions precisely** - Use explicit extensions for clarity
   - Precise: {"pattern": "**/*.test.ts"} (test files only)
   - Imprecise: {"pattern": "**/*test*"} (may match non-test files)

6. **Check result count before processing** - Use Data.count to validate matches
   - Pattern not found returns "No files found" with count = 0`
}

func (t *GlobTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern (e.g., '*.go', 'src/**/*.ts')",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Base path to search (default: current directory)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GlobTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
	var p struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	basePath := p.Path
	if basePath == "" {
		basePath = "."
	}

	var matches []string
	var err error

	// Check if pattern contains ** (recursive glob)
	if strings.Contains(p.Pattern, "**") {
		matches, err = t.recursiveGlob(basePath, p.Pattern)
	} else {
		// Use standard filepath.Glob for non-recursive patterns
		fullPattern := filepath.Join(basePath, p.Pattern)
		matches, err = filepath.Glob(fullPattern)
	}

	if err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("glob failed: %v", err),
		}, nil
	}

	if len(matches) == 0 {
		return &tool.Result{
			Success: true,
			Output:  "No files found",
			Data: map[string]any{
				"count": 0,
				"files": []string{},
			},
		}, nil
	}

	// Sort for deterministic output
	sort.Strings(matches)

	return &tool.Result{
		Success: true,
		Output:  strings.Join(matches, "\n"),
		Data: map[string]any{
			"count": len(matches),
			"files": matches,
		},
	}, nil
}

// recursiveGlob implements ** recursive pattern matching using filepath.WalkDir
func (t *GlobTool) recursiveGlob(basePath, pattern string) ([]string, error) {
	var matches []string

	// Split pattern by ** to get the suffix pattern
	// e.g., "**/*.go" -> suffix = "*.go"
	// e.g., "src/**/*.ts" -> prefix = "src", suffix = "*.ts"
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ** pattern: %s (only one ** supported)", pattern)
	}

	prefix := strings.Trim(parts[0], "/")
	suffix := strings.Trim(parts[1], "/")

	// Determine the starting directory
	searchRoot := basePath
	if prefix != "" {
		searchRoot = filepath.Join(basePath, prefix)
	}

	// Walk the directory tree
	err := filepath.WalkDir(searchRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't read
			return nil
		}

		// Skip directories in matching (only match files)
		if d.IsDir() {
			return nil
		}

		// Match against the suffix pattern
		if suffix == "" {
			// "**" matches all files
			matches = append(matches, path)
		} else {
			// Get relative path from searchRoot
			relPath, err := filepath.Rel(searchRoot, path)
			if err != nil {
				return nil
			}

			// Match the relative path against the suffix pattern
			matched, err := filepath.Match(suffix, filepath.Base(path))
			if err != nil {
				return err
			}

			// Also try matching the full relative path for patterns like "test/*.go"
			if !matched {
				matched, err = filepath.Match(suffix, relPath)
				if err != nil {
					return err
				}
			}

			if matched {
				matches = append(matches, path)
			}
		}

		return nil
	})

	return matches, err
}
