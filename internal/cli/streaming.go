package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"finta/internal/llm"
)

// StreamingWriter provides utilities for writing streaming content to output
type StreamingWriter struct {
	writer    io.Writer
	colorMode bool
	verbose   bool
}

func NewStreamingWriter(w io.Writer) *StreamingWriter {
	if w == nil {
		w = os.Stdout
	}
	return &StreamingWriter{
		writer:    w,
		colorMode: true,
		verbose:   false,
	}
}

func (sw *StreamingWriter) SetColorMode(enabled bool) {
	sw.colorMode = enabled
}

func (sw *StreamingWriter) SetVerbose(enabled bool) {
	sw.verbose = enabled
}

// Write writes content to the output
func (sw *StreamingWriter) Write(content string) {
	fmt.Fprint(sw.writer, content)
}

// WriteLine writes a line to the output
func (sw *StreamingWriter) WriteLine(content string) {
	fmt.Fprintln(sw.writer, content)
}

// WriteColored writes colored content if color mode is enabled
func (sw *StreamingWriter) WriteColored(content, color string) {
	if sw.colorMode {
		fmt.Fprintf(sw.writer, "%s%s%s", color, content, ColorReset)
	} else {
		fmt.Fprint(sw.writer, content)
	}
}

// Flush ensures all content is written (useful for buffered writers)
func (sw *StreamingWriter) Flush() {
	if flusher, ok := sw.writer.(interface{ Flush() error }); ok {
		flusher.Flush()
	}
}

// ANSI Color codes
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

// StreamRenderer handles rendering streaming LLM responses
type StreamRenderer struct {
	writer     *StreamingWriter
	currentPos int
}

func NewStreamRenderer(writer *StreamingWriter) *StreamRenderer {
	return &StreamRenderer{
		writer:     writer,
		currentPos: 0,
	}
}

// RenderDelta renders a single delta from the stream
func (sr *StreamRenderer) RenderDelta(delta *llm.Delta) {
	if delta.Content != "" {
		sr.writer.Write(delta.Content)
		sr.currentPos += len(delta.Content)
	}
}

// RenderComplete indicates the stream is complete
func (sr *StreamRenderer) RenderComplete() {
	sr.writer.WriteLine("")
	sr.currentPos = 0
}

// StreamContent streams content from a reader and renders it
func (sr *StreamRenderer) StreamContent(ctx context.Context, reader llm.StreamReader) (llm.Message, error) {
	defer reader.Close()

	var accumulatedMsg llm.Message
	accumulatedMsg.Role = llm.RoleAssistant
	accumulatedMsg.Content = ""

	for {
		select {
		case <-ctx.Done():
			return accumulatedMsg, ctx.Err()
		default:
		}

		delta, err := reader.Recv()
		if err != nil {
			return accumulatedMsg, err
		}

		if delta.Done {
			break
		}

		// Render the delta
		sr.RenderDelta(delta)

		// Accumulate content
		if delta.Content != "" {
			accumulatedMsg.Content += delta.Content
		}

		// Accumulate tool calls
		if len(delta.ToolCalls) > 0 {
			accumulatedMsg.ToolCalls = delta.ToolCalls
		}
	}

	sr.RenderComplete()
	return accumulatedMsg, nil
}

// MarkdownRenderer provides markdown-aware streaming rendering
type MarkdownRenderer struct {
	writer        *StreamingWriter
	buffer        strings.Builder
	inCodeBlock   bool
	codeBlockLang string
}

func NewMarkdownRenderer(writer *StreamingWriter) *MarkdownRenderer {
	return &MarkdownRenderer{
		writer: writer,
	}
}

// RenderDelta renders markdown-formatted delta
func (mr *MarkdownRenderer) RenderDelta(delta *llm.Delta) {
	if delta.Content == "" {
		return
	}

	// Add to buffer
	mr.buffer.WriteString(delta.Content)

	// Check for code block markers
	content := mr.buffer.String()

	// Detect code block start
	if strings.Contains(content, "```") && !mr.inCodeBlock {
		mr.inCodeBlock = true
		// Extract language if present
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				mr.codeBlockLang = strings.TrimPrefix(line, "```")
				break
			}
		}
	}

	// Detect code block end
	if mr.inCodeBlock && strings.Count(content, "```") >= 2 {
		mr.inCodeBlock = false
		mr.codeBlockLang = ""
	}

	// Render based on context
	if mr.inCodeBlock {
		mr.writer.WriteColored(delta.Content, ColorCyan)
	} else {
		mr.writer.Write(delta.Content)
	}
}

// ProgressIndicator shows a simple progress indicator during streaming
type ProgressIndicator struct {
	writer  *StreamingWriter
	frames  []string
	current int
	active  bool
}

func NewProgressIndicator(writer *StreamingWriter) *ProgressIndicator {
	return &ProgressIndicator{
		writer:  writer,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		current: 0,
		active:  false,
	}
}

// Show displays the progress indicator
func (pi *ProgressIndicator) Show(message string) {
	if !pi.active {
		return
	}
	frame := pi.frames[pi.current%len(pi.frames)]
	pi.writer.WriteColored(fmt.Sprintf("\r%s %s", frame, message), ColorCyan)
	pi.current++
}

// Start starts the progress indicator
func (pi *ProgressIndicator) Start() {
	pi.active = true
	pi.current = 0
}

// Stop stops and clears the progress indicator
func (pi *ProgressIndicator) Stop() {
	pi.active = false
	pi.writer.Write("\r\033[K") // Clear line
}

// InteractiveStreamer provides an interactive streaming experience
type InteractiveStreamer struct {
	writer     *StreamingWriter
	renderer   *StreamRenderer
	indicator  *ProgressIndicator
	showThink  bool
}

func NewInteractiveStreamer(writer *StreamingWriter) *InteractiveStreamer {
	return &InteractiveStreamer{
		writer:    writer,
		renderer:  NewStreamRenderer(writer),
		indicator: NewProgressIndicator(writer),
		showThink: false,
	}
}

// SetShowThinking enables/disables thinking display
func (is *InteractiveStreamer) SetShowThinking(show bool) {
	is.showThink = show
}

// StreamResponse streams and renders a response with optional thinking indicator
func (is *InteractiveStreamer) StreamResponse(ctx context.Context, reader llm.StreamReader) (llm.Message, error) {
	// Show thinking indicator briefly
	if is.showThink {
		is.indicator.Start()
		is.indicator.Show("Thinking...")
		is.indicator.Stop()
	}

	// Stream and render content
	return is.renderer.StreamContent(ctx, reader)
}
