package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"finta/internal/tool"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPToolAdapter adapts an MCP tool to Finta's Tool interface
type MCPToolAdapter struct {
	client         *Client
	mcpTool        *mcp.Tool
	namespacedName string // e.g., "filesystem_read_file"
}

// NewMCPToolAdapter creates an adapter for an MCP tool
func NewMCPToolAdapter(client *Client, mcpTool *mcp.Tool) *MCPToolAdapter {
	return &MCPToolAdapter{
		client:         client,
		mcpTool:        mcpTool,
		namespacedName: fmt.Sprintf("%s_%s", client.Name(), mcpTool.Name),
	}
}

// Name returns the namespaced tool name (server_tool)
func (a *MCPToolAdapter) Name() string {
	return a.namespacedName
}

// Description returns the MCP tool description
func (a *MCPToolAdapter) Description() string {
	desc := a.mcpTool.Description
	if desc == "" {
		desc = fmt.Sprintf("MCP tool from %s server", a.client.Name())
	}

	// Add source information
	return fmt.Sprintf("%s\n\n[MCP Server: %s]", desc, a.client.Name())
}

// BestPractices returns empty (MCP tools don't have this concept)
func (a *MCPToolAdapter) BestPractices() string {
	return ""
}

// Parameters returns the MCP tool's input schema
func (a *MCPToolAdapter) Parameters() map[string]any {
	// The InputSchema is of type `any` in the SDK
	// We need to type assert it to map[string]any
	if a.mcpTool.InputSchema == nil {
		// Return empty object schema if no schema provided
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	// Try to convert to map[string]any
	if schema, ok := a.mcpTool.InputSchema.(map[string]any); ok {
		return schema
	}

	// If it's not a map, try to marshal/unmarshal to convert
	schemaBytes, err := json.Marshal(a.mcpTool.InputSchema)
	if err != nil {
		// Return empty schema on error
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	return schema
}

// Execute calls the MCP server to execute the tool
func (a *MCPToolAdapter) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
	// Unmarshal params to map[string]interface{}
	var args map[string]interface{}
	if err := json.Unmarshal(params, &args); err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	// Call MCP server via client.CallTool()
	result, err := a.client.CallTool(ctx, a.mcpTool.Name, args)
	if err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("MCP tool execution failed: %v", err),
		}, nil
	}

	// Convert MCP result to Finta result
	if result.IsError {
		return &tool.Result{
			Success: false,
			Error:   formatMCPError(result),
		}, nil
	}

	return &tool.Result{
		Success: true,
		Output:  formatMCPContent(result.Content),
		Data: map[string]any{
			"mcp_server": a.client.Name(),
			"mcp_tool":   a.mcpTool.Name,
		},
	}, nil
}

// formatMCPContent converts MCP content array to string
func formatMCPContent(content []mcp.Content) string {
	var parts []string

	for _, item := range content {
		// Use type assertion to check content type
		switch c := item.(type) {
		case *mcp.TextContent:
			parts = append(parts, c.Text)

		case *mcp.ImageContent:
			parts = append(parts, fmt.Sprintf("[Image: %s]", c.MIMEType))

		case *mcp.AudioContent:
			parts = append(parts, fmt.Sprintf("[Audio: %s]", c.MIMEType))

		default:
			// Unknown content type - try to marshal to JSON
			data, err := json.Marshal(item)
			if err != nil {
				parts = append(parts, fmt.Sprintf("[Unknown content type: %T]", item))
			} else {
				parts = append(parts, string(data))
			}
		}
	}

	return strings.Join(parts, "\n")
}

// formatMCPError extracts error message from MCP result
func formatMCPError(result *mcp.CallToolResult) string {
	// Try to extract error message from content
	if len(result.Content) > 0 {
		return formatMCPContent(result.Content)
	}

	return "MCP tool returned an error"
}
