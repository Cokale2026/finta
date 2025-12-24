package main

import (
	"context"
	"fmt"
	"os"

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

	// Create LLM client
	log.Debug("Creating LLM client (model: %s)", model)
	llmClient := openai.NewClient(apiKey, model, apiBaseURL)

	// Create tool registry
	log.Debug("Registering built-in tools")
	registry := tool.NewRegistry()
	registry.Register(builtin.NewReadTool())
	registry.Register(builtin.NewBashTool())

	log.Info("Registered %d tools: read, bash", 2)

	// Create Agent
	systemPrompt := `You are a helpful AI assistant with access to tools.
You can read files and execute bash commands.
Always provide clear, concise responses.`

	log.Debug("Agent created with max_turns=%d, temperature=%.2f", maxTurns, temperature)

	ag := agent.NewBaseAgent("general", systemPrompt, llmClient, registry, &agent.Config{
		Model:       model,
		Temperature: temperature,
		MaxTurns:    maxTurns,
	})

	// Run Agent (pass Logger to agent)
	_, err := ag.Run(context.Background(), &agent.Input{
		Task:        task,
		Temperature: temperature,
		Logger:      log,
	})
	if err != nil {
		log.Error("Agent execution failed: %v", err)
		return err
	}

	log.Debug("Agent completed successfully")

	return nil
}
