package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newHistoryCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "history <url>",
		Short: "Show past probe results for an endpoint",
		Long:  "Query the local SQLite store for historical probe verdicts. Requires persistence mode (not --stateless).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := buildDeps()
			if err != nil {
				return fmt.Errorf("init: %w", err)
			}
			defer deps.Store.Close()

			ctx := context.Background()
			results, err := deps.Store.GetHistory(ctx, args[0], limit)
			if err != nil {
				return fmt.Errorf("get history: %w", err)
			}

			if len(results) == 0 {
				fmt.Fprintln(os.Stderr, "No probe history found for", args[0])
				return nil
			}

			if cfgOutput == "text" {
				fmt.Printf("%-20s  %-7s  %-6s  %-6s  %-8s\n", "TIMESTAMP", "VERDICT", "CONF", "SCORE", "P95(ms)")
				for _, v := range results {
					score := 0.0
					if v.Freshness != nil {
						score = v.Freshness.FreshnessScore
					}
					var p95 int64
					if v.Liveness != nil {
						p95 = v.Liveness.P95Ms
					}
					fmt.Printf("%-20s  %-7s  %-6.2f  %-6.2f  %-8d\n",
						v.ProbeMetadata.Timestamp.Format("2006-01-02 15:04:05"),
						v.Verdict,
						v.Confidence,
						score,
						p95,
					)
				}
				return nil
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(results)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Max number of results to show")

	return cmd
}
