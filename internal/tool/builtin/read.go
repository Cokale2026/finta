package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"finta/internal/tool"
)

type ReadTool struct{}

func NewReadTool() *ReadTool {
	return &ReadTool{}
}

func (t *ReadTool) Name() string {
	return "read"
}

func (t *ReadTool) Description() string {
	return "Read contents of a file"
}

func (t *ReadTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path to the file to read",
			},
		},
		"required": []string{"file_path"},
	}
}

func (t *ReadTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
	var p struct {
		FilePath string `json:"file_path"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	content, err := os.ReadFile(p.FilePath)
	if err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}, nil
	}

	return &tool.Result{
		Success: true,
		Output:  string(content),
	}, nil
}
