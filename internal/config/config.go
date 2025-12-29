package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the complete Finta configuration
type Config struct {
	MCP   MCPConfig   `yaml:"mcp"`
	Hooks HooksConfig `yaml:"hooks"`
}

// HooksConfig contains hook-related settings
type HooksConfig struct {
	// BashConfirm enables user confirmation before bash commands
	BashConfirm bool `yaml:"bash_confirm"`
	// ToolConfirm enables user confirmation before specified tools
	ToolConfirm []string `yaml:"tool_confirm"`
}

// MCPConfig contains MCP-specific settings
type MCPConfig struct {
	Servers []MCPServerConfig `yaml:"servers"`
}

// MCPServerConfig defines a single MCP server
type MCPServerConfig struct {
	Name      string            `yaml:"name"`      // Unique server identifier
	Transport string            `yaml:"transport"` // "stdio" (only supported initially)
	Command   string            `yaml:"command"`   // Executable to run
	Args      []string          `yaml:"args"`      // Command arguments
	Env       map[string]string `yaml:"env"`       // Environment variables with ${VAR} support
	Disabled  bool              `yaml:"disabled"`  // Skip this server if true
}

// Load reads and parses the YAML config file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// LoadWithDefaults loads config with fallback to default locations
// Checks: ./finta.yaml, ~/config/finta/finta.yaml, /etc/finta/finta.yaml
func LoadWithDefaults() (*Config, error) {
	// Try config locations in order
	locations := []string{
		"./finta.yaml",
		"./configs/finta.yaml",
	}

	// Add user config directory if available
	if home, err := os.UserHomeDir(); err == nil {
		locations = append(locations, filepath.Join(home, ".config", "finta", "finta.yaml"))
	}

	// Add system-wide config
	locations = append(locations, "/etc/finta/finta.yaml")

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return Load(loc)
		}
	}

	// No config found - return empty config (not an error)
	return &Config{}, nil
}

// Validate checks config correctness
func (c *Config) Validate() error {
	if len(c.MCP.Servers) == 0 {
		// Empty config is valid
		return nil
	}

	// Check for duplicate server names
	names := make(map[string]bool)
	for i, server := range c.MCP.Servers {
		if server.Name == "" {
			return fmt.Errorf("server #%d: name cannot be empty", i+1)
		}

		if names[server.Name] {
			return fmt.Errorf("duplicate server name: %s", server.Name)
		}
		names[server.Name] = true

		// Validate server config
		if err := server.Validate(); err != nil {
			return fmt.Errorf("server %s: %w", server.Name, err)
		}
	}

	return nil
}

// Validate checks a single server config
func (s *MCPServerConfig) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Validate server name matches OpenAI tool name requirements
	// Pattern: ^[a-zA-Z0-9_-]+$
	for _, ch := range s.Name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return fmt.Errorf("server name '%s' contains invalid character '%c' (only alphanumeric, underscore, and hyphen allowed)", s.Name, ch)
		}
	}

	if s.Transport == "" {
		return fmt.Errorf("transport is required")
	}

	if s.Transport != "stdio" {
		return fmt.Errorf("unsupported transport: %s (only 'stdio' is supported)", s.Transport)
	}

	if s.Command == "" {
		return fmt.Errorf("command is required")
	}

	return nil
}
