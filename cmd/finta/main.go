package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"finta/internal/agent"
	"finta/internal/llm/openai"
	"finta/internal/logger"
	"finta/internal/tool"
	"finta/internal/tool/builtin"

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
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "finta",
		Short: "Finta AI Agent Framework",
		Long:  "A flexible AI agent framework inspired by ClaudeCode",
	}

	chatCmd := &cobra.Command{
		Use:   "chat [task]",
		Short: "Chat with an AI agent",
		Args:  cobra.MinimumNArgs(1),
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

	rootCmd.AddCommand(chatCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runChat(cmd *cobra.Command, args []string) error {
	if apiKey == "" {
		return fmt.Errorf("OpenAI API key required (set OPENAI_API_KEY or use --api-key)")
	}

	task := args[0]

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
	log.Info("  Task: %s", task)
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

	// Create agent factory
	factory := agent.NewDefaultFactory(llmClient, registry)

	// Register Task tool with factory
	taskTool := builtin.NewTaskTool(factory)
	registry.Register(taskTool)

	log.Info("Registered %d tools: read, bash, write, glob, grep, TodoWrite, task", 7)

	// Create agent based on type
	var ag agent.Agent
	var err error
	ag, err = factory.CreateAgent(agent.AgentType(agentType))
	if err != nil {
		log.Error("Failed to create agent: %v", err)
		return err
	}

	log.Debug("Created %s agent with max_turns=%d, temperature=%.2f, parallel=%v", agentType, maxTurns, temperature, parallel)

	// Build input - only override defaults if flags were explicitly set
	input := &agent.Input{
		Task:   task,
		Logger: log,
	}

	// Only override temperature if explicitly set by user
	if cmd.Flags().Changed("temperature") {
		input.Temperature = temperature
		log.Debug("Overriding agent temperature with CLI value: %.2f", temperature)
	}

	// Only override max turns if explicitly set by user
	if cmd.Flags().Changed("max-turns") {
		input.MaxTurns = maxTurns
		log.Debug("Overriding agent max turns with CLI value: %d", maxTurns)
	}

	// Run Agent (pass Logger to agent)
	if streaming {
		log.Info("Running in streaming mode")
		streamChan := make(chan string, 100)

		// Start goroutine to print streamed content
		go func() {
			for content := range streamChan {
				fmt.Print(content)
			}
		}()

		input.EnableStreaming = true
		_, err := ag.RunStreaming(context.Background(), input, streamChan)
		if err != nil {
			log.Error("Agent execution failed: %v", err)
			return err
		}
	} else {
		_, err := ag.Run(context.Background(), input)
		if err != nil {
			log.Error("Agent execution failed: %v", err)
			return err
		}
	}

	log.Debug("Agent completed successfully")

	return nil
}

// maskAPIKey masks the API key for logging, showing only first 8 and last 4 characters
func maskAPIKey(key string) string {
	if len(key) <= 12 {
		return "***" // Too short to safely show any part
	}
	return key[:8] + strings.Repeat("*", len(key)-12) + key[len(key)-4:]
}
