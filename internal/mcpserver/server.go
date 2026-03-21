package mcpserver

import (
	"context"

	"github.com/Sudhan30/freshprobe/internal/policy"
	"github.com/Sudhan30/freshprobe/internal/probe"
	"github.com/Sudhan30/freshprobe/internal/store"
	"github.com/Sudhan30/freshprobe/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Run starts the MCP server on stdio transport.
func Run(ctx context.Context, engine *probe.Engine, st store.Store, loader *policy.Loader) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "freshprobe",
		Version: version.Version,
	}, nil)

	registerTools(server, engine, st, loader)

	return server.Run(ctx, &mcp.StdioTransport{})
}
