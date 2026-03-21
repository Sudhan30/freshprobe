package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sudhan30/freshprobe/internal/httpserver"
	"github.com/Sudhan30/freshprobe/internal/mcpserver"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var (
		mode string
		addr string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start freshprobe as a server (MCP or HTTP)",
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := buildDeps()
			if err != nil {
				return fmt.Errorf("init: %w", err)
			}
			defer deps.Store.Close()

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			switch mode {
			case "mcp":
				return mcpserver.Run(ctx, deps.Engine, deps.Store, deps.PolicyLoader)
			case "http":
				fmt.Fprintf(os.Stderr, "freshprobe HTTP server listening on %s\n", addr)
				return httpserver.Run(ctx, deps.Engine, deps.Store, deps.PolicyLoader, addr)
			default:
				return fmt.Errorf("unknown mode %q, use 'mcp' or 'http'", mode)
			}
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "mcp", "Server mode: mcp or http")
	cmd.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")

	return cmd
}
