package transport

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

const (
	// shutdownTimeout is how long to wait for graceful shutdown
	shutdownTimeout = 5 * time.Second
)

// StdioTransport implements Transport via stdin/stdout of a child process
type StdioTransport struct {
	ctx     context.Context
	cancel  context.CancelFunc
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
}

// NewStdioTransport creates a new stdio transport
// command: executable name (e.g., "npx", "node", "/path/to/server")
// args: command arguments
// env: additional environment variables (merged with os.Environ())
func NewStdioTransport(ctx context.Context, command string, args []string, env map[string]string) (*StdioTransport, error) {
	// Create cancellable context for process management
	ctx, cancel := context.WithCancel(ctx)

	// Create command
	cmd := exec.CommandContext(ctx, command, args...)

	// Merge environment variables
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Get stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Get stderr pipe for debugging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	return &StdioTransport{
		ctx:    ctx,
		cancel: cancel,
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}, nil
}

// Start initiates the transport by starting the child process
func (t *StdioTransport) Start() error {
	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// TODO: Optionally capture and log stderr in background
	// For now, stderr is available but not automatically logged

	return nil
}

// Reader returns the input stream from the MCP server (stdout of child process)
func (t *StdioTransport) Reader() io.ReadCloser {
	return t.stdout
}

// Writer returns the output stream to the MCP server (stdin of child process)
func (t *StdioTransport) Writer() io.WriteCloser {
	return t.stdin
}

// Close terminates the transport and cleans up resources
// Implements graceful shutdown with timeout, then force kill
func (t *StdioTransport) Close() error {
	// Cancel context to signal shutdown
	t.cancel()

	// Close stdin to signal EOF to child process
	if t.stdin != nil {
		t.stdin.Close()
	}

	// Wait for process to exit gracefully (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- t.cmd.Wait()
	}()

	select {
	case err := <-done:
		// Process exited gracefully
		if err != nil && err.Error() != "signal: killed" {
			return fmt.Errorf("process exited with error: %w", err)
		}
		return nil

	case <-time.After(shutdownTimeout):
		// Timeout - force kill the process
		if t.cmd.Process != nil {
			if err := t.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
		// Wait for kill to complete
		<-done
		return nil
	}
}
