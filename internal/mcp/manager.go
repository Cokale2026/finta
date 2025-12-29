package mcp

import (
	"context"
	"fmt"
	"sync"

	"finta/internal/config"
	"finta/internal/tool"
)

// Manager coordinates multiple MCP servers
type Manager struct {
	servers  map[string]*Server
	registry *tool.Registry
	mu       sync.RWMutex
}

// NewManager creates a new MCP manager
func NewManager(registry *tool.Registry) *Manager {
	return &Manager{
		servers:  make(map[string]*Server),
		registry: registry,
	}
}

// Initialize starts all MCP servers from config
func (m *Manager) Initialize(ctx context.Context, cfg config.MCPConfig) error {
	if len(cfg.Servers) == 0 {
		return nil // No servers to initialize
	}

	// Validate no duplicate server names
	names := make(map[string]bool)
	for _, serverCfg := range cfg.Servers {
		if serverCfg.Disabled {
			continue
		}
		if names[serverCfg.Name] {
			return fmt.Errorf("duplicate server name: %s", serverCfg.Name)
		}
		names[serverCfg.Name] = true
	}

	// Start servers concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, len(cfg.Servers))
	successChan := make(chan string, len(cfg.Servers))

	for _, serverCfg := range cfg.Servers {
		if serverCfg.Disabled {
			continue
		}

		wg.Add(1)
		go func(cfg config.MCPServerConfig) {
			defer wg.Done()
			if err := m.startServer(ctx, cfg); err != nil {
				errChan <- fmt.Errorf("server %s: %w", cfg.Name, err)
			} else {
				successChan <- cfg.Name
			}
		}(serverCfg)
	}

	wg.Wait()
	close(errChan)
	close(successChan)

	// Collect results
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	var successNames []string
	for name := range successChan {
		successNames = append(successNames, name)
	}

	// Return error if ALL servers failed
	if len(errs) > 0 && len(successNames) == 0 {
		return fmt.Errorf("all MCP servers failed to initialize: %v", errs)
	}

	// If some servers failed but some succeeded, just log warnings
	// (caller should log these warnings)
	if len(errs) > 0 {
		// Partial failure is acceptable - we'll work with available servers
		return fmt.Errorf("some MCP servers failed (loaded %d/%d): %v", len(successNames), len(successNames)+len(errs), errs)
	}

	return nil
}

// startServer initializes a single MCP server
func (m *Manager) startServer(ctx context.Context, serverCfg config.MCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create server instance
	server, err := NewServer(ctx, serverCfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Get tools from server
	tools := server.Client().Tools()

	// Create adapters for each tool and register them
	for _, mcpTool := range tools {
		adapter := NewMCPToolAdapter(server.Client(), mcpTool)

		if err := m.registry.Register(adapter); err != nil {
			// If tool registration fails, close the server and return error
			server.Close()
			return fmt.Errorf("failed to register tool %s: %w", adapter.Name(), err)
		}
	}

	// Store server
	m.servers[serverCfg.Name] = server

	return nil
}

// Close shuts down all MCP servers
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	// Close all servers concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, len(m.servers))

	for name, server := range m.servers {
		wg.Add(1)
		go func(name string, s *Server) {
			defer wg.Done()
			if err := s.Close(); err != nil {
				errChan <- fmt.Errorf("server %s: %w", name, err)
			}
		}(name, server)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	for err := range errChan {
		errs = append(errs, err)
	}

	// Clear servers map
	m.servers = make(map[string]*Server)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing servers: %v", errs)
	}

	return nil
}

// GetServer returns a server by name
func (m *Manager) GetServer(name string) (*Server, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, ok := m.servers[name]
	return server, ok
}

// ListServers returns all active server names
func (m *Manager) ListServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	return names
}

// ServerCount returns the number of active servers
func (m *Manager) ServerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.servers)
}
