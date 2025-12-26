package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"finta/internal/tool"
)

type WriteTool struct{}

func NewWriteTool() *WriteTool {
	return &WriteTool{}
}

func (t *WriteTool) Name() string {
	return "write"
}

func (t *WriteTool) Description() string {
	return "Write content to a file (creates or overwrites)"
}

func (t *WriteTool) BestPractices() string {
	return ""
}

func (t *WriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"file_path", "content"},
	}
}

func (t *WriteTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
	var p struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(p.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("failed to create directory: %v", err),
		}, nil
	}

	// Write file
	if err := os.WriteFile(p.FilePath, []byte(p.Content), 0644); err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}, nil
	}

	return &tool.Result{
		Success: true,
		Output:  fmt.Sprintf("Successfully wrote %d bytes to %s", len(p.Content), p.FilePath),
	}, nil
}
