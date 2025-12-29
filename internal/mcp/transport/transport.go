package transport

import "io"

// Transport defines the interface for MCP communication
type Transport interface {
	// Reader returns the input stream from the MCP server
	Reader() io.ReadCloser

	// Writer returns the output stream to the MCP server
	Writer() io.WriteCloser

	// Close terminates the transport and cleans up resources
	Close() error

	// Start initiates the transport (spawns process for stdio)
	Start() error
}
