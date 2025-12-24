package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Level represents the log level
type Level int

const (
	LevelDebug Level = iota // Debug information (only shown with --verbose)
	LevelInfo               // Important steps
	LevelTool               // Tool call related
	LevelAgent              // Agent response
	LevelError              // Error messages
)

// ANSI color codes for terminal output
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorGray    = "\033[90m"
	ColorBold    = "\033[1m"
)

// Logger provides structured logging for the agent framework
type Logger struct {
	writer    io.Writer
	level     Level
	showTime  bool
	colorMode bool
}

// NewLogger creates a new Logger instance
func NewLogger(w io.Writer, level Level) *Logger {
	if w == nil {
		w = os.Stdout
	}
	return &Logger{
		writer:    w,
		level:     level,
		showTime:  true,
		colorMode: true,
	}
}

// SetColorMode enables or disables colored output
func (l *Logger) SetColorMode(enabled bool) {
	l.colorMode = enabled
}

// SetShowTime enables or disables timestamp display
func (l *Logger) SetShowTime(enabled bool) {
	l.showTime = enabled
}

// Debug logs debug information (only shown in verbose mode)
func (l *Logger) Debug(format string, args ...any) {
	if l.level <= LevelDebug {
		l.log(ColorGray, "DEBUG", format, args...)
	}
}

// Info logs general information
func (l *Logger) Info(format string, args ...any) {
	if l.level <= LevelInfo {
		l.log(ColorBlue, "INFO", format, args...)
	}
}

// Error logs error messages
func (l *Logger) Error(format string, args ...any) {
	l.log(ColorRed, "ERROR", format, args...)
}

// AgentResponse logs the agent's response with structured formatting
func (l *Logger) AgentResponse(content string) {
	if l.level <= LevelAgent {
		l.printSection(ColorGreen, "💬 Agent Response", content)
	}
}

// ToolCall logs a tool call with its parameters
func (l *Logger) ToolCall(toolName string, params string) {
	if l.level <= LevelTool {
		formattedParams := l.formatJSON(params)
		l.printSection(ColorCyan, fmt.Sprintf("🔧 Tool Call: %s", toolName), formattedParams)
	}
}

// ToolResult logs a tool execution result
func (l *Logger) ToolResult(toolName string, success bool, output string, duration time.Duration) {
	if l.level <= LevelTool {
		status := "✅ Success"
		color := ColorGreen
		if !success {
			status = "❌ Failed"
			color = ColorRed
		}

		// Limit output to maximum 2 lines and 500 characters
		const maxLines = 2
		const maxLength = 500

		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		displayOutput := output
		truncatedLines := false

		// First, limit to maximum 2 lines
		if len(lines) > maxLines {
			displayOutput = strings.Join(lines[:maxLines], "\n")
			truncatedLines = true
		}

		// Then, limit to maximum 500 characters
		if len(displayOutput) > maxLength {
			displayOutput = displayOutput[:maxLength] + "..."
		} else if truncatedLines {
			// Add ellipsis if we truncated lines but not characters
			displayOutput += "\n..."
		}

		header := fmt.Sprintf("📊 Tool Result: %s [%s] (%s)", toolName, status, duration)
		l.printSection(color, header, displayOutput)
	}
}

// SessionStart logs the beginning of an agent session
func (l *Logger) SessionStart(task string) {
	l.printBanner(ColorCyan, "🚀 Session Started", task)
}

// SessionEnd logs the completion of an agent session with statistics
func (l *Logger) SessionEnd(duration time.Duration, toolCallCount int) {
	summary := fmt.Sprintf("Duration: %s | Tool Calls: %d", duration.Round(time.Millisecond), toolCallCount)
	l.printBanner(ColorGreen, "✨ Session Completed", summary)
}

// Progress displays a progress bar (optional implementation)
func (l *Logger) Progress(current, total int, message string) {
	if l.level > LevelInfo {
		return
	}

	bar := l.progressBar(current, total, 30)
	fmt.Fprintf(l.writer, "\r%s [%d/%d] %s", bar, current, total, message)

	if current == total {
		fmt.Fprintln(l.writer)
	}
}

// log is the core logging method
func (l *Logger) log(color, level, format string, args ...any) {
	timestamp := ""
	if l.showTime {
		timestamp = time.Now().Format("15:04:05") + " "
	}

	msg := fmt.Sprintf(format, args...)

	if l.colorMode {
		fmt.Fprintf(l.writer, "%s%s[%s]%s %s\n",
			color, timestamp, level, ColorReset, msg)
	} else {
		fmt.Fprintf(l.writer, "%s[%s] %s\n", timestamp, level, msg)
	}
}

// printSection prints a formatted section with header and content
func (l *Logger) printSection(color, header, content string) {
	separator := strings.Repeat("─", 60)

	if l.colorMode {
		fmt.Fprintf(l.writer, "\n%s%s%s%s\n", ColorBold, color, header, ColorReset)
		fmt.Fprintf(l.writer, "%s%s%s\n", color, separator, ColorReset)
		fmt.Fprintf(l.writer, "%s\n", content)
		fmt.Fprintf(l.writer, "%s%s%s\n\n", color, separator, ColorReset)
	} else {
		fmt.Fprintf(l.writer, "\n%s\n%s\n%s\n%s\n\n", header, separator, content, separator)
	}
}

// printBanner prints a prominent banner for session start/end
func (l *Logger) printBanner(color, title, subtitle string) {
	separator := strings.Repeat("═", 70)

	if l.colorMode {
		fmt.Fprintf(l.writer, "\n%s%s%s%s\n", ColorBold, color, separator, ColorReset)
		fmt.Fprintf(l.writer, "%s%s  %s%s\n", ColorBold, color, title, ColorReset)
		if subtitle != "" {
			fmt.Fprintf(l.writer, "%s  %s%s\n", color, subtitle, ColorReset)
		}
		fmt.Fprintf(l.writer, "%s%s%s%s\n\n", ColorBold, color, separator, ColorReset)
	} else {
		fmt.Fprintf(l.writer, "\n%s\n  %s\n", separator, title)
		if subtitle != "" {
			fmt.Fprintf(l.writer, "  %s\n", subtitle)
		}
		fmt.Fprintf(l.writer, "%s\n\n", separator)
	}
}

// progressBar generates a progress bar string
func (l *Logger) progressBar(current, total, width int) string {
	if total == 0 {
		return ""
	}

	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	if l.colorMode {
		return fmt.Sprintf("%s%s%s %.0f%%", ColorCyan, bar, ColorReset, percent*100)
	}
	return fmt.Sprintf("%s %.0f%%", bar, percent*100)
}

// formatJSON formats JSON strings adaptively based on length
// Short JSON (< 80 chars) stays compact, long JSON gets pretty-printed
func (l *Logger) formatJSON(jsonStr string) string {
	// Trim whitespace
	compact := strings.TrimSpace(jsonStr)

	// If it's short, keep it compact
	if len(compact) < 80 {
		return compact
	}

	// Otherwise, pretty-print it
	var obj interface{}
	if err := json.Unmarshal([]byte(compact), &obj); err != nil {
		// If parsing fails, return original
		return compact
	}

	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return compact
	}

	return string(pretty)
}
