package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"

	"finta/internal/agent"
	"finta/internal/config"
	"finta/internal/hook"
	"finta/internal/hook/handlers"
	"finta/internal/llm"
	"finta/internal/llm/openai"
	"finta/internal/logger"
	"finta/internal/mcp"
	"finta/internal/tool"
	"finta/internal/tool/builtin"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

var (
	apiBaseURL  string
	apiKey      string
	model       string
	temperature float32
	maxTurns    int
	verbose     bool
	noColor     bool
	streaming   bool
	parallel    bool
	agentType   string
	configPath  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "finta",
		Short: "Finta AI Agent Framework",
		Long:  "A flexible AI agent framework inspired by ClaudeCode",
	}

	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Start interactive chat with an AI agent",
		Args:  cobra.NoArgs,
		RunE:  runChat,
	}

	chatCmd.Flags().StringVar(&apiBaseURL, "api-base-url", os.Getenv("OPENAI_API_BASE_URL"), "OpenAI API base URL")
	chatCmd.Flags().StringVar(&apiKey, "api-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key")
	chatCmd.Flags().StringVar(&model, "model", "gpt-4-turbo", "Model to use")
	chatCmd.Flags().Float32Var(&temperature, "temperature", 0.7, "Temperature")
	chatCmd.Flags().IntVar(&maxTurns, "max-turns", 10, "Maximum conversation turns")
	chatCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output (debug mode)")
	chatCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	chatCmd.Flags().BoolVar(&streaming, "streaming", false, "Enable streaming output")
	chatCmd.Flags().BoolVar(&parallel, "parallel", true, "Enable parallel tool execution (default: true)")
	chatCmd.Flags().StringVar(&agentType, "agent-type", "general", "Agent type to use (general, explore, plan, execute)")
	chatCmd.Flags().StringVar(&configPath, "config", "", "Path to config file (default: auto-detect)")

	rootCmd.AddCommand(chatCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runChat(cmd *cobra.Command, _ []string) error {
	if apiKey == "" {
		return fmt.Errorf("OpenAI API key required (set OPENAI_API_KEY or use --api-key)")
	}

	// Create Logger
	logLevel := logger.LevelInfo
	if verbose {
		logLevel = logger.LevelDebug
	}
	log := logger.NewLogger(os.Stdout, logLevel)
	if noColor {
		log.SetColorMode(false)
	}

	// Print configuration with masked sensitive data
	log.Info("Configuration:")
	log.Info("  Model: %s", model)
	log.Info("  Agent Type: %s", agentType)
	log.Info("  Temperature: %.2f", temperature)
	log.Info("  Max Turns: %d", maxTurns)
	log.Info("  Parallel: %v", parallel)
	log.Info("  Streaming: %v", streaming)
	log.Info("  Verbose: %v", verbose)
	log.Info("  API Key: %s", maskAPIKey(apiKey))
	if apiBaseURL != "" {
		log.Info("  API Base URL: %s", apiBaseURL)
	}
	log.Info("")

	// Create LLM client
	log.Debug("Creating LLM client (model: %s)", model)
	llmClient := openai.NewClient(apiKey, model, apiBaseURL)

	// Create tool registry
	log.Debug("Registering built-in tools")
	registry := tool.NewRegistry()
	registry.Register(builtin.NewReadTool())
	registry.Register(builtin.NewBashTool())
	registry.Register(builtin.NewWriteTool())
	registry.Register(builtin.NewGlobTool())
	registry.Register(builtin.NewGrepTool())
	registry.Register(builtin.NewTodoWriteTool())

	builtinToolCount := 6

	// Load MCP configuration
	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Info("Warning: Failed to load config: %v (continuing without MCP servers)", err)
			cfg = &config.Config{}
		}
	} else {
		var err error
		cfg, err = config.LoadWithDefaults()
		if err != nil {
			log.Debug("No config file found (continuing without MCP servers)")
			cfg = &config.Config{}
		}
	}

	// Initialize MCP manager
	mcpManager := mcp.NewManager(registry)
	mcpToolCount := 0

	if len(cfg.MCP.Servers) > 0 {
		log.Info("Initializing MCP servers...")
		if err := mcpManager.Initialize(context.Background(), cfg.MCP); err != nil {
			log.Info("Warning: MCP initialization had errors: %v", err)
		}

		servers := mcpManager.ListServers()
		if len(servers) > 0 {
			log.Info("Loaded %d MCP servers: %v", len(servers), servers)

			// Count MCP tools
			allTools := registry.List()
			mcpToolCount = len(allTools) - builtinToolCount
		}
	}

	// Ensure cleanup on exit
	defer mcpManager.Close()

	// Create agent factory
	factory := agent.NewDefaultFactory(llmClient, registry)

	// Register Task tool with factory
	taskTool := builtin.NewTaskTool(factory)
	registry.Register(taskTool)

	totalTools := builtinToolCount + 1 + mcpToolCount // built-in + task + MCP
	if mcpToolCount > 0 {
		log.Info("Registered %d tools: %d built-in (read, bash, write, glob, grep, TodoWrite, task) + %d MCP tools", totalTools, builtinToolCount+1, mcpToolCount)
	} else {
		log.Info("Registered %d tools: read, bash, write, glob, grep, TodoWrite, task", builtinToolCount+1)
	}

	// Create agent based on type
	var ag agent.Agent
	var err error
	ag, err = factory.CreateAgent(agent.AgentType(agentType))
	if err != nil {
		log.Error("Failed to create agent: %v", err)
		return err
	}

	log.Debug("Created %s agent with max_turns=%d, temperature=%.2f, parallel=%v", agentType, maxTurns, temperature, parallel)

	// Initialize hook manager based on configuration
	hookManager := hook.NewManager()

	if cfg.Hooks.BashConfirm {
		hookManager.Register(handlers.NewBashConfirmHandler())
		log.Info("Hooks: bash command confirmation enabled")
	}

	if len(cfg.Hooks.ToolConfirm) > 0 {
		hookManager.Register(handlers.NewToolConfirmHandler(cfg.Hooks.ToolConfirm...))
		log.Info("Hooks: tool confirmation enabled for: %v", cfg.Hooks.ToolConfirm)
	}

	// Set hook manager on agent if it supports it
	if baseAgent, ok := ag.(*agent.BaseAgent); ok {
		baseAgent.SetHookManager(hookManager)
	}

	// Setup context with signal handling for Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		fmt.Println("\nExiting...")
		cancel()
	}()

	// Message history for continuous conversation
	var history []llm.Message

	// Helper function to run a single task
	runTask := func(task string) error {
		input := &agent.Input{
			Task:     task,
			Messages: history,
			Logger:   log,
		}

		// Only override temperature if explicitly set by user
		if cmd.Flags().Changed("temperature") {
			input.Temperature = temperature
		}

		// Only override max turns if explicitly set by user
		if cmd.Flags().Changed("max-turns") {
			input.MaxTurns = maxTurns
		}

		var output *agent.Output
		var err error

		if streaming {
			streamChan := make(chan string, 100)
			var streamedContent strings.Builder
			var lineCount int
			var mu sync.Mutex
			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				defer wg.Done()
				for content := range streamChan {
					fmt.Print(content)
					mu.Lock()
					streamedContent.WriteString(content)
					lineCount += strings.Count(content, "\n")
					mu.Unlock()
				}
			}()

			input.EnableStreaming = true
			output, err = ag.RunStreaming(ctx, input, streamChan)

			// Wait for streaming goroutine to finish
			wg.Wait()

			// Clear streamed content and re-render with markdown
			if err == nil && streamedContent.Len() > 0 {
				mu.Lock()
				content := streamedContent.String()
				lines := lineCount
				mu.Unlock()

				// Move cursor up and clear (add 1 for the line without newline at end)
				if lines > 0 || len(content) > 0 {
					fmt.Printf("\033[%dA", lines+1) // Move up
					fmt.Print("\033[J")              // Clear to end of screen
				}

				// Re-render with markdown formatting
				log.AgentResponse(content)
			}
		} else {
			output, err = ag.Run(ctx, input)
		}

		if err != nil {
			return err
		}

		// Update history (filter out system messages as agent adds them automatically)
		history = filterSystemMessages(output.Messages)
		return nil
	}

	// Interactive loop with readline support
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     "", // No history file
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		log.Error("Failed to initialize readline: %v", err)
		return err
	}
	defer rl.Close()

	for {
		// Check if context was cancelled
		if ctx.Err() != nil {
			break
		}

		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue // Ctrl+C clears line, continue
			}
			if err == io.EOF {
				break // Ctrl+D exits
			}
			break
		}

		task := strings.TrimSpace(line)
		if task == "" {
			continue
		}

		if err := runTask(task); err != nil {
			if ctx.Err() != nil {
				break // Graceful exit on Ctrl+C
			}
			log.Error("Error: %v", err)
			continue
		}
	}

	log.Debug("Session ended")
	return nil
}

// filterSystemMessages removes system messages from history
// since the agent automatically adds system prompt
func filterSystemMessages(messages []llm.Message) []llm.Message {
	filtered := make([]llm.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.Role != llm.RoleSystem {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// maskAPIKey masks the API key for logging, showing only first 8 and last 4 characters
func maskAPIKey(key string) string {
	if len(key) <= 12 {
		return "***" // Too short to safely show any part
	}
	return key[:8] + strings.Repeat("*", len(key)-12) + key[len(key)-4:]
}
