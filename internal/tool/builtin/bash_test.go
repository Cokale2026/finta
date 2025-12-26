package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestBashTool_SuccessWithOutput(t *testing.T) {
	tool := NewBashTool()

	params, _ := json.Marshal(map[string]any{
		"command": "echo 'hello world'",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Output, "hello world") {
		t.Errorf("Expected output to contain 'hello world', got: %s", result.Output)
	}
}

func TestBashTool_SuccessWithNoOutput(t *testing.T) {
	tool := NewBashTool()

	// Command that produces no output (true command)
	params, _ := json.Marshal(map[string]any{
		"command": "true",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	// CRITICAL: Output should NOT be empty (this is the fix for issue #1)
	if result.Output == "" {
		t.Error("Output should not be empty - this causes LLM API errors")
	}

	// Should have a placeholder message
	expectedMsg := "(Command executed successfully with no output)"
	if result.Output != expectedMsg {
		t.Errorf("Expected placeholder message, got: %s", result.Output)
	}
}

func TestBashTool_SuccessWithEmptyOutput(t *testing.T) {
	tool := NewBashTool()

	// Command that explicitly produces no output
	params, _ := json.Marshal(map[string]any{
		"command": "ls /nonexistent_directory_that_does_not_exist 2>/dev/null || true",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	// Output should not be empty
	if result.Output == "" {
		t.Error("Output should not be empty - this causes LLM API errors")
	}
}

func TestBashTool_Failure(t *testing.T) {
	tool := NewBashTool()

	// Command that fails
	params, _ := json.Marshal(map[string]any{
		"command": "exit 1",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for 'exit 1' command")
	}

	if result.Error == "" {
		t.Error("Expected error message")
	}
}

func TestBashTool_InvalidParameters(t *testing.T) {
	tool := NewBashTool()

	// Invalid JSON
	result, err := tool.Execute(context.Background(), []byte("invalid json"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for invalid JSON")
	}

	if !strings.Contains(result.Error, "invalid parameters") {
		t.Errorf("Expected 'invalid parameters' error, got: %s", result.Error)
	}
}

func TestBashTool_Timeout(t *testing.T) {
	tool := NewBashTool()

	// Command that should timeout (sleep 10 seconds with 100ms timeout)
	params, _ := json.Marshal(map[string]any{
		"command": "sleep 10",
		"timeout": 100,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure due to timeout")
	}

	if result.Error == "" {
		t.Error("Expected timeout error message")
	}
}
