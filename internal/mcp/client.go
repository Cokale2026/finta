package mcp

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client wraps the official MCP SDK client and session
type Client struct {
	name    string
	client  *mcp.Client
	session *mcp.ClientSession
	tools   []*mcp.Tool
}

// NewClient creates and initializes a new MCP client
func NewClient(ctx context.Context, name string, command string, args []string, env map[string]string) (*Client, error) {
	// Create command for MCP server
	cmd := exec.Command(command, args...)

	// Set environment variables
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), formatEnvVars(env)...)
	}

	// Create MCP client
	impl := &mcp.Implementation{
		Name:    "finta",
		Version: "1.0.0",
	}
	client := mcp.NewClient(impl, nil)

	// Create transport
	transport := &mcp.CommandTransport{Command: cmd}

	// Connect to server
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	// Collect tools from server
	var tools []*mcp.Tool
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			session.Close()
			return nil, fmt.Errorf("failed to list tools: %w", err)
		}
		tools = append(tools, tool)
	}

	return &Client{
		name:    name,
		client:  client,
		session: session,
		tools:   tools,
	}, nil
}

// formatEnvVars converts env map to KEY=VALUE slice
func formatEnvVars(env map[string]string) []string {
	result := make([]string, 0, len(env))
	for key, value := range env {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}
	return result
}

// Name returns the server name
func (c *Client) Name() string {
	return c.name
}

// Tools returns the cached list of tools
func (c *Client) Tools() []*mcp.Tool {
	return c.tools
}

// CallTool executes a tool with given parameters
func (c *Client) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	params := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	}

	result, err := c.session.CallTool(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("call tool request failed: %w", err)
	}

	return result, nil
}

// Close shuts down the client and session
func (c *Client) Close() error {
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}
