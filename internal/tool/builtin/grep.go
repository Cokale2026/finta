package builtin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"finta/internal/tool"
)

type GrepTool struct{}

func NewGrepTool() *GrepTool {
	return &GrepTool{}
}

func (t *GrepTool) Name() string {
	return "grep"
}

func (t *GrepTool) Description() string {
	return "Search for content in files using regex patterns"
}

func (t *GrepTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Regular expression pattern to search for",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File or directory to search in",
			},
			"case_insensitive": map[string]any{
				"type":        "boolean",
				"description": "Case-insensitive search (default: false)",
			},
			"file_pattern": map[string]any{
				"type":        "string",
				"description": "Filter files by pattern (e.g., '*.go')",
			},
		},
		"required": []string{"pattern", "path"},
	}
}

func (t *GrepTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
	var p struct {
		Pattern         string `json:"pattern"`
		Path            string `json:"path"`
		CaseInsensitive bool   `json:"case_insensitive"`
		FilePattern     string `json:"file_pattern"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	// Compile regex pattern
	regexPattern := p.Pattern
	if p.CaseInsensitive {
		regexPattern = "(?i)" + regexPattern
	}

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("invalid regex pattern: %v", err),
		}, nil
	}

	// Check if path exists
	info, err := os.Stat(p.Path)
	if err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("path not found: %v", err),
		}, nil
	}

	var results []string

	if info.IsDir() {
		// Search directory
		err = filepath.Walk(p.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files with errors
			}

			// Check context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Filter by file pattern if specified
			if p.FilePattern != "" {
				matched, err := filepath.Match(p.FilePattern, filepath.Base(path))
				if err != nil || !matched {
					return nil
				}
			}

			// Skip binary files
			if isBinaryFile(path) {
				return nil
			}

			// Search file
			matches := t.searchFile(path, re)
			results = append(results, matches...)

			return nil
		})

		if err != nil {
			return &tool.Result{
				Success: false,
				Error:   fmt.Sprintf("directory walk failed: %v", err),
			}, nil
		}
	} else {
		// Search single file
		if !isBinaryFile(p.Path) {
			results = t.searchFile(p.Path, re)
		}
	}

	if len(results) == 0 {
		return &tool.Result{
			Success: true,
			Output:  "No matches found",
		}, nil
	}

	return &tool.Result{
		Success: true,
		Output:  strings.Join(results, "\n"),
		Data: map[string]any{
			"count": len(results),
		},
	}, nil
}

func (t *GrepTool) searchFile(path string, re *regexp.Regexp) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var results []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if re.MatchString(line) {
			results = append(results, fmt.Sprintf("%s:%d:%s", path, lineNum, line))
		}
	}

	return results
}

// isBinaryFile checks if a file is binary by reading the first 512 bytes
func isBinaryFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return true // Assume binary if can't open
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return true
	}

	// Check for null bytes (indicator of binary file)
	return bytes.Contains(buf[:n], []byte{0})
}
