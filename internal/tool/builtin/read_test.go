package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTool_SingleFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test1.txt")
	content := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadTool()

	params := `{"files": [{"file_path": "` + file1 + `"}]}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected success, got error: %s", result.Error)
	}

	if result.Output != content {
		t.Errorf("Content mismatch:\nExpected: %q\nGot: %q", content, result.Output)
	}

	if result.Data["file_count"] != 1 {
		t.Errorf("Expected file_count=1, got %v", result.Data["file_count"])
	}
}

func TestReadTool_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "test1.txt")
	file2 := filepath.Join(tmpDir, "test2.txt")
	file3 := filepath.Join(tmpDir, "test3.txt")

	os.WriteFile(file1, []byte("content A"), 0644)
	os.WriteFile(file2, []byte("content B"), 0644)
	os.WriteFile(file3, []byte("content C"), 0644)

	tool := NewReadTool()

	params := `{"files": [
		{"file_path": "` + file1 + `"},
		{"file_path": "` + file2 + `"},
		{"file_path": "` + file3 + `"}
	]}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected success, got error: %s", result.Error)
	}

	// Check that all files are included
	if !strings.Contains(result.Output, "content A") {
		t.Error("Output should contain content A")
	}
	if !strings.Contains(result.Output, "content B") {
		t.Error("Output should contain content B")
	}
	if !strings.Contains(result.Output, "content C") {
		t.Error("Output should contain content C")
	}

	// Check headers
	if !strings.Contains(result.Output, "File 1/3") {
		t.Error("Output should contain file counter")
	}

	if result.Data["file_count"] != 3 {
		t.Errorf("Expected file_count=3, got %v", result.Data["file_count"])
	}
}

func TestReadTool_LineRange(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test.txt")

	content := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\nline 10\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadTool()

	// Test reading lines 3-6
	params := `{"files": [{"file_path": "` + file1 + `", "from": 3, "to": 6}]}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected success, got error: %s", result.Error)
	}

	// When single file has line range, it includes a header
	if !strings.Contains(result.Output, "line 3") {
		t.Error("Output should contain line 3")
	}
	if !strings.Contains(result.Output, "line 4") {
		t.Error("Output should contain line 4")
	}
	if !strings.Contains(result.Output, "line 5") {
		t.Error("Output should contain line 5")
	}
	if !strings.Contains(result.Output, "line 6") {
		t.Error("Output should contain line 6")
	}

	// Should contain header with line range info
	if !strings.Contains(result.Output, "lines 3-6") {
		t.Error("Output should contain line range info")
	}

	if result.Data["total_lines"] != 4 {
		t.Errorf("Expected total_lines=4, got %v", result.Data["total_lines"])
	}
}

func TestReadTool_LineRangeFromOnly(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test.txt")

	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadTool()

	// Test reading from line 3 to end
	params := `{"files": [{"file_path": "` + file1 + `", "from": 3}]}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected success, got error: %s", result.Error)
	}

	// Check content
	if !strings.Contains(result.Output, "line 3") {
		t.Error("Output should contain line 3")
	}
	if !strings.Contains(result.Output, "line 4") {
		t.Error("Output should contain line 4")
	}
	if !strings.Contains(result.Output, "line 5") {
		t.Error("Output should contain line 5")
	}

	// Should contain header with "from-end" info
	if !strings.Contains(result.Output, "3-end") {
		t.Error("Output should contain '3-end' line range info")
	}

	// Should NOT contain lines before range
	if strings.Contains(result.Output, "line 1") || strings.Contains(result.Output, "line 2") {
		t.Error("Output should not contain lines 1 or 2")
	}
}

func TestReadTool_MaxFilesLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Try to read 9 files (exceeds limit of 8)
	var files []string
	for i := 1; i <= 9; i++ {
		files = append(files, `{"file_path": "`+filepath.Join(tmpDir, "test.txt")+`"}`)
	}

	params := `{"files": [` + strings.Join(files, ",") + `]}`

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Fatal("Expected failure for too many files")
	}

	if !strings.Contains(result.Error, "too many files") {
		t.Errorf("Error should mention too many files: %s", result.Error)
	}
}

func TestReadTool_InvalidLineRange(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file1, []byte("line 1\nline 2\n"), 0644)

	tool := NewReadTool()

	// Test invalid range: from > to
	params := `{"files": [{"file_path": "` + file1 + `", "from": 10, "to": 5}]}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Fatal("Expected failure for invalid line range")
	}

	if !strings.Contains(result.Error, "invalid line range") {
		t.Errorf("Error should mention invalid line range: %s", result.Error)
	}
}

func TestReadTool_FileNotFound(t *testing.T) {
	tool := NewReadTool()

	params := `{"files": [{"file_path": "/nonexistent/file.txt"}]}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Fatal("Expected failure for nonexistent file")
	}

	if !strings.Contains(result.Error, "failed to open file") {
		t.Errorf("Error should mention file opening failure: %s", result.Error)
	}
}

func TestReadTool_EmptyFilePath(t *testing.T) {
	tool := NewReadTool()

	params := `{"files": [{"file_path": ""}]}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Fatal("Expected failure for empty file path")
	}

	if !strings.Contains(result.Error, "file_path cannot be empty") {
		t.Errorf("Error should mention empty file path: %s", result.Error)
	}
}

func TestReadTool_NoFiles(t *testing.T) {
	tool := NewReadTool()

	params := `{"files": []}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Fatal("Expected failure for no files")
	}

	if !strings.Contains(result.Error, "at least one file") {
		t.Errorf("Error should mention at least one file: %s", result.Error)
	}
}

func TestReadTool_MultipleFilesWithLineRanges(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	content1 := "A1\nA2\nA3\nA4\nA5\n"
	content2 := "B1\nB2\nB3\nB4\nB5\n"

	os.WriteFile(file1, []byte(content1), 0644)
	os.WriteFile(file2, []byte(content2), 0644)

	tool := NewReadTool()

	params := `{"files": [
		{"file_path": "` + file1 + `", "from": 2, "to": 3},
		{"file_path": "` + file2 + `", "from": 1, "to": 2}
	]}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected success, got error: %s", result.Error)
	}

	// Check that correct lines are included
	if !strings.Contains(result.Output, "A2") || !strings.Contains(result.Output, "A3") {
		t.Error("Output should contain A2 and A3")
	}

	if !strings.Contains(result.Output, "B1") || !strings.Contains(result.Output, "B2") {
		t.Error("Output should contain B1 and B2")
	}

	// Should NOT contain these lines
	if strings.Contains(result.Output, "A1") || strings.Contains(result.Output, "A4") {
		t.Error("Output should not contain A1 or A4")
	}

	if strings.Contains(result.Output, "B3") || strings.Contains(result.Output, "B4") {
		t.Error("Output should not contain B3 or B4")
	}
}

func TestReadTool_LineRangeOutOfBounds(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test.txt")

	content := "line 1\nline 2\nline 3\n"
	os.WriteFile(file1, []byte(content), 0644)

	tool := NewReadTool()

	// Request lines beyond file length
	params := `{"files": [{"file_path": "` + file1 + `", "from": 10, "to": 20}]}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Fatal("Expected failure for out of bounds line range")
	}

	if !strings.Contains(result.Error, "no lines in specified range") {
		t.Errorf("Error should mention no lines in range: %s", result.Error)
	}
}

func TestReadTool_Parameters(t *testing.T) {
	tool := NewReadTool()
	params := tool.Parameters()

	// Verify structure
	if params["type"] != "object" {
		t.Error("Expected object type")
	}

	props := params["properties"].(map[string]any)
	filesParam := props["files"].(map[string]any)

	if filesParam["type"] != "array" {
		t.Error("Expected files to be array type")
	}

	if filesParam["maxItems"] != maxFilesPerRead {
		t.Errorf("Expected maxItems=%d, got %v", maxFilesPerRead, filesParam["maxItems"])
	}

	// Check required fields
	required := params["required"].([]string)
	if len(required) != 1 || required[0] != "files" {
		t.Error("Expected 'files' to be required")
	}
}
