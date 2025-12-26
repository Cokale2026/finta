package tool

import (
	"context"
	"encoding/json"
	"time"
)

// Tool defines the interface that all tools must implement
type Tool interface {
	// Name returns the unique identifier for this tool
	Name() string

	// Description returns a brief description of what this tool does
	Description() string

	// BestPractices returns usage guidelines for this tool
	// Returns empty string if no special guidance is needed
	BestPractices() string

	// Parameters returns the JSON schema for the tool's parameters
	Parameters() map[string]any

	// Execute runs the tool with the given parameters
	Execute(ctx context.Context, params json.RawMessage) (*Result, error)
}

type Result struct {
	Success bool
	Output  string
	Error   string
	Data    map[string]any
}

type CallResult struct {
	ToolName  string
	CallID    string
	Params    json.RawMessage
	Result    *Result
	StartTime time.Time
	EndTime   time.Time
}
