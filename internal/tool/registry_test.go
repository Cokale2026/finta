package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// MockToolWithBestPractices is a mock tool that implements ToolWithBestPractices
type MockToolWithBestPractices struct{}

func (t *MockToolWithBestPractices) Name() string {
	return "mock_tool_with_bp"
}

func (t *MockToolWithBestPractices) Description() string {
	return "A mock tool with best practices"
}

func (t *MockToolWithBestPractices) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"param": map[string]any{
				"type": "string",
			},
		},
	}
}

func (t *MockToolWithBestPractices) Execute(ctx context.Context, params json.RawMessage) (*Result, error) {
	return &Result{Success: true, Output: "mock output"}, nil
}

func (t *MockToolWithBestPractices) BestPractices() string {
	return `**Mock Tool Best Practices**:
1. Always use param X
2. Never use param Y
3. Check results carefully`
}

// MockToolWithoutBestPractices is a mock tool that does NOT implement best practices
type MockToolWithoutBestPractices struct{}

func (t *MockToolWithoutBestPractices) Name() string {
	return "mock_tool_without_bp"
}

func (t *MockToolWithoutBestPractices) Description() string {
	return "A mock tool without best practices"
}

func (t *MockToolWithoutBestPractices) BestPractices() string {
	return ""
}

func (t *MockToolWithoutBestPractices) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
	}
}

func (t *MockToolWithoutBestPractices) Execute(ctx context.Context, params json.RawMessage) (*Result, error) {
	return &Result{Success: true, Output: "mock output"}, nil
}

func TestRegistry_GetToolBestPractices_Empty(t *testing.T) {
	registry := NewRegistry()

	// No tools registered
	practices := registry.GetToolBestPractices()
	if practices != "" {
		t.Errorf("Expected empty string when no tools registered, got: %s", practices)
	}
}

func TestRegistry_GetToolBestPractices_NoToolsWithBP(t *testing.T) {
	registry := NewRegistry()

	// Register tool without best practices
	tool := &MockToolWithoutBestPractices{}
	if err := registry.Register(tool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	practices := registry.GetToolBestPractices()
	if practices != "" {
		t.Errorf("Expected empty string when no tools have best practices, got: %s", practices)
	}
}

func TestRegistry_GetToolBestPractices_WithBP(t *testing.T) {
	registry := NewRegistry()

	// Register tool with best practices
	tool := &MockToolWithBestPractices{}
	if err := registry.Register(tool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	practices := registry.GetToolBestPractices()

	// Should contain header
	if !strings.Contains(practices, "# Tool Usage Best Practices") {
		t.Error("Best practices should contain header")
	}

	// Should contain the mock tool's best practices
	if !strings.Contains(practices, "Mock Tool Best Practices") {
		t.Error("Best practices should contain mock tool's practices")
	}

	if !strings.Contains(practices, "Always use param X") {
		t.Error("Best practices should contain specific practice text")
	}
}

func TestRegistry_GetToolBestPractices_Mixed(t *testing.T) {
	registry := NewRegistry()

	// Register both tools
	tool1 := &MockToolWithBestPractices{}
	tool2 := &MockToolWithoutBestPractices{}

	if err := registry.Register(tool1); err != nil {
		t.Fatalf("Failed to register tool1: %v", err)
	}
	if err := registry.Register(tool2); err != nil {
		t.Fatalf("Failed to register tool2: %v", err)
	}

	practices := registry.GetToolBestPractices()

	// Should contain header
	if !strings.Contains(practices, "# Tool Usage Best Practices") {
		t.Error("Best practices should contain header")
	}

	// Should contain practices from tool with BP
	if !strings.Contains(practices, "Mock Tool Best Practices") {
		t.Error("Best practices should contain mock tool's practices")
	}

	// Should NOT contain references to tool without BP
	if strings.Contains(practices, "mock_tool_without_bp") {
		t.Error("Best practices should not reference tools without best practices")
	}
}
