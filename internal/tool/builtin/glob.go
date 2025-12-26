package builtin

import (
	"context"
	"encoding/json"
	"fmt"
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
	return ""
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

	// Construct full pattern
	fullPattern := filepath.Join(basePath, p.Pattern)

	// Use filepath.Glob for matching
	matches, err := filepath.Glob(fullPattern)
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
