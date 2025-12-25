package llm

import "time"

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role
	Reason     string
	Content    string
	ToolCalls  []*ToolCall
	ToolCallID string
	Name       string
	Timestamp  time.Time

	// ReAct pattern support: Reasoning-Action-Observation
	ReActTrace *ReActTrace `json:"react_trace,omitempty"`
}

type ToolCall struct {
	ID       string
	Type     string
	Function *FunctionCall
}

type FunctionCall struct {
	Name      string
	Arguments string
}

type StopReason string

const (
	StopReasonStop      StopReason = "stop"
	StopReasonLength    StopReason = "length"
	StopReasonToolCalls StopReason = "tool_calls"
)

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// ReActTrace records a complete Thought-Action-Observation cycle
// This implements the ReAct (Reasoning and Acting) pattern for transparent agent behavior
type ReActTrace struct {
	Thought     string         `json:"thought"`               // Why is this action being taken?
	Action      string         `json:"action"`                // What operation is being performed?
	Observation string         `json:"observation"`           // What was the result?
	Metadata    map[string]any `json:"metadata,omitempty"`    // Additional info (duration, tokens, etc.)
}
