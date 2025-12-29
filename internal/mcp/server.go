package mcp

import (
	"context"
	"fmt"

	"finta/internal/config"
)

// Server represents a running MCP server instance
type Server struct {
	config config.MCPServerConfig
	client *Client
}

// NewServer creates and starts an MCP server
func NewServer(ctx context.Context, cfg config.MCPServerConfig) (*Server, error) {
	// Expand environment variables in the config
	expandedEnv := config.ExpandEnvMap(cfg.Env)

	// Create and initialize MCP client
	// The client now handles transport internally
	client, err := NewClient(ctx, cfg.Name, cfg.Command, cfg.Args, expandedEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	return &Server{
		config: cfg,
		client: client,
	}, nil
}

// Name returns the server name
func (s *Server) Name() string {
	return s.config.Name
}

// Client returns the MCP client
func (s *Server) Client() *Client {
	return s.client
}

// Close shuts down the server
func (s *Server) Close() error {
	return s.client.Close()
}

// Health checks if the server is still responsive
func (s *Server) Health(ctx context.Context) error {
	// Check if we still have tools (simple health check)
	if len(s.client.Tools()) == 0 {
		return fmt.Errorf("server has no tools available")
	}
	return nil
}
