package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"finta/internal/tool"
)

type BashTool struct{}

func NewBashTool() *BashTool {
	return &BashTool{}
}

func (t *BashTool) Name() string {
	return "bash"
}

func (t *BashTool) Description() string {
	return "Execute a bash command"
}

func (t *BashTool) BestPractices() string {
	return ""
}

func (t *BashTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The bash command to execute",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Timeout in milliseconds (default: 120000)",
			},
		},
		"required": []string{"command"},
	}
}

func (t *BashTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
	var p struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	// Default timeout: 2 minutes
	timeout := 120000
	if p.Timeout > 0 {
		timeout = p.Timeout
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", p.Command)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return &tool.Result{
			Success: false,
			Output:  string(output),
			Error:   err.Error(),
		}, nil
	}

	return &tool.Result{
		Success: true,
		Output:  string(output),
	}, nil
}
