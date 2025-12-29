package hook

import (
	"context"
	"time"
)

// HookPoint defines when a hook is triggered
type HookPoint string

const (
	// Tool execution hooks
	BeforeToolExecution HookPoint = "before_tool_execution"
	AfterToolExecution  HookPoint = "after_tool_execution"

	// Bash-specific hooks (for user confirmation)
	BeforeBashCommand HookPoint = "before_bash_command"
	AfterBashCommand  HookPoint = "after_bash_command"

	// Agent lifecycle hooks
	OnAgentStart HookPoint = "on_agent_start"
	OnAgentEnd   HookPoint = "on_agent_end"
)

// HookData carries context-specific information for hooks
type HookData struct {
	Point     HookPoint
	Timestamp time.Time
	ToolName  string
	Data      map[string]any
}

// NewHookData creates a new HookData instance
func NewHookData(point HookPoint, toolName string) *HookData {
	return &HookData{
		Point:     point,
		Timestamp: time.Now(),
		ToolName:  toolName,
		Data:      make(map[string]any),
	}
}

// Set sets a data field
func (d *HookData) Set(key string, value any) *HookData {
	d.Data[key] = value
	return d
}

// Get retrieves a data field
func (d *HookData) Get(key string) any {
	return d.Data[key]
}

// GetString retrieves a string data field
func (d *HookData) GetString(key string) string {
	if v, ok := d.Data[key].(string); ok {
		return v
	}
	return ""
}

// Feedback is returned by handlers to control execution flow
type Feedback struct {
	Allow    bool   // Whether to allow the operation to continue
	Message  string // Optional message to display
	Modified any    // Modified data (e.g., modified command)
}

// AllowFeedback creates an allow feedback
func AllowFeedback() *Feedback {
	return &Feedback{Allow: true}
}

// DenyFeedback creates a deny feedback with message
func DenyFeedback(message string) *Feedback {
	return &Feedback{Allow: false, Message: message}
}

// Handler is the interface for hook handlers
type Handler interface {
	// Name returns the handler name
	Name() string

	// Points returns which hook points this handler listens to
	Points() []HookPoint

	// Handle processes the hook event and returns feedback
	Handle(ctx context.Context, data *HookData) (*Feedback, error)

	// Priority returns the handler priority (higher = earlier execution)
	Priority() int
}
