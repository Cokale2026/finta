package agent

import (
	"context"
	"finta/pkg/llm"
	"finta/pkg/tool"
)

type Agent interface {
	Name() string
	Run(ctx context.Context, input *Input) (*Output, error)
}

type Input struct {
	Messages    []llm.Message
	Task        string
	MaxTurns    int
	Temperature float32
}

type Output struct {
	Messages  []llm.Message
	Result    string
	ToolCalls []*tool.CallResult
}

type Config struct {
	Model       string
	Temperature float32
	MaxTokens   int
	MaxTurns    int
}
