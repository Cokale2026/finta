package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestGlobTool_SimplePattern(t *testing.T) {
	tool := NewGlobTool()

	params, _ := json.Marshal(map[string]any{
		"pattern": "*.go",
		"path":    ".",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	// Should find Go files in current directory
	if result.Output == "No files found" {
		t.Error("Expected to find Go files")
	}
}

func TestGlobTool_RecursivePattern(t *testing.T) {
	tool := NewGlobTool()

	// Test **/*.go pattern from project root (../../..)
	params, _ := json.Marshal(map[string]any{
		"pattern": "**/*.go",
		"path":    "../../..",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	// Should find Go files recursively
	if result.Output == "No files found" {
		t.Error("Expected to find Go files recursively")
	}

	// Verify it found files in subdirectories (paths should contain /)
	if !strings.Contains(result.Output, "/") {
		t.Error("Expected to find files in subdirectories (paths should contain /)")
	}

	// Check data structure
	count, ok := result.Data["count"].(int)
	if !ok || count == 0 {
		t.Error("Expected count > 0")
	}

	files, ok := result.Data["files"].([]string)
	if !ok || len(files) == 0 {
		t.Error("Expected files array to have items")
	}
}

func TestGlobTool_RecursiveWithPrefix(t *testing.T) {
	tool := NewGlobTool()

	// Test internal/**/*.go pattern from project root
	params, _ := json.Marshal(map[string]any{
		"pattern": "internal/**/*.go",
		"path":    "../../..",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	// Should find Go files in internal/ subdirectory
	if result.Output == "No files found" {
		t.Error("Expected to find Go files in internal/")
	}

	// All files should contain internal/
	lines := strings.Split(result.Output, "\n")
	for _, line := range lines {
		if line != "" && !strings.Contains(line, "internal/") {
			t.Errorf("Expected file to be in internal/, got: %s", line)
		}
	}
}

func TestGlobTool_NoMatches(t *testing.T) {
	tool := NewGlobTool()

	// Pattern that shouldn't match anything
	params, _ := json.Marshal(map[string]any{
		"pattern": "**/*.nonexistent",
		"path":    ".",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success even with no matches, got error: %s", result.Error)
	}

	if result.Output != "No files found" {
		t.Errorf("Expected 'No files found', got: %s", result.Output)
	}

	count, ok := result.Data["count"].(int)
	if !ok {
		t.Fatal("Expected count field in data")
	}
	if count != 0 {
		t.Errorf("Expected count = 0, got %d", count)
	}
}

func TestGlobTool_InvalidParameters(t *testing.T) {
	tool := NewGlobTool()

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

func TestGlobTool_DoubleStarOnly(t *testing.T) {
	tool := NewGlobTool()

	// Pattern with just **
	params, _ := json.Marshal(map[string]any{
		"pattern": "**",
		"path":    ".",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	// Should find all files
	if result.Output == "No files found" {
		t.Error("Expected to find files")
	}
}
