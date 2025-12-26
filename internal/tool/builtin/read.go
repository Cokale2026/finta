package builtin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"finta/internal/tool"
)

const (
	maxFilesPerRead = 8 // Maximum number of files to read in one call
)

// FileReadRequest represents a request to read a single file
type FileReadRequest struct {
	FilePath string `json:"file_path"`
	From     int    `json:"from,omitempty"` // Optional: starting line number (1-based)
	To       int    `json:"to,omitempty"`   // Optional: ending line number (1-based, inclusive)
}

type ReadTool struct{}

func NewReadTool() *ReadTool {
	return &ReadTool{}
}

func (t *ReadTool) Name() string {
	return "read"
}

func (t *ReadTool) BestPractices() string {
	return `**Read Tool Best Practices**:

1. **Use line ranges for large files** - Instead of reading entire large files, specify 'from' and 'to' to read only relevant sections
   - Example: {"files": [{"file_path": "large.go", "from": 100, "to": 150}]}

2. **Read multiple related files together** - Batch related files in a single call (max 8) for better context
   - Good: Read handler.go, handler_test.go, and types.go together
   - Avoid: Making 3 separate read calls

3. **Read function implementations precisely** - Use line ranges to read specific functions
   - Find function start/end with grep, then read that range
   - Avoid reading entire files when you only need one function

4. **Prefer targeted reads over full file reads** - Only read what you need
   - For imports: Read lines 1-30
   - For specific function: Use line range
   - For entire file: Only when genuinely needed

5. **Read files before modifying** - Always read a file before using write or edit tools on it`
}

func (t *ReadTool) Description() string {
	return `Read contents of one or more files (max 8 files per call).

Supports reading entire files or specific line ranges.

Examples:
- Read entire file: {"files": [{"file_path": "config.yaml"}]}
- Read lines 10-20: {"files": [{"file_path": "main.go", "from": 10, "to": 20}]}
- Read multiple files: {"files": [{"file_path": "a.txt"}, {"file_path": "b.txt"}]}`
}

func (t *ReadTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"files": map[string]any{
				"type":        "array",
				"description": "Array of files to read (max 8 files)",
				"minItems":    1,
				"maxItems":    maxFilesPerRead,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file_path": map[string]any{
							"type":        "string",
							"description": "Path to the file to read",
						},
						"from": map[string]any{
							"type":        "integer",
							"description": "Optional: starting line number (1-based, inclusive)",
							"minimum":     1,
						},
						"to": map[string]any{
							"type":        "integer",
							"description": "Optional: ending line number (1-based, inclusive)",
							"minimum":     1,
						},
					},
					"required": []string{"file_path"},
				},
			},
		},
		"required": []string{"files"},
	}
}

func (t *ReadTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
	var p struct {
		Files []FileReadRequest `json:"files"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	// Validate number of files
	if len(p.Files) == 0 {
		return &tool.Result{
			Success: false,
			Error:   "at least one file must be specified",
		}, nil
	}

	if len(p.Files) > maxFilesPerRead {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("too many files requested (max %d, got %d)", maxFilesPerRead, len(p.Files)),
		}, nil
	}

	// Validate each file request
	for i, req := range p.Files {
		if strings.TrimSpace(req.FilePath) == "" {
			return &tool.Result{
				Success: false,
				Error:   fmt.Sprintf("file #%d: file_path cannot be empty", i+1),
			}, nil
		}

		// Validate line range
		if req.From > 0 && req.To > 0 && req.From > req.To {
			return &tool.Result{
				Success: false,
				Error:   fmt.Sprintf("file #%d (%s): invalid line range: from (%d) > to (%d)", i+1, req.FilePath, req.From, req.To),
			}, nil
		}
	}

	// Read all files
	var results []string
	totalLines := 0

	for i, req := range p.Files {
		content, lines, err := t.readFile(req)
		if err != nil {
			return &tool.Result{
				Success: false,
				Error:   fmt.Sprintf("file #%d (%s): %v", i+1, req.FilePath, err),
			}, nil
		}

		totalLines += lines

		// Format output for this file
		var header string
		if len(p.Files) > 1 {
			if req.From > 0 || req.To > 0 {
				header = fmt.Sprintf("=== File %d/%d: %s (lines %s) ===", i+1, len(p.Files), req.FilePath, t.formatLineRange(req, lines))
			} else {
				header = fmt.Sprintf("=== File %d/%d: %s ===", i+1, len(p.Files), req.FilePath)
			}
			results = append(results, header)
		} else {
			// Single file - no header needed unless line range specified
			if req.From > 0 || req.To > 0 {
				header = fmt.Sprintf("File: %s (lines %s)", req.FilePath, t.formatLineRange(req, lines))
				results = append(results, header)
			}
		}

		results = append(results, content)

		// Add separator between files (except for last file)
		if i < len(p.Files)-1 {
			results = append(results, "") // Empty line separator
		}
	}

	output := strings.Join(results, "\n")

	return &tool.Result{
		Success: true,
		Output:  output,
		Data: map[string]any{
			"file_count":  len(p.Files),
			"total_lines": totalLines,
		},
	}, nil
}

// readFile reads a single file with optional line range
func (t *ReadTool) readFile(req FileReadRequest) (string, int, error) {
	file, err := os.Open(req.FilePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Determine if we need line-based reading
	needLineRange := req.From > 0 || req.To > 0

	if !needLineRange {
		// Read entire file
		content, err := os.ReadFile(req.FilePath)
		if err != nil {
			return "", 0, fmt.Errorf("failed to read file: %w", err)
		}
		lines := strings.Count(string(content), "\n")
		if len(content) > 0 && content[len(content)-1] != '\n' {
			lines++ // Count last line if doesn't end with newline
		}
		return string(content), lines, nil
	}

	// Read with line range
	scanner := bufio.NewScanner(file)
	var lines []string
	lineNum := 0
	totalLines := 0

	from := req.From
	if from <= 0 {
		from = 1 // Default to start from line 1
	}

	to := req.To
	if to <= 0 {
		to = -1 // Read until end
	}

	for scanner.Scan() {
		lineNum++
		totalLines++

		// Check if we're in the desired range
		if lineNum >= from && (to < 0 || lineNum <= to) {
			lines = append(lines, scanner.Text())
		}

		// Stop early if we've reached the end of range
		if to > 0 && lineNum >= to {
			// Continue scanning to count total lines
			for scanner.Scan() {
				totalLines++
			}
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", 0, fmt.Errorf("error reading file: %w", err)
	}

	if len(lines) == 0 {
		return "", totalLines, fmt.Errorf("no lines in specified range (file has %d lines)", totalLines)
	}

	return strings.Join(lines, "\n"), len(lines), nil
}

// formatLineRange formats the line range for display
func (t *ReadTool) formatLineRange(req FileReadRequest, actualLines int) string {
	from := req.From
	if from <= 0 {
		from = 1
	}

	if req.To > 0 {
		return fmt.Sprintf("%d-%d, returned %d lines", from, req.To, actualLines)
	}

	return fmt.Sprintf("%d-end, returned %d lines", from, actualLines)
}
