package openai

import (
	"context"
	"fmt"
	"io"
	"strings"

	"finta/internal/llm"

	openai "github.com/sashabaranov/go-openai"
)

type StreamReader struct {
	stream         *openai.ChatCompletionStream
	accumulatedMsg llm.Message
	toolCallsMap   map[int]*llm.ToolCall // Track tool calls by index
}

func (c *Client) ChatStream(ctx context.Context, req *llm.ChatRequest) (llm.StreamReader, error) {
	messages := c.convertMessages(req.Messages)
	tools := c.convertTools(req.Tools)

	stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		Tools:       tools,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      true,
	})
	if err != nil {
		return nil, err
	}

	return &StreamReader{
		stream: stream,
		accumulatedMsg: llm.Message{
			Role:      llm.RoleAssistant,
			Reason:    "",
			Content:   "",
			ToolCalls: nil,
		},
		toolCallsMap: make(map[int]*llm.ToolCall),
	}, nil
}

func (s *StreamReader) Recv() (*llm.Delta, error) {
	resp, err := s.stream.Recv()
	if err == io.EOF {
		// Stream complete, return final accumulated message
		return &llm.Delta{
			Done: true,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in stream response")
	}

	delta := resp.Choices[0].Delta

	result := &llm.Delta{
		Role:    llm.Role(delta.Role),
		Reason:  delta.ReasoningContent,
		Content: delta.Content,
		Done:    false,
	}

	// Accumulate reason
	if delta.ReasoningContent != "" {
		s.accumulatedMsg.Reason += delta.ReasoningContent
	}

	// Accumulate content
	if delta.Content != "" {
		s.accumulatedMsg.Content += delta.Content
	}

	// Handle tool calls - they come in chunks
	if len(delta.ToolCalls) > 0 {
		result.ToolCalls = make([]*llm.ToolCall, 0)

		for _, tc := range delta.ToolCalls {
			index := 0
			if tc.Index != nil {
				index = *tc.Index
			}

			// Get or create tool call at this index
			toolCall, exists := s.toolCallsMap[index]
			if !exists {
				toolCall = &llm.ToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: &llm.FunctionCall{
						Name:      "",
						Arguments: "",
					},
				}
				s.toolCallsMap[index] = toolCall
			}

			// Accumulate function name
			if tc.Function.Name != "" {
				toolCall.Function.Name += tc.Function.Name
			}

			// Accumulate function arguments
			if tc.Function.Arguments != "" {
				toolCall.Function.Arguments += tc.Function.Arguments
			}

			// Update ID if provided
			if tc.ID != "" {
				toolCall.ID = tc.ID
			}

			result.ToolCalls = append(result.ToolCalls, toolCall)
		}
	}

	// Check if this is the final chunk
	finishReason := resp.Choices[0].FinishReason
	if finishReason == openai.FinishReasonStop ||
		finishReason == openai.FinishReasonToolCalls ||
		finishReason == openai.FinishReasonLength {
		// Finalize accumulated tool calls
		if len(s.toolCallsMap) > 0 {
			s.accumulatedMsg.ToolCalls = make([]*llm.ToolCall, 0, len(s.toolCallsMap))
			// Sort by index
			for i := 0; i < len(s.toolCallsMap); i++ {
				if tc, ok := s.toolCallsMap[i]; ok {
					s.accumulatedMsg.ToolCalls = append(s.accumulatedMsg.ToolCalls, tc)
				}
			}
		}
	}

	return result, nil
}

func (s *StreamReader) Close() error {
	s.stream.Close()
	return nil
}

// GetAccumulatedMessage returns the fully accumulated message
// This should be called after the stream is complete
func (s *StreamReader) GetAccumulatedMessage() llm.Message {
	return s.accumulatedMsg
}

// StreamToString is a helper that accumulates all content chunks into a single string
func StreamToString(ctx context.Context, reader llm.StreamReader) (string, error) {
	defer reader.Close()

	var builder strings.Builder

	for {
		delta, err := reader.Recv()
		if err != nil {
			return "", err
		}

		if delta.Done {
			break
		}

		if delta.Reason != "" {
			builder.WriteString(delta.Reason)
		}

		if delta.Content != "" {
			builder.WriteString(delta.Content)
		}
	}

	return builder.String(), nil
}

// StreamToChannel sends content deltas to a channel
func StreamToChannel(ctx context.Context, reader llm.StreamReader, ch chan<- string) error {
	defer reader.Close()
	defer close(ch)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		delta, err := reader.Recv()
		if err != nil {
			return err
		}

		if delta.Done {
			break
		}

		if delta.Content != "" {
			select {
			case ch <- delta.Content:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}
