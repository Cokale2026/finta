package agent

import (
	"context"
	"fmt"
	"time"

	"finta/internal/llm"
	"finta/internal/tool"
)

type BaseAgent struct {
	name         string
	systemPrompt string
	llmClient    llm.Client
	toolRegistry *tool.Registry
	config       *Config
}

func NewBaseAgent(name, systemPrompt string, client llm.Client, registry *tool.Registry, cfg *Config) *BaseAgent {
	if cfg == nil {
		cfg = &Config{
			Model:       "gpt-4-turbo",
			Temperature: 0.7,
			MaxTokens:   4096,
			MaxTurns:    20,
		}
	}

	return &BaseAgent{
		name:         name,
		systemPrompt: systemPrompt,
		llmClient:    client,
		toolRegistry: registry,
		config:       cfg,
	}
}

func (a *BaseAgent) Name() string {
	return a.name
}

func (a *BaseAgent) Run(ctx context.Context, input *Input) (*Output, error) {
	// Initialize message list
	messages := make([]llm.Message, 0, len(input.Messages)+1)

	// Add system prompt
	if a.systemPrompt != "" {
		messages = append(messages, llm.Message{
			Role:    llm.RoleSystem,
			Content: a.systemPrompt,
		})
	}

	// Add history messages
	messages = append(messages, input.Messages...)

	// Add user task
	if input.Task != "" {
		messages = append(messages, llm.Message{
			Role:      llm.RoleUser,
			Content:   input.Task,
			Timestamp: time.Now(),
		})
	}

	maxTurns := input.MaxTurns
	if maxTurns == 0 {
		maxTurns = a.config.MaxTurns
	}

	allToolCalls := make([]*tool.CallResult, 0)

	// Agent run loop
	for turn := 0; turn < maxTurns; turn++ {
		// Call LLM
		resp, err := a.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    messages,
			Tools:       a.toolRegistry.GetToolDefinitions(),
			Temperature: input.Temperature,
			MaxTokens:   a.config.MaxTokens,
		})
		if err != nil {
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		// Add assistant message
		messages = append(messages, resp.Message)

		// Check if done
		if resp.StopReason == llm.StopReasonStop {
			return &Output{
				Messages:  messages,
				Result:    resp.Message.Content,
				ToolCalls: allToolCalls,
			}, nil
		}

		// Handle tool calls
		if resp.StopReason == llm.StopReasonToolCalls {
			toolResults, err := a.executeTools(ctx, resp.Message.ToolCalls)
			if err != nil {
				return nil, fmt.Errorf("tool execution failed: %w", err)
			}

			allToolCalls = append(allToolCalls, toolResults...)

			// Add tool result messages
			for _, tr := range toolResults {
				messages = append(messages, llm.Message{
					Role:       llm.RoleTool,
					ToolCallID: tr.CallID,
					Content:    tr.Result.Output,
					Name:       tr.ToolName,
					Timestamp:  tr.EndTime,
				})
			}

			continue
		}

		// If stopped due to length limit
		if resp.StopReason == llm.StopReasonLength {
			return &Output{
				Messages:  messages,
				Result:    resp.Message.Content + "\n[Response truncated due to length limit]",
				ToolCalls: allToolCalls,
			}, nil
		}
	}

	return nil, fmt.Errorf("max turns (%d) exceeded", maxTurns)
}

func (a *BaseAgent) executeTools(ctx context.Context, toolCalls []*llm.ToolCall) ([]*tool.CallResult, error) {
	results := make([]*tool.CallResult, len(toolCalls))

	for i, tc := range toolCalls {
		startTime := time.Now()

		t, err := a.toolRegistry.Get(tc.Function.Name)
		if err != nil {
			results[i] = &tool.CallResult{
				ToolName: tc.Function.Name,
				CallID:   tc.ID,
				Result: &tool.Result{
					Success: false,
					Error:   fmt.Sprintf("tool not found: %v", err),
				},
				StartTime: startTime,
				EndTime:   time.Now(),
			}
			continue
		}

		result, err := t.Execute(ctx, []byte(tc.Function.Arguments))
		if err != nil {
			results[i] = &tool.CallResult{
				ToolName: tc.Function.Name,
				CallID:   tc.ID,
				Result: &tool.Result{
					Success: false,
					Error:   fmt.Sprintf("execution error: %v", err),
				},
				StartTime: startTime,
				EndTime:   time.Now(),
			}
			continue
		}

		results[i] = &tool.CallResult{
			ToolName:  tc.Function.Name,
			CallID:    tc.ID,
			Params:    []byte(tc.Function.Arguments),
			Result:    result,
			StartTime: startTime,
			EndTime:   time.Now(),
		}
	}

	return results, nil
}
