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
