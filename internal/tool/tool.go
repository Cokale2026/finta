package tool

import (
	"context"
	"encoding/json"
	"time"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
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
