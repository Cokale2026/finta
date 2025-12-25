package llm

import "context"

type Client interface {
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req *ChatRequest) (StreamReader, error)
	Provider() string
	Model() string
}

type ChatRequest struct {
	Messages    []Message
	Tools       []*ToolDefinition
	Temperature float32
	MaxTokens   int
}

type ChatResponse struct {
	Message    Message
	StopReason StopReason
	Usage      Usage
}

type ToolDefinition struct {
	Type     string
	Function *FunctionDef
}

type FunctionDef struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type StreamReader interface {
	Recv() (*Delta, error)
	Close() error
}

type Delta struct {
	Role      Role
	Reason    string
	Content   string
	ToolCalls []*ToolCall
	Done      bool
}
