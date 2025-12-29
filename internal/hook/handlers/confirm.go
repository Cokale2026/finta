package handlers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"finta/internal/hook"
)

// BashConfirmHandler prompts user for confirmation before executing bash commands
type BashConfirmHandler struct {
	reader io.Reader
	writer io.Writer
}

// NewBashConfirmHandler creates a new bash confirmation handler
func NewBashConfirmHandler() *BashConfirmHandler {
	return &BashConfirmHandler{
		reader: os.Stdin,
		writer: os.Stdout,
	}
}

// NewBashConfirmHandlerWithIO creates a handler with custom IO (for testing)
func NewBashConfirmHandlerWithIO(reader io.Reader, writer io.Writer) *BashConfirmHandler {
	return &BashConfirmHandler{
		reader: reader,
		writer: writer,
	}
}

func (h *BashConfirmHandler) Name() string {
	return "bash_confirm"
}

func (h *BashConfirmHandler) Points() []hook.HookPoint {
	return []hook.HookPoint{hook.BeforeBashCommand}
}

func (h *BashConfirmHandler) Priority() int {
	return 100 // High priority - runs first
}

func (h *BashConfirmHandler) Handle(ctx context.Context, data *hook.HookData) (*hook.Feedback, error) {
	command := data.GetString("command")
	if command == "" {
		return hook.AllowFeedback(), nil
	}

	// Display confirmation prompt
	fmt.Fprintf(h.writer, "\n\033[33m⚠️  Bash command requires confirmation:\033[0m\n")
	fmt.Fprintf(h.writer, "    \033[1m%s\033[0m\n\n", command)
	fmt.Fprintf(h.writer, "Allow? [y/N]: ")

	// Read user input
	scanner := bufio.NewScanner(h.reader)
	if !scanner.Scan() {
		return hook.DenyFeedback("No input received"), nil
	}

	input := strings.TrimSpace(strings.ToLower(scanner.Text()))

	switch input {
	case "y", "yes":
		fmt.Fprintf(h.writer, "\033[32m✓ Allowed\033[0m\n\n")
		return hook.AllowFeedback(), nil
	default:
		fmt.Fprintf(h.writer, "\033[31m✗ Denied\033[0m\n\n")
		return hook.DenyFeedback("User denied command execution"), nil
	}
}

// ToolConfirmHandler prompts user for confirmation before executing any tool
type ToolConfirmHandler struct {
	reader    io.Reader
	writer    io.Writer
	toolNames map[string]bool // Only confirm these tools (empty = all)
}

// NewToolConfirmHandler creates a new tool confirmation handler
func NewToolConfirmHandler(tools ...string) *ToolConfirmHandler {
	toolNames := make(map[string]bool)
	for _, t := range tools {
		toolNames[t] = true
	}
	return &ToolConfirmHandler{
		reader:    os.Stdin,
		writer:    os.Stdout,
		toolNames: toolNames,
	}
}

func (h *ToolConfirmHandler) Name() string {
	return "tool_confirm"
}

func (h *ToolConfirmHandler) Points() []hook.HookPoint {
	return []hook.HookPoint{hook.BeforeToolExecution}
}

func (h *ToolConfirmHandler) Priority() int {
	return 100
}

func (h *ToolConfirmHandler) Handle(ctx context.Context, data *hook.HookData) (*hook.Feedback, error) {
	// If specific tools are configured, check if this tool needs confirmation
	if len(h.toolNames) > 0 && !h.toolNames[data.ToolName] {
		return hook.AllowFeedback(), nil
	}

	params := data.GetString("params")

	fmt.Fprintf(h.writer, "\n\033[33m⚠️  Tool '%s' requires confirmation:\033[0m\n", data.ToolName)
	if params != "" {
		fmt.Fprintf(h.writer, "    Parameters: %s\n", params)
	}
	fmt.Fprintf(h.writer, "\nAllow? [y/N]: ")

	scanner := bufio.NewScanner(h.reader)
	if !scanner.Scan() {
		return hook.DenyFeedback("No input received"), nil
	}

	input := strings.TrimSpace(strings.ToLower(scanner.Text()))

	switch input {
	case "y", "yes":
		fmt.Fprintf(h.writer, "\033[32m✓ Allowed\033[0m\n\n")
		return hook.AllowFeedback(), nil
	default:
		fmt.Fprintf(h.writer, "\033[31m✗ Denied\033[0m\n\n")
		return hook.DenyFeedback("User denied tool execution"), nil
	}
}
