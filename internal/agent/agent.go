package agent

import (
	"context"
	"finta/internal/llm"
	"finta/internal/logger"
	"finta/internal/tool"
)

type Agent interface {
	Name() string
	Run(ctx context.Context, input *Input) (*Output, error)
	RunStreaming(ctx context.Context, input *Input, streamChan chan<- string) (*Output, error)
}

type Input struct {
	Messages       []llm.Message
	Task           string
	MaxTurns       int
	Temperature    float32
	Logger         *logger.Logger
	EnableStreaming bool
}

type Output struct {
	Messages  []llm.Message
	Result    string
	ToolCalls []*tool.CallResult
}

type Config struct {
	Model              string
	Temperature        float32
	MaxTokens          int
	MaxTurns           int
	EnableParallelTools bool
	ToolExecutionMode   tool.ExecutionMode
}
