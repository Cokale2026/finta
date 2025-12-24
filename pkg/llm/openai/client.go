package openai

import (
	"context"
	"finta/pkg/llm"

	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	client *openai.Client
	model  string
}

// NewClient creates a new OpenAI client with the given API key and model.
// If baseURL is empty, it uses the default OpenAI API endpoint.
// If baseURL is provided, it uses the custom endpoint (useful for OpenAI-compatible APIs).
func NewClient(apiKey, model string, baseURL ...string) *Client {
	var client *openai.Client

	if len(baseURL) > 0 && baseURL[0] != "" {
		// Use custom base URL
		config := openai.DefaultConfig(apiKey)
		config.BaseURL = baseURL[0]
		client = openai.NewClientWithConfig(config)
	} else {
		// Use default OpenAI endpoint
		client = openai.NewClient(apiKey)
	}

	return &Client{
		client: client,
		model:  model,
	}
}

func (c *Client) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	// Convert message format
	messages := c.convertMessages(req.Messages)

	// Convert tool definitions
	tools := c.convertTools(req.Tools)

	// Call OpenAI API
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		Tools:       tools,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
	if err != nil {
		return nil, err
	}

	// Convert response
	return c.convertResponse(resp), nil
}

func (c *Client) ChatStream(ctx context.Context, req *llm.ChatRequest) (llm.StreamReader, error) {
	// Streaming implementation will be added in Phase 2
	return nil, nil
}

func (c *Client) Provider() string {
	return "openai"
}

func (c *Client) Model() string {
	return c.model
}

// Helper method: message format conversion
func (c *Client) convertMessages(msgs []llm.Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, len(msgs))
	for i, msg := range msgs {
		ocMsg := openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}

		// Convert tool calls
		if len(msg.ToolCalls) > 0 {
			ocMsg.ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				ocMsg.ToolCalls[j] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		// Tool response message
		if msg.Role == llm.RoleTool {
			ocMsg.ToolCallID = msg.ToolCallID
		}

		result[i] = ocMsg
	}
	return result
}

// Helper method: tool definition conversion
func (c *Client) convertTools(tools []*llm.ToolDefinition) []openai.Tool {
	result := make([]openai.Tool, len(tools))
	for i, t := range tools {
		result[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		}
	}
	return result
}

// Helper method: response conversion
func (c *Client) convertResponse(resp openai.ChatCompletionResponse) *llm.ChatResponse {
	choice := resp.Choices[0]
	msg := choice.Message

	result := &llm.ChatResponse{
		Message: llm.Message{
			Role:    llm.Role(msg.Role),
			Content: msg.Content,
		},
		Usage: llm.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// Convert tool calls
	if len(msg.ToolCalls) > 0 {
		result.Message.ToolCalls = make([]*llm.ToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			result.Message.ToolCalls[i] = &llm.ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: &llm.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
		result.StopReason = llm.StopReasonToolCalls
	} else {
		result.StopReason = llm.StopReason(choice.FinishReason)
	}

	return result
}
